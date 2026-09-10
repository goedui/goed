package fileops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleOnDiskDetectsExternalModification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if StaleOnDisk(doc) {
		t.Fatal("freshly loaded document must not be stale")
	}

	// 确保mtime 前进（FAT 粒度 2s，NTFS 100ns；这里统一推进 2s）。
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	if !StaleOnDisk(doc) {
		t.Fatal("document must be stale after external touch")
	}
}

func TestMissingOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if MissingOnDisk(doc) {
		t.Fatal("existing file reported missing")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if !MissingOnDisk(doc) {
		t.Fatal("removed file not detected as missing")
	}
	if MissingOnDisk(Document{}) {
		t.Fatal("empty document must not be reported missing")
	}
}
