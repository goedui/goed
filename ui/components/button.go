package components

import (
	"fmt"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("button", renderer.Widget{
		Measure: measureButton,
		Paint:   paintButton,
	})
}

// Button 对应 h('button', props?, '点我')。
//
//	Button("点我")
//	Button(runtime.Props{OnClick: fn}, "点我")
func Button(first any, rest ...any) *runtime.VNode {
	var p runtime.Props
	label := ""
	if pp, ok := first.(runtime.Props); ok {
		p = pp
		if len(rest) > 0 {
			label = fmt.Sprint(rest[0])
		}
	} else if s, ok := first.(string); ok {
		label = s
	} else if first != nil {
		label = fmt.Sprint(first)
	}
	n := runtime.H("button", p)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	if _, ok := n.Attrs["label"]; !ok {
		n.Attrs["label"] = label
	}
	return n
}

func buttonLabel(n *runtime.VNode) string {
	if s := renderer.AttrString(n, "label"); s != "" {
		return s
	}
	for _, c := range n.Children {
		if c != nil && c.Kind == runtime.KindText {
			return c.Text
		}
	}
	return ""
}

func measureButton(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = maxH
	_ = th
	label := buttonLabel(n)
	padX, padY := float32(16), float32(8)
	if n.Style.Padding > 0 {
		padX, padY = n.Style.Padding, n.Style.Padding
	}
	limit := maxW
	if limit > 2*padX {
		limit -= 2 * padX
	}
	tw, textH := ctx.MeasureText(label, limit, style)
	if textH < style.FontSize {
		textH = style.FontSize
	}
	w := tw + 2*padX
	h := textH + 2*padY
	if h < 32 && n.Style.Height <= 0 {
		h = 32
	}
	if maxW > 0 && w > maxW && n.Style.Width <= 0 {
		w = maxW
	}
	return w, h
}

func paintButton(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	variant := renderer.AttrString(n, "variant")
	selected := renderer.AttrBool(n, "selected")
	enabled := renderer.Enabled(n)
	bg := renderer.ColorFrom(th.Button)
	fg := renderer.ColorFrom(th.ButtonForeground)
	if fg.A == 0 {
		fg = renderer.RGB(255, 255, 255)
	}
	hover := renderer.ColorFrom(th.ButtonHover)
	pressed := renderer.ColorFrom(th.ButtonPressed)
	if variant == "secondary" || variant == "ghost" {
		bg = renderer.ColorFrom(th.Titlebar)
		if bg.A == 0 {
			bg = renderer.RGB(240, 240, 240)
		}
		fg = renderer.ColorFrom(th.Foreground)
		hover = renderer.ColorFrom(th.TitlebarHover)
		pressed = renderer.ColorFrom(th.Separator)
	}
	if selected {
		bg = renderer.ColorFrom(th.Button)
		fg = renderer.ColorFrom(th.ButtonForeground)
		if fg.A == 0 {
			fg = renderer.RGB(255, 255, 255)
		}
	}
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
		hover = bg
		pressed = bg
		// 自定义底色时按亮度选字色：白底胶囊不能再沿用 primary 的白字。
		// selected 已经是白字（蓝底），不要覆盖。
		if !selected && !renderer.ColorSet(n.Style.Color) && lightBackground(bg) {
			fg = renderer.ColorFrom(th.Foreground)
			if fg.A == 0 {
				fg = renderer.RGB(24, 30, 38)
			}
		}
	}
	if enabled && renderer.PointerIn(n.X, n.Y, n.W, n.H) {
		if renderer.PointerDown() {
			bg = pressed
		} else if !renderer.ColorSet(n.Style.Background) {
			bg = hover
		}
	}
	if !enabled {
		bg = renderer.RGB(232, 234, 238)
		fg = renderer.RGB(95, 102, 115)
	}
	// 白底描边按钮（胶囊标签、次级操作）没有主题 hover 底色，单独给一档轻微
	// 提亮，否则鼠标移上去完全没有反馈。放在 disabled 之后，禁用态不被覆盖。
	if enabled && renderer.ColorSet(n.Style.Border) && renderer.PointerIn(n.X, n.Y, n.W, n.H) {
		if renderer.PointerDown() {
			bg = renderer.RGB(232, 236, 241)
		} else {
			bg = renderer.RGB(245, 247, 250)
		}
	}
	r := n.Style.Radius
	// 描边用「外圈圆角矩形 + 内嵌底色」画：DrawRect 只有直角，
	// 直接描直角会把圆角按钮的四个角切掉。
	if renderer.ColorSet(n.Style.Border) && n.W > 2 && n.H > 2 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, renderer.ColorFrom(n.Style.Border))
		inner := r - 1
		if inner < 0 {
			inner = 0
		}
		ctx.FillRoundedRect(n.X+1, n.Y+1, n.W-2, n.H-2, inner, bg)
	} else if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, bg)
	}
	label := buttonLabel(n)
	if label == "" {
		return
	}
	if renderer.ColorSet(n.Style.Color) {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	weight := style.Weight
	if weight <= 0 {
		weight = 600
	}
	// align="start" 让文字靠左（菜单项这类需要左对齐的场景）；
	// 默认居中，与普通按钮一致。
	align := renderer.AlignCenter
	textX, textW := n.X, n.W
	if renderer.AttrString(n, "align") == "start" {
		align = renderer.AlignStart
		pad := n.Style.Padding
		if pad <= 0 {
			pad = 10
		}
		textX += pad
		textW -= 2 * pad
		if textW < 0 {
			textW = 0
		}
	}
	ctx.DrawText(label, textX, n.Y, textW, n.H, renderer.TextStyle{
		FontFamily: style.FontFamily,
		FontSize:   style.FontSize,
		Color:      fg,
		Weight:     weight,
		Align:      align,
		VAlign:     renderer.AlignCenter,
		NoWrap:     true,
	})

	// 快捷键占右侧一列（菜单项的常见排版），颜色比标签淡一档。
	if shortcut := renderer.AttrString(n, "shortcut"); shortcut != "" {
		pad := n.Style.Padding
		if pad <= 0 {
			pad = 10
		}
		shortFg := fg
		shortFg.A *= 0.75
		ctx.DrawText(shortcut, n.X+pad, n.Y, n.W-2*pad, n.H, renderer.TextStyle{
			FontFamily: style.FontFamily,
			FontSize:   style.FontSize,
			Color:      shortFg,
			Weight:     weight,
			Align:      renderer.AlignEnd,
			VAlign:     renderer.AlignCenter,
			NoWrap:     true,
		})
	}
}

// lightBackground 报告自定义底色是否偏浅。白底胶囊标签、描边按钮走这条
// 分支，字色改成前景色而不是 primary 的白字。
func lightBackground(c renderer.Color) bool {
	return 0.299*c.R+0.587*c.G+0.114*c.B > 0.72
}
