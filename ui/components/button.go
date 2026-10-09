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

// buttonIcon 取按钮的图标名。Props.Icon 走 Attrs["icon"]（applyProps 写入），
// 与 empty / tabs 的读取方式一致。
func buttonIcon(n *runtime.VNode) string { return renderer.AttrString(n, "icon") }

// buttonIconSize 图标边长跟随字号（下限 14，上限 18），与文字视觉等高。
func buttonIconSize(style renderer.TextStyle) float32 {
	s := style.FontSize
	if s < 14 {
		s = 14
	}
	if s > 18 {
		s = 18
	}
	return s
}

const buttonIconGap = float32(7)

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
	iconW := float32(0)
	if buttonIcon(n) != "" {
		iconW = buttonIconSize(style)
		if label != "" {
			iconW += buttonIconGap
		}
	}
	if limit > iconW {
		limit -= iconW
	}
	tw, textH := ctx.MeasureText(label, limit, style)
	if textH < style.FontSize {
		textH = style.FontSize
	}
	w := tw + iconW + 2*padX
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
		// 填充 / 文字都从主题来。以前这里（以及 select、textedit）各自硬编码
		// #E8EAEE，主题切到暗色时禁用按钮会变成一块近白色。
		dbg, dfg := th.DisabledColors()
		bg = renderer.ColorFrom(dbg)
		fg = renderer.ColorFrom(dfg)
	}
	// 描边按钮的自定义底色会盖掉主题 hover。按当前底色微调，浅底压暗、深底提亮，
	// 避免暗色主题下闪成一块近白。放在 disabled 之后，禁用态不被覆盖。
	if enabled && renderer.ColorSet(n.Style.Border) && renderer.PointerIn(n.X, n.Y, n.W, n.H) {
		bg = nudgeBackground(bg, renderer.PointerDown())
		if lightBackground(bg) && (renderer.ColorSet(n.Style.Color) && lightBackground(fg) || !renderer.ColorSet(n.Style.Color)) {
			fg = renderer.ColorFrom(th.Foreground)
			if fg.A == 0 || lightBackground(fg) {
				fg = renderer.RGB(24, 30, 38)
			}
		}
	}
	if enabled && renderer.PointerIn(n.X, n.Y, n.W, n.H) && lightBackground(bg) && lightBackground(fg) {
		fg = renderer.RGB(24, 30, 38)
	}
	r := n.Style.Radius
	// 投影：FillShadowRoundedRect 一次画完「阴影 + 本体」。有投影的按钮（生成）
	// 没有描边，两条路径互斥 —— 描边优先，描边按钮不投影。
	if n.Props.Shadow != nil && !renderer.ColorSet(n.Style.Border) {
		blur, offY := n.Props.Shadow.Blur, n.Props.Shadow.OffsetY
		if blur <= 0 {
			blur = 8
		}
		shadowC := renderer.RGB(15, 23, 42)
		if renderer.ColorSet(n.Props.Shadow.Color) {
			shadowC = renderer.ColorFrom(n.Props.Shadow.Color)
		}
		renderer.FillShadowRoundedRect(ctx, n.X, n.Y, n.W, n.H, r, blur, offY, bg, shadowC)
	} else if enabled && renderer.ColorSet(n.Style.Border) && n.W > 2 && n.H > 2 {
		// 描边用「外圈圆角矩形 + 内嵌底色」画：DrawRect 只有直角，
		// 直接描直角会把圆角按钮的四个角切掉。禁用时不描边，和灰底按钮同一套。
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
	// 渐变 / 投影按钮：在纯色本体上再叠渐变（端点色一致时观感是「本体被渐变
	// 替换」），投影已在上面的 FillShadow 分支处理不到 —— 按钮自己不调
	// FillShadowRoundedRect，避免与描边画法打架；生成按钮的投影由调用方把
	// Shadow 放在外层容器上（卡片式按钮）或直接省略。
	if g := n.Props.Gradient; g != nil && enabled {
		from, to := bg, bg
		if renderer.ColorSet(g.From) {
			from = renderer.ColorFrom(g.From)
		}
		if renderer.ColorSet(g.To) {
			to = renderer.ColorFrom(g.To)
		}
		renderer.FillGradientRoundedRect(ctx, n.X, n.Y, n.W, n.H, r, from, to, g.Vertical)
	}
	label := buttonLabel(n)
	icon := buttonIcon(n)
	if label == "" && icon == "" {
		return
	}
	if enabled && renderer.ColorSet(n.Style.Color) {
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
	// 图标与文字成组居中：先按「图标宽 + 间距 + 文字宽」算整组宽度，图标画在
	// 组首，文字从图标后开始。align=start（菜单项）时图标单独贴左、文字贴
	// padding —— 两列排版，和快捷键列并存。
	iconW := float32(0)
	if icon != "" {
		iconW = buttonIconSize(style)
	}
	if icon != "" && label == "" {
		// 纯图标按钮：图标居中（或贴左）。
		gx := n.X + (n.W-iconW)/2
		if renderer.AttrString(n, "align") == "start" {
			gx = n.X + padStart(n)
		}
		PaintGlyph(ctx, icon, gx, n.Y+(n.H-iconW)/2, iconW, fg)
		return
	}
	if icon != "" && align == renderer.AlignCenter {
		// 居中：整组 = 图标 + 间距 + 文字。漏掉图标宽会把内容整体挤到左边。
		tw, _ := ctx.MeasureText(label, textW, renderer.TextStyle{
			FontFamily: style.FontFamily, FontSize: style.FontSize,
			Weight: weight, NoWrap: true,
		})
		group := iconW + buttonIconGap + tw
		start := n.X + (n.W-group)/2
		PaintGlyph(ctx, icon, start, n.Y+(n.H-iconW)/2, iconW, fg)
		textX = start + iconW + buttonIconGap
		textW = n.X + n.W - textX
		if textW < 0 {
			textW = 0
		}
		align = renderer.AlignStart
	} else if icon != "" {
		PaintGlyph(ctx, icon, textX, n.Y+(n.H-iconW)/2, iconW, fg)
		textX += iconW + buttonIconGap
		textW -= iconW + buttonIconGap
		if textW < 0 {
			textW = 0
		}
	}
	if label != "" {
		ctx.DrawText(label, textX, n.Y, textW, n.H, renderer.TextStyle{
			FontFamily: style.FontFamily,
			FontSize:   style.FontSize,
			Color:      fg,
			Weight:     weight,
			Align:      align,
			VAlign:     renderer.AlignCenter,
			NoWrap:     true,
		})
	}

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

// nudgeBackground 给描边按钮一个跟当前底色同侧的 hover。浅底压暗，深底提亮。
func nudgeBackground(c renderer.Color, pressed bool) renderer.Color {
	delta := float32(0.08)
	if pressed {
		delta = 0.14
	}
	if lightBackground(c) {
		delta = -delta
	}
	return renderer.Color{
		R: clamp01(c.R + delta),
		G: clamp01(c.G + delta),
		B: clamp01(c.B + delta),
		A: c.A,
	}
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// padStart 是 align=start 文字列的左内边距（与测量端一致的缺省值）。
func padStart(n *runtime.VNode) float32 {
	if n.Style.Padding > 0 {
		return n.Style.Padding
	}
	return 10
}
