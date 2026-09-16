package onlyoffice

// Read-only SQL backend of the unified file client (epic #34, F2 #36).
//
// The goal is to read files and folders straight from the Community Server
// database, without the REST layer. Research on the live portal (VM
// `onlyoffice-v2`) showed the server runs on **MySQL 8.0** (`files_file`,
// `files_folder`, `files_folder_tree`, tenant `tenants_tenants`), not
// PostgreSQL — see docs/community-server-db.md. The store below therefore
// speaks `database/sql` and selects its driver from the DSN, so it works
// against the live MySQL today and against PostgreSQL if the portal is ever
// migrated. Every query is a SELECT; the write methods of FileStore return
// ErrReadOnly.
//
// Downloads follow the portal's S3/MinIO object layout through the shared
// MinIO helper in storage_fallback.go — no HTTP file endpoint is used.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Provider names for the SQL backend. ProviderPG is the value Name reports for
// a PostgreSQL connection and ProviderMySQL for MySQL.
const (
	ProviderPG    = "postgres"
	ProviderMySQL = "mysql"
)

// ErrReadOnly is returned by every FileStore write method of the SQL backend.
var ErrReadOnly = errors.New("onlyoffice: sql file store is read-only")

const (
	pgConnectTimeout = 10 * time.Second
	pgSearchLimit    = 50
	pgSearchMaxLimit = 500
)

// PGConfig configures the read-only SQL store. DSN is a driver DSN:
// `user:pass@tcp(host:port)/onlyoffice?parseTime=true` for MySQL or a
// `postgres://` / libpq keyword string for PostgreSQL. Driver, when set,
// forces the engine ("postgres" or "mysql"); otherwise it is detected from the
// DSN. Tenant filters rows (empty means all tenants).
type PGConfig struct {
	DSN    string
	Driver string
	Tenant string
}

// PGConfigFromEnv reads ONLYOFFICE_DSN (or the ONLYOFFICE_PG_* parts),
// ONLYOFFICE_PG_DRIVER and the tenant from ONLYOFFICE_PG_TENANT /
// ONLYOFFICE_TENANT. The library never loads dotfiles — the CLI does that.
func PGConfigFromEnv() PGConfig {
	dsn := strings.TrimSpace(os.Getenv("ONLYOFFICE_DSN"))
	if dsn == "" {
		dsn = pgDSNFromParts()
	}
	return PGConfig{
		DSN:    dsn,
		Driver: strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_DRIVER")),
		Tenant: firstNonEmpty(os.Getenv("ONLYOFFICE_PG_TENANT"), os.Getenv("ONLYOFFICE_TENANT")),
	}
}

// pgDSNFromParts builds a libpq keyword DSN from ONLYOFFICE_PG_* variables.
// It returns "" unless a host is set, which keeps the MySQL path (ONLYOFFICE_DSN)
// the default.
func pgDSNFromParts() string {
	host := strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_HOST"))
	if host == "" {
		return ""
	}
	port := firstNonEmpty(os.Getenv("ONLYOFFICE_PG_PORT"), "5432")
	dbname := firstNonEmpty(os.Getenv("ONLYOFFICE_PG_DBNAME"), "onlyoffice")
	sslmode := firstNonEmpty(os.Getenv("ONLYOFFICE_PG_SSLMODE"), "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, os.Getenv("ONLYOFFICE_PG_USER"), os.Getenv("ONLYOFFICE_PG_PASSWORD"), dbname, sslmode)
}

// pgStore is a read-only FileStore/Searcher over the Community Server database.
type pgStore struct {
	db        *sql.DB
	driver    string
	tenantID  int64
	hasTenant bool
	http      *http.Client
}

var (
	_ FileStore = (*pgStore)(nil)
	_ Searcher  = (*pgStore)(nil)
)

// NewPGStore opens the database and verifies connectivity. It never writes.
func NewPGStore(cfg PGConfig) (*pgStore, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("onlyoffice: sql file store: empty DSN (set ONLYOFFICE_DSN)")
	}
	driver := pgDriver(dsn, cfg.Driver)
	dsn, err := normalizeSQLDSN(driver, dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(sqlDriverName(driver), dsn)
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: sql file store: open %s: %w", driver, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), pgConnectTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("onlyoffice: sql file store: ping %s: %w", driver, err)
	}
	s := &pgStore{db: db, driver: driver, http: &http.Client{}}
	if t := strings.TrimSpace(cfg.Tenant); t != "" {
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("onlyoffice: sql file store: non-numeric tenant %q", t)
		}
		s.tenantID, s.hasTenant = n, true
	}
	return s, nil
}

