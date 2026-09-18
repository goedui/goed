package storage

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "github.com/goedui/goed", "storage.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestStoreSetPersistsAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "github.com/goedui/goed", "storage.json")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Set("theme", "Light"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set("fontSize", float64(16)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// 重新打开同一文件验证落盘。
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.GetString("theme", "Dark"); got != "Light" {
		t.Fatalf("theme = %q, want Light", got)
	}
	if got := reopened.GetInt("fontSize", 0); got != 16 {
		t.Fatalf("fontSize = %d, want 16", got)
	}
}

func TestStoreTypedGettersFallBackToDefaults(t *testing.T) {
	s := newTestStore(t)
	if got := s.GetString("missing", "def"); got != "def" {
		t.Fatalf("GetString = %q, want def", got)
	}
	if got := s.GetInt("missing", 7); got != 7 {
		t.Fatalf("GetInt = %d, want 7", got)
	}
	if got := s.GetBool("missing", true); !got {
		t.Fatal("GetBool = false, want true")
	}
	_ = s.Set("flag", "not-a-bool")
	if got := s.GetBool("flag", true); !got {
		t.Fatal("mistyped bool should fall back to default")
	}
}

func TestStoreOverwrite(t *testing.T) {
	s := newTestStore(t)
	if err := s.Set("theme", "Dark"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set("theme", "Tabby"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := s.GetString("theme", ""); got != "Tabby" {
		t.Fatalf("theme = %q, want Tabby", got)
	}
}
