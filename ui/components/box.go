package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("box", renderer.Widget{
		Measure: measureBox,
		Layout:  layoutBox,
		Paint:   paintBox,
	})
	renderer.Register("layer", renderer.Widget{
		Measure: measureLayer,
		Layout:  layoutLayer,
		Paint:   paintLayer,
	})
}

// Box 矩形容器。
func Box(parts ...any) *runtime.VNode {
	return runtime.H("box", parts...)
}

func measureBox(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	w, h := n.Style.Width, n.Style.Height
	if w <= 0 {
		w = maxW
	}
	if h <= 0 && n.Style.Flex > 0 {
		h = maxH
	}
	if h <= 0 {
		h = 1
	}
	_ = ctx
	_ = style
	_ = th
	return w, h
}

func layoutBox(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	p := n.Style.Padding
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		renderer.LayoutNode(ctx, c, x+p, y+p, w-2*p, h-2*p, style, th)
	}
}

func paintBox(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
}

func measureLayer(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	w, h := maxW, maxH
	if w <= 0 && ctx != nil {
		w = ctx.Width()
	}
	if h <= 0 && ctx != nil {
		h = ctx.Height()
	}
	_ = style
	_ = th
	_ = n
	return w, h
}

func layoutLayer(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	// 父级常用 maxH=0 重新度量，layer 会按整窗高度返回；标题栏占 32 DIP
	// 后若仍用这个高度往下排，底栏会被顶出窗口。布局时改铺满 y 以下的画布。
	if ctx != nil {
		if w <= 0 {
			w = ctx.Width()
		}
		if bottom := ctx.Height() - y; bottom > 0 {
			h = bottom
		}
	}
	n.X, n.Y, n.W, n.H = x, y, w, h
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if first {
			renderer.LayoutNode(ctx, c, x, y, w, h, style, th)
			first = false
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, w, h, style, th)
		renderer.LayoutNode(ctx, c, x, y, cw, ch, style, th)
	}
}

func paintLayer(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
}
