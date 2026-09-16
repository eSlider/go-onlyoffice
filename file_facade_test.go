package onlyoffice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// fakeStore is a FileStore test double; it records which backend served a call
// and returns a canned result or error.
type fakeStore struct {
	name    string
	entries []Entry
	err     error
	calls   *[]string
}

func (f *fakeStore) record(op string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, op+":"+f.name)
	}
}

func (f *fakeStore) Name() string { return f.name }

func (f *fakeStore) List(_ context.Context, _ string) ([]Entry, error) {
	f.record("list")
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func (f *fakeStore) Stat(_ context.Context, id string) (Entry, error) {
	f.record("stat")
	if f.err != nil {
		return Entry{}, f.err
	}
	return Entry{ID: id, Title: "t-" + f.name, Provider: f.name}, nil
}

func (f *fakeStore) CreateFolder(_ context.Context, _, title string) (Entry, error) {
	f.record("mkdir")
	if f.err != nil {
		return Entry{}, f.err
	}
	return Entry{ID: "new", Title: title, Provider: f.name}, nil
}

func (f *fakeStore) Upload(_ context.Context, _, title string, _ io.Reader) (Entry, error) {
	f.record("upload")
	return Entry{ID: "up", Title: title, Provider: f.name}, f.err
}

func (f *fakeStore) Download(_ context.Context, _ string, _ io.Writer) (int64, error) {
	f.record("download")
	return 0, f.err
}

func (f *fakeStore) Move(_ context.Context, _ []string, _ string) error {
	f.record("move")
	return f.err
}

func (f *fakeStore) Copy(_ context.Context, _ []string, _ string) error {
	f.record("copy")
	return f.err
}

func (f *fakeStore) Rename(_ context.Context, _, _ string) error {
	f.record("rename")
	return f.err
}

func (f *fakeStore) Delete(_ context.Context, _ []string) error {
	f.record("delete")
	return f.err
}

type fakeSearcher struct{ name string }

func (s *fakeSearcher) Name() string { return s.name }

func (s *fakeSearcher) Search(_ context.Context, _ SearchQuery) ([]SearchHit, error) {
	return []SearchHit{{Entry: Entry{Title: s.name}}}, nil
}

func newFacadeTestClient(stores map[string]FileStore, read, write []string) *FileClient {
	return &FileClient{
		stores:     stores,
		searchers:  map[string]Searcher{},
		readOrder:  read,
		writeOrder: write,
	}
}

// TestFileClientIsFileStore guarantees the facade can stand in for the
// interface anywhere a plain FileStore is expected.
func TestFileClientIsFileStore(t *testing.T) {
	var _ FileStore = (*FileClient)(nil)
}

func TestClientFilesPrefersRESTForReadsAndWrites(t *testing.T) {
	c := NewClient(Credentials{})
	f := c.Files()
	if got := f.Read().Name(); got != ProviderREST {
		t.Errorf("Read().Name() = %q, want %q", got, ProviderREST)
	}
	if got := f.Write().Name(); got != ProviderREST {
		t.Errorf("Write().Name() = %q, want %q", got, ProviderREST)
	}
	if got := f.Name(); got != ProviderREST {
		t.Errorf("Name() = %q, want %q", got, ProviderREST)
	}
}

func TestFileClientPostgresTakesReadPriority(t *testing.T) {
	pg := &fakeStore{name: ProviderPG}
	f := newFacadeTestClient(
		map[string]FileStore{ProviderREST: &fakeStore{name: ProviderREST}, ProviderPG: pg},
		[]string{ProviderPG, ProviderREST},
		[]string{ProviderREST},
	)
	if got := f.Read().Name(); got != ProviderPG {
		t.Errorf("Read().Name() = %q, want %q", got, ProviderPG)
	}
	if got := f.Write().Name(); got != ProviderREST {
		t.Errorf("Write().Name() = %q, want %q (PG is read-only)", got, ProviderREST)
	}
}

func TestFileClientRegisterStoreNormalizesName(t *testing.T) {
	pg := &fakeStore{name: "pg"}
	f := newFacadeTestClient(map[string]FileStore{}, []string{ProviderPG}, nil)
	f.RegisterStore("  POSTGRES  ", pg)
	if got := f.Read(); got != pg {
		t.Fatalf("Read() = %v, want registered postgres store", got)
	}
	f.RegisterStore("", pg)
	f.RegisterStore("pg", nil)
}

func TestFileClientReadFallsBackOnlyOnTransient(t *testing.T) {
	var calls []string
	primary := &fakeStore{name: "primary", err: fmt.Errorf("onlyoffice: list: 503 unavailable"), calls: &calls}
	secondary := &fakeStore{name: "secondary", entries: []Entry{{ID: "1"}}, calls: &calls}
	f := newFacadeTestClient(
		map[string]FileStore{"primary": primary, "secondary": secondary},
		[]string{"primary", "secondary"},
		nil,
	)
	got, err := f.List(context.Background(), "root")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("List() = %+v, want secondary entry", got)
	}
	want := []string{"list:primary", "list:secondary"}
	if fmt.Sprint(calls) != fmt.Sprint(want) {
		t.Fatalf("call order = %v, want %v", calls, want)
	}
}

