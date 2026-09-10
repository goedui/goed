package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// ScrollHost clips both painting and pointer routing while preserving child hosts.
type ScrollHost struct {
	core.BaseComponent
	Offset, ContentHeight int32
	toEnd                 bool
}

func (s *ScrollHost) EventClip() core.Rect                              { return s.Bounds }
func (s *ScrollHost) MeasureWidth(core.DrawingContext, int32) core.Size { return core.Size{} }
func (s *ScrollHost) ScrollToEnd()                                      { s.toEnd = true }
func (s *ScrollHost) Render(ctx core.DrawingContext) {
	if !s.Visible {
		return
	}
	s.ContentHeight = 0
	for _, c := range s.Children {
		size := core.MeasureComponent(ctx, c, max(0, s.Bounds.W-8))
		s.ContentHeight += size.Height
	}
	limit := max(0, s.ContentHeight-s.Bounds.H)
	if s.toEnd {
		s.Offset = limit
		s.toEnd = false
	}
	s.Offset = max(0, min(s.Offset, limit))
	y := s.Bounds.Y - s.Offset
	core.WithClip(ctx, s.Bounds, func() {
		for _, c := range s.Children {
			b := c.GetBounds()
			c.SetBounds(core.Rect{X: s.Bounds.X, Y: y, W: max(0, s.Bounds.W-8), H: b.H})
			c.Render(ctx)
			y += b.H
		}
	})
	if limit > 0 {
		h := max(int32(28), s.Bounds.H*s.Bounds.H/max(1, s.ContentHeight))
		y := s.Bounds.Y + s.Offset*(s.Bounds.H-h)/limit
		ctx.DrawRoundedRect(s.Bounds.X+s.Bounds.W-5, y, 4, h, 2, theme.GetTheme().SearchBorder, true)
	}
}
func (s *ScrollHost) HandleEvent(e core.Event) bool {
	if e.Type == core.EventMouseWheel && s.Contains(e.X, e.Y) {
		s.Offset = max(0, min(max(0, s.ContentHeight-s.Bounds.H), s.Offset-e.Delta/3))
		return true
	}
	return false
}

// Reveal keeps tab-focused descendants in the visible viewport.
func (s *ScrollHost) Reveal(target core.Component) {
	found := false
	runtime.WalkTree(s, func(c core.Component) bool {
		if c == target {
			found = true
		}
		return found
	})
	if !found {
		return
	}
	b := target.GetBounds()
	if b.Y < s.Bounds.Y {
		s.Offset -= s.Bounds.Y - b.Y
	} else if b.Y+b.H > s.Bounds.Y+s.Bounds.H {
		s.Offset += b.Y + b.H - s.Bounds.Y - s.Bounds.H
	}
	s.Offset = max(0, min(max(0, s.ContentHeight-s.Bounds.H), s.Offset))
}

type scrollElement struct{}

func (scrollElement) Tag() string { return "scroll" }
func (scrollElement) Create(p runtime.Props) core.Component {
	s := &ScrollHost{BaseComponent: *core.NewBaseComponent()}
	scrollElement{}.Update(s, nil, p)
	return s
}
func (scrollElement) Update(c core.Component, old, p runtime.Props) {
	s := c.(*ScrollHost)
	if p.Int("scrollToEnd", 0) != old.Int("scrollToEnd", 0) {
		s.toEnd = true
	}
}
func init() { runtime.RegisterElement(scrollElement{}) }
