//go:build windows

package windows

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBytesToDirUsesUniqueName(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("png-bytes")
	first, err := writeBytesToDir(dir, "imgen.png", raw)
	if err != nil || filepath.Base(first) != "imgen.png" {
		t.Fatalf("first save: path=%s err=%v", first, err)
	}
	got, err := os.ReadFile(first)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("first file content")
	}
	second, err := writeBytesToDir(dir, "imgen.png", []byte("other"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(second) != "imgen (2).png" {
		t.Fatalf("expected unique name, got %s", second)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal("original overwritten")
	}
}

func TestSanitizeFileNameStripsPath(t *testing.T) {
	if got := sanitizeFileName(`C:\secret\..\imgen.png`); got != "imgen.png" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeFileName("  "); got != "download" {
		t.Fatalf("empty: %q", got)
	}
}

func TestWriteBytesToDirRejectsEmpty(t *testing.T) {
	if _, err := writeBytesToDir(t.TempDir(), "a.png", nil); err == nil {
		t.Fatal("empty data accepted")
	}
}

func TestUniquePathIncrements(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.png")
	if err := os.WriteFile(path, []byte("1"), 0600); err != nil {
		t.Fatal(err)
	}
	got := uniquePath(path)
	if !strings.HasSuffix(got, "a (2).png") {
		t.Fatalf("got %s", got)
	}
}