func TestFileClientReadStopsOnPermanentError(t *testing.T) {
	var calls []string
	primary := &fakeStore{name: "primary", err: errors.New("onlyoffice: not found"), calls: &calls}
	secondary := &fakeStore{name: "secondary", entries: []Entry{{ID: "1"}}, calls: &calls}
	f := newFacadeTestClient(
		map[string]FileStore{"primary": primary, "secondary": secondary},
		[]string{"primary", "secondary"},
		nil,
	)
	if _, err := f.List(context.Background(), "root"); err == nil {
		t.Fatal("expected permanent error to be returned")
	}
	if len(calls) != 1 || calls[0] != "list:primary" {
		t.Fatalf("secondary backend must not run on a permanent error: %v", calls)
	}
}

func TestFileClientWriteUsesWriteBackend(t *testing.T) {
	var calls []string
	rest := &fakeStore{name: ProviderREST, calls: &calls}
	dav := &fakeStore{name: ProviderDAV, calls: &calls}
	f := newFacadeTestClient(
		map[string]FileStore{ProviderREST: rest, ProviderDAV: dav},
		[]string{ProviderREST},
		[]string{ProviderREST, ProviderDAV},
	)
	if _, err := f.CreateFolder(context.Background(), "p", "t"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if _, err := f.Upload(context.Background(), "p", "t", nil); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if len(calls) != 2 || calls[0] != "mkdir:rest" || calls[1] != "upload:rest" {
		t.Fatalf("write calls = %v, want REST", calls)
	}
}

func TestFileClientWriteWithoutBackend(t *testing.T) {
	f := newFacadeTestClient(map[string]FileStore{}, nil, nil)
	if err := f.Delete(context.Background(), []string{"1"}); !errors.Is(err, errNoWriteBackend) {
		t.Fatalf("Delete err = %v, want errNoWriteBackend", err)
	}
	if _, err := f.List(context.Background(), "root"); !errors.Is(err, errNoReadBackend) {
		t.Fatalf("List err = %v, want errNoReadBackend", err)
	}
}

func TestNewFileClientReadOrderIncludesSQL(t *testing.T) {
	f := NewClient(Credentials{}).newFileClient()
	want := []string{ProviderPG, ProviderMySQL, ProviderREST, ProviderDAV}
	if fmt.Sprint(f.readOrder) != fmt.Sprint(want) {
		t.Fatalf("readOrder = %v, want %v", f.readOrder, want)
	}
}

// TestClientFileStoreSQLRoutingWithoutDSN checks that the SQL backend names are
// recognised and never yield nil: without a DSN the returned store surfaces the
// open error on use.
func TestClientFileStoreSQLRoutingWithoutDSN(t *testing.T) {
	t.Setenv("ONLYOFFICE_DSN", "")
	t.Setenv("ONLYOFFICE_PG_HOST", "")
	c := NewClient(Credentials{})
	for _, name := range []string{"pg", "sql", ProviderPG, ProviderMySQL} {
		s := c.FileStore(name)
		if s == nil {
			t.Fatalf("FileStore(%q) = nil", name)
		}
		if _, err := s.Stat(context.Background(), "1"); err == nil {
			t.Errorf("FileStore(%q).Stat without DSN: want error", name)
		}
	}
}

func TestFileClientMySQLStoreIsPreferredForReads(t *testing.T) {
	mysql := &fakeStore{name: ProviderMySQL}
	f := newFacadeTestClient(
		map[string]FileStore{ProviderREST: &fakeStore{name: ProviderREST}, ProviderMySQL: mysql},
		[]string{ProviderPG, ProviderMySQL, ProviderREST},
		[]string{ProviderREST},
	)
	if got := f.Read().Name(); got != ProviderMySQL {
		t.Errorf("Read().Name() = %q, want %q", got, ProviderMySQL)
	}
}

func TestFileClientSearchSelection(t *testing.T) {
	f := &FileClient{searchers: map[string]Searcher{}, searchOrder: []string{ProviderES}}
	_, err := f.Search()
	if err == nil || !strings.Contains(err.Error(), "ONLYOFFICE_ES_URL") {
		t.Fatalf("Search without backend = %v, want ONLYOFFICE_ES_URL hint", err)
	}
	es := &fakeSearcher{name: "fake-es"}
	f.RegisterSearcher(ProviderES, es)
	got, err := f.Search()
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got.Name() != "fake-es" {
		t.Fatalf("searcher = %q, want fake-es", got.Name())
	}
}
