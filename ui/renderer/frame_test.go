package renderer

import (
	"testing"

	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// 标题栏那一条决定应用树从哪里开始排：框架自绘时占 32 DIP，应用自绘时不占。
//
// 这条断言值钱在：「应用自绘标题栏」是两件独立的事 —— 框架少占高度（在这里）、
// 宿主不再把整条判成 HTCAPTION（在 ui/platform/windows 的 hitTest 里）。少了任何
// 一件都不会报错：前者漏了，应用把标签条画在 y=0 上、内容却仍从 32 起排，
// 现象是「顶部那一条被压掉一半」；后者漏了，标签条画对了但点不动。
func TestTitleBarOffsetOwnership(t *testing.T) {
	for _, c := range []struct {
		name  string
		attrs runtime.Attrs
		wantY float32
	}{
		{"框架自绘标题栏", runtime.Attrs{"title": "T"}, TitleBarHeight},
		{"应用自绘标题栏", runtime.Attrs{"title": "T", "apptitle": true}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			const (
				w = float32(400)
				h = float32(300)
			)
			ctx := &mockContext{w: w, h: h}
			tb := runtime.Tag("titlebar", c.attrs)
			root := runtime.H("vstack", runtime.Props{Style: runtime.Style{Flex: 1}})
			PaintTree(ctx, []*runtime.VNode{tb, root}, theme.Light)

			if root.Y != c.wantY {
				t.Fatalf("应用树从 y=%.0f 起，期望 %.0f", root.Y, c.wantY)
			}
			if want := h - c.wantY; root.H != want {
				t.Fatalf("应用树高度 %.0f，期望 %.0f（客户区 %.0f − 标题栏 %.0f）",
					root.H, want, h, c.wantY)
			}
		})
	}
}
