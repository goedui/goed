package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

type StatusBar struct {
	Height    int32
	LeftText  string
	RightText string
}

func NewStatusBar() *StatusBar {
	return &StatusBar{
		Height: 24,
	}
}

func (s *StatusBar) Render(ctx core.DrawingContext, x, y, width int32) {
	th := theme.GetTheme()
	// 字体阶梯：Secondary 档（字体规范.md 第 4 节）。
	ctx.SetFont(th.UIFontFamily, th.FontSecondary())

	// Draw background - VSCode 蓝色
	ctx.DrawRect(x, y, width, s.Height, th.StatusBarBg, true)

	rightWidth := int32(0)
	if s.RightText != "" {
		rightWidth, _ = ctx.MeasureText(s.RightText)
	}
	leftText := fitStatusText(ctx, s.LeftText, width-rightWidth-36)

	// Draw left text - 垂直居中
	textY := y + (s.Height-14)/2
	ctx.DrawText(x+12, textY, leftText, th.StatusBarFg)

	// Draw right text
	if s.RightText != "" {
		textWidth, _ := ctx.MeasureText(s.RightText)
		ctx.DrawText(x+width-textWidth-12, textY, s.RightText, th.StatusBarFg)
	}
}

func fitStatusText(ctx core.DrawingContext, text string, maxWidth int32) string {
	if text == "" || maxWidth <= 0 {
		return ""
	}
	if measured, _ := ctx.MeasureText(text); measured <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		candidate := "..." + string(runes)
		if measured, _ := ctx.MeasureText(candidate); measured <= maxWidth {
			return candidate
		}
		runes = runes[1:]
	}
	return ""
}

func (s *StatusBar) SetLeft(text string) {
	s.LeftText = text
}

func (s *StatusBar) SetRight(text string) {
	s.RightText = text
}