// Close releases the database handle.
func (s *pgStore) Close() error { return s.db.Close() }

// Name implements FileStore and Searcher.
func (s *pgStore) Name() string { return s.driver }

// pgDriver resolves the engine: the explicit value wins, otherwise the DSN
// shape decides. A leading postgres:// scheme or a libpq keyword DSN (which
// always carries '=') selects PostgreSQL; anything else is MySQL.
func pgDriver(dsn, explicit string) string {
	switch strings.ToLower(strings.TrimSpace(explicit)) {
	case ProviderPG, "pg", "postgresql", "pgx":
		return ProviderPG
	case ProviderMySQL, "mariadb":
		return ProviderMySQL
	}
	l := strings.ToLower(strings.TrimSpace(dsn))
	switch {
	case strings.HasPrefix(l, "postgres://"), strings.HasPrefix(l, "postgresql://"):
		return ProviderPG
	case strings.HasPrefix(l, "mysql://"), strings.Contains(l, "@tcp("), strings.Contains(l, "@unix("):
		return ProviderMySQL
	case strings.Contains(l, "="):
		return ProviderPG
	default:
		return ProviderMySQL
	}
}

// sqlDriverName maps the engine to its registered database/sql driver.
func sqlDriverName(driver string) string {
	if driver == ProviderPG {
		return "pgx"
	}
	return "mysql"
}

// normalizeSQLDSN converts a mysql:// URL to the go-sql-driver form and forces
// parseTime so datetime columns scan into time.Time. PostgreSQL DSNs pass
// through untouched.
func normalizeSQLDSN(driver, dsn string) (string, error) {
	if driver != ProviderMySQL {
		return dsn, nil
	}
	if strings.HasPrefix(strings.ToLower(dsn), "mysql://") {
		converted, err := mysqlDSNFromURL(dsn)
		if err != nil {
			return "", err
		}
		dsn = converted
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("onlyoffice: sql file store: parse mysql DSN: %w", err)
	}
	cfg.ParseTime = true
	return cfg.FormatDSN(), nil
}

// mysqlDSNFromURL turns mysql://user:pass@host:port/db into the driver DSN.
func mysqlDSNFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("onlyoffice: sql file store: bad mysql URL %q", raw)
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			user += ":" + p
		}
	}
	q := u.Query()
	q.Set("parseTime", "true")
	return fmt.Sprintf("%s@tcp(%s)/%s?%s", user, u.Host, strings.TrimPrefix(u.Path, "/"), q.Encode()), nil
}

