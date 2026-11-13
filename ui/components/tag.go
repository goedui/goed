package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("tag", renderer.Widget{
		Measure: measureTag,
		Paint:   paintTag,
	})
}

// Tag 状态胶囊：圆角底 + 居中单行短文本。
//
// 与「HStack 包 Text」的手搓 pill 相比，Tag 自己管测量和绘制：
// 文本垂直居中在胶囊里，不会被 Text 节点的 Height-Padding 公式裁掉下半；
// 宽度按文本自然宽 + 左右 padding。
//
// Attrs:
//
//	label  文案
//	fg     文字色（缺省走前景色）
//	bg     底色（缺省按主题给一块浅底）
//	fill   文字在胶囊里是否横向居中（对齐 calc 的显示屏 / 结果条）
//
// 尺寸：Style.Height 为 0 时按字号取「行高 + 上下内边距」的自然高，
// 写了 Height 就只用它，宽度同理。
func Tag(parts ...any) *runtime.VNode {
	return runtime.H("tag", parts...)
}

func measureTag(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = maxH
	_ = th
	label := renderer.AttrString(n, "label")
	font := n.Style.FontSize
	if font <= 0 {
		font = 11
	}
	ts := style
	ts.FontSize = font
	w, h := float32(len([]rune(label)))*font*0.62+16, float32(20)
	if ctx != nil {
		if tw, th2 := ctx.MeasureText(label, 1e6, ts); tw > 0 {
			w, h = tw+16, th2+8
		}
	}
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if n.Style.Height > 0 {
		h = n.Style.Height
	}
	if w > maxW && maxW > 24 {
		w = maxW
	}
	return w, h
}

func paintTag(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	fg, bg := tagColors(n, th)
	r := n.Style.Radius
	if r <= 0 {
		r = 999
	}
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	label := renderer.AttrString(n, "label")
	if label == "" {
		return
	}
	font := n.Style.FontSize
	if font <= 0 {
		font = 11
	}
	ts := style
	ts.FontSize = font
	ts.Color = fg
	ts.NoWrap = true
	if renderer.AttrBool(n, "fill") {
		ts.Align = renderer.AlignEnd
	}
	// 文本在胶囊里垂直居中：量出单行高，再往上让出居中偏移。
	_, th2 := ctx.MeasureText(label, 1e6, ts)
	if th2 <= 0 {
		th2 = n.H
	}
	ty := n.Y + (n.H-th2)/2
	if ty < n.Y {
		ty = n.Y
	}
	ctx.DrawText(label, n.X+8, ty, n.W-16, th2, ts)
}

func tagColors(n *runtime.VNode, th theme.Theme) (fg, bg renderer.Color) {
	if c, ok := renderer.Attr(n, "fg").(runtime.Color); ok {
		fg = renderer.ColorFrom(c)
	}
	if c, ok := renderer.Attr(n, "bg").(runtime.Color); ok {
		bg = renderer.ColorFrom(c)
	}
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	if bg.A == 0 {
		bg = renderer.RGB(240, 241, 244)
		if th.Dark {
			bg = renderer.RGB(48, 52, 60)
		}
	}
	return fg, bg
}
