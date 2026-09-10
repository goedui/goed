package linux

import (
	"image"
	"testing"
)

func TestClipStackPushIntersectsAndPopRestores(t *testing.T) {
	g := NewGDIContext(0, 40, 40)

	g.PushClipRect(2, 4, 20, 20)
	if g.clip != image.Rect(2, 4, 22, 24) {
		t.Fatalf("clip = %v, want (2,4)-(22,24)", g.clip)
	}

	g.PushClipRect(16, 18, 20, 20)
	if g.clip != image.Rect(16, 18, 22, 24) {
		t.Fatalf("nested clip = %v, want the intersection (16,18)-(22,24)", g.clip)
	}

	g.PopClip()
	if g.clip != image.Rect(2, 4, 22, 24) {
		t.Fatalf("clip after pop = %v, want the outer clip back", g.clip)
	}

	g.PopClip()
	if !g.clip.Empty() {
		t.Fatalf("clip after the last pop = %v, want no clipping", g.clip)
	}

	// 多余的 PopClip 不应改变状态
	g.PopClip()
	if !g.clip.Empty() || len(g.clipStack) != 0 {
		t.Fatal("an unbalanced pop must be a no-op")
	}
}

func TestPushClipOutsideSurfaceClipsEverything(t *testing.T) {
	g := NewGDIContext(0, 8, 8)
	g.BeginDraw()
	g.PushClipRect(20, 20, 5, 5)
	g.DrawRect(0, 0, 8, 8, 0x00FF0000, true)
	for i, p := range g.pixels {
		if p != g.clearColor {
			t.Fatalf("pixel %d = %#x, want the clear color: an empty clip must hide everything", i, p)
		}
	}
}

func TestSetClipRectResetsStack(t *testing.T) {
	g := NewGDIContext(0, 16, 16)
	g.PushClipRect(1, 1, 4, 4)
	g.SetClipRect(6, 6, 4, 4)
	if g.clip != image.Rect(6, 6, 10, 10) || len(g.clipStack) != 0 {
		t.Fatalf("clip = %v, stack = %d, want the hard clip to replace the stack", g.clip, len(g.clipStack))
	}
	g.ResetClip()
	if !g.clip.Empty() || len(g.clipStack) != 0 {
		t.Fatal("ResetClip must drop every stacked clip")
	}
}

func TestClippedFillOnlyTouchesClipArea(t *testing.T) {
	g := NewGDIContext(0, 10, 10)
	g.BeginDraw()
	g.SetClipRect(3, 3, 4, 4)
	g.DrawRect(0, 0, 10, 10, 0x00FF0000, true)

	painted := func(x, y int) bool { return g.pixels[y*10+x] == 0x00FF0000 }
	if painted(2, 2) || painted(7, 7) || painted(5, 2) {
		t.Fatal("drawing leaked outside the clip rectangle")
	}
	if !painted(3, 3) || !painted(6, 6) {
		t.Fatal("the clip rectangle was not filled")
	}
}