// rebind rewrites '?' placeholders to PostgreSQL's $1..$n. MySQL keeps them.
func rebind(query, driver string) string {
	if driver != ProviderPG {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// List returns the folders and files directly below parentID, folders first.
func (s *pgStore) List(ctx context.Context, parentID string) ([]Entry, error) {
	pid, err := parseEntryID(parentID)
	if err != nil {
		return nil, err
	}
	folders, err := s.queryFolders(ctx, "parent_id = ?", pid)
	if err != nil {
		return nil, err
	}
	files, err := s.queryFiles(ctx, "folder_id = ? AND current_version = 1", pid)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(folders)+len(files))
	for _, f := range folders {
		out = append(out, folderRowToEntry(f, s.Name()))
	}
	for _, f := range files {
		out = append(out, fileRowToEntry(f, s.Name()))
	}
	return out, nil
}

// Stat resolves a folder or file id to an Entry. Folders win when both id
// spaces overlap (they never do on a real portal, but the lookup is cheap).
func (s *pgStore) Stat(ctx context.Context, id string) (Entry, error) {
	n, err := parseEntryID(id)
	if err != nil {
		return Entry{}, err
	}
	folders, err := s.queryFolders(ctx, "id = ?", n)
	if err != nil {
		return Entry{}, err
	}
	if len(folders) > 0 {
		return folderRowToEntry(folders[0], s.Name()), nil
	}
	files, err := s.queryFiles(ctx, "id = ? AND current_version = 1", n)
	if err != nil {
		return Entry{}, err
	}
	if len(files) == 0 {
		return Entry{}, fmt.Errorf("onlyoffice: sql file store: id %s not found", id)
	}
	return fileRowToEntry(files[0], s.Name()), nil
}

// Download streams the file's current version from the portal's S3/MinIO store.
// The object key is reconstructed from the file id and version; the parent
// folder id is not part of the key.
func (s *pgStore) Download(ctx context.Context, id string, w io.Writer) (int64, error) {
	n, err := parseEntryID(id)
	if err != nil {
		return 0, err
	}
	files, err := s.queryFiles(ctx, "id = ? AND current_version = 1", n)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, fmt.Errorf("onlyoffice: sql file store: file %s not found", id)
	}
	f := files[0]
	key := csObjectKey(s.tenantID, f.id, f.version, filepath.Ext(f.title))
	return downloadMinioObject(ctx, s.http, key, w)
}

// CreateFolder is unavailable: the SQL backend is read-only.
func (s *pgStore) CreateFolder(context.Context, string, string) (Entry, error) {
	return Entry{}, ErrReadOnly
}

// Upload is unavailable: the SQL backend is read-only.
func (s *pgStore) Upload(context.Context, string, string, io.Reader) (Entry, error) {
	return Entry{}, ErrReadOnly
}

// Move is unavailable: the SQL backend is read-only.
func (s *pgStore) Move(context.Context, []string, string) error { return ErrReadOnly }

// Copy is unavailable: the SQL backend is read-only.
func (s *pgStore) Copy(context.Context, []string, string) error { return ErrReadOnly }

// Rename is unavailable: the SQL backend is read-only.
func (s *pgStore) Rename(context.Context, string, string) error { return ErrReadOnly }

// Delete is unavailable: the SQL backend is read-only.
func (s *pgStore) Delete(context.Context, []string) error { return ErrReadOnly }

// Search matches file titles by substring. Content search lives in the
// Elasticsearch backend; q.InContent is ignored here.
func (s *pgStore) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	text := strings.TrimSpace(q.Text)
	if text == "" {
		return nil, fmt.Errorf("onlyoffice: empty search query")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = pgSearchLimit
	}
	if limit > pgSearchMaxLimit {
		limit = pgSearchMaxLimit
	}

	where := "title LIKE ? AND current_version = 1"
	args := []any{"%" + text + "%"}
	if s.hasTenant {
		where += " AND tenant_id = ?"
		args = append(args, s.tenantID)
	}
	if fid := strings.TrimSpace(q.FolderID); fid != "" {
		n, err := parseEntryID(fid)
		if err != nil {
			return nil, err
		}
		where += " AND folder_id = ?"
		args = append(args, n)
	}
	for _, ext := range normalizeExtensions(q.Extensions) {
		where += " AND LOWER(title) LIKE ?"
		args = append(args, "%."+ext)
	}
	query := rebind(`SELECT id, folder_id, title, content_length, version, create_on, modified_on
        FROM files_file WHERE `+where+` ORDER BY modified_on DESC, id DESC LIMIT ?`, s.driver)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: sql search: %w", err)
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		f, err := scanFileRow(rows)
		if err != nil {
			return nil, err
		}
		hits = append(hits, SearchHit{Entry: fileRowToEntry(f, s.Name())})
	}
	return hits, rows.Err()
}

