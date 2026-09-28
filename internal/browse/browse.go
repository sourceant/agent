// Package browse lists directories, so a person can pick one to index.
//
// A browser cannot give a page the absolute path of a folder somebody chose:
// its file picker deliberately withholds it. The agent is already on the
// machine, so it walks it and the page navigates what it lists.
package browse

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one directory that can be opened or picked.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Repository says the directory is a git working tree, which is what
	// somebody is nearly always looking for.
	Repository bool `json:"repository"`
}

// Listing is one directory and what is under it.
type Listing struct {
	Path    string  `json:"path"`
	Parent  string  `json:"parent"`
	Entries []Entry `json:"entries"`
}

// Home is where browsing starts when nowhere was named.
func Home() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return string(filepath.Separator)
	}
	return home
}

// At lists the directories inside path.
//
// Only directories, and only their names: this exists to choose somewhere to
// index, and reading file contents is the indexer's job, not the picker's.
func At(path string) (Listing, error) {
	if path == "" {
		path = Home()
	}
	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return Listing{}, err
	}

	items, err := os.ReadDir(resolved)
	if err != nil {
		return Listing{}, err
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() || strings.HasPrefix(item.Name(), ".") {
			continue
		}
		full := filepath.Join(resolved, item.Name())
		entries = append(entries, Entry{
			Name:       item.Name(),
			Path:       full,
			Repository: isRepository(full),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	parent := filepath.Dir(resolved)
	if parent == resolved {
		parent = ""
	}
	return Listing{Path: resolved, Parent: parent, Entries: entries}, nil
}

func isRepository(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// How far down the walk goes, and how much it answers with.
const (
	Depth   = 6
	Results = 40
)

// skipped names a directory whose contents nobody is looking for.
var skipped = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".venv":        true,
	"venv":         true,
}

// Find looks for directories whose name contains term, starting at home.
func Find(term string) ([]Entry, error) {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return nil, nil
	}

	root := Home()
	found := make([]Entry, 0, Results)
	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			// A directory this user cannot read is not a reason to stop.
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if path != root && (strings.HasPrefix(name, ".") || skipped[name]) {
			return filepath.SkipDir
		}
		if depthOf(root, path) > Depth {
			return filepath.SkipDir
		}
		if path != root && strings.Contains(strings.ToLower(name), term) {
			found = append(found, Entry{Name: name, Path: path, Repository: isRepository(path)})
			if len(found) >= Results {
				return filepath.SkipAll
			}
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil, err
	}

	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Repository != found[j].Repository {
			return found[i].Repository
		}
		left, right := depthOf(root, found[i].Path), depthOf(root, found[j].Path)
		if left != right {
			return left < right
		}
		return found[i].Path < found[j].Path
	})
	return found, nil
}

func depthOf(root, path string) int {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	return len(strings.Split(relative, string(filepath.Separator)))
}
