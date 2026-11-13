package components

import (
	"testing"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestWindowCaptionPaintsHoverFromFrame(t *testing.T) {
	old := platform.CaptionButtons
	platform.CaptionButtons = true
	t.Cleanup(func() { platform.CaptionButtons = old })

	platform.SetFrame(platform.FrameState{Hover: platform.CaptionMin, Active: true})
	n := WindowCaption()
	if n == nil || n.Tag != "caption" || n.OnClick != nil {
		t.Fatalf("系统按钮不该带 OnClick：%+v", n)
	}
	ctx := &mockContext{w: 800, h: 32}
	n.X, n.Y, n.W, n.H = 800-3*46, 0, 3*46, 32
	paintCaption(ctx, n, renderer.TextStyle{}, theme.Dark)

	hover := renderer.ColorFrom(theme.Dark.TitlebarHover)
	wantX := float32(800 - 3*renderer.CaptionButtonWidth)
	for _, f := range ctx.fills {
		if f.c == hover && f.x == wantX {
			return
		}
	}
	t.Fatalf("最小化悬停没画在 x=%.0f：%v", wantX, ctx.fills)
}

func TestWindowCaptionNilWithoutHostButtons(t *testing.T) {
	old := platform.CaptionButtons
	platform.CaptionButtons = false
	t.Cleanup(func() { platform.CaptionButtons = old })
	if WindowCaption() != nil {
		t.Fatal("浏览器上不该再画三个死按钮")
	}
}

func TestMenuBarReservesCaptionStrip(t *testing.T) {
	old := platform.CaptionButtons
	platform.CaptionButtons = true
	t.Cleanup(func() { platform.CaptionButtons = old })

	bar := MenuBar(runtime.Props{
		Style: runtime.Style{Height: 32, Padding: 3, Position: runtime.PositionRelative},
		Menus: []Menu{{ID: "file", Label: "文件"}},
	}, WindowCaption())
	ctx := &mockContext{w: 800, h: 600}
	renderer.LayoutNode(ctx, bar, 0, 0, 800, 32, renderer.TextStyle{FontSize: 13}, theme.Dark)

	var cap *runtime.VNode
	runtime.Walk([]*runtime.VNode{bar}, func(n *runtime.VNode) bool {
		if n != nil && n.Tag == "caption" {
			cap = n
		}
		return false
	})
	menu := runtime.FindByID([]*runtime.VNode{bar}, "file")
	hitLeft := float32(800 - 3*renderer.CaptionButtonWidth)
	if cap == nil || cap.X != hitLeft || cap.Y != 0 {
		t.Fatalf("三按钮没钉在窗口右缘：%+v", cap)
	}
	if menu == nil {
		t.Fatal("菜单项丢了")
	}
	if menu.X+menu.W > hitLeft {
		t.Fatalf("菜单项伸进了系统按钮命中区：menu 右缘 %.0f，命中区左缘 %.0f", menu.X+menu.W, hitLeft)
	}
	platform.SetFrame(platform.FrameState{Hover: platform.CaptionClose, Active: true})
	paintCaption(ctx, cap, renderer.TextStyle{}, theme.Dark)
	for _, f := range ctx.fills {
		if f.c == renderer.ColorFrom(theme.Dark.CloseHover) && f.x == hitLeft+2*float32(renderer.CaptionButtonWidth) {
			return
		}
	}
	t.Fatalf("关闭按钮没画在窗口右缘 %.0f：%v", hitLeft+2*float32(renderer.CaptionButtonWidth), ctx.fills)
}
