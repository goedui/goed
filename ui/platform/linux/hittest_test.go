//go:build linux

package linux

import "testing"

func TestHitTestCaptionButtons(t *testing.T) {
	const w, h, border, title, btn = 800, 600, 8, 32, 46

	if g := hitTest(10, 16, w, h, border, title, btn, false); g != hitCaption {
		t.Fatalf("标题栏空白应为 caption，得到 %d", g)
	}
	if g := hitTest(w-10, 16, w, h, border, title, btn, false); g != hitClose {
		t.Fatalf("关闭按钮应为 close，得到 %d", g)
	}
	if g := hitTest(w-btn-10, 16, w, h, border, title, btn, false); g != hitMax {
		t.Fatalf("最大化按钮应为 max，得到 %d", g)
	}
	if g := hitTest(w-2*btn-10, 16, w, h, border, title, btn, false); g != hitMin {
		t.Fatalf("最小化按钮应为 min，得到 %d", g)
	}
	if g := hitTest(400, 200, w, h, border, title, btn, false); g != hitClient {
		t.Fatalf("内容区应为 client，得到 %d", g)
	}
	if g := hitTest(0, 100, w, h, border, title, btn, false); g != hitLeft {
		t.Fatalf("左边缘应为 left，得到 %d", g)
	}
	if g := hitTest(10, 0, w, h, border, title, btn, false); g != hitTop {
		t.Fatalf("上边缘应为 top，得到 %d", g)
	}
	if g := hitTest(0, 0, w, h, border, title, btn, false); g != hitTopLeft {
		t.Fatalf("左上角应为 top-left，得到 %d", g)
	}
	if g := hitTest(0, 16, w, h, border, title, btn, true); g != hitCaption {
		t.Fatalf("最大化后边缘不应再缩放，得到 %d", g)
	}
}

func TestHitToCaption(t *testing.T) {
	if hitToCaption(hitMin) != 1 || hitToCaption(hitMax) != 2 || hitToCaption(hitClose) != 3 {
		t.Fatal("caption 编号应对齐 platform.CaptionButton")
	}
	if hitToCaption(hitCaption) != 0 || hitToCaption(hitClient) != 0 {
		t.Fatal("非按钮命中不应映射成 caption button")
	}
}
