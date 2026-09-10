package components

import "testing"

// 1000 行、视口 20 行、minimap 高 300：
// 总高 1500 > 300 => scale=0.2, rowPx=0.3。

func TestMinimapMouseDownOutsideViewportCenters(t *testing.T) {
	m := NewMinimap()
	// 点击 y=150 => 行 500，居中 => 500-10=490
	if got := m.HandleMouseDown(150, 300, 1000, 20, 0); got != 490 {
		t.Fatalf("scroll after outside click = %d, want 490", got)
	}
}

func TestMinimapMouseDownInsideViewportKeepsScroll(t *testing.T) {
	m := NewMinimap()
	// scroll=100 => 视口顶 30px、高 6px；y=32 落在视口内，滚动不变
	if got := m.HandleMouseDown(32, 300, 1000, 20, 100); got != 100 {
		t.Fatalf("scroll after inside click = %d, want 100", got)
	}
	// 从按下点向下拖 30px => 100 行
	if got := m.HandleDragMove(62, 300, 1000, 20); got != 200 {
		t.Fatalf("scroll after drag = %d, want 200", got)
	}
}

func TestMinimapDragClampedToDocument(t *testing.T) {
	m := NewMinimap()
	m.HandleMouseDown(5, 300, 1000, 20, 0) // 视口内（顶部 0..6px）
	if got := m.HandleDragMove(-500, 300, 1000, 20); got != 0 {
		t.Fatalf("top clamp = %d, want 0", got)
	}
	if got := m.HandleDragMove(100000, 300, 1000, 20); got != 980 {
		t.Fatalf("bottom clamp = %d, want 980", got)
	}
}
