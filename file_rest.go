package onlyoffice

// restStore implements FileStore on top of the REST Documents methods in
// files.go. It is a thin adapter: no endpoint logic lives here, and every call
// is wrapped in DoRetry.

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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

// Stat returns file or folder metadata. Folders are resolved through the
// listing endpoint (their own id appears as the listing's Current); other ids
// fall back to the file metadata API.
func (s *restStore) Stat(ctx context.Context, id string) (Entry, error) {
	return s.stat(ctx, id)
}

// stat resolves a single id to a folder or file Entry.
func (s *restStore) stat(ctx context.Context, id string) (Entry, error) {
	var out Entry
	err := retryStoreOp(ctx, func() error {
		if l, err := s.c.ListDavFolder(ctx, id); err == nil {
			if l != nil && l.Current.ID != "" && l.Current.ID == id {
				out = DavFolderToEntry(l.Current, ProviderREST)
				return nil
			}
		} else if Transient(err) {
			return err
		}
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

// Move moves folders and/or files into parentID. Ids are classified through
// stat so folder moves use folderIds and file moves use fileIds on the shared
// fileops/move endpoint.
func (s *restStore) Move(ctx context.Context, ids []string, parentID string) error {
	folders, files, err := s.split(ctx, ids)
	if err != nil {
		return err
	}
	if len(folders) == 0 && len(files) == 0 {
		return nil
	}
	return retryStoreOp(ctx, func() error {
		return s.c.MoveDavItems(ctx, folders, files, parentID)
	})
}

// Copy copies folders and/or files into parentID. files.go has no copy method,
// so the shared REST fileops copy endpoint (CopyDavItems) is used.
func (s *restStore) Copy(ctx context.Context, ids []string, parentID string) error {
	folders, files, err := s.split(ctx, ids)
	if err != nil {
		return err
	}
	if len(folders) == 0 && len(files) == 0 {
		return nil
	}
	return retryStoreOp(ctx, func() error {
		return s.c.CopyDavItems(ctx, folders, files, parentID)
	})
}

// Rename sets a new title (including extension) for a file or folder.
func (s *restStore) Rename(ctx context.Context, id, title string) error {
	e, err := s.stat(ctx, id)
	if err != nil {
		return err
	}
	if e.Kind == Folder {
		return retryStoreOp(ctx, func() error {
			return s.c.RenameDavFolder(ctx, id, title)
		})
	}
	return retryStoreOp(ctx, func() error {
		_, err := s.c.RenameFile(ctx, id, title)
		return err
	})
}

// Delete permanently deletes folders and/or files.
func (s *restStore) Delete(ctx context.Context, ids []string) error {
	folders, files, err := s.split(ctx, ids)
	if err != nil {
		return err
	}
	if len(folders) == 0 && len(files) == 0 {
		return nil
	}
	return retryStoreOp(ctx, func() error {
		return s.c.DeleteDavItems(ctx, folders, files)
	})
}

// split classifies ids into folder and file id lists.
func (s *restStore) split(ctx context.Context, ids []string) (folders, files []string, err error) {
	for _, id := range ids {
		e, err := s.stat(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if e.Kind == Folder {
			folders = append(folders, id)
		} else {
			files = append(files, id)
		}
	}
	return folders, files, nil
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
