package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// rowsOf 把 []*VNode 转成 ...any 需要的形状。
func rowsOf(n int) []any {
	rows := overflowRows(n)
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, r)
	}
	return out
}

// 铺满整窗的装饰层（自绘页面背景）不能吞掉滚轮。
// 它是 layer 的第一个孩子、盖住后面的滚动容器，但自己既不可滚也没有 onWheel。
func TestWheelPassesThroughDecorativeFullSizeLayer(t *testing.T) {
	ResetScroll("wd-host")
	t.Cleanup(func() { ResetScroll("wd-host") })
	decor := runtime.H("canvas", runtime.Props{
		ID:    "wd-decor",
		Style: runtime.Style{Flex: 1},
	})
	host := VStack(append([]any{runtime.Props{
		ID:    "wd-host",
		Style: runtime.Style{Flex: 1, OverflowY: runtime.OverflowAuto},
	}}, rowsOf(12)...)...)
	root := runtime.H("layer", runtime.Props{
		Style: runtime.Style{Width: 300, Height: 200},
	}, decor, host)

	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{root}, theme.Light)
	HandleWheel([]*runtime.VNode{root}, host.X+20, host.Y+20, -3)
	if got := ScrollOffsetForTest("wd-host"); got <= 0 {
		t.Fatalf("滚轮应穿过装饰层落到可滚容器：offset=%.1f", got)
	}
}

// VStack{OverflowY:auto} 也要能拖滚动条：布局阶段统一补回调。
func TestOverflowBoxGetsScrollDrag(t *testing.T) {
	ResetScroll("wd-drag")
	t.Cleanup(func() { ResetScroll("wd-drag") })
	host := VStack(append([]any{runtime.Props{
		ID:    "wd-drag",
		Style: runtime.Style{Width: 200, Height: 80, OverflowY: runtime.OverflowAuto},
	}}, rowsOf(12)...)...)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{host}, theme.Light)

	down, ok := host.Attrs["onPointerDown"].(func())
	if !ok {
		t.Fatal("overflow 容器应被补上 onPointerDown")
	}
	drag, ok := host.Attrs["onPointerDrag"].(func())
	if !ok {
		t.Fatal("overflow 容器应被补上 onPointerDrag")
	}
	if !scrollDragNode(host) {
		t.Fatal("引擎补的回调应带 scrollDrag 标记")
	}

	barY, barH, maxOff, shown := scrollBarGeom(host)
	if !shown || maxOff <= 0 {
		t.Fatalf("样本应可滚：shown=%v maxOff=%.1f", shown, maxOff)
	}
	defer renderer.SetPointer(0, 0, false)

	// 按住滑块，往下拖到视口底部。
	gutterX := host.X + host.W - scrollGutter(host)/2
	renderer.SetPointer(gutterX, barY+barH/2, true)
	down()
	renderer.SetPointer(gutterX, host.Y+host.H-1, true)
	drag()

	got := ScrollOffsetForTest("wd-drag")
	if got <= 0 {
		t.Fatalf("拖动滚动条应改变偏移：%.1f", got)
	}
	if got > maxOff+0.5 {
		t.Fatalf("偏移超出可滚上限：%.1f > %.1f", got, maxOff)
	}
	EndScrollDrag()
	if got := ScrollOffsetForTest("wd-drag"); got <= 0 {
		t.Fatal("松手不应回滚偏移")
	}
}

// Scroll 标签的右侧纵向滚动条也要能按下拖动（预览正文就是这种容器）。
func TestScrollTagVerticalBarDrags(t *testing.T) {
	ResetScroll("wd-scroll-drag")
	t.Cleanup(func() { ResetScroll("wd-scroll-drag") })
	host := Scroll(runtime.Props{
		ID:    "wd-scroll-drag",
		Style: runtime.Style{Width: 200, Height: 80},
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, overflowRows(12)))
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{host}, theme.Light)

	down, ok := host.Attrs["onPointerDown"].(func())
	if !ok {
		t.Fatal("Scroll 容器应被补上 onPointerDown")
	}
	drag, ok := host.Attrs["onPointerDrag"].(func())
	if !ok {
		t.Fatal("Scroll 容器应被补上 onPointerDrag")
	}
	barY, barH, maxOff, shown := scrollBarGeom(host)
	if !shown || maxOff <= 0 {
		t.Fatalf("Scroll 应可滚动：shown=%v maxOff=%.1f", shown, maxOff)
	}
	defer renderer.SetPointer(0, 0, false)
	defer EndScrollDrag()

	gutterX := host.X + host.W - scrollGutter(host)/2
	renderer.SetPointer(gutterX, barY+barH/2, true)
	down()
	renderer.SetPointer(gutterX, host.Y+host.H-1, true)
	drag()

	if got := ScrollOffsetForTest("wd-scroll-drag"); got <= 0 {
		t.Fatalf("拖动 Scroll 滚动条应改变偏移：%.1f", got)
	}
}

// 点轨道空白处应直接把滑块跳过去，而不是「没反应」。
func TestScrollTrackClickJumps(t *testing.T) {
	ResetScroll("wd-jump")
	t.Cleanup(func() { ResetScroll("wd-jump") })
	host := VStack(append([]any{runtime.Props{
		ID:    "wd-jump",
		Style: runtime.Style{Width: 200, Height: 80, OverflowY: runtime.OverflowAuto},
	}}, rowsOf(12)...)...)
	renderer.PaintTree(&mockContext{w: 300, h: 200}, []*runtime.VNode{host}, theme.Light)

	barY, barH, maxOff, shown := scrollBarGeom(host)
	if !shown || maxOff <= 0 {
		t.Fatalf("样本应可滚：shown=%v", shown)
	}
	if barY+barH >= host.Y+host.H-2 {
		t.Skip("滑块已铺满轨道，没有可点的空白")
	}
	down := host.Attrs["onPointerDown"].(func())
	defer renderer.SetPointer(0, 0, false)
	renderer.SetPointer(host.X+host.W-4, host.Y+host.H-2, true)
	down()
	if got := ScrollOffsetForTest("wd-jump"); got <= maxOff*0.5 {
		t.Fatalf("点轨道底部应跳到接近底部：offset=%.1f max=%.1f", got, maxOff)
	}
}

// 应用可以给滚动条换品牌色；未设置时仍走主题前景色 18%。
func TestScrollbarUsesThemeColor(t *testing.T) {
	ResetScroll("wd-tint")
	t.Cleanup(func() { ResetScroll("wd-tint") })
	rows := overflowRows(12)
	host := Scroll(runtime.Props{
		Style: runtime.Style{Width: 200, Height: 80},
		ID:    "wd-tint",
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))

	th := theme.Light
	want := runtime.RGBA(116, 142, 210, 96)
	th.Scrollbar = want
	ctx := &mockContext{w: 300, h: 200}
	renderer.PaintTree(ctx, []*runtime.VNode{host}, th)

	found := false
	for _, r := range ctx.rounds {
		if r.x > host.X+host.W-20 && r.w <= 8 && r.h > 12 {
			found = true
			if r.c != renderer.ColorFrom(want) {
				t.Fatalf("滚动条应使用主题色：got=%+v want=%+v", r.c, renderer.ColorFrom(want))
			}
		}
	}
	if !found {
		t.Fatalf("没有画滚动条：%+v", ctx.rounds)
	}
}

func scrollDragNode(n *runtime.VNode) bool {
	if n == nil || n.Attrs == nil {
		return false
	}
	v, _ := n.Attrs["scrollDrag"].(bool)
	return v
}
