package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
	"strings"
	"unicode"
)

// TextRun preserves inline formatting while a paragraph wraps across lines.
type TextRun struct {
	Text                                  string
	Color                                 runtime.Color
	Bold, Italic, Code, Underline, Strike bool
	OnClick                               func()
}

type richFragment struct {
	text       string
	x, y, w, h float32
	style      renderer.TextStyle
	run        TextRun
}

func init() {
	renderer.Register("richtext", renderer.Widget{Measure: measureRichText, Layout: layoutRichText, Paint: paintRichText})
}

func RichText(p runtime.Props, runs ...TextRun) *runtime.VNode {
	n := runtime.H("richtext", p)
	n.Attrs["runs"] = runs
	var text strings.Builder
	for _, run := range runs {
		text.WriteString(run.Text)
	}
	n.Text = text.String()
	return n
}

func richStyle(base renderer.TextStyle, run TextRun) renderer.TextStyle {
	if base.FontSize <= 0 {
		base.FontSize = 13
	}
	if run.Bold {
		base.Weight = 700
	}
	base.Italic = run.Italic
	if run.Color.A > 0 {
		base.Color = renderer.ColorFrom(run.Color)
	}
	if run.Code {
		base.FontFamily = "Consolas"
		base.FontSize *= .95
	}
	base.NoWrap = true
	return base
}

// Keep words together, with rune boundaries for CJK and overlong identifiers.
func richTokens(text string) []string {
	var tokens []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}
	for _, r := range text {
		switch {
		case r == '\r':
			continue
		case r == '\n' || unicode.IsSpace(r):
			flush()
			tokens = append(tokens, string(r))
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			flush()
			tokens = append(tokens, string(r))
		default:
			word.WriteRune(r)
		}
	}
	flush()
	return tokens
}

func richFlow(c renderer.Context, n *runtime.VNode, width float32, base renderer.TextStyle) ([]richFragment, float32) {
	pad := n.Style.Padding
	width -= 2 * pad
	if width < 1 {
		width = 1
	}
	runs, _ := n.Attrs["runs"].([]TextRun)
	lineH := base.FontSize * 1.6
	if lineH < 20 {
		lineH = 20
	}
	probe := base
	probe.NoWrap = true
	_, measuredH := c.MeasureText("Mg中", 100000, probe)
	if measuredH > lineH {
		lineH = measuredH
	}
	x, y := float32(0), float32(0)
	var result []richFragment
	var add func(string, TextRun, renderer.TextStyle)
	add = func(text string, run TextRun, style renderer.TextStyle) {
		if text == "\n" {
			x = 0
			y += lineH
			return
		}
		w, h := c.MeasureText(text, 100000, style)
		if w > width && len([]rune(text)) > 1 {
			for _, r := range text {
				add(string(r), run, style)
			}
			return
		}
		if x > 0 && x+w > width {
			x = 0
			y += lineH
		}
		if count := len(result); count > 0 {
			last := &result[count-1]
			if last.y == y && last.x+last.w == x && last.style == style && last.run.Code == run.Code && last.run.Underline == run.Underline && last.run.Strike == run.Strike && last.run.OnClick == nil && run.OnClick == nil {
				last.text += text
				last.w += w
				x += w
				return
			}
		}
		if h < lineH {
			h = lineH
		}
		result = append(result, richFragment{text: text, x: x, y: y, w: w, h: h, style: style, run: run})
		x += w
	}
	for _, run := range runs {
		style := richStyle(base, run)
		for _, token := range richTokens(run.Text) {
			add(token, run, style)
		}
	}
	if len(result) == 0 {
		return nil, 0
	}
	return result, y + lineH + 2*pad
}

func measureRichText(c renderer.Context, n *runtime.VNode, maxW, _ float32, style renderer.TextStyle, _ theme.Theme) (float32, float32) {
	width := renderer.InnerWidth(n, maxW) + 2*n.Style.Padding
	if n.Style.Width <= 0 && maxW <= 0 {
		var line float32
		runs, _ := n.Attrs["runs"].([]TextRun)
		for _, run := range runs {
			for i, part := range strings.Split(run.Text, "\n") {
				if i > 0 {
					width = max(width, line+2*n.Style.Padding)
					line = 0
				}
				w, _ := c.MeasureText(part, 100000, richStyle(style, run))
				line += w
			}
		}
		width = max(width, line+2*n.Style.Padding)
	}
	if n.Style.MaxWidth > 0 {
		width = min(width, n.Style.MaxWidth)
	}
	_, height := richFlow(c, n, width, style)
	return width, height
}

func layoutRichText(c renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, _ theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	fragments, _ := richFlow(c, n, w, style)
	n.Attrs["fragments"] = fragments
	n.Children = nil
	for _, fragment := range fragments {
		if fragment.run.OnClick == nil {
			continue
		}
		child := runtime.Text(runtime.Props{OnClick: fragment.run.OnClick}, fragment.text)
		child.X, child.Y, child.W, child.H = x+n.Style.Padding+fragment.x, y+n.Style.Padding+fragment.y, fragment.w, fragment.h
		n.Children = append(n.Children, child)
	}
}

func paintRichText(c renderer.Context, n *runtime.VNode, _ renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(c, n)
	fragments, _ := n.Attrs["fragments"].([]richFragment)
	for _, f := range fragments {
		x, y := n.X+n.Style.Padding+f.x, n.Y+n.Style.Padding+f.y
		if f.run.Code {
			c.FillRoundedRect(x-1, y+1, f.w+2, f.h-2, 3, renderer.ColorFrom(th.Separator))
		}
		c.DrawText(f.text, x, y, f.w+.5, f.h, f.style)
		if f.run.Underline {
			c.DrawLine(x, y+f.h-3, x+f.w, y+f.h-3, f.style.Color, 1)
		}
		if f.run.Strike {
			c.DrawLine(x, y+f.h*.55, x+f.w, y+f.h*.55, f.style.Color, 1)
		}
	}
}
