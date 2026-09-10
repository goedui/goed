package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

type ButtonState int

const (
	StateNormal ButtonState = iota
	StateHover
	StatePressed
	StateDisabled
)

type Button struct {
	Text    string
	X, Y    int32
	Width   int32
	Height  int32
	Enabled bool
	OnClick func()

	state ButtonState
}

func NewButton(text string, x, y, width, height int32) *Button {
	return &Button{
		Text:    text,
		X:       x,
		Y:       y,
		Width:   width,
		Height:  height,
		Enabled: true,
		state:   StateNormal,
	}
}

func (b *Button) Render(ctx core.DrawingContext) {
	th := theme.GetTheme()
	ctx.SetFont(th.UIFontFamily, th.UIFontSize)
	var bgColor, fgColor uint32

	if !b.Enabled {
		bgColor = th.ButtonDisabledBg
		fgColor = th.ButtonDisabledFg
	} else {
		switch b.state {
		case StateNormal:
			bgColor = th.ButtonBg
			fgColor = th.ButtonFg
		case StateHover:
			bgColor = th.ButtonHoverBg
			fgColor = th.ButtonFg
		case StatePressed:
			bgColor = th.ButtonPressBg
			fgColor = th.ButtonFg
		}
	}

	// Draw button background
	ctx.DrawRect(b.X, b.Y, b.Width, b.Height, bgColor, true)

	// Draw border
	ctx.DrawRect(b.X, b.Y, b.Width, b.Height, th.ButtonBorder, false)

	// Draw text (centered)
	textWidth, textHeight := ctx.MeasureText(b.Text)
	textX := b.X + (b.Width-textWidth)/2
	textY := b.Y + (b.Height-textHeight)/2
	ctx.DrawText(textX, textY, b.Text, fgColor)
}

func (b *Button) HandleMouseMove(x, y int32) {
	if !b.Enabled {
		return
	}

	if b.Contains(x, y) {
		if b.state == StateNormal {
			b.state = StateHover
		}
	} else {
		if b.state == StateHover {
			b.state = StateNormal
		}
	}
}

func (b *Button) HandleMouseDown(x, y int32) bool {
	if !b.Enabled {
		return false
	}

	if b.Contains(x, y) {
		b.state = StatePressed
		return true
	}
	return false
}

func (b *Button) HandleMouseUp(x, y int32) bool {
	if !b.Enabled {
		return false
	}

	if b.state == StatePressed {
		if b.Contains(x, y) {
			if b.OnClick != nil {
				b.OnClick()
			}
		}
		b.state = StateHover
		return true
	}
	return false
}

func (b *Button) Contains(x, y int32) bool {
	return x >= b.X && x < b.X+b.Width &&
		y >= b.Y && y < b.Y+b.Height
}

// SetHovered 供宿主组件（如 Dialog）在无完整鼠标事件流时驱动悬停态。
func (b *Button) SetHovered(hovered bool) {
	if hovered && b.state == StateNormal {
		b.state = StateHover
	} else if !hovered && b.state == StateHover {
		b.state = StateNormal
	}
}

// Pressed 报告按钮是否处于按下态。
func (b *Button) Pressed() bool { return b.state == StatePressed }

// HandleClick 命中即触发 OnClick（单事件点击语义，供模态对话框等
// 只接收 MouseDown 的宿主复用按钮渲染与命中判定）。
func (b *Button) HandleClick(x, y int32) bool {
	if !b.Enabled || !b.Contains(x, y) {
		return false
	}
	if b.OnClick != nil {
		b.OnClick()
	}
	return true
}
