package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("tooltip", renderer.Widget{
		Measure: measureTooltip,
		Layout:  layoutTooltip,
		Paint:   paintTooltip,
	})
}

// Tooltip 是锚在某个节点上方的单行提示。Attrs: anchor, label。
// 它是浮层，画在锚点盒子外面，不会被锚点自己的宽高裁掉。
func Tooltip(parts ...any) *runtime.VNode {
	n := runtime.H("tooltip", parts...)
	// 脱离文档流：锚点常在 absolute 工作区里，若按普通子节点先排，
	// 悬停首帧读到的锚点还是零坐标。absolute 兄弟排在正文之后，坐标已就位。
	n.Style.Position = runtime.PositionAbsolute
	return n
}

func measureTooltip(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	label := renderer.AttrString(n, "label")
	st := renderer.ApplyStyle(style, n.Style)
	if st.FontSize <= 0 {
		st.FontSize = 12
	}
	w := float32(0)
	if ctx != nil {
		w, _ = ctx.MeasureText(label, 0, st)
	}
	if w < 8 {
		w = 8
	}
	_ = maxW
	_ = maxH
	_ = th
	return w + 20, st.FontSize + 12
}

func layoutTooltip(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	nw, nh := measureTooltip(ctx, n, 0, 0, style, th)
	if nw > 0 {
		w = nw
	}
	if nh > 0 {
		h = nh
	}
	ax, ay, aw := x, y, float32(0)
	if id := renderer.AttrString(n, "anchor"); id != "" {
		if a := runtime.FindByID(renderer.LayoutRoot(), id); a != nil {
			ax, ay, aw = a.X, a.Y, a.W
		}
	}
	px := ax + (aw-w)/2
	py := ay - h - 6
	if ctx != nil {
		if px+w > ctx.Width()-8 {
			px = ctx.Width() - w - 8
		}
		if py < 8 {
			py = ay + 22
		}
	}
	if px < 8 {
		px = 8
	}
	if py < 8 {
		py = 8
	}
	n.X, n.Y, n.W, n.H = px, py, w, h
	_ = style
	_ = th
}

func paintTooltip(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	bg := renderer.RGB(30, 30, 30)
	fg := renderer.RGB(255, 255, 255)
	if th.Dark {
		bg = renderer.RGB(230, 230, 230)
		fg = renderer.RGB(30, 30, 30)
	}
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, 6, bg)
	st := renderer.ApplyStyle(style, n.Style)
	st.Color = fg
	if st.FontSize <= 0 {
		st.FontSize = 12
	}
	ctx.DrawText(renderer.AttrString(n, "label"), n.X+10, n.Y+4, n.W-16, n.H-6, st)
}
