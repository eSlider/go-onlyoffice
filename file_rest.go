package onlyoffice

// restStore implements FileStore on top of the REST Documents methods in
// files.go. It is a thin adapter: no endpoint logic lives here, and every call
// is wrapped in DoRetry.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// restStore is a FileStore over the REST Documents API.
type restStore struct{ c *Client }

// Name reports the backend name.
func (s *restStore) Name() string { return ProviderREST }

// List returns the files and folders directly below parentID.
func (s *restStore) List(ctx context.Context, parentID string) ([]Entry, error) {
	var out []Entry
	err := retryStoreOp(ctx, func() error {
		raw, err := s.c.ListFolder(ctx, parentID)
		if err != nil {
			return err
		}
		entries, err := entriesFromFolderMap(raw, ProviderREST)
		if err != nil {
			return err
		}
		out = entries
		return nil
	})
	return out, err
}

// Stat returns file metadata. The REST adapter resolves files only; folders
// are listed by their parent (use List).
func (s *restStore) Stat(ctx context.Context, id string) (Entry, error) {
	var out Entry
	err := retryStoreOp(ctx, func() error {
		f, err := s.c.GetFile(ctx, id)
		if err != nil {
			return err
		}
		out = FileEntryToEntry(f, ProviderREST)
		return nil
	})
	return out, err
}

// CreateFolder creates a subfolder under parentID.
func (s *restStore) CreateFolder(ctx context.Context, parentID, title string) (Entry, error) {
	var out Entry
	err := retryStoreOp(ctx, func() error {
		m, err := s.c.CreateFolder(ctx, parentID, title)
		if err != nil {
			return err
		}
		e, err := folderEntryFromMap(m, parentID, ProviderREST)
		if err != nil {
			return err
		}
		if e.ParentID == "" {
			e.ParentID = parentID
		}
		if e.Title == "" {
			e.Title = title
		}
		out = e
		return nil
	})
	return out, err
}

// Upload streams r into parentID as title. UploadToFolder is path based, so
// the reader is spooled to a temporary file first (ponytail: OnlyOffice
// multipart upload buffers the whole body anyway).
func (s *restStore) Upload(ctx context.Context, parentID, title string, r io.Reader) (Entry, error) {
	dir, err := os.MkdirTemp("", "oo-rest-upload-")
	if err != nil {
		return Entry{}, err
	}
	defer os.RemoveAll(dir)

	local := filepath.Join(dir, SafeLocalFileName(title))
	f, err := os.Create(local)
	if err != nil {
		return Entry{}, err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return Entry{}, err
	}
	if err := f.Close(); err != nil {
		return Entry{}, err
	}

	var out Entry
	err = retryStoreOp(ctx, func() error {
		fe, err := s.c.UploadToFolder(ctx, parentID, local)
		if err != nil {
			return err
		}
		out = FileEntryToEntry(fe, ProviderREST)
		return nil
	})
	return out, err
}

// Download streams the file bytes into w.
func (s *restStore) Download(ctx context.Context, id string, w io.Writer) (int64, error) {
	var n int64
	err := retryStoreOp(ctx, func() error {
		var e error
		n, e = s.c.DownloadFile(ctx, id, w)
		return e
	})
	return n, err
}

// Move moves file ids into parentID. The REST MoveFiles endpoint handles files
// only; folder moves are not exposed by this adapter.
func (s *restStore) Move(ctx context.Context, ids []string, parentID string) error {
	dest, err := strconv.Atoi(strings.TrimSpace(parentID))
	if err != nil {
		return fmt.Errorf("onlyoffice: rest store: move: non-numeric destination folder id %q", parentID)
	}
	fileIDs, err := numericIDs(ids)
	if err != nil {
		return err
	}
	return retryStoreOp(ctx, func() error {
		_, err := s.c.MoveFiles(ctx, dest, fileIDs)
		return err
	})
}

// Copy copies file ids into parentID. files.go has no copy method, so the
// shared REST fileops copy endpoint (CopyDavItems) is used.
func (s *restStore) Copy(ctx context.Context, ids []string, parentID string) error {
	if len(ids) == 0 {
		return nil
	}
	return retryStoreOp(ctx, func() error {
		return s.c.CopyDavItems(ctx, nil, ids, parentID)
	})
}

// Rename sets a new title (including extension) for a file.
func (s *restStore) Rename(ctx context.Context, id, title string) error {
	return retryStoreOp(ctx, func() error {
		_, err := s.c.RenameFile(ctx, id, title)
		return err
	})
}

// Delete permanently deletes file ids.
func (s *restStore) Delete(ctx context.Context, ids []string) error {
	fileIDs, err := numericIDs(ids)
	if err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return nil
	}
	return retryStoreOp(ctx, func() error {
		return s.c.DeleteFiles(ctx, fileIDs)
	})
}

// entriesFromFolderMap converts a ListFolder response map into canonical
// entries, reusing the DavFile/DavFolder decoders for robust size handling.
func entriesFromFolderMap(m map[string]any, provider string) ([]Entry, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var listing DavListing
	if err := json.Unmarshal(b, &listing); err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(listing.Folders)+len(listing.Files))
	for _, f := range listing.Folders {
		out = append(out, DavFolderToEntry(f, provider))
	}
	for _, f := range listing.Files {
		out = append(out, DavFileToEntry(f, provider))
	}
	return out, nil
}

// folderEntryFromMap converts a CreateFolder response map into a folder Entry.
func folderEntryFromMap(m map[string]any, parentID, provider string) (Entry, error) {
	e := Entry{Kind: Folder, Provider: provider, ParentID: parentID}
	if m == nil {
		return e, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return e, err
	}
	var f DavFolder
	if err := json.Unmarshal(b, &f); err != nil {
		return e, err
	}
	e = DavFolderToEntry(f, provider)
	return e, nil
}

// numericIDs parses Documents numeric ids from strings.
func numericIDs(ids []string) ([]int, error) {
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		n, err := strconv.Atoi(strings.TrimSpace(id))
		if err != nil {
			return nil, fmt.Errorf("onlyoffice: rest store: non-numeric id %q", id)
		}
		out = append(out, n)
	}
	return out, nil
}
