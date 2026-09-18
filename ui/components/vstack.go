package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("vstack", renderer.Widget{
		Measure: measureVStack,
		Layout:  layoutVStack,
		Paint:   paintVStack,
	})
}

// VStack 垂直堆叠。对应 h('vstack', props?, children...)。
func VStack(parts ...any) *runtime.VNode {
	return runtime.H("vstack", parts...)
}

func measureVStack(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	innerW := maxW - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	var contentH, contentW float32
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, innerW, 0, style, th)
		if cw > contentW {
			contentW = cw
		}
		if !first {
			contentH += gap
		}
		first = false
		contentH += ch
	}
	w := contentW + 2*pad
	if maxW > 0 {
		w = maxW
	}
	h := contentH + 2*pad
	if n.Style.Flex > 0 && maxH > 0 && h < maxH {
		h = maxH
	}
	return w, h
}

func layoutVStack(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	align := n.Style.Align
	justify := n.Style.Justify
	innerW := w - 2*pad
	if innerW < 0 {
		innerW = 0
	}

	type item struct {
		n      *runtime.VNode
		iw, ih float32
		flex   float32
	}
	items := make([]item, 0, len(n.Children))
	var contentH, flexSum float32
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		iw, ih := renderer.MeasureNode(ctx, c, innerW, 0, style, th)
		if len(items) > 0 {
			contentH += gap
		}
		contentH += ih
		flexSum += c.Style.Flex
		items = append(items, item{c, iw, ih, c.Style.Flex})
	}

	innerH := h - 2*pad
	extra := innerH - contentH
	cy := y + pad
	if extra > 0 && flexSum == 0 {
		switch justify {
		case runtime.Center:
			cy += extra / 2
		case runtime.End:
			cy += extra
		}
	}
	for _, it := range items {
		cx := x + pad
		cw := it.iw
		ch := it.ih
		if extra != 0 && flexSum > 0 && it.flex > 0 {
			ch += extra * (it.flex / flexSum)
			if ch < 0 {
				ch = 0
			}
		}
		switch align {
		case runtime.Center:
			cx += (innerW - cw) / 2
		case runtime.End:
			cx += innerW - cw
		case runtime.Stretch:
			cw = innerW
		}
		if it.flex > 0 && align != runtime.Center && align != runtime.End {
			cw = innerW
		}
		renderer.LayoutNode(ctx, it.n, cx, cy, cw, ch, style, th)
		cy += ch + gap
	}
}

func paintVStack(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
}
