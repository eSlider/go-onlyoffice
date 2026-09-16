package onlyoffice

// davStore implements FileStore on top of the Documents/WebDAV methods in
// files_webdav.go. The Documents fileops calls need folder and file ids
// separated, so ids are classified through Stat before move/copy/rename/delete.

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// davStore is a FileStore over the WebDAV-oriented Documents API.
type davStore struct{ c *Client }

// Name reports the backend name.
func (s *davStore) Name() string { return ProviderDAV }

// List returns the files and folders directly below parentID.
func (s *davStore) List(ctx context.Context, parentID string) ([]Entry, error) {
	var out []Entry
	err := retryStoreOp(ctx, func() error {
		l, err := s.c.ListDavFolder(ctx, parentID)
		if err != nil {
			return err
		}
		entries := make([]Entry, 0, len(l.Folders)+len(l.Files))
		for _, f := range l.Folders {
			entries = append(entries, DavFolderToEntry(f, ProviderDAV))
		}
		for _, f := range l.Files {
			entries = append(entries, DavFileToEntry(f, ProviderDAV))
		}
		out = entries
		return nil
	})
	return out, err
}

// Stat resolves a folder or file entry by id. A folder answers ListDavFolder
// with its own metadata in Current; otherwise the file metadata API is used.
func (s *davStore) Stat(ctx context.Context, id string) (Entry, error) {
	return s.stat(ctx, id)
}

// CreateFolder creates a subfolder under parentID.
func (s *davStore) CreateFolder(ctx context.Context, parentID, title string) (Entry, error) {
	var out Entry
	err := retryStoreOp(ctx, func() error {
		f, err := s.c.CreateDavFolder(ctx, parentID, title)
		if err != nil {
			return err
		}
		if f == nil {
			return fmt.Errorf("onlyoffice: dav store: empty create-folder response")
		}
		out = DavFolderToEntry(*f, ProviderDAV)
		return nil
	})
	return out, err
}

// Upload streams r into parentID as title. The reader is buffered once so a
// retry re-sends the same bytes instead of an exhausted stream.
func (s *davStore) Upload(ctx context.Context, parentID, title string, r io.Reader) (Entry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Entry{}, err
	}
	var out Entry
	err = retryStoreOp(ctx, func() error {
		f, err := s.c.UploadDavFile(ctx, parentID, title, bytes.NewReader(data))
		if err != nil {
			return err
		}
		if f == nil {
			return fmt.Errorf("onlyoffice: dav store: empty upload response")
		}
		out = DavFileToEntry(*f, ProviderDAV)
		return nil
	})
	return out, err
}

// Download streams the file bytes into w.
func (s *davStore) Download(ctx context.Context, id string, w io.Writer) (int64, error) {
	var n int64
	err := retryStoreOp(ctx, func() error {
		var e error
		n, e = s.c.DownloadDavFile(ctx, id, w)
		return e
	})
	return n, err
}

// Move moves ids into parentID, splitting folders from files.
func (s *davStore) Move(ctx context.Context, ids []string, parentID string) error {
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

// Copy copies ids into parentID, splitting folders from files.
func (s *davStore) Copy(ctx context.Context, ids []string, parentID string) error {
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

// Rename renames a folder or file.
func (s *davStore) Rename(ctx context.Context, id, title string) error {
	e, err := s.stat(ctx, id)
	if err != nil {
		return err
	}
	return retryStoreOp(ctx, func() error {
		if e.Kind == Folder {
			return s.c.RenameDavFolder(ctx, id, title)
		}
		return s.c.RenameDavFile(ctx, id, title)
	})
}

// Delete removes ids, splitting folders from files.
func (s *davStore) Delete(ctx context.Context, ids []string) error {
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

// stat resolves a single id to a folder or file Entry.
func (s *davStore) stat(ctx context.Context, id string) (Entry, error) {
	var out Entry
	err := retryStoreOp(ctx, func() error {
		if l, err := s.c.ListDavFolder(ctx, id); err == nil {
			if l != nil && l.Current.ID != "" && l.Current.ID == id {
				out = DavFolderToEntry(l.Current, ProviderDAV)
				return nil
			}
		} else if Transient(err) {
			return err
		}
		f, err := s.c.GetFile(ctx, id)
		if err != nil {
			return err
		}
		out = FileEntryToEntry(f, ProviderDAV)
		return nil
	})
	return out, err
}

// split classifies ids into folder and file id lists.
func (s *davStore) split(ctx context.Context, ids []string) (folders, files []string, err error) {
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
