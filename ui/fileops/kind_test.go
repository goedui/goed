package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectKindByExtension(t *testing.T) {
	dir := t.TempDir()
	if got := DetectKind(write(t, dir, "a.png", []byte("x"))); got != KindImage {
		t.Fatalf("png = %v", got)
	}
	if got := DetectKind(write(t, dir, "a.JPEG", []byte("x"))); got != KindImage {
		t.Fatalf("JPEG (case-insensitive) = %v", got)
	}
	if got := DetectKind(write(t, dir, "a.exe", []byte("text?"))); got != KindBinary {
		t.Fatalf("exe = %v", got)
	}
	if got := DetectKind(write(t, dir, "a.zip", []byte("PK"))); got != KindBinary {
		t.Fatalf("zip = %v", got)
	}
}

func TestDetectKindByContentSniff(t *testing.T) {
	dir := t.TempDir()
	// 无扩展名 + NUL 字节 → 二进制。
	bin := write(t, dir, "blob", []byte{'M', 'Z', 0x00, 0x01})
	if got := DetectKind(bin); got != KindBinary {
		t.Fatalf("nul sniff = %v, want KindBinary", got)
	}
	// 无 NUL → 文本。
	txt := write(t, dir, "note", []byte("hello world\n"))
	if got := DetectKind(txt); got != KindText {
		t.Fatalf("text sniff = %v, want KindText", got)
	}
}

func TestLoadPreviewTruncates(t *testing.T) {
	dir := t.TempDir()
	content := make([]byte, 100)
	for i := range content {
		content[i] = byte(i)
	}
	path := write(t, dir, "data.bin", content)

	data, total, truncated, err := LoadPreview(path, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 40 || !truncated || total != 100 {
		t.Fatalf("len=%d truncated=%v total=%d, want 40/true/100", len(data), truncated, total)
	}
	data, total, truncated, err = LoadPreview(path, 100)
	if err != nil || len(data) != 100 || truncated || total != 100 {
		t.Fatalf("exact fit: len=%d truncated=%v total=%d err=%v", len(data), truncated, total, err)
	}
}

func TestFormatSize(t *testing.T) {
	if got := FormatSize(512); got != "512 B" {
		t.Fatalf("512 = %q", got)
	}
	if got := FormatSize(2048); got != "2.0 KB" {
		t.Fatalf("2048 = %q", got)
	}
	if got := FormatSize(3 * 1024 * 1024); got != "3.0 MB" {
		t.Fatalf("3MB = %q", got)
	}
}
