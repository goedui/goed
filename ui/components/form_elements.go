package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func fitLabel(ctx core.DrawingContext, text string, width int32) string {
	if w, _ := ctx.MeasureText(text); w <= width {
		return text
	}
	r := []rune(text)
	for len(r) > 0 {
		r = r[:len(r)-1]
		v := string(r) + "…"
		if w, _ := ctx.MeasureText(v); w <= width {
			return v
		}
	}
	return ""
}

// SwitchHost shares keyboard and pointer behavior with buttons.
type SwitchHost struct {
	ButtonHost
	Value    bool
	OnChange func(bool)
}

func (s *SwitchHost) MeasureContent(core.DrawingContext) { s.Bounds.W, s.Bounds.H = 34, 20 }
func (s *SwitchHost) Render(ctx core.DrawingContext) {
	if !s.Visible {
		return
	}
	r := s.Bounds
	bg := theme.GetTheme().SearchBorder
	if s.Value {
		bg = theme.GetTheme().ButtonBg
	}
	ctx.DrawRoundedRect(r.X, r.Y, r.W, r.H, r.H/2, bg, true)
	x := r.X + 2
	if s.Value {
		x = r.X + r.W - r.H + 2
	}
	ctx.DrawEllipse(x, r.Y+2, r.H-4, r.H-4, 0xFFFFFF, true)
	if s.focused {
		ctx.DrawRoundedRect(r.X-2, r.Y-2, r.W+4, r.H+4, r.H/2+2, theme.GetTheme().ButtonBg, false)
	}
}

type switchElement struct{}

func (switchElement) Tag() string { return "switch" }
func (switchElement) Create(p runtime.Props) core.Component {
	s := &SwitchHost{ButtonHost: ButtonHost{BaseComponent: *core.NewBaseComponent(), Enabled: true}}
	s.OnClick = func() {
		s.Value = !s.Value
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
	}
	switchElement{}.Update(s, nil, p)
	return s
}
func (switchElement) Update(c core.Component, _, p runtime.Props) {
	s := c.(*SwitchHost)
	s.Value = p.Bool("value", false)
	s.Enabled = p.Bool("enabled", true)
	s.OnChange, _ = p["onChange"].(func(bool))
}
func init() { runtime.RegisterElement(switchElement{}) }
