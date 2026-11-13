package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func overflowRows(n int) []*runtime.VNode {
	rows := make([]*runtime.VNode, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "行"))
	}
	return rows
}

func TestOverflowAutoScrollsAndClipsHit(t *testing.T) {
	ResetScroll("ov-auto")
	t.Cleanup(func() { ResetScroll("ov-auto") })
	row := runtime.H("button", runtime.Props{
		Style:   runtime.Style{Height: 24},
		OnClick: func() {},
	}, runtime.Text("点我"))
	host := VStack(runtime.Props{
		ID: "ov-auto",
		Style: runtime.Style{
			Width: 200, Height: 80, OverflowY: runtime.OverflowAuto,
		},
	}, row, runtime.Text(runtime.Props{Style: runtime.Style{Height: 200}}, "余下"))
	ctx := &mockContext{w: 300, h: 200}
	renderer.PaintTree(ctx, []*runtime.VNode{host}, theme.Light)
	if row.Y >= host.Y+host.H {
		t.Fatalf("首行应在视口内: row=%+v host=%+v", row, host)
	}
	HandleWheel([]*runtime.VNode{host}, host.X+8, host.Y+8, -2)
	renderer.PaintTree(ctx, []*runtime.VNode{host}, theme.Light)
	if got := ScrollOffset("ov-auto"); got <= 0 {
		t.Fatalf("auto 容器应记下纵向偏移: %v", got)
	}
	if row.Y >= host.Y {
		t.Fatalf("滚过后首行应移出视口顶: row.Y=%.1f host.Y=%.1f", row.Y, host.Y)
	}
	if runtime.HitNode([]*runtime.VNode{host}, row.X+4, host.Y-4) != nil {
		t.Fatal("滚出裁剪区的按钮不应命中")
	}
}

func TestOverflowHiddenDoesNotScroll(t *testing.T) {
	ResetScroll("ov-hidden")
	t.Cleanup(func() { ResetScroll("ov-hidden") })
	parts := []any{runtime.Props{
		ID: "ov-hidden",
		Style: runtime.Style{
			Width: 200, Height: 40, Overflow: runtime.OverflowHidden,
		},
	}}
	for _, row := range overflowRows(6) {
		parts = append(parts, row)
	}
	host := VStack(parts...)
	ctx := &mockContext{w: 300, h: 200}
	renderer.PaintTree(ctx, []*runtime.VNode{host}, theme.Light)
	before := host.Children[0].Y
	HandleWheel([]*runtime.VNode{host}, host.X+8, host.Y+8, -3)
	renderer.PaintTree(ctx, []*runtime.VNode{host}, theme.Light)
	if ScrollOffset("ov-hidden") != 0 || host.Children[0].Y != before {
		t.Fatal("hidden 只裁不滚")
	}
}

func TestOverflowWheelBubblesWhenInnerAtEnd(t *testing.T) {
	ResetScroll("ov-inner")
	ResetScroll("ov-outer")
	t.Cleanup(func() {
		ResetScroll("ov-inner")
		ResetScroll("ov-outer")
	})
	inner := VStack(runtime.Props{
		ID:    "ov-inner",
		Style: runtime.Style{Width: 160, Height: 40, OverflowY: runtime.OverflowAuto},
	}, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "短"))
	outer := VStack(runtime.Props{
		ID:    "ov-outer",
		Style: runtime.Style{Width: 200, Height: 80, OverflowY: runtime.OverflowAuto},
	}, inner, runtime.Text(runtime.Props{Style: runtime.Style{Height: 200}}, "外层内容"))
	ctx := &mockContext{w: 300, h: 240}
	renderer.PaintTree(ctx, []*runtime.VNode{outer}, theme.Light)
	HandleWheel([]*runtime.VNode{outer}, inner.X+4, inner.Y+4, -2)
	renderer.PaintTree(ctx, []*runtime.VNode{outer}, theme.Light)
	if ScrollOffset("ov-inner") != 0 {
		t.Fatalf("内层没有溢出，不该滚: %v", ScrollOffset("ov-inner"))
	}
	if ScrollOffset("ov-outer") <= 0 {
		t.Fatal("内层到头后滚轮应交给外层")
	}
}

