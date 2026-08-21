package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScanProjectsRoot finds git roots under root (max depth) and emits company rows.
func ScanProjectsRoot(root string, maxDepth int) (*Document, error) {
	root = filepath.Clean(root)
	if maxDepth <= 0 {
		maxDepth = 4
	}
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}

	var entries []Entry
	err = walkGitRoots(root, root, 0, maxDepth, &entries)
	if err != nil {
		return nil, err
	}

	// Also top-level dirs as company stubs (even without git) — coverage 1B.
	dents, _ := os.ReadDir(root)
	for _, d := range dents {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		name := d.Name()
		skip := map[string]struct{}{
			"node_modules": {}, "vendor": {}, ".cache": {},
		}
		if _, ok := skip[name]; ok {
			continue
		}
		path := filepath.Join(root, name)
		id := EntryID("company", "", name)
		role, zone := classifyProjectName(name, "")
		entries = append(entries, Entry{
			ID:      id,
			Kind:    "company",
			Name:    name,
			Sources: []string{path},
			Zone:    zone,
			Role:    role,
			Status:  "new",
			Notes:   "top_level_dir",
		})
	}

	return MergeDocs(&Document{Entries: entries}), nil
}

func walkGitRoots(root, dir string, depth, maxDepth int, out *[]Entry) error {
	if depth > maxDepth {
		return nil
	}
	// is this a git root?
	if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
		name := filepath.Base(dir)
		remote := gitRemoteOrigin(dir)
		role, zone := classifyProjectName(name, remote)
		*out = append(*out, Entry{
			ID:      EntryID("company", "", name),
			Kind:    "company",
			Name:    name,
			Sources: []string{dir},
			Remote:  remote,
			GitRoot: dir,
			Zone:    zone,
			Role:    role,
			Status:  "new",
		})
		return nil // do not descend into nested repos from a git root
	}
	dents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, d := range dents {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		if name == ".git" || name == "node_modules" || name == "vendor" || name == ".venv" || name == "dist" {
			continue
		}
		_ = walkGitRoots(root, filepath.Join(dir, name), depth+1, maxDepth, out)
	}
	return nil
}

func gitRemoteOrigin(dir string) string {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func classifyProjectName(name, remote string) (role, zone string) {
	lower := strings.ToLower(name)
	remoteL := strings.ToLower(remote)
	switch {
	case strings.Contains(lower, "experiment") || strings.HasPrefix(lower, "test"):
		return "experiment", "cold"
	case lower == "mama" || lower == "personal" || strings.Contains(lower, "private"):
		return "personal", "private"
	case strings.Contains(remoteL, "git.produktor.io") || strings.Contains(remoteL, "github.com/eslider"):
		return "work", "hot"
	case strings.Contains(lower, "produktor") || strings.Contains(lower, "eslider") ||
		strings.Contains(lower, "asesoria") || strings.Contains(lower, "dyvenia") ||
		strings.Contains(lower, "onlyoffice"):
		return "work", "warm"
	default:
		return "unknown", "warm"
	}
}
