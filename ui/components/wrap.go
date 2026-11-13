package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("wrap", renderer.Widget{
		Measure: measureWrap,
		Layout:  layoutWrap,
		Paint:   paintWrap,
	})
}

// Wrap 横向流式排列：一行放不下就换行，等价于 CSS flex-wrap: wrap。
//
// 子节点保持自然尺寸，行高取该行最高的那个；行内从左往右摆，剩余空间留在
// 右侧（格子视图正是这个语义）。子节点的 Flex 在这里没有意义，会被忽略。
func Wrap(parts ...any) *runtime.VNode {
	return runtime.H("wrap", parts...)
}

// wrapFlow 遍历子节点并回调每项的自然尺寸（含外边距）。
//
// 测量与布局必须用同一套换行判断，否则「量三行、摆四行」，最后一行被祖先
// 的裁剪框切掉。
func wrapFlow(ctx renderer.Context, n *runtime.VNode, innerW float32, style renderer.TextStyle, th theme.Theme,
	fn func(c *runtime.VNode, cw, ch float32)) {
	gap := n.Style.Gap
	var rowW, rowH float32
	started := false
	for _, c := range n.Children {
		if c == nil || runtime.IsAbsolute(c) {
			continue
		}
		iw, ih := renderer.MeasureNode(ctx, c, innerW, 0, style, th)
		m := 2 * c.Style.Margin
		cw, ch := iw+m, ih+m
		if started && innerW > 0 && rowW+gap+cw > innerW {
			rowW, rowH = 0, 0
			started = false
		}
		if started {
			rowW += gap
		}
		rowW += cw
		if ch > rowH {
			rowH = ch
		}
		started = true
		fn(c, cw, ch)
	}
}

func measureWrap(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	innerW := renderer.InnerWidth(n, maxW)
	var totalW, totalH, rowW, rowH float32
	started := false
	wrapFlow(ctx, n, innerW, style, th, func(_ *runtime.VNode, cw, ch float32) {
		if started && innerW > 0 && rowW+gap+cw > innerW {
			totalH += rowH + gap
			rowW, rowH = 0, 0
			started = false
		}
		if started {
			rowW += gap
		}
		rowW += cw
		if ch > rowH {
			rowH = ch
		}
		if rowW > totalW {
			totalW = rowW
		}
		started = true
	})
	if started {
		totalH += rowH
	}
	w := totalW + 2*pad
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}
	h := totalH + 2*pad
	if n.Style.Height > 0 {
		h = n.Style.Height
	}
	if n.Style.Flex > 0 && maxH > 0 && h < maxH {
		h = maxH
	}
	return w, h
}

func layoutWrap(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	innerW := w - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	cx, cy := x+pad, y+pad
	var rowH float32
	started := false
	wrapFlow(ctx, n, innerW, style, th, func(c *runtime.VNode, cw, ch float32) {
		if started && innerW > 0 && cx-x-pad+cw > innerW {
			cx = x + pad
			cy += rowH + gap
			rowH = 0
			started = false
		}
		m := c.Style.Margin
		renderer.LayoutNode(ctx, c, cx+m, cy+m, cw-2*m, ch-2*m, style, th)
		cx += cw + gap
		if ch > rowH {
			rowH = ch
		}
		started = true
	})
}

func paintWrap(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
}
