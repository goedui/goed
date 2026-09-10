package composables

import (
	"path/filepath"
	"testing"

	"github.com/goedui/goed/ui/storage"
)

func TestUseStorageReadsInitialAndWritesThrough(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "goed", "storage.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Set("theme", "Light"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 初始值来自存储而非默认值。
	themeName, setTheme := UseStorage(store, "theme", "Dark")
	if got := themeName.Get(); got != "Light" {
		t.Fatalf("initial = %q, want Light", got)
	}

	// 写入同时更新 ref 与落盘。
	setTheme("Tabby")
	if got := themeName.Get(); got != "Tabby" {
		t.Fatalf("ref = %q, want Tabby", got)
	}
	reopened, err := storage.Open(store.Path())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.GetString("theme", ""); got != "Tabby" {
		t.Fatalf("persisted = %q, want Tabby", got)
	}
}

func TestUseStorageNilStoreFallsBackToMemoryOnly(t *testing.T) {
	value, set := UseStorage[int](nil, "counter", 5)
	if got := value.Get(); got != 5 {
		t.Fatalf("initial = %d, want 5", got)
	}
	set(9) // 不得 panic
	if got := value.Get(); got != 9 {
		t.Fatalf("ref = %d, want 9", got)
	}
}

func TestUseStorageTypeMismatchFallsBackToDefault(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "goed", "storage.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = store.Set("flag", "yes") // 类型不匹配
	value, _ := UseStorage(store, "flag", true)
	if got := value.Get(); got != true {
		t.Fatalf("value = %v, want default true", got)
	}
}
