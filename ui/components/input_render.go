package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
	"strings"
)

func (i *InputHost) buildLines(ctx core.DrawingContext, text string) {
	r := []rune(text)
	i.lines = nil
	line := inputLine{widths: []int32{0}}
	limit := max(int32(1), i.Bounds.W-22-i.RightPadding)
	for n, c := range r {
		if c == '\n' && i.Multiline {
			line.end = n
			i.lines = append(i.lines, line)
			line = inputLine{start: n + 1, widths: []int32{0}}
			continue
		}
		w, _ := ctx.MeasureText(string(r[line.start : n+1]))
		if i.Multiline && w > limit && n > line.start {
			line.end = n
			i.lines = append(i.lines, line)
			line = inputLine{start: n, widths: []int32{0}}
			w, _ = ctx.MeasureText(string(c))
		}
		line.widths = append(line.widths, w)
	}
	line.end = len(r)
	i.lines = append(i.lines, line)
}
func (i *InputHost) caretRow() int {
	for n := len(i.lines) - 1; n >= 0; n-- {
		if i.caret >= i.lines[n].start {
			return n
		}
	}
	return 0
}
func (i *InputHost) hitCaret(x, y int32) {
	if len(i.lines) == 0 {
		i.caret = len([]rune(i.Value))
		return
	}
	row := 0
	if i.Multiline {
		row = int((y - i.Bounds.Y - 10 + i.scrollY) / max(1, i.lineHeight))
	}
	row = max(0, min(row, len(i.lines)-1))
	l := i.lines[row]
	px := x - i.Bounds.X - 10 + i.scrollX
	col := len(l.widths) - 1
	for n := 1; n < len(l.widths); n++ {
		if px < (l.widths[n-1]+l.widths[n])/2 {
			col = n - 1
			break
		}
	}
	i.caret = l.start + col
}

// Render draws wrapped text, selection and a caret clipped to the field.
func (i *InputHost) Render(ctx core.DrawingContext) {
	if !i.Visible {
		return
	}
	th := theme.GetTheme()
	ctx.SetFont(th.UIFontFamily, th.UIFontSize)
	bg, fg, border := th.SearchBg, th.TextPrimary, th.SearchBorder
	if i.focused {
		border = th.ButtonBg
	}
	if !i.Enabled {
		fg = th.TextDisabled
	}
	r := i.Bounds
	ctx.DrawRoundedRect(r.X, r.Y, r.W, r.H, i.Radius, bg, true)
	ctx.DrawRoundedRect(r.X, r.Y, r.W, r.H, i.Radius, border, false)
	text := i.Value
	if i.Password {
		text = strings.Repeat("•", len([]rune(text)))
	}
	i.caret = clampCaret(i.caret, i.Value)
	i.anchor = clampCaret(i.anchor, i.Value)
	i.buildLines(ctx, text)
	_, fontH := ctx.MeasureText("Ag")
	i.lineHeight = max(fontH, th.UIFontSize) + 4
	row := i.caretRow()
	l := i.lines[row]
	col := min(i.caret-l.start, len(l.widths)-1)
	cw := l.widths[max(0, col)]
	y := r.Y + (r.H-fontH)/2
	if i.Multiline {
		y = r.Y + 10
		i.scrollX = 0
		if i.focused {
			i.scrollY = max(0, min(i.scrollY, int32(row)*i.lineHeight))
			i.scrollY = max(i.scrollY, int32(row+1)*i.lineHeight-(r.H-20))
		}
	} else {
		i.scrollY = 0
		if i.focused {
			i.scrollX = max(0, min(i.scrollX, cw))
			i.scrollX = max(i.scrollX, cw-max(1, r.W-24-i.RightPadding))
		}
	}
	if stack, ok := ctx.(clipStack); ok {
		stack.PushClipRect(r.X+5, r.Y+3, max(0, r.W-10-i.RightPadding), max(0, r.H-6))
		defer stack.PopClip()
	} else {
		ctx.SetClipRect(r.X+5, r.Y+3, max(0, r.W-10-i.RightPadding), max(0, r.H-6))
		defer ctx.ResetClip()
	}
	if text == "" {
		ctx.DrawText(r.X+10, y, i.Placeholder, th.TextSecondary)
	} else {
		a, b := i.selection()
		runes := []rune(text)
		for n, line := range i.lines {
			ly := y + int32(n)*i.lineHeight - i.scrollY
			if ly+i.lineHeight < r.Y || ly > r.Y+r.H {
				continue
			}
			if i.focused && a != b {
				start, end := max(a, line.start), min(b, line.end)
				if start < end {
					sx, ex := line.widths[start-line.start], line.widths[end-line.start]
					ctx.DrawRect(r.X+10+sx-i.scrollX, ly, ex-sx, i.lineHeight, th.EditorSelectionBg, true)
				}
			}
			ctx.DrawText(r.X+10-i.scrollX, ly, string(runes[line.start:line.end]), fg)
		}
	}
	if i.focused && i.Enabled {
		cx := r.X + 10 + cw - i.scrollX
		cy := y + int32(row)*i.lineHeight - i.scrollY
		ctx.DrawLine(cx, cy, cx, cy+fontH, th.EditorCursorColor, 1)
	}
}
