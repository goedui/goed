package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

var layoutRoot []*runtime.VNode

func layoutContent(ctx Context, nodes []*runtime.VNode, y float32, style TextStyle, th theme.Theme) {
	if ctx == nil {
		return
	}
	layoutRoot = nodes
	w := ctx.Width()
	h := ctx.Height() - y
	if h < 0 {
		h = 0
	}
	// 单个根（Flex 或组件）铺满标题栏以下的客户区。
	// 否则会走「主题 Padding + 自然高度」分支：layer 在 maxH=0 时按整窗
	// 高度度量，再从标题栏下方往下排，状态栏会被顶出窗口。
	if len(nodes) == 1 && fills(nodes[0]) {
		layoutNode(ctx, nodes[0], 0, y, w, h, style, th)
		return
	}
	pad := th.Padding
	maxW := w - pad*2
	if maxW < 0 {
		maxW = 0
	}
	cy := y + pad
	for _, n := range nodes {
		if n == nil {
			continue
		}
		_, nh := measureNode(ctx, n, maxW, 0, style, th)
		layoutNode(ctx, n, pad, cy, maxW, nh, style, th)
		cy += nh
	}
}

func fills(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	if n.Style.Flex > 0 {
		return true
	}
	// CreateApp 包一层组件时，Flex 在内层根上；外层仍应铺满客户区。
	return n.Kind == runtime.KindComponent
}

func measureNode(ctx Context, n *runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (float32, float32) {
	if n == nil {
		return 0, 0
	}
	st := applyStyle(style, n.Style)
	var w, h float32
	switch n.Kind {
	case runtime.KindText:
		w, h = measureText(ctx, n, maxW, st)
	case runtime.KindComponent:
		expandComponent(n)
		w, h = measureFlow(ctx, n.Children, innerWidth(n, maxW), 0, st, th)
		w, h = addPadding(n, w, h)
	case runtime.KindElement:
		if widget, ok := Lookup(n.Tag); ok && widget.Measure != nil {
			w, h = widget.Measure(ctx, n, maxW, maxH, st, th)
		} else {
			w, h = measureFlow(ctx, n.Children, innerWidth(n, maxW), 0, st, th)
			w, h = addPadding(n, w, h)
		}
	}
	return applySize(n, w, h, maxW, maxH)
}

func measureText(ctx Context, n *runtime.VNode, maxW float32, style TextStyle) (float32, float32) {
	if n.Text == "" {
		return 0, 0
	}
	pad := n.Style.Padding
	limit := maxW
	if limit > 2*pad {
		limit -= 2 * pad
	}
	w, h := ctx.MeasureText(n.Text, limit, style)
	return w + 2*pad, h + 2*pad
}

func measureFlow(ctx Context, nodes []*runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (float32, float32) {
	var w, h float32
	for _, c := range nodes {
		if c == nil {
			continue
		}
		cw, ch := measureNode(ctx, c, maxW, 0, style, th)
		if cw > w {
			w = cw
		}
		h += ch
	}
	if maxW > 0 {
		w = maxW
	}
	_ = maxH
	return w, h
}

func innerWidth(n *runtime.VNode, maxW float32) float32 {
	inner := maxW - 2*n.Style.Padding
	if inner < 0 {
		return 0
	}
	return inner
}

func addPadding(n *runtime.VNode, w, h float32) (float32, float32) {
	p := n.Style.Padding
	return w + 2*p, h + 2*p
}

func applySize(n *runtime.VNode, w, h, maxW, maxH float32) (float32, float32) {
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if n.Style.Height > 0 {
		h = n.Style.Height
	}
	if n.Style.MinWidth > 0 && w < n.Style.MinWidth {
		w = n.Style.MinWidth
	}
	if n.Style.MinHeight > 0 && h < n.Style.MinHeight {
		h = n.Style.MinHeight
	}
	if n.Style.MaxWidth > 0 && w > n.Style.MaxWidth {
		w = n.Style.MaxWidth
	}
	if n.Style.MaxHeight > 0 && h > n.Style.MaxHeight {
		h = n.Style.MaxHeight
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}
	return w, h
}

func layoutNode(ctx Context, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	if n == nil {
		return
	}
	n.X, n.Y, n.W, n.H = x, y, w, h
	st := applyStyle(style, n.Style)
	switch n.Kind {
	case runtime.KindText:
		return
	case runtime.KindComponent:
		expandComponent(n)
		layoutPadded(ctx, n, st, th)
	case runtime.KindElement:
		if widget, ok := Lookup(n.Tag); ok && widget.Layout != nil {
			widget.Layout(ctx, n, x, y, w, h, st, th)
			return
		}
		layoutPadded(ctx, n, st, th)
	}
}

func layoutPadded(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme) {
	p := n.Style.Padding
	layoutFlow(ctx, n.Children, n.X+p, n.Y+p, n.W-2*p, n.H-2*p, style, th)
}

func layoutFlow(ctx Context, nodes []*runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	var visible []*runtime.VNode
	for _, c := range nodes {
		if c != nil {
			visible = append(visible, c)
		}
	}
	if h > 0 && len(visible) == 1 && fills(visible[0]) {
		layoutNode(ctx, visible[0], x, y, w, h, style, th)
		return
	}
	cy := y
	for _, c := range visible {
		_, ch := measureNode(ctx, c, w, 0, style, th)
		layoutNode(ctx, c, x, cy, w, ch, style, th)
		cy += ch
	}
}

func expandComponent(n *runtime.VNode) {
	runtime.ResolveComponent(n)
}

// LayoutRoot 返回本帧正在布局的根节点，供 popover 找锚点。
func LayoutRoot() []*runtime.VNode { return layoutRoot }
