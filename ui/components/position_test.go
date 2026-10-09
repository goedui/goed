package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestMarginOffsetsFlowChild(t *testing.T) {
	child := runtime.H("box", runtime.Props{Style: runtime.Style{Width: 40, Height: 20, Margin: 8}})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{Style: runtime.Style{Width: 200, Height: 80}}, child),
	)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	if child.X != host.X+8 || child.Y != host.Y+8 {
		t.Fatalf("margin 应把子节点推开: child=%+v host=%+v", child, host)
	}
	if child.W != 40 {
		t.Fatalf("显式宽度不应被 margin 吃掉: %.1f", child.W)
	}
}

func TestAbsoluteLeavesFlowAndUsesContainingBlock(t *testing.T) {
	abs := runtime.H("box", runtime.Props{
		Style: runtime.Style{
			Position: runtime.PositionAbsolute,
			Width:    30, Height: 16,
			Top: 4, Right: 6,
		},
	})
	flow := runtime.H("box", runtime.Props{Style: runtime.Style{Height: 20}})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{
			Style: runtime.Style{Width: 200, Height: 100, Position: runtime.PositionRelative, Padding: 10},
		}, abs, flow),
	)
	renderer.PaintTree(&mockContext{w: 400, h: 300}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	if flow.Y != host.Y+10 {
		t.Fatalf("绝对定位不应把后面的流式节点顶下去: flow.Y=%.1f host.Y=%.1f", flow.Y, host.Y)
	}
	wantX := host.X + 10 + (host.W - 20) - 6 - 30
	wantY := host.Y + 10 + 4
	if abs.X != wantX || abs.Y != wantY || abs.W != 30 || abs.H != 16 {
		t.Fatalf("应相对定位祖先的 padding 边钉右上: got (%.1f,%.1f,%.1f,%.1f) want (%.1f,%.1f,30,16)",
			abs.X, abs.Y, abs.W, abs.H, wantX, wantY)
	}
}

func TestAbsoluteStretchWhenBothInsetsSet(t *testing.T) {
	abs := runtime.H("box", runtime.Props{
		Style: runtime.Style{
			Position: runtime.PositionAbsolute,
			Left:     8, Right: 8, Top: 4, Bottom: 4,
			Inset: runtime.InsetAll,
		},
	})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{
			Style: runtime.Style{Width: 100, Height: 60, Position: runtime.PositionRelative},
		}, abs),
	)
	renderer.PaintTree(&mockContext{w: 200, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	if abs.X != host.X+8 || abs.W != 84 || abs.Y != host.Y+4 || abs.H != 52 {
		t.Fatalf("对边都写了应拉满: %+v host=%+v", abs, host)
	}
}

func TestRelativeShiftsWithoutChangingFlowSlot(t *testing.T) {
	a := runtime.H("box", runtime.Props{Style: runtime.Style{Height: 20, Position: runtime.PositionRelative, Left: 12, Top: 5}})
	b := runtime.H("box", runtime.Props{Style: runtime.Style{Height: 20}})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{Style: runtime.Style{Width: 100}}, a, b),
	)
	renderer.PaintTree(&mockContext{w: 200, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	if a.X != host.X+12 || a.Y != host.Y+5 {
		t.Fatalf("relative 应平移: %+v", a)
	}
	if b.Y != host.Y+20 {
		t.Fatalf("relative 仍应占原来的流位置，后继不应跟着移: b.Y=%.1f", b.Y)
	}
}
