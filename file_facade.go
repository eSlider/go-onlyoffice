package onlyoffice

// Single file client (epic #34, F4 #38). FileClient composes the registered
// FileStore and Searcher backends and picks one per operation: REST/DAV for
// writes, PostgreSQL (when registered) for fast reads, Elasticsearch for name
// and content search. Client.Files returns the facade; it also implements
// FileStore, so existing callers keep compiling.

import (
	"context"
	"errors"
	"io"
	"strings"
)

// ProviderES is the composed Elasticsearch searcher. The SQL store owns
// ProviderPG/ProviderMySQL (file_pg.go); the facade references ProviderPG in
// readOrder.
const ProviderES = "elasticsearch"

var (
	errNoReadBackend  = errors.New("onlyoffice: no file backend registered for reads")
	errNoWriteBackend = errors.New("onlyoffice: no file backend registered for writes")
	errNoSearcher     = errors.New("onlyoffice: no search backend registered (set ONLYOFFICE_ES_URL)")
)

// FileClient is the single entry point for file operations. It holds the
// registered backends and the order in which each operation tries them.
type FileClient struct {
	stores    map[string]FileStore
	searchers map[string]Searcher

	readOrder   []string
	writeOrder  []string
	searchOrder []string
}

// newFileClient builds the facade over the built-in REST and DAV stores. The
// Elasticsearch searcher is registered when ONLYOFFICE_ES_URL is set; the
// missing-credential case is left to Search so read-only commands still work.
func (c *Client) newFileClient() *FileClient {
	f := &FileClient{
		stores: map[string]FileStore{
			ProviderREST: &restStore{c: c},
			ProviderDAV:  &davStore{c: c},
		},
		searchers:   map[string]Searcher{},
		readOrder:   []string{ProviderPG, ProviderMySQL, ProviderREST, ProviderDAV},
		writeOrder:  []string{ProviderREST, ProviderDAV},
		searchOrder: []string{ProviderES},
	}
	if cfg := ESConfigFromEnv(); cfg.URL != "" {
		if es, err := NewESSearcher(cfg); err == nil {
			f.searchers[ProviderES] = es
		}
	}
	return f
}

// RegisterStore adds or replaces a named backend (for example the PostgreSQL
// read store). The name is matched case-insensitively.
func (f *FileClient) RegisterStore(name string, s FileStore) {
	if f == nil || s == nil {
		return
	}
	name = normalizeProvider(name)
	if name == "" {
		return
	}
	if f.stores == nil {
		f.stores = map[string]FileStore{}
	}
	f.stores[name] = s
}

// RegisterSearcher adds or replaces a named search backend.
func (f *FileClient) RegisterSearcher(name string, s Searcher) {
	if f == nil || s == nil {
		return
	}
	name = normalizeProvider(name)
	if name == "" {
		return
	}
	if f.searchers == nil {
		f.searchers = map[string]Searcher{}
	}
	f.searchers[name] = s
}

// Read returns the preferred backend for reads: the SQL store (PostgreSQL or
// MySQL) when registered, then REST, then WebDAV.
func (f *FileClient) Read() FileStore { return f.firstStore(f.readOrder) }

// Write returns the preferred backend for writes: REST, then WebDAV.
func (f *FileClient) Write() FileStore { return f.firstStore(f.writeOrder) }

// Search returns the preferred name/content searcher (Elasticsearch), or an
// error when no search backend is configured.
func (f *FileClient) Search() (Searcher, error) {
	if f == nil {
		return nil, errNoSearcher
	}
	for _, name := range f.searchOrder {
		if s := f.searchers[normalizeProvider(name)]; s != nil {
			return s, nil
		}
	}
	return nil, errNoSearcher
}

// firstStore returns the first registered store in the order.
func (f *FileClient) firstStore(order []string) FileStore {
	if f == nil {
		return nil
	}
	for _, name := range order {
		if s := f.stores[normalizeProvider(name)]; s != nil {
			return s
		}
	}
	return nil
}

// orderedStores returns the registered stores in the order.
func (f *FileClient) orderedStores(order []string) []FileStore {
	if f == nil {
		return nil
	}
	out := make([]FileStore, 0, len(order))
	for _, name := range order {
		if s := f.stores[normalizeProvider(name)]; s != nil {
			out = append(out, s)
		}
	}
	return out
}

func normalizeProvider(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Name implements FileStore and reports the preferred read backend.
func (f *FileClient) Name() string {
	if s := f.Read(); s != nil {
		return s.Name()
	}
	return ""
}

// List reads from the preferred backend, falling back to the next read backend
// only on a transient error (429/502/503/504).
func (f *FileClient) List(ctx context.Context, parentID string) ([]Entry, error) {
	return fallbackRead(ctx, f.orderedStores(f.readOrder), func(s FileStore) ([]Entry, error) {
		return s.List(ctx, parentID)
	})
}

// Stat reads from the preferred backend, with the same transient fallback.
func (f *FileClient) Stat(ctx context.Context, id string) (Entry, error) {
	return fallbackRead(ctx, f.orderedStores(f.readOrder), func(s FileStore) (Entry, error) {
		return s.Stat(ctx, id)
	})
}

// Download streams file bytes. It does not fall back: a failed attempt may have
// already written partial bytes into w, so a second backend would append.
func (f *FileClient) Download(ctx context.Context, id string, w io.Writer) (int64, error) {
	s := f.Read()
	if s == nil {
		return 0, errNoReadBackend
	}
	return s.Download(ctx, id, w)
}

// CreateFolder writes to the preferred write backend.
func (f *FileClient) CreateFolder(ctx context.Context, parentID, title string) (Entry, error) {
	s := f.Write()
	if s == nil {
		return Entry{}, errNoWriteBackend
	}
	return s.CreateFolder(ctx, parentID, title)
}

// Upload writes to the preferred write backend.
func (f *FileClient) Upload(ctx context.Context, parentID, title string, r io.Reader) (Entry, error) {
	s := f.Write()
	if s == nil {
		return Entry{}, errNoWriteBackend
	}
	return s.Upload(ctx, parentID, title, r)
}

// Move writes to the preferred write backend.
func (f *FileClient) Move(ctx context.Context, ids []string, parentID string) error {
	s := f.Write()
	if s == nil {
		return errNoWriteBackend
	}
	return s.Move(ctx, ids, parentID)
}

// Copy writes to the preferred write backend.
func (f *FileClient) Copy(ctx context.Context, ids []string, parentID string) error {
	s := f.Write()
	if s == nil {
		return errNoWriteBackend
	}
	return s.Copy(ctx, ids, parentID)
}

// Rename writes to the preferred write backend.
func (f *FileClient) Rename(ctx context.Context, id, title string) error {
	s := f.Write()
	if s == nil {
		return errNoWriteBackend
	}
	return s.Rename(ctx, id, title)
}

// Delete writes to the preferred write backend.
func (f *FileClient) Delete(ctx context.Context, ids []string) error {
	s := f.Write()
	if s == nil {
		return errNoWriteBackend
	}
	return s.Delete(ctx, ids)
}

// fallbackRead runs op against each store in order, moving on only when the
// error is transient. Non-transient errors (not found, forbidden) are final.
func fallbackRead[T any](ctx context.Context, stores []FileStore, op func(FileStore) (T, error)) (T, error) {
	var zero T
	if len(stores) == 0 {
		return zero, errNoReadBackend
	}
	var err error
	for i, s := range stores {
		var v T
		v, err = op(s)
		if err == nil {
			return v, nil
		}
		if i == len(stores)-1 || !Transient(err) {
			return zero, err
		}
	}
	return zero, err
}
