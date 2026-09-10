package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "note.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	doc, err := SaveAs(path, "hello\n世界")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Path != filepath.Clean(path) || doc.Content != "hello\n世界" {
		t.Fatalf("saved document = %#v", doc)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != doc {
		t.Fatalf("loaded document = %#v, want %#v", loaded, doc)
	}
}

func TestDocumentRequiresPath(t *testing.T) {
	if err := Save("", "text"); err != ErrPathRequired {
		t.Fatalf("Save empty path error = %v", err)
	}
}
