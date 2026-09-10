package windows

import "testing"

func TestUTF16FilterBufferIsDoubleNulTerminated(t *testing.T) {
	buffer := utf16Buffer("Text\x00*.txt\x00All\x00*.*")
	if len(buffer) < 2 || buffer[len(buffer)-1] != 0 || buffer[len(buffer)-2] != 0 {
		t.Fatalf("filter buffer is not double-NUL terminated: %#v", buffer)
	}
}

func TestSanitizeWindowsTextRemovesEmbeddedNUL(t *testing.T) {
	got := sanitizeWindowsText("before\x00after")
	if got != "before\uFFFDafter" {
		t.Fatalf("sanitizeWindowsText() = %q", got)
	}
}