func TestOverflowXWheelScrollsWideRow(t *testing.T) {
	ResetScroll("ov-row")
	t.Cleanup(func() { ResetScroll("ov-row") })
	kids := make([]any, 0, 9)
	kids = append(kids, runtime.Props{
		ID: "ov-row",
		Style: runtime.Style{
			Width: 120, Height: 32, Gap: 4,
			OverflowX: runtime.OverflowAuto,
			OverflowY: runtime.OverflowHidden,
		},
	})
	for i := 0; i < 8; i++ {
		kids = append(kids, runtime.H("box", runtime.Props{Style: runtime.Style{Width: 40, Height: 20}}))
	}
	host := HStack(kids...)
	ctx := &mockContext{w: 400, h: 80}
	wrap := VStack(runtime.Props{Style: runtime.Style{Width: 160, Height: 40}}, host)
	renderer.PaintTree(ctx, []*runtime.VNode{wrap}, theme.Light)
	HandleWheel([]*runtime.VNode{wrap}, host.X+4, host.Y+4, -2)
	if got := ScrollOffsetAxis("ov-row", true); got <= 0 {
		t.Fatalf("横向溢出应能滚: off=%v host.W=%.1f", got, host.W)
	}
}

func TestOverflowXPaintsHorizontalBarAndDrags(t *testing.T) {
	ResetScroll("ov-row-bar")
	t.Cleanup(func() {
		ResetScroll("ov-row-bar")
		EndScrollDrag()
		renderer.SetPointer(0, 0, false)
	})
	kids := make([]any, 0, 9)
	kids = append(kids, runtime.Props{
		ID: "ov-row-bar",
		Style: runtime.Style{
			Width: 120, Height: 32, Gap: 4,
			OverflowX: runtime.OverflowAuto,
			OverflowY: runtime.OverflowHidden,
		},
	})
	for i := 0; i < 8; i++ {
		kids = append(kids, runtime.H("box", runtime.Props{Style: runtime.Style{Width: 40, Height: 20}}))
	}
	host := HStack(kids...)
	wrap := VStack(runtime.Props{Style: runtime.Style{Width: 160, Height: 40}}, host)
	ctx := &mockContext{w: 400, h: 80}
	renderer.PaintTree(ctx, []*runtime.VNode{wrap}, theme.Light)

	barX, barW, maxOff, shown := scrollBarGeomH(host)
	if !shown || maxOff <= 0 || barW <= 0 {
		t.Fatalf("横向滚动条几何无效：shown=%v maxOff=%.1f barW=%.1f", shown, maxOff, barW)
	}
	found := false
	for _, r := range ctx.rounds {
		if r.y >= host.Y+host.H-12 && r.h <= 8 && r.w >= 20 && r.x >= host.X {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("横向溢出应在底部画滚动条：rounds=%+v host=%+v", ctx.rounds, host)
	}

	down, ok := host.Attrs["onPointerDown"].(func())
	if !ok {
		t.Fatal("横向 overflow 容器应被补上 onPointerDown")
	}
	drag, ok := host.Attrs["onPointerDrag"].(func())
	if !ok {
		t.Fatal("横向 overflow 容器应被补上 onPointerDrag")
	}
	renderer.SetPointer(barX+barW/2, host.Y+host.H-4, true)
	down()
	renderer.SetPointer(host.X+host.W-2, host.Y+host.H-4, true)
	drag()
	if got := ScrollOffsetAxis("ov-row-bar", true); got <= 0 {
		t.Fatalf("拖动横向滚动条应改变偏移：%.1f", got)
	}
}

func TestOverflowXOverridesOverflow(t *testing.T) {
	wide := runtime.H("box", runtime.Props{Style: runtime.Style{Width: 400, Height: 20}})
	host := HStack(runtime.Props{
		ID: "ov-x",
		Style: runtime.Style{
			Width: 120, Height: 40,
			Overflow:  runtime.OverflowHidden,
			OverflowX: runtime.OverflowVisible,
		},
	}, wide)
	renderer.PaintTree(&mockContext{w: 500, h: 200}, []*runtime.VNode{host}, theme.Light)
	if wide.W < 400-1 {
		t.Fatalf("OverflowX=visible 时子项宽度不该被夹: %.1f", wide.W)
	}
}
