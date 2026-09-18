package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("divider", renderer.Widget{
		Measure: measureDivider,
		Layout:  layoutDivider,
		Paint:   paintDivider,
	})
}

// Divider 1px 分隔线。
//
//	Divider()            水平
//	Divider(true)        竖直
//	Divider(props)       用 Style.Width/Height 或 Attrs["vertical"]
func Divider(parts ...any) *runtime.VNode {
	vertical := false
	filtered := make([]any, 0, len(parts))
	for _, p := range parts {
		if b, ok := p.(bool); ok {
			vertical = b
			continue
		}
		filtered = append(filtered, p)
	}
	n := runtime.H("divider", filtered...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	if vertical {
		n.Attrs["vertical"] = true
	}
	return n
}

func dividerVertical(n *runtime.VNode) bool {
	if n.Style.Width > 0 && n.Style.Height == 0 {
		return true
	}
	return renderer.AttrBool(n, "vertical")
}

func measureDivider(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = style
	_ = th
	if dividerVertical(n) {
		w := n.Style.Width
		if w <= 0 {
			w = 1
		}
		h := n.Style.Height
		if h <= 0 {
			h = maxH
		}
		if h <= 0 {
			h = 1
		}
		return w, h
	}
	w := n.Style.Width
	if w <= 0 {
		w = maxW
	}
	if w <= 0 {
		w = 1
	}
	h := n.Style.Height
	if h <= 0 {
		h = 1
	}
	return w, h
}

func layoutDivider(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	_ = ctx
	_ = style
	_ = th
	n.X, n.Y, n.W, n.H = x, y, w, h
}

func paintDivider(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	_ = style
	if ctx == nil || n == nil {
		return
	}
	c := renderer.ColorFrom(th.Separator)
	if renderer.ColorSet(n.Style.Background) {
		c = renderer.ColorFrom(n.Style.Background)
	} else if renderer.ColorSet(n.Style.Color) {
		c = renderer.ColorFrom(n.Style.Color)
	}
	if c.A == 0 {
		c = renderer.RGB(228, 231, 236)
	}
	ctx.FillRect(n.X, n.Y, n.W, n.H, c)
}
