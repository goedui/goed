package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("hstack", renderer.Widget{
		Measure: measureHStack,
		Layout:  layoutHStack,
		Paint:   paintHStack,
	})
}

// HStack 水平堆叠。对应 h('hstack', props?, children...)。
func HStack(parts ...any) *runtime.VNode {
	return runtime.H("hstack", parts...)
}

func measureHStack(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	innerW := maxW - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	var contentW, contentH float32
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, 0, maxH-2*pad, style, th)
		if ch > contentH {
			contentH = ch
		}
		if !first {
			contentW += gap
		}
		first = false
		contentW += cw
	}
	w := contentW + 2*pad
	if n.Style.Flex > 0 && maxW > 0 {
		w = maxW
	}
	h := contentH + 2*pad
	if n.Style.Flex > 0 && maxH > 0 && h < maxH {
		h = maxH
	}
	return w, h
}

func layoutHStack(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	align := n.Style.Align
	justify := n.Style.Justify
	innerW := w - 2*pad
	innerH := h - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	type item struct {
		n      *runtime.VNode
		iw, ih float32
		flex   float32
	}
	items := make([]item, 0, len(n.Children))
	var contentW, flexSum float32
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		iw, ih := renderer.MeasureNode(ctx, c, 0, innerH, style, th)
		if len(items) > 0 {
			contentW += gap
		}
		contentW += iw
		flex := c.Style.Flex
		flexSum += flex
		items = append(items, item{c, iw, ih, flex})
	}

	extra := innerW - contentW
	if extra < 0 {
		extra = 0
	}
	cx := x + pad
	if extra > 0 && flexSum == 0 {
		switch justify {
		case runtime.Center:
			cx += extra / 2
		case runtime.End:
			cx += extra
		}
	}
	for _, it := range items {
		cw := it.iw
		if extra > 0 && flexSum > 0 && it.flex > 0 {
			cw += extra * (it.flex / flexSum)
		}
		ch := it.ih
		cy := y + pad
		switch align {
		case runtime.Center:
			cy += (innerH - ch) / 2
		case runtime.End:
			cy += innerH - ch
		case runtime.Stretch:
			ch = innerH
		}
		renderer.LayoutNode(ctx, it.n, cx, cy, cw, ch, style, th)
		cx += cw + gap
	}
}

func paintHStack(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if n.OnClick != nil && renderer.Enabled(n) && renderer.PointerIn(n.X, n.Y, n.W, n.H) && renderer.ColorSet(n.Style.Background) {
		paintHStackHover(ctx, n)
	} else {
		renderer.PaintBackground(ctx, n)
	}
	renderer.PaintChildren(ctx, n, style, th)
}

// paintHStackHover 可点行（历史卡）悬停时略压暗底色，让「整行可点」看得见。
func paintHStackHover(ctx renderer.Context, n *runtime.VNode) {
	c := renderer.ColorFrom(n.Style.Background)
	c.R *= 0.97
	c.G *= 0.975
	c.B *= 0.98
	r := n.Style.Radius
	if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, c)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, c)
	}
	if renderer.ColorSet(n.Style.Border) {
		ctx.DrawRect(n.X, n.Y, n.W, n.H, renderer.ColorFrom(n.Style.Border), 1)
	}
}
