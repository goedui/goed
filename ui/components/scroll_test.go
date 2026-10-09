package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// 内容超出高度时，Scroll 要在右侧画一根细滚动条；
// 内容装得下就不画。
func TestScrollPaintsScrollbar(t *testing.T) {
	ResetScroll("sc-bar")
	rows := make([]*runtime.VNode, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "行"))
	}
	ctx := &mockContext{w: 300, h: 240}
	n := Scroll(runtime.Props{
		Style: runtime.Style{Width: 220, Height: 100},
		ID:    "sc-bar",
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)

	found := false
	for _, r := range ctx.rounds {
		if r.w >= 4 && r.w <= 8 && r.h > 12 && r.x > n.X+n.W-20 {
			found = true
		}
	}
	if !found {
		t.Fatalf("内容溢出时应画滚动条，圆角矩形：%+v", ctx.rounds)
	}

	// 内容装得下 → 不画
	ResetScroll("sc-bar2")
	few := []*runtime.VNode{runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "一行")}
	ctx2 := &mockContext{w: 300, h: 240}
	n2 := Scroll(runtime.Props{
		Style: runtime.Style{Width: 220, Height: 100},
		ID:    "sc-bar2",
	}, VStack(runtime.Props{}, few))
	renderer.PaintTree(ctx2, []*runtime.VNode{n2}, theme.Dark)
	for _, r := range ctx2.rounds {
		if r.w >= 4 && r.w <= 8 && r.h > 12 && r.x > n2.X+n2.W-20 {
			t.Fatalf("内容不溢出时不该画滚动条：%+v", r)
		}
	}
}

// 滚轮能把偏移推上去，滚动条位置随之变化。
func TestScrollWheelMovesOffset(t *testing.T) {
	ResetScroll("sc-wheel")
	rows := make([]*runtime.VNode, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "行"))
	}
	n := Scroll(runtime.Props{
		Style: runtime.Style{Width: 220, Height: 100},
		ID:    "sc-wheel",
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))
	ctx := &mockContext{w: 300, h: 240}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	before := ScrollOffsetForTest("sc-wheel")
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, -3)
	if got := ScrollOffsetForTest("sc-wheel"); got <= before {
		t.Fatalf("向上滚应增大偏移：%v -> %v", before, got)
	}
}

func TestScrollPaintDoesNotRemeasureContent(t *testing.T) {
	ResetScroll("sc-cache")
	t.Cleanup(func() { ResetScroll("sc-cache") })
	rows := make([]*runtime.VNode, 0, 8)
	for i := 0; i < 8; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "行内容"))
	}
	n := Scroll(runtime.Props{
		Style: runtime.Style{Width: 220, Height: 80},
		ID:    "sc-cache",
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))
	warm := &countingMeasure{w: 300, h: 240}
	renderer.PaintTree(warm, []*runtime.VNode{n}, theme.Dark)
	ctx := &countingMeasure{w: 300, h: 240}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if ctx.n != 0 {
		t.Fatalf("滚动条绘制应复用 contentH，得到 %d 次 MeasureText", ctx.n)
	}
}

func TestScrollSkipsLayoutFarFromViewport(t *testing.T) {
	ResetScroll("sc-cull")
	t.Cleanup(func() { ResetScroll("sc-cull") })
	parts := make([]any, 0, 41)
	parts = append(parts, runtime.Props{
		Style: runtime.Style{Width: 220, Height: 80},
		ID:    "sc-cull",
	})
	for i := 0; i < 40; i++ {
		parts = append(parts, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "row"))
	}
	n := Scroll(parts...)
	ctx := &countingMeasure{w: 300, h: 240}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	SetScrollOffset("sc-cull", 400)
	ctx.n = 0
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if ctx.n != 0 {
		t.Fatalf("scrolled frame measured text %d times", ctx.n)
	}
	var near int
	for _, row := range n.Children {
		if row.H > 0 && row.Y+row.H > n.Y-n.H && row.Y < n.Y+2*n.H {
			near++
		}
	}
	if near == 0 || near >= len(n.Children) {
		t.Fatalf("viewport rows=%d total=%d", near, len(n.Children))
	}
}

// 预览区是「一个很高的内容块」。滚过两屏后这块的顶已经在视口上方，
// 但块本身还盖着视口。不能把它收成高度 0，否则可滚上限塌掉，偏移被钳回 0。
func TestScrollKeepsTallChildWhenTopIsAboveViewport(t *testing.T) {
	ResetScroll("sc-tall")
	t.Cleanup(func() { ResetScroll("sc-tall") })
	rows := make([]*runtime.VNode, 0, 40)
	for i := 0; i < 40; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "row"))
	}
	n := Scroll(runtime.Props{
		Style: runtime.Style{Width: 220, Height: 100},
		ID:    "sc-tall",
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))
	ctx := &mockContext{w: 400, h: 300}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	max := ScrollMax("sc-tall")
	if max < 300 {
		t.Fatalf("样本应能滚过三屏：max=%.1f", max)
	}
	SetScrollOffset("sc-tall", 250)
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if got := ScrollOffsetForTest("sc-tall"); got < 200 {
		t.Fatalf("滚过两屏后偏移被钳回开头：offset=%.1f max=%.1f", got, ScrollMax("sc-tall"))
	}
	if got := ScrollMax("sc-tall"); got < max-1 {
		t.Fatalf("内容高度不应塌掉：max %.1f -> %.1f", max, got)
	}
}

func TestTextLayoutCacheSkipsMeasure(t *testing.T) {
	const id = "te-layout-cache"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })
	n := Textarea(runtime.Props{
		ID:    id,
		Value: "hello\nworld",
		Style: runtime.Style{Width: 200, Height: 80},
	})
	ctx := &countingMeasure{w: 400, h: 200}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if ctx.n == 0 {
		t.Fatal("first layout should measure")
	}
	ctx.n = 0
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if ctx.n != 0 {
		t.Fatalf("cached layout measured %d times", ctx.n)
	}
}

func TestWideBarLeavesGutterForContent(t *testing.T) {
	ResetScroll("sc-gutter")
	t.Cleanup(func() { ResetScroll("sc-gutter") })
	child := runtime.H("box", runtime.Props{Style: runtime.Style{Height: 40}})
	n := Scroll(runtime.Props{
		ID:      "sc-gutter",
		WideBar: true,
		Style:   runtime.Style{Width: 300, Height: 80},
	}, child)
	renderer.PaintTree(&mockContext{w: 400, h: 200}, []*runtime.VNode{n}, theme.Dark)
	if child.W > n.W-22+0.5 {
		t.Fatalf("wide bar should leave a gutter, child.W=%.1f scroll.W=%.1f", child.W, n.W)
	}
}

