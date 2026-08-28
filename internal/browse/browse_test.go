package browse

import (
	"os"
	"path/filepath"
	"testing"
)

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"work", "work/billing", "work/billing/.git", "work/notes", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("building the tree: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "work", "a-file"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing a file: %v", err)
	}
	return root
}

func TestItListsDirectoriesAndNotFiles(t *testing.T) {
	root := tree(t)

	listing, err := At(filepath.Join(root, "work"))
	if err != nil {
		t.Fatalf("listing: %v", err)
	}

	names := make([]string, 0, len(listing.Entries))
	for _, entry := range listing.Entries {
		names = append(names, entry.Name)
	}
	if len(names) != 2 || names[0] != "billing" || names[1] != "notes" {
		t.Errorf("got %v, want the two directories in order", names)
	}
}

func TestItMarksAWorkingTree(t *testing.T) {
	root := tree(t)

	listing, _ := At(filepath.Join(root, "work"))

	for _, entry := range listing.Entries {
		if entry.Name == "billing" && !entry.Repository {
			t.Error("a directory with .git in it was not marked as a repository")
		}
		if entry.Name == "notes" && entry.Repository {
			t.Error("a directory with no .git was marked as a repository")
		}
	}
}

func TestHiddenDirectoriesAreLeftOut(t *testing.T) {
	root := tree(t)

	listing, _ := At(root)

	for _, entry := range listing.Entries {
		if entry.Name == ".hidden" {
			t.Error("listed a hidden directory")
		}
	}
}

func TestItSaysWhereUpIs(t *testing.T) {
	root := tree(t)

	listing, _ := At(filepath.Join(root, "work"))

	if listing.Parent != root {
		t.Errorf("got parent %q, want %q", listing.Parent, root)
	}
}

func TestTheTopOfTheTreeHasNoParent(t *testing.T) {
	listing, err := At(string(filepath.Separator))
	if err != nil {
		t.Fatalf("listing the root: %v", err)
	}

	if listing.Parent != "" {
		t.Errorf("got parent %q at the top of the tree, want none", listing.Parent)
	}
}

func TestNowhereNamedStartsAtHome(t *testing.T) {
	listing, err := At("")
	if err != nil {
		t.Fatalf("listing: %v", err)
	}

	if listing.Path != Home() {
		t.Errorf("got %q, want %q", listing.Path, Home())
	}
}

func TestADirectoryThatIsNotThereIsAnError(t *testing.T) {
	if _, err := At(filepath.Join(t.TempDir(), "nowhere")); err == nil {
		t.Fatal("listed a directory that does not exist")
	}
}
