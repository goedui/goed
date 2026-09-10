//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"testing"
)

func guiObjectCount(t *testing.T) uintptr {
	t.Helper()
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	count, _, _ := user32.NewProc("GetGuiResources").Call(uintptr(process), 0)
	return count
}

func TestNestedClipDoesNotLeakGDIRegions(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc := GetDC(0)
	if dc == 0 {
		t.Skip("no GDI device context")
	}
	defer ReleaseDC(0, dc)
	g := NewGDIContext(dc, 80, 80)
	defer g.Cleanup()
	g.SetClipRect(0, 0, 70, 70)
	// Warm up GDI's lazy allocations and drain its per-thread command batch.
	for n := 0; n < 40; n++ {
		g.PushClipRect(5, 5, 60, 60)
		g.PushClipRect(10, 10, 20, 20)
		g.PopClip()
		g.PopClip()
	}
	gdi32.NewProc("GdiFlush").Call()
	before := guiObjectCount(t)
	for n := 0; n < 400; n++ {
		g.PushClipRect(5, 5, 60, 60)
		g.PushClipRect(10, 10, 20, 20)
		g.PopClip()
		g.PopClip()
	}
	gdi32.NewProc("GdiFlush").Call()
	after := guiObjectCount(t)
	if after > before+2 {
		t.Fatalf("nested clips leaked GDI objects: before=%d after=%d growth=%d", before, after, after-before)
	}
}

func TestNestedClipRestoresParentAndEmptyRegions(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc := GetDC(0)
	if dc == 0 {
		t.Skip("no GDI device context")
	}
	defer ReleaseDC(0, dc)
	g := NewGDIContext(dc, 80, 80)
	defer g.Cleanup()
	g.SetBackgroundColor(0xFFFFFF)
	g.BeginDraw()
	g.SetClipRect(5, 5, 60, 60)
	g.PushClipRect(10, 10, 40, 40)
	g.PushClipRect(100, 100, 10, 10)
	g.DrawRect(0, 0, 80, 80, 0x0000FF, true)
	g.PopClip()
	g.DrawRect(0, 0, 80, 80, 0x00FF00, true)
	g.PopClip()
	g.DrawRect(55, 55, 25, 25, 0xFF0000, true)
	g.ResetClip()
	getPixel := gdi32.NewProc("GetPixel")
	for _, tc := range []struct {
		x, y int
		want uintptr
	}{{2, 2, 0xFFFFFF}, {15, 15, 0x00FF00}, {60, 60, 0xFF0000}, {75, 75, 0xFFFFFF}} {
		got, _, _ := getPixel.Call(uintptr(g.memDC), uintptr(tc.x), uintptr(tc.y))
		if got != tc.want {
			t.Fatalf("pixel(%d,%d)=%x want %x", tc.x, tc.y, got, tc.want)
		}
	}
}
