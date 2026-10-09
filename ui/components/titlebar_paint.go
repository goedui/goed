package components

import (
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func paintTitleBar(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil {
		return
	}
	w := ctx.Width()
	h := float32(renderer.TitleBarHeight)
	if w <= 0 {
		return
	}

	// 应用自绘标题栏：这一条整个归应用（标签条就画在这儿），框架什么都不画 ——
	// 连右侧系统按钮也不画，因为那三个位置的颜色要与应用自己的标签条底色相配。
	// 宿主仍然把那三个位置判成 HTMINBUTTON / HTMAXBUTTON / HTCLOSE，所以应用只要
	// 把图形画在对齐的位置上，点击照样由系统执行。
	if appTitleBarNode(n) {
		return
	}

	bg := renderer.ColorFrom(th.Titlebar)
	if bg.A == 0 {
		bg = renderer.ColorFrom(th.Background)
	}
	if !activeTitle(n) && renderer.ColorSet(th.TitlebarInactive) {
		bg = renderer.ColorFrom(th.TitlebarInactive)
	}
	if n != nil && renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	}
	ctx.FillRect(0, 0, w, h, bg)

	btnW := float32(renderer.CaptionButtonWidth)
	closeX := w - btnW
	maxX := closeX - btnW
	minX := maxX - btnW
	titleX := float32(12)
	// 没有系统标题栏按钮的宿主（浏览器）上，右侧那三个位置不该占着：
	// 按钮画出来是死的，标题还会被挤掉一截。
	titleRight := minX
	if !platform.CaptionButtons {
		minX, maxX, closeX = w, w, w
		titleRight = w
	}

	hover := renderer.AttrInt(n, "hover")
	pressed := renderer.AttrInt(n, "pressed")
	maximized := renderer.AttrBool(n, "maximized")
	active := activeTitle(n)

	fg := renderer.ColorFrom(th.TitlebarForeground)
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	if n != nil && renderer.ColorSet(n.Style.Color) {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	if !active {
		fg = renderer.Color{R: fg.R, G: fg.G, B: fg.B, A: fg.A * 0.6}
	}
	if platform.CaptionButtons {
		paintCaptionButton(ctx, minX, 0, btnW, h, hover == 1, pressed == 1, false, th, bg, fg, captionMin)
		paintCaptionButton(ctx, maxX, 0, btnW, h, hover == 2, pressed == 2, false, th, bg, fg, captionMaxRestore(maximized))
		paintCaptionButton(ctx, closeX, 0, btnW, h, hover == 3, pressed == 3, true, th, bg, fg, captionClose)
	}

	if paint := titleBarIcon(n); paint != nil {
		icon := float32(renderer.TitleBarIconSize)
		iy := (h - icon) / 2
		paint(ctx, 10, iy, icon, fg, bg)
		titleX = 10 + icon + renderer.TitleBarIconGap
	}

	title := renderer.AttrString(n, "title")
	if title != "" {
		maxTitleW := titleRight - titleX - 8
		if maxTitleW < 0 {
			maxTitleW = 0
		}
		font := th.FontFamily
		if style.FontFamily != "" {
			font = style.FontFamily
		}
		size := float32(12)
		if style.FontSize > 0 {
			size = style.FontSize
		}
		weight := 600
		if style.Weight > 0 {
			weight = style.Weight
		}
		ctx.DrawText(title, titleX, 0, maxTitleW, h, renderer.TextStyle{
			FontFamily: font,
			FontSize:   size,
			Color:      fg,
			Weight:     weight,
			Align:      renderer.AlignStart,
			VAlign:     renderer.AlignCenter,
			NoWrap:     true,
		})
	}

	sep := renderer.ColorFrom(th.Separator)
	if sep.A == 0 {
		sep = renderer.Color{R: fg.R, G: fg.G, B: fg.B, A: 0.08}
	}
	ctx.FillRect(0, h-1, w, 1, sep)
}

func activeTitle(n *runtime.VNode) bool {
	if n == nil || n.Attrs == nil {
		return true
	}
	if _, ok := n.Attrs["active"]; ok {
		return renderer.AttrBool(n, "active")
	}
	return true
}

// appTitleBarNode 报告应用自绘标题栏（platform.AppTitleBar）。
func appTitleBarNode(n *runtime.VNode) bool {
	return n != nil && renderer.AttrBool(n, "apptitle")
}

func titleBarIcon(n *runtime.VNode) renderer.IconPaint {
	v, _ := renderer.Attr(n, "icon").(renderer.IconPaint)
	return v
}

type captionKind int

const (
	captionMin captionKind = iota
	captionMax
	captionRestore
	captionClose
)

func captionMaxRestore(maximized bool) captionKind {
	if maximized {
		return captionRestore
	}
	return captionMax
}

func paintCaptionButton(ctx renderer.Context, x, y, w, h float32, hover, pressed, isClose bool, th theme.Theme, barBg, fg renderer.Color, kind captionKind) {
	bg := barBg
	if hover || pressed {
		bg = renderer.ColorFrom(th.TitlebarHover)
		if isClose {
			bg = renderer.ColorFrom(th.CloseHover)
			if pressed {
				bg = renderer.RGB(150, 30, 20)
			}
		} else if pressed {
			bg = renderer.Color{R: bg.R * 0.88, G: bg.G * 0.88, B: bg.B * 0.88, A: 1}
		}
		ctx.FillRect(x, y, w, h, bg)
	}
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.TitlebarForeground)
	}
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	if isClose && (hover || pressed) {
		fg = renderer.RGB(255, 255, 255)
	}
	cx := x + w/2
	cy := y + h/2
	paintCaptionGlyph(ctx, cx, cy, fg, bg, kind)
}

func paintCaptionGlyph(ctx renderer.Context, cx, cy float32, fg, bg renderer.Color, kind captionKind) {
	const stroke float32 = 1.15
	switch kind {
	case captionMin:
		ctx.FillRect(cx-5, cy, 10, stroke, fg)
	case captionMax:
		drawOutlineRect(ctx, cx-5, cy-5, 10, 10, stroke, fg)
	case captionRestore:
		drawOutlineRect(ctx, cx-3, cy-6, 8, 8, stroke, fg)
		ctx.FillRect(cx-6, cy-3, 8, 8, bg)
		drawOutlineRect(ctx, cx-6, cy-3, 8, 8, stroke, fg)
	case captionClose:
		drawCloseX(ctx, cx, cy, 4.6, stroke, fg)
	}
}

func drawCloseX(ctx renderer.Context, cx, cy, arm, s float32, c renderer.Color) {
	n := int(arm * 2)
	if n < 6 {
		n = 6
	}
	for i := 0; i <= n; i++ {
		t := float32(i)/float32(n)*2 - 1
		x := cx + t*arm
		y1 := cy + t*arm
		y2 := cy - t*arm
		ctx.FillRect(x-s/2, y1-s/2, s, s, c)
		ctx.FillRect(x-s/2, y2-s/2, s, s, c)
	}
}

func drawOutlineRect(ctx renderer.Context, x, y, w, h, s float32, c renderer.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	ctx.FillRect(x, y, w, s, c)
	ctx.FillRect(x, y+h-s, w, s, c)
	ctx.FillRect(x, y, s, h, c)
	ctx.FillRect(x+w-s, y, s, h, c)
}
