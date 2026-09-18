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
		if r.w > 2.5 && r.w < 5 && r.h > 12 && r.x > n.X+n.W-12 {
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
		if r.w > 2.5 && r.w < 5 && r.h > 12 && r.x > n2.X+n2.W-12 {
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
