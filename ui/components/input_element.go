package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
	"strings"
)

// InputHost is a focusable text field with optional wrapping and password masking.
type InputHost struct {
	core.BaseComponent
	Value, Placeholder  string
	OnInput, OnChange   func(string)
	OnFocus, OnBlur     func()
	Multiline, Password bool
	Height, Radius      int32
	// RightPadding reserves room for an adjacent dropdown arrow.
	RightPadding     int32
	ReadClipboard    func() string
	WriteClipboard   func(string)
	focused          bool
	caret, anchor    int
	scrollX, scrollY int32
	lines            []inputLine
	lineHeight       int32
}
type inputLine struct {
	start, end int
	widths     []int32
}

// NewInput creates a text field. Clipboard providers are supplied by its host.
func NewInput(value string) *InputHost {
	return &InputHost{BaseComponent: *core.NewBaseComponent(), Value: value, caret: len([]rune(value)), anchor: len([]rune(value))}
}
func (i *InputHost) SetFocused(v bool) {
	if i.focused == v {
		return
	}
	i.focused = v
	if v {
		i.caret = len([]rune(i.Value))
		i.anchor = i.caret
		if i.OnFocus != nil {
			i.OnFocus()
		}
	} else if i.OnBlur != nil {
		i.OnBlur()
	}
}
func (i *InputHost) Focused() bool { return i.focused }
func (i *InputHost) MeasureContent(ctx core.DrawingContext) {
	ctx.SetFont(theme.GetTheme().UIFontFamily, theme.GetTheme().UIFontSize)
	if i.Bounds.W == 0 {
		i.Bounds.W = 160
	}
	if i.Bounds.H == 0 {
		i.Bounds.H = i.Height
		if i.Bounds.H <= 0 {
			i.Bounds.H = 32
		}
	}
}
func (i *InputHost) selection() (int, int) { return min(i.caret, i.anchor), max(i.caret, i.anchor) }
func (i *InputHost) emitInput() {
	if i.OnInput != nil {
		i.OnInput(i.Value)
	}
}
func (i *InputHost) insert(value []rune) {
	a, b := i.selection()
	r := []rune(i.Value)
	result := append([]rune{}, r[:a]...)
	result = append(result, value...)
	result = append(result, r[b:]...)
	i.Value = string(result)
	i.caret = a + len(value)
	i.anchor = i.caret
	i.emitInput()
}
func (i *InputHost) HandleEvent(e core.Event) bool {
	if !i.Visible || !i.Enabled {
		return false
	}
	i.caret = clampCaret(i.caret, i.Value)
	i.anchor = clampCaret(i.anchor, i.Value)
	switch e.Type {
	case core.EventFocus:
		i.SetFocused(true)
		return true
	case core.EventBlur:
		i.SetFocused(false)
		return true
	case core.EventMouseDown:
		if !i.Contains(e.X, e.Y) {
			return false
		}
		i.hitCaret(e.X, e.Y)
		if !e.Shift {
			i.anchor = i.caret
		}
		return true
	case core.EventKeyChar:
		if !i.focused || e.Ctrl || e.Char < ' ' {
			return false
		}
		i.insert([]rune{e.Char})
		return true
	case core.EventKeyDown:
		if !i.focused {
			return false
		}
		if e.Ctrl {
			a, b := i.selection()
			switch e.Key {
			case 'A':
				i.anchor = 0
				i.caret = len([]rune(i.Value))
				return true
			case 'C', 'X':
				if !i.Password && a != b && i.WriteClipboard != nil {
					i.WriteClipboard(string([]rune(i.Value)[a:b]))
					if e.Key == 'X' {
						i.insert(nil)
					}
				}
				return true
			case 'V':
				if i.ReadClipboard != nil {
					v := strings.ReplaceAll(i.ReadClipboard(), "\r\n", "\n")
					v = strings.ReplaceAll(v, "\r", "\n")
					if !i.Multiline {
						v = strings.ReplaceAll(v, "\n", " ")
					}
					i.insert([]rune(v))
				}
				return true
			}
		}
		switch e.Key {
		case core.KeyBackspace:
			a, b := i.selection()
			if a == b && i.caret > 0 {
				i.anchor = i.caret - 1
			}
			i.insert(nil)
		case core.KeyDelete:
			a, b := i.selection()
			if a == b && i.caret < len([]rune(i.Value)) {
				i.anchor = i.caret + 1
			}
			i.insert(nil)
		case core.KeyLeft:
			a, b := i.selection()
			if a != b && !e.Shift {
				i.caret = a
			} else {
				i.caret = max(0, i.caret-1)
			}
			if !e.Shift {
				i.anchor = i.caret
			}
		case core.KeyRight:
			a, b := i.selection()
			if a != b && !e.Shift {
				i.caret = b
			} else {
				i.caret = min(len([]rune(i.Value)), i.caret+1)
			}
			if !e.Shift {
				i.anchor = i.caret
			}
		case core.KeyHome, core.KeyEnd:
			row := i.caretRow()
			i.caret = 0
			if e.Key == core.KeyEnd {
				i.caret = len([]rune(i.Value))
			}
			if !e.Ctrl && len(i.lines) > 0 {
				if e.Key == core.KeyHome {
					i.caret = i.lines[row].start
				} else {
					i.caret = i.lines[row].end
				}
			}
			if !e.Shift {
				i.anchor = i.caret
			}
		case 38, 40:
			if !i.Multiline {
				return false
			}
			row := i.caretRow()
			if len(i.lines) > 0 {
				col := i.caret - i.lines[row].start
				if e.Key == 38 {
					row = max(0, row-1)
				} else {
					row = min(len(i.lines)-1, row+1)
				}
				i.caret = min(i.lines[row].end, i.lines[row].start+col)
			}
			if !e.Shift {
				i.anchor = i.caret
			}
		case core.KeyEnter:
			if i.Multiline {
				i.insert([]rune{'\n'})
			} else if i.OnChange != nil {
				i.OnChange(i.Value)
			}
		default:
			return false
		}
		return true
	}
	return false
}
func clampCaret(c int, v string) int { return max(0, min(c, len([]rune(v)))) }

type inputElement struct{}

func (inputElement) Tag() string { return "input" }
func (inputElement) Create(p runtime.Props) core.Component {
	v, _ := inputValue(p)
	i := NewInput(v)
	inputElement{}.Update(i, nil, p)
	return i
}
func (inputElement) Update(c core.Component, _, p runtime.Props) {
	i := c.(*InputHost)
	if v, ok := inputValue(p); ok {
		i.Value = v
		i.caret = clampCaret(i.caret, v)
		i.anchor = clampCaret(i.anchor, v)
	}
	i.Placeholder = p.Str("placeholder", "")
	i.OnInput = p.StringFn("onInput")
	i.OnChange = p.StringFn("onChange")
	i.OnFocus = p.Fn("onFocus")
	i.OnBlur = p.Fn("onBlur")
	i.Multiline = p.Bool("multiline", false)
	i.Password = p.Bool("password", false)
	i.Height = p.Int("height", 0)
	i.Radius = p.Int("radius", 0)
	i.RightPadding = p.Int("rightPadding", 0)
	i.Visible = p.Bool("visible", true)
	i.Enabled = p.Bool("enabled", true)
}
func inputValue(p runtime.Props) (string, bool) {
	if v, ok := p["modelValue"].(string); ok {
		return v, true
	}
	v, ok := p["value"].(string)
	return v, ok
}
