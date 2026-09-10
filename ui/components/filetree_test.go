package components

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeListsDirectoriesBeforeFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := NewFileTree(root)
	if len(tree.Entries) != 3 || !tree.Entries[1].Directory || tree.Entries[2].Directory {
		t.Fatalf("entries = %#v", tree.Entries)
	}
}

func TestFileTreeOpensFileOnClick(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := NewFileTree(root)
	opened := ""
	tree.OnOpen = func(p string) { opened = p }
	// root row is at y=0, first child starts at y=58; account for row height.
	if !tree.HandleMouseDown(20, 58+tree.RowHeight, 0, 0, 300, 300) {
		t.Fatal("click not handled")
	}
	if opened != path {
		t.Fatalf("opened = %q, want %q", opened, path)
	}
}

func TestFileTreeStatusCanBeSet(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := NewFileTree(root)
	tree.SetStatus(path, FileStatusModified)
	for _, entry := range tree.Entries {
		if entry.Path == path && entry.Status != FileStatusModified {
			t.Fatalf("status = %v", entry.Status)
		}
	}
}
