package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestTableAlignsColumnsWithEmptyCells(t *testing.T) {
	cell := func(id, text string) *runtime.VNode {
		return runtime.H("hstack", runtime.Props{
			ID:    id,
			Style: runtime.Style{Padding: 6, Border: runtime.RGB(200, 200, 200)},
		}, runtime.Text(text))
	}
	n := Table(runtime.Props{Style: runtime.Style{Width: 360, Gap: 0, Border: runtime.RGB(180, 180, 180)}},
		HStack(cell("h0", "code"), cell("h1", "desc"), cell("h2", "schema")),
		HStack(cell("r0c0", "200"), cell("r0c1", "OK"), cell("r0c2", "R")),
		HStack(cell("r1c0", "201"), cell("r1c1", "Created"), cell("r1c2", "")),
		HStack(cell("r2c0", "401"), cell("r2c1", ""), cell("r2c2", "")),
	)
	ctx := &mockContext{w: 400, h: 300}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)

	h0 := runtime.FindByID([]*runtime.VNode{n}, "h0")
	h1 := runtime.FindByID([]*runtime.VNode{n}, "h1")
	h2 := runtime.FindByID([]*runtime.VNode{n}, "h2")
	r1c0 := runtime.FindByID([]*runtime.VNode{n}, "r1c0")
	r1c1 := runtime.FindByID([]*runtime.VNode{n}, "r1c1")
	r1c2 := runtime.FindByID([]*runtime.VNode{n}, "r1c2")
	r2c1 := runtime.FindByID([]*runtime.VNode{n}, "r2c1")
	r2c2 := runtime.FindByID([]*runtime.VNode{n}, "r2c2")
	if h0 == nil || r1c0 == nil || r2c2 == nil {
		t.Fatal("表格单元格应全部挂载")
	}
	if abs32(h0.X-r1c0.X) > 0.5 || abs32(h1.X-r1c1.X) > 0.5 || abs32(h2.X-r1c2.X) > 0.5 {
		t.Fatalf("列应对齐：header=(%.1f,%.1f,%.1f) row=(%.1f,%.1f,%.1f)", h0.X, h1.X, h2.X, r1c0.X, r1c1.X, r1c2.X)
	}
	if abs32(h0.W-r1c0.W) > 0.5 || abs32(h1.W-r1c1.W) > 0.5 || abs32(h2.W-r1c2.W) > 0.5 {
		t.Fatalf("列宽应一致：header=(%.1f,%.1f,%.1f) row=(%.1f,%.1f,%.1f)", h0.W, h1.W, h2.W, r1c0.W, r1c1.W, r1c2.W)
	}
	if r1c2.W < tableMinCol || r2c1.W < tableMinCol || r2c2.W < tableMinCol {
		t.Fatalf("空单元格不应塌缩：r1c2=%.1f r2c1=%.1f r2c2=%.1f", r1c2.W, r2c1.W, r2c2.W)
	}
	if abs32((h0.W+h1.W+h2.W)-n.W) > 1 {
		t.Fatalf("列宽应铺满表格：cols=%.1f table=%.1f", h0.W+h1.W+h2.W, n.W)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
