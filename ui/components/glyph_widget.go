package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("glyph", renderer.Widget{
		Measure: measureGlyph,
		Layout:  layoutGlyphWidget,
		Paint:   paintGlyphWidget,
	})
}

// Glyph 单个矢量图标节点：Icon 指定图标名，Size（Style.Width/Height）缺省 16。
// 字段标签、行内装饰用它 —— 不给 label 的按钮也能干这活，但语义上图标不是按钮。
//
//	Glyph(runtime.Props{Icon: "edit", Style: runtime.Style{Width: 16, Height: 16, Color: blue}})
func Glyph(parts ...any) *runtime.VNode {
	return runtime.H("glyph", parts...)
}

func glyphSize(n *runtime.VNode) float32 {
	s := n.Style.Width
	if n.Style.Height > s {
		s = n.Style.Height
	}
	if s <= 0 {
		s = 16
	}
	return s
}

func measureGlyph(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = maxW
	_ = maxH
	_ = style
	_ = th
	s := glyphSize(n)
	return s, s
}

func layoutGlyphWidget(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	_ = ctx
	_ = style
	_ = th
	s := glyphSize(n)
	// 给多大画多大：父级把行高撑起来时图标保持自身尺寸并靠上（行内基线附近）。
	n.X, n.Y, n.W, n.H = x, y, s, s
	if h > s {
		_ = w
	}
}

func paintGlyphWidget(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	fg := style.Color
	if renderer.ColorSet(n.Style.Color) {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	PaintGlyph(ctx, renderer.AttrString(n, "icon"), n.X, n.Y, n.H, fg)
}
