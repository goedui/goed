package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("table", renderer.Widget{
		Measure: measureTable,
		Layout:  layoutTable,
		Paint:   paintTable,
	})
}

const tableMinCol = float32(28)

// Table 按列对齐的网格。子节点是行（hstack / 任意容器），行的子节点是单元格。
// 列宽取各行该列自然宽的最大值，再按行 flex 把剩余宽度均分；空单元格仍占一列。
func Table(parts ...any) *runtime.VNode {
	return runtime.H("table", parts...)
}

func tableRows(n *runtime.VNode) []*runtime.VNode {
	if n == nil {
		return nil
	}
	out := make([]*runtime.VNode, 0, len(n.Children))
	for _, row := range n.Children {
		if row != nil {
			out = append(out, row)
		}
	}
	return out
}

func tableCells(row *runtime.VNode) []*runtime.VNode {
	if row == nil {
		return nil
	}
	out := make([]*runtime.VNode, 0, len(row.Children))
	for _, cell := range row.Children {
		if cell != nil {
			out = append(out, cell)
		}
	}
	return out
}

func tableColCount(rows []*runtime.VNode) int {
	cols := 0
	for _, row := range rows {
		if n := len(tableCells(row)); n > cols {
			cols = n
		}
	}
	return cols
}

func measureTable(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	rows := tableRows(n)
	cols := tableColCount(rows)
	colW := make([]float32, cols)
	rowH := make([]float32, len(rows))
	for i, row := range rows {
		cells := tableCells(row)
		var h float32
		for c := 0; c < cols; c++ {
			var cw, ch float32
			if c < len(cells) {
				cw, ch = renderer.MeasureNode(ctx, cells[c], 0, 0, style, th)
			}
			if cw < tableMinCol {
				cw = tableMinCol
			}
			if cw > colW[c] {
				colW[c] = cw
			}
			if ch > h {
				h = ch
			}
		}
		if min := row.Style.MinHeight; min > 0 && h < min {
			h = min
		}
		if h < 1 {
			h = 1
		}
		rowH[i] = h
	}

	contentW := float32(0)
	for i, w := range colW {
		if i > 0 {
			contentW += gap
		}
		contentW += w
	}
	w := contentW + 2*pad
	if n.Style.Flex > 0 && maxW > 0 {
		w = maxW
	}
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}

	innerW := w - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	colW = distributeTableCols(colW, gap, innerW)

	var contentH float32
	for i, row := range rows {
		h := rowH[i]
		cells := tableCells(row)
		for c := 0; c < cols && c < len(cells); c++ {
			if _, ch := renderer.MeasureNode(ctx, cells[c], colW[c], 0, style, th); ch > h {
				h = ch
			}
		}
		if min := row.Style.MinHeight; min > 0 && h < min {
			h = min
		}
		rowH[i] = h
		if i > 0 {
			contentH += gap
		}
		contentH += h
	}
	h := contentH + 2*pad
	if n.Style.Flex > 0 && maxH > 0 && h < maxH {
		h = maxH
	}
	return w, h
}

func layoutTable(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	pad := n.Style.Padding
	gap := n.Style.Gap
	innerW := w - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	rows := tableRows(n)
	cols := tableColCount(rows)
	colW := make([]float32, cols)
	rowH := make([]float32, len(rows))
	for i, row := range rows {
		cells := tableCells(row)
		var rh float32
		for c := 0; c < cols; c++ {
			var cw, ch float32
			if c < len(cells) {
				cw, ch = renderer.MeasureNode(ctx, cells[c], 0, 0, style, th)
			}
			if cw < tableMinCol {
				cw = tableMinCol
			}
			if cw > colW[c] {
				colW[c] = cw
			}
			if ch > rh {
				rh = ch
			}
		}
		if min := row.Style.MinHeight; min > 0 && rh < min {
			rh = min
		}
		rowH[i] = rh
	}
	colW = distributeTableCols(colW, gap, innerW)

	cy := y + pad
	for i, row := range rows {
		cells := tableCells(row)
		hRow := rowH[i]
		for c := 0; c < cols && c < len(cells); c++ {
			if _, ch := renderer.MeasureNode(ctx, cells[c], colW[c], 0, style, th); ch > hRow {
				hRow = ch
			}
		}
		if min := row.Style.MinHeight; min > 0 && hRow < min {
			hRow = min
		}
		renderer.LayoutNode(ctx, row, x+pad, cy, innerW, hRow, style, th)
		cx := x + pad
		for c := 0; c < cols; c++ {
			cw := colW[c]
			if c < len(cells) {
				renderer.LayoutNode(ctx, cells[c], cx, cy, cw, hRow, style, th)
			}
			cx += cw + gap
		}
		cy += hRow + gap
	}
}

func distributeTableCols(natural []float32, gap, innerW float32) []float32 {
	n := len(natural)
	if n == 0 {
		return natural
	}
	flex := make([]float32, n)
	for i := range flex {
		flex[i] = 1
	}
	widths, _ := renderer.Distribute(natural, flex, gap, innerW)
	for i := range widths {
		if widths[i] < tableMinCol {
			widths[i] = tableMinCol
		}
	}
	return widths
}

func paintTable(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
}
