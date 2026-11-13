package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestMarginLeftAutoPushesGroupRight(t *testing.T) {
	a := runtime.H("box", runtime.Props{Style: runtime.Style{Width: 40, Height: 20}})
	b := runtime.H("box", runtime.Props{Style: runtime.Style{Width: 30, Height: 20, MarginLeftAuto: true}})
	c := runtime.H("box", runtime.Props{Style: runtime.Style{Width: 20, Height: 20}})
	root := HStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		HStack(runtime.Props{Style: runtime.Style{Width: 200, Height: 40, Gap: 10}}, a, b, c),
	)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	// a 居左不动；b 吃掉剩余宽度（200 - 40 - 30 - 20 - 2*10 = 90），b/c 贴最右。
	if a.X != host.X {
		t.Fatalf("auto 前面的兄弟应居左: a.X=%.1f host.X=%.1f", a.X, host.X)
	}
	wantBX := host.X + 40 + 10 + 90
	wantCX := wantBX + 30 + 10
	if b.X != wantBX || c.X != wantCX {
		t.Fatalf("margin-left:auto 应把 b/c 推到最右: b.X=%.1f want %.1f, c.X=%.1f want %.1f",
			b.X, wantBX, c.X, wantCX)
	}
}

func TestMarginAutoCentersFixedWidthChild(t *testing.T) {
	child := runtime.H("box", runtime.Props{
		Style: runtime.Style{Width: 60, Height: 20, MarginAuto: true},
	})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{Style: runtime.Style{Width: 200, Height: 80}}, child),
	)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	wantX := host.X + (host.W-60)/2
	if child.X != wantX || child.W != 60 {
		t.Fatalf("margin:0 auto 应水平居中: got x=%.1f w=%.1f want x=%.1f", child.X, child.W, wantX)
	}
}

func TestMarginAutoYieldsToAlignCenter(t *testing.T) {
	child := runtime.H("box", runtime.Props{
		Style: runtime.Style{Width: 60, Height: 20, MarginAuto: true},
	})
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}},
		VStack(runtime.Props{Style: runtime.Style{Width: 200, Height: 80, Align: runtime.End}}, child),
	)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{root}, theme.Light)
	host := root.Children[0]
	wantX := host.X + host.W - 60
	if child.X != wantX {
		t.Fatalf("显式 Align 应盖过 MarginAuto: got %.1f want %.1f", child.X, wantX)
	}
}
