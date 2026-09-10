//go:build windows

package windows

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestFontQualityOptIn(t *testing.T) {
	dc := GetDC(0)
	if dc == 0 {
		t.Skip("no GDI device context")
	}
	defer ReleaseDC(0, dc)
	g := NewGDIContext(dc, 100, 50)
	defer g.Cleanup()
	check := func(clear bool, want uint32) {
		g.SetClearType(clear)
		g.SetFont("Microsoft YaHei UI", 15)
		h := g.fonts[fontKey{name: "Microsoft YaHei UI", size: 15, quality: want}]
		if h == 0 {
			t.Fatalf("font quality %d not cached", want)
		}
		// LOGFONTW: five LONGs, eight BYTEs, then a 32-WCHAR face name.
		var lf struct {
			Height, Width, Escapement, Orientation, Weight                                     int32
			Italic, Underline, StrikeOut, CharSet, OutPrecision, ClipPrecision, Quality, Pitch byte
			Face                                                                               [32]uint16
		}
		n, _, _ := gdi32.NewProc("GetObjectW").Call(uintptr(h), unsafe.Sizeof(lf), uintptr(unsafe.Pointer(&lf)))
		if n == 0 || uint32(lf.Quality) != want || lf.Weight != 400 {
			t.Fatalf("font quality=%d weight=%d read=%d", lf.Quality, lf.Weight, n)
		}
	}
	check(false, fontAntialiased)
	check(true, fontClearType)
	check(false, fontAntialiased)
	// Legacy callers retain grayscale anti-aliasing.
	f := CreateFont("Consolas", 15)
	if f == syscall.Handle(0) {
		t.Fatal("legacy CreateFont failed")
	}
	DeleteObject(f)
}
