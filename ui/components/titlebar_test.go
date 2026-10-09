package components

import (
	"testing"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestTitleBarRendersFrame(t *testing.T) {
	platform.SetFrame(platform.FrameState{
		Title:     "HelloWorld",
		Maximized: true,
		Active:    true,
		Hover:     platform.CaptionClose,
	})
	inst := runtime.Mount(TitleBar, nil)
	nodes := inst.Render()
	if len(nodes) != 1 || nodes[0].Tag != "titlebar" {
		t.Fatalf("期望 titlebar 元素，得到 %+v", nodes)
	}
	if nodes[0].Attrs["title"] != "HelloWorld" {
		t.Fatalf("标题未传入: %+v", nodes[0].Attrs)
	}
	if nodes[0].Attrs["maximized"] != true {
		t.Fatal("最大化状态未传入")
	}
	if nodes[0].Attrs["hover"] != int(platform.CaptionClose) {
		t.Fatal("关闭按钮悬停状态未传入")
	}
	if nodes[0].Attrs["icon"] == nil {
		t.Fatal("默认应用图标未传入标题栏")
	}
}

func TestTitleBarAcceptsStyle(t *testing.T) {
	platform.SetFrame(platform.FrameState{Title: "imgen生图", Active: true})
	bg := runtime.RGB(246, 248, 252)
	fg := runtime.RGB(24, 30, 38)
	inst := runtime.MountWith(TitleBar, nil, runtime.Props{Style: runtime.Style{
		Background: bg, Color: fg, FontFamily: "Microsoft YaHei UI", FontSize: 13, Weight: 600,
	}})
	nodes := inst.Render()
	if len(nodes) != 1 || nodes[0].Style.Background != bg || nodes[0].Style.Color != fg {
		t.Fatalf("style 未落到标题栏: %+v", nodes)
	}
	ctx := &mockContext{w: 800, h: 600}
	paintTitleBar(ctx, nodes[0], renderer.TextStyle{FontFamily: "Microsoft YaHei UI", FontSize: 13, Weight: 600}, theme.Light)
	if len(ctx.fills) == 0 || ctx.fills[0].c != renderer.ColorFrom(bg) {
		t.Fatalf("标题栏底色未用 style: %+v", ctx.fills)
	}
	found := false
	for _, d := range ctx.draws {
		if d.text == "imgen生图" && d.style.Color == renderer.ColorFrom(fg) && d.style.FontSize == 13 {
			found = true
		}
	}
	if !found {
		t.Fatalf("标题文字未按 style 绘制: %+v", ctx.draws)
	}
}

// TestMeasureTitleBarWidthFallback 标题栏注册表里那个 Measure 的宽度来源：
// maxW → ctx.Width() → 0；高度恒为 renderer.TitleBarHeight。
//
// 为什么值得单测：**正常渲染路径根本不调它**。PaintTree 认出标题栏后走
// renderer.paintFrame（frame.go），那里直接取常量 TitleBarHeight 并调 Paint，
// 既不 Measure 也不 Layout。所以这个函数只在「有人显式 MeasureNode 一个
// titlebar」时才会执行 —— 从前的测试一次都没走到过。
//
// 留这条用例是为了把它的高度与**框架实际使用的高度**钉在一起：将来谁把框架
// 改成走 Measure，两边不一致会立刻红，而不是让标题栏高度悄悄跳一下。
func TestMeasureTitleBarWidthFallback(t *testing.T) {
	ctx := &mockContext{w: 500, h: 400}
	n := runtime.Tag("titlebar", runtime.Attrs{"title": "x"})
	st := renderer.TextStyle{FontSize: 13}

	cases := []struct {
		name  string
		ctx   renderer.Context
		maxW  float32
		wantW float32
	}{
		{"有可用宽度就用它", ctx, 300, 300},
		{"可用宽度为 0 → 退回窗口宽度", ctx, 0, 500},
		{"可用宽度为负 → 退回窗口宽度", ctx, -5, 500},
		{"ctx 为 nil 时没得退 → 0", nil, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, h := measureTitleBar(c.ctx, n, c.maxW, 100, st, theme.Dark)
			if w != c.wantW {
				t.Fatalf("宽度 = %g，期望 %g", w, c.wantW)
			}
			if h != renderer.TitleBarHeight {
				t.Fatalf("高度 = %g，期望与框架实际用的 renderer.TitleBarHeight=%d 一致",
					h, renderer.TitleBarHeight)
			}
		})
	}
}
