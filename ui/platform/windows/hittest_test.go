//go:build windows

package windows

import "testing"

// hit 是 hitTest 的简写：默认「应用不占用标题栏」（clientHit 为 nil，dpi 96）。
func hit(px, py, cw, ch, border, titleH, btnW int32, maximized bool) uintptr {
	return hitTest(px, py, cw, ch, border, titleH, btnW, maximized, nil, 96)
}

func TestHitTestCaptionButtons(t *testing.T) {
	const w, h, border, title, btn = int32(800), int32(600), int32(8), int32(32), int32(46)

	if g := hit(10, 16, w, h, border, title, btn, false); g != htCaption {
		t.Fatalf("标题栏空白应为 HTCAPTION，得到 %d", g)
	}
	if g := hit(w-10, 16, w, h, border, title, btn, false); g != htClose {
		t.Fatalf("关闭按钮应为 HTCLOSE，得到 %d", g)
	}
	if g := hit(w-btn-10, 16, w, h, border, title, btn, false); g != htMaxButton {
		t.Fatalf("最大化按钮应为 HTMAXBUTTON，得到 %d", g)
	}
	if g := hit(w-2*btn-10, 16, w, h, border, title, btn, false); g != htMinButton {
		t.Fatalf("最小化按钮应为 HTMINBUTTON，得到 %d", g)
	}
	if g := hit(400, 200, w, h, border, title, btn, false); g != htClient {
		t.Fatalf("内容区应为 HTCLIENT，得到 %d", g)
	}
	if g := hit(0, 100, w, h, border, title, btn, false); g != htLeft {
		t.Fatalf("左边缘应为 HTLEFT，得到 %d", g)
	}
	if g := hit(10, 0, w, h, border, title, btn, false); g != htTop {
		t.Fatalf("上边缘应为 HTTOP，得到 %d", g)
	}
	if g := hit(0, 0, w, h, border, title, btn, false); g != htTopLeft {
		t.Fatalf("左上角应为 HTTOPLEFT，得到 %d", g)
	}
	if g := hit(0, 16, w, h, border, title, btn, true); g != htCaption {
		t.Fatalf("最大化后边缘不应再缩放，得到 %d", g)
	}
}

// TestHitTestClientHitInCaption 是「应用自绘标题栏」这条路的判据。
//
// 应用把标签条画进顶部那一条之后，宿主如果还把整条判成 HTCAPTION，标签就永远
// 点不动 —— 而且症状很隐蔽：拖动窗口正常、系统按钮正常，只有标签没反应。
// 所以这里逐条钉住：应用声明过的地方给 HTCLIENT，没声明的仍然是 HTCAPTION，
// 三个系统按钮的位置**优先于**应用声明（它们由系统执行，不能被应用抢走）。
func TestHitTestClientHitInCaption(t *testing.T) {
	const w, h, border, title, btn = int32(800), int32(600), int32(8), int32(32), int32(46)

	// 应用声明 x < 300 是自己的控件（模拟一条标签条）
	clientHit := func(x, y float32) bool { return x < 300 }
	at := func(px, py int32) uintptr {
		return hitTest(px, py, w, h, border, title, btn, false, clientHit, 96)
	}

	if g := at(100, 16); g != htClient {
		t.Fatalf("应用声明过的地方应给 HTCLIENT，得到 %d", g)
	}
	if g := at(400, 16); g != htCaption {
		t.Fatalf("应用没声明的地方仍应是 HTCAPTION（否则窗口拖不动），得到 %d", g)
	}
	// 系统按钮优先：应用就算把整条都声明了，也抢不走这三个位置
	always := func(x, y float32) bool { return true }
	for _, c := range []struct {
		px   int32
		want uintptr
		name string
	}{
		{w - 10, htClose, "关闭"},
		{w - btn - 10, htMaxButton, "最大化"},
		{w - 2*btn - 10, htMinButton, "最小化"},
	} {
		if g := hitTest(c.px, 16, w, h, border, title, btn, false, always, 96); g != c.want {
			t.Errorf("应用声明整条时 %s 按钮仍应是 %d，得到 %d", c.name, c.want, g)
		}
	}
	// 标题栏区之外不受影响
	if g := at(100, 200); g != htClient {
		t.Fatalf("内容区应为 HTCLIENT，得到 %d", g)
	}
}

// TestHitTestClientHitDPI hitTest 收的是**物理像素**，而 ClientHitTest 收的是 DIP。
// 这条用例钉住那个换算：忘了换的话 125% 缩放下「声明过的区域」会整体偏右下，
// 表现为「标签条只有最左边一小截能点」。
func TestHitTestClientHitDPI(t *testing.T) {
	const w, h, border, title, btn = int32(1585), int32(966), int32(10), int32(40), int32(58)

	var gotX, gotY float32
	clientHit := func(x, y float32) bool { gotX, gotY = x, y; return true }
	hitTest(250, 20, w, h, border, title, btn, false, clientHit, 125)

	// 期望值写成字面量而不是再算一遍 96/dpi：算一遍就成了同义反复。
	// 250px @125% = 192 DIP，20px @125% = 15.36 DIP。
	const tol = float32(0.01)
	if gotX < 192-tol || gotX > 192+tol {
		t.Errorf("x 应换算成 DIP：得到 %.2f，期望 192", gotX)
	}
	if gotY < 15.36-tol || gotY > 15.36+tol {
		t.Errorf("y 应换算成 DIP：得到 %.2f，期望 15.36", gotY)
	}
}
