//go:build windows

package windows

import "testing"

func TestHitTestCaptionButtons(t *testing.T) {
	const w, h, border, title, btn = int32(800), int32(600), int32(8), int32(32), int32(46)

	if g := hitTest(10, 16, w, h, border, title, btn, false); g != htCaption {
		t.Fatalf("标题栏空白应为 HTCAPTION，得到 %d", g)
	}
	if g := hitTest(w-10, 16, w, h, border, title, btn, false); g != htClose {
		t.Fatalf("关闭按钮应为 HTCLOSE，得到 %d", g)
	}
	if g := hitTest(w-btn-10, 16, w, h, border, title, btn, false); g != htMaxButton {
		t.Fatalf("最大化按钮应为 HTMAXBUTTON，得到 %d", g)
	}
	if g := hitTest(w-2*btn-10, 16, w, h, border, title, btn, false); g != htMinButton {
		t.Fatalf("最小化按钮应为 HTMINBUTTON，得到 %d", g)
	}
	if g := hitTest(400, 200, w, h, border, title, btn, false); g != htClient {
		t.Fatalf("内容区应为 HTCLIENT，得到 %d", g)
	}
	if g := hitTest(0, 100, w, h, border, title, btn, false); g != htLeft {
		t.Fatalf("左边缘应为 HTLEFT，得到 %d", g)
	}
	if g := hitTest(10, 0, w, h, border, title, btn, false); g != htTop {
		t.Fatalf("上边缘应为 HTTOP，得到 %d", g)
	}
	if g := hitTest(0, 0, w, h, border, title, btn, false); g != htTopLeft {
		t.Fatalf("左上角应为 HTTOPLEFT，得到 %d", g)
	}
	if g := hitTest(0, 16, w, h, border, title, btn, true); g != htCaption {
		t.Fatalf("最大化后边缘不应再缩放，得到 %d", g)
	}
}
