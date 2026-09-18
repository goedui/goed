package renderer

import "github.com/goedui/goed/ui/runtime"

func applyStyle(base TextStyle, s runtime.Style) TextStyle {
	if s.FontFamily != "" {
		base.FontFamily = s.FontFamily
	}
	if s.FontSize > 0 {
		base.FontSize = s.FontSize
	}
	if s.Weight > 0 {
		base.Weight = s.Weight
	}
	if colorSet(s.Color) {
		base.Color = colorFrom(s.Color)
	}
	return base
}

func colorFrom(c runtime.Color) Color {
	return Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

func colorSet(c runtime.Color) bool {
	return c.A > 0
}

func paintBackground(ctx Context, n *runtime.VNode) {
	if ctx == nil || n == nil {
		return
	}
	if colorSet(n.Style.Background) {
		c := colorFrom(n.Style.Background)
		if n.Style.Radius > 0 {
			ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, n.Style.Radius, c)
		} else {
			ctx.FillRect(n.X, n.Y, n.W, n.H, c)
		}
	}
	if colorSet(n.Style.Border) {
		ctx.DrawRect(n.X, n.Y, n.W, n.H, colorFrom(n.Style.Border), 1)
	}
}
