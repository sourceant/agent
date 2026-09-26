package browse

import (
	"os"
	"path/filepath"
	"strings"
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

func TestItFindsAFolderByName(t *testing.T) {
	root := tree(t)
	t.Setenv("HOME", root)

	found, err := Find("bill")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	if len(found) != 1 || found[0].Name != "billing" {
		t.Fatalf("got %v, want the billing directory", found)
	}
	if !found[0].Repository {
		t.Error("the billing directory is a working tree and was not marked as one")
	}
}

func TestItAnswersWithWorkingTreesFirst(t *testing.T) {
	root := tree(t)
	if err := os.MkdirAll(filepath.Join(root, "notes-billing"), 0o755); err != nil {
		t.Fatalf("building the tree: %v", err)
	}
	t.Setenv("HOME", root)

	found, err := Find("billing")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	if len(found) != 2 || !found[0].Repository || found[0].Name != "billing" {
		t.Fatalf("got %v, want the working tree first", found)
	}
}

func TestItLeavesDependenciesAndHiddenDirectoriesAlone(t *testing.T) {
	root := tree(t)
	for _, dir := range []string{"work/node_modules/billing-ui", ".hidden/billing-old"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("building the tree: %v", err)
		}
	}
	t.Setenv("HOME", root)

	found, err := Find("billing")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	for _, entry := range found {
		if strings.Contains(entry.Path, "node_modules") || strings.Contains(entry.Path, ".hidden") {
			t.Errorf("%s should not have been walked", entry.Path)
		}
	}
}

func TestItAnswersWithNothingForAnEmptyTerm(t *testing.T) {
	root := tree(t)
	t.Setenv("HOME", root)

	found, err := Find("   ")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("got %v, want nothing", found)
	}
}
