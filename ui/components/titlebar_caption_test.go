package components

import (
	"testing"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// platform.CaptionButtons 是个「失灵了也不会有任何断言失败」的开关：
// 浏览器里多出三个点不动的死按钮、桌面上少掉三个真按钮，只能靠人眼在截图里
// 发现。这三条用例把它钉住。
//
// 判据为什么是「有没有往画布外画」而不是「右侧那一带有没有墨迹」：
// 守卫失效时按钮并不是留在原地照画，而是被算到 x = w（画布外），所有后端都会
// 裁掉，区间检查查不出来。第一版就是这么写的，故意把守卫改成 `if true` 验证时
// 三条全绿——空转的。

const titleBarTestW float32 = 800

// paintTitleForTest 在指定开关下画一次标题栏，返回记录到的绘制。
func paintTitleForTest(t *testing.T, captionButtons bool, hover int) *mockContext {
	t.Helper()
	old := platform.CaptionButtons
	platform.CaptionButtons = captionButtons
	t.Cleanup(func() { platform.CaptionButtons = old })

	ctx := &mockContext{w: titleBarTestW, h: 600}
	n := runtime.H("titlebar", runtime.Props{Title: "Goed Editor"})
	if hover != 0 {
		if n.Attrs == nil {
			n.Attrs = runtime.Attrs{}
		}
		n.Attrs["hover"] = hover
	}
	paintTitleBar(ctx, n, renderer.TextStyle{}, theme.Dark)
	return ctx
}

// outsideFills 挑出画到画布右边的填充。
func outsideFills(ctx *mockContext, w float32) []mockRound {
	var out []mockRound
	for _, f := range ctx.fills {
		if f.x >= w {
			out = append(out, f)
		}
	}
	return out
}

func TestTitleBarShowsCaptionButtonsByDefault(t *testing.T) {
	if !platform.CaptionButtons {
		t.Fatal("platform.CaptionButtons 的零值/默认应为 true（桌面端有系统按钮）")
	}

	// hover=1 是最小化按钮：它应当画出一块悬停底色，位置在最右三个格子的第一个。
	ctx := paintTitleForTest(t, true, 1)
	btnW := float32(renderer.CaptionButtonWidth)
	wantX := titleBarTestW - 3*btnW
	hover := renderer.ColorFrom(theme.Dark.TitlebarHover)

	for _, f := range ctx.fills {
		if f.c == hover && f.x == wantX {
			return
		}
	}
	t.Fatalf("最小化按钮的悬停底色没画在 x=%.1f：fills=%v", wantX, ctx.fills)
}

func TestTitleBarHidesCaptionButtonsWhenHostHasNone(t *testing.T) {
	ctx := paintTitleForTest(t, false, 1)
	if got := outsideFills(ctx, titleBarTestW); len(got) != 0 {
		t.Fatalf("宿主没有标题栏按钮时仍往画布外画了 %d 个矩形：%v", len(got), got)
	}
}

func TestTitleBarCaptionHoverFollowsSwitch(t *testing.T) {
	closeHover := renderer.ColorFrom(theme.Dark.CloseHover)

	// 开关打开：hover=3 是关闭按钮，给的是关闭专用悬停色。
	on := paintTitleForTest(t, true, 3)
	found := false
	for _, f := range on.fills {
		if f.c == closeHover {
			found = true
		}
	}
	if !found {
		t.Fatalf("开关打开时关闭按钮没画关闭悬停色：fills=%v", on.fills)
	}

	// 开关关闭：一个都不该画。
	off := paintTitleForTest(t, false, 3)
	for _, f := range off.fills {
		if f.c == closeHover {
			t.Fatalf("开关关闭时仍画了关闭按钮的悬停色")
		}
	}
	if got := outsideFills(off, titleBarTestW); len(got) != 0 {
		t.Fatalf("开关关闭时仍往画布外画了 %d 个矩形：%v", len(got), got)
	}
}
