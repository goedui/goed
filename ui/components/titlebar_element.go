package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

const (
	TitleBarHeight          int32 = 40
	TitleBarButtonWidth     int32 = 46
	defaultTitleBarIconSize int32 = 14
)

// TitleBarButton identifies one of the native-window commands.
type TitleBarButton int

const (
	TitleBarMinimize TitleBarButton = iota
	TitleBarMaximize
	TitleBarClose
)

// TitleBarHost is the default native-app title bar. It is deliberately an
// element so applications can replace it with their own top-level VNode.
type TitleBarHost struct {
	core.BaseComponent
	Title      string
	OnMinimize func()
	OnMaximize func()
	OnClose    func()

	hoveredBtn TitleBarButton
	pressedBtn TitleBarButton
	hasHover   bool
	hasPress   bool
}

func (b *TitleBarHost) MeasureContent(core.DrawingContext) {
	if b.Bounds.H == 0 {
		b.Bounds.H = TitleBarHeight
	}
}

func (b *TitleBarHost) Render(ctx core.DrawingContext) {
	if !b.Visible {
		return
	}
	width := b.Bounds.W
	if width <= 0 {
		return
	}
	th := theme.GetTheme()
	ctx.SetFont(th.UIFontFamily, titleBarFontSize(th))
	ctx.DrawRect(b.Bounds.X, b.Bounds.Y, width, TitleBarHeight, th.TitleBarBg, true)

	// Small built-in mark used as the default application icon. The native
	// window class also receives the compiled icon for the taskbar/title bar.
	iconX := b.Bounds.X + 12
	iconY := b.Bounds.Y + (TitleBarHeight-defaultTitleBarIconSize)/2
	iconColor := th.TitleToolbarActive
	if iconColor == 0 {
		iconColor = th.TitleBarFg
	}
	ctx.DrawRect(iconX, iconY, defaultTitleBarIconSize, defaultTitleBarIconSize, iconColor, true)
	ctx.DrawLine(iconX+3, iconY+4, iconX+defaultTitleBarIconSize-3, iconY+4, th.TitleBarBg, 1)
	ctx.DrawLine(iconX+3, iconY+8, iconX+defaultTitleBarIconSize-3, iconY+8, th.TitleBarBg, 1)

	textY := b.Bounds.Y + (TitleBarHeight-titleBarFontSize(th))/2 - 1
	ctx.DrawText(b.Bounds.X+34, textY, b.Title, th.TitleBarFg)

	for button := TitleBarMinimize; button <= TitleBarClose; button++ {
		x := b.Bounds.X + width - TitleBarButtonWidth*int32(TitleBarClose-button+1)
		bg := th.TitleBarBg
		if b.hasHover && b.hoveredBtn == button {
			bg = th.TitleBarBtnHover
			if button == TitleBarClose {
				bg = th.TitleBarCloseHover
			}
		}
		if b.hasPress && b.pressedBtn == button {
			bg = th.TitleBarBtnPress
			if button == TitleBarClose {
				bg = th.TitleBarClosePress
			}
		}
		ctx.DrawRect(x, b.Bounds.Y, TitleBarButtonWidth, TitleBarHeight, bg, true)
		drawTitleBarGlyph(ctx, button, x, b.Bounds.Y, th.TitleBarFg)
	}
}

func titleBarFontSize(th *theme.Theme) int32 {
	if th.TitleBarFontSize > 0 {
		return th.TitleBarFontSize
	}
	if th.UIFontSize > 0 {
		return th.UIFontSize
	}
	return 12
}

func drawTitleBarGlyph(ctx core.DrawingContext, button TitleBarButton, x, y int32, color uint32) {
	cx := x + TitleBarButtonWidth/2
	cy := y + TitleBarHeight/2
	switch button {
	case TitleBarMinimize:
		ctx.DrawLine(cx-7, cy, cx+7, cy, color, 1)
	case TitleBarMaximize:
		ctx.DrawRect(cx-6, cy-5, 12, 10, color, false)
	case TitleBarClose:
		ctx.DrawLine(cx-6, cy-6, cx+6, cy+6, color, 1)
		ctx.DrawLine(cx+6, cy-6, cx-6, cy+6, color, 1)
	}
}

func (b *TitleBarHost) buttonAt(x, y int32) (TitleBarButton, bool) {
	x -= b.Bounds.X
	y -= b.Bounds.Y
	start := b.Bounds.W - TitleBarButtonWidth*3
	if y < 0 || y >= TitleBarHeight || x < start || x >= b.Bounds.W {
		return 0, false
	}
	return TitleBarButton((x - start) / TitleBarButtonWidth), true
}

// IsDragRegion reports whether a point belongs to the draggable title area.
// The app shell uses it to leave the three command buttons as client pixels.
func (b *TitleBarHost) IsDragRegion(x, y int32) bool {
	_, button := b.buttonAt(x, y)
	return y >= b.Bounds.Y && y < b.Bounds.Y+TitleBarHeight && !button
}

func (b *TitleBarHost) HandleEvent(event core.Event) bool {
	if !b.Visible {
		return false
	}
	button, inside := b.buttonAt(event.X, event.Y)
	switch event.Type {
	case core.EventMouseLeave:
		changed := b.hasHover || b.hasPress
		b.hasHover, b.hasPress = false, false
		return changed
	case core.EventMouseMove:
		changed := b.hasHover != inside || (inside && b.hoveredBtn != button)
		b.hasHover, b.hoveredBtn = inside, button
		return changed
	case core.EventMouseDown:
		if !inside || event.Button != 0 {
			return false
		}
		b.hasPress, b.pressedBtn = true, button
		return true
	case core.EventMouseUp:
		if !b.hasPress {
			return false
		}
		pressed := b.pressedBtn
		b.hasPress = false
		if inside && button == pressed {
			switch pressed {
			case TitleBarMinimize:
				if b.OnMinimize != nil {
					b.OnMinimize()
				}
			case TitleBarMaximize:
				if b.OnMaximize != nil {
					b.OnMaximize()
				}
			case TitleBarClose:
				if b.OnClose != nil {
					b.OnClose()
				}
			}
		}
		return true
	}
	return false
}

type titleBarElement struct{}

func (titleBarElement) Tag() string { return "titlebar" }

func (titleBarElement) Create(props runtime.Props) core.Component {
	b := &TitleBarHost{
		Title:      props.Str("title", ""),
		OnMinimize: props.Fn("onMinimize"),
		OnMaximize: props.Fn("onMaximize"),
		OnClose:    props.Fn("onClose"),
		hoveredBtn: TitleBarClose + 1,
		pressedBtn: TitleBarClose + 1,
	}
	b.Visible = props.Bool("visible", true)
	return b
}

func (titleBarElement) Update(host core.Component, old, next runtime.Props) {
	b := host.(*TitleBarHost)
	b.Title = next.Str("title", "")
	b.OnMinimize = next.Fn("onMinimize")
	b.OnMaximize = next.Fn("onMaximize")
	b.OnClose = next.Fn("onClose")
	b.Visible = next.Bool("visible", true)
}

func init() { runtime.RegisterElement(titleBarElement{}) }
