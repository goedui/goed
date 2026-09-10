//go:build windows

package windows

import (
	"testing"
)

func TestDrawEllipseHonorsFilled(t *testing.T) {
	dc := GetDC(0)
	if dc == 0 {
		t.Skip("no GDI device context")
	}
	defer ReleaseDC(0, dc)
	g := NewGDIContext(dc, 40, 40)
	defer g.Cleanup()
	g.SetBackgroundColor(0xFFFFFF)
	g.BeginDraw()
	getPixel := gdi32.NewProc("GetPixel")
	g.DrawEllipse(5, 5, 30, 30, 0x0000FF, false)
	center, _, _ := getPixel.Call(uintptr(g.memDC), 20, 20)
	if center != 0xFFFFFF {
		t.Fatalf("outline filled center: %x", center)
	}
	g.DrawEllipse(5, 5, 30, 30, 0x0000FF, true)
	center, _, _ = getPixel.Call(uintptr(g.memDC), 20, 20)
	if center != 0x0000FF {
		t.Fatalf("filled ellipse center: %x", center)
	}
}
