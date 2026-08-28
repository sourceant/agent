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