// queryFolders runs a folder SELECT with the tenant filter applied.
func (s *pgStore) queryFolders(ctx context.Context, where string, arg any) ([]pgFolderRow, error) {
	args := []any{arg}
	if s.hasTenant {
		where += " AND tenant_id = ?"
		args = append(args, s.tenantID)
	}
	query := rebind(`SELECT id, parent_id, title, create_on, modified_on
        FROM files_folder WHERE `+where+` ORDER BY title, id`, s.driver)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: sql list folders: %w", err)
	}
	defer rows.Close()
	var out []pgFolderRow
	for rows.Next() {
		var r pgFolderRow
		if err := rows.Scan(&r.id, &r.parentID, &r.title, &r.created, &r.modified); err != nil {
			return nil, fmt.Errorf("onlyoffice: sql folder row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// queryFiles runs a file SELECT for the current version with the tenant filter.
func (s *pgStore) queryFiles(ctx context.Context, where string, arg any) ([]pgFileRow, error) {
	args := []any{arg}
	if s.hasTenant {
		where += " AND tenant_id = ?"
		args = append(args, s.tenantID)
	}
	query := rebind(`SELECT id, folder_id, title, content_length, version, create_on, modified_on
        FROM files_file WHERE `+where+` ORDER BY title, id`, s.driver)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: sql list files: %w", err)
	}
	defer rows.Close()
	var out []pgFileRow
	for rows.Next() {
		f, err := scanFileRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// pgFileRow is one current files_file row.
type pgFileRow struct {
	id       int64
	folderID int64
	title    string
	size     int64
	version  int
	created  time.Time
	modified time.Time
}

// pgFolderRow is one files_folder row.
type pgFolderRow struct {
	id       int64
	parentID int64
	title    string
	created  time.Time
	modified time.Time
}

// scanFileRow reads the canonical file column order.
func scanFileRow(rows *sql.Rows) (pgFileRow, error) {
	var f pgFileRow
	if err := rows.Scan(&f.id, &f.folderID, &f.title, &f.size, &f.version, &f.created, &f.modified); err != nil {
		return f, fmt.Errorf("onlyoffice: sql file row: %w", err)
	}
	return f, nil
}

// fileRowToEntry maps a files_file row to the canonical model.
func fileRowToEntry(f pgFileRow, provider string) Entry {
	return Entry{
		ID:       strconv.FormatInt(f.id, 10),
		ParentID: strconv.FormatInt(f.folderID, 10),
		Title:    f.title,
		Kind:     File,
		Size:     f.size,
		MIME:     mimeForTitle(f.title, ""),
		Created:  f.created.UTC(),
		Modified: f.modified.UTC(),
		Version:  f.version,
		Provider: provider,
	}
}

// folderRowToEntry maps a files_folder row to the canonical model.
func folderRowToEntry(f pgFolderRow, provider string) Entry {
	return Entry{
		ID:       strconv.FormatInt(f.id, 10),
		ParentID: strconv.FormatInt(f.parentID, 10),
		Title:    f.title,
		Kind:     Folder,
		Created:  f.created.UTC(),
		Modified: f.modified.UTC(),
		Provider: provider,
	}
}

// csObjectKey reconstructs the object key the portal's S3 consumer uses:
//
//	00/00/<tenant>/files/folder_<shard>/file_<id>/v<version>/content.<ext>
//
// The shard is the next thousand above the file id (file 3727 -> folder_4000),
// NOT the parent folder id — verified live against the MinIO bucket.
func csObjectKey(tenant int64, fileID int64, version int, ext string) string {
	shard := (fileID/1000 + 1) * 1000
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
	if ext == "" {
		ext = "bin"
	}
	if version < 1 {
		version = 1
	}
	if tenant <= 0 {
		tenant = 1
	}
	return fmt.Sprintf("00/00/%02d/files/folder_%d/file_%d/v%d/content.%s", tenant, shard, fileID, version, ext)
}

// parseEntryID parses a numeric OnlyOffice id or returns a store error.
func parseEntryID(id string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("onlyoffice: sql file store: non-numeric id %q", id)
	}
	return n, nil
}
