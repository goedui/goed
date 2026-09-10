package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// LayerHost composes normal content with anchored popovers, painted last.
type LayerHost struct {
	core.BaseComponent
	OnKeyDown func(core.Event) bool
}

func (l *LayerHost) CaptureEvent(e core.Event) bool {
	return e.Type == core.EventKeyDown && l.OnKeyDown != nil && l.OnKeyDown(e)
}
func (l *LayerHost) Render(ctx core.DrawingContext) {
	if !l.Visible {
		return
	}
	for _, c := range l.Children {
		c.SetBounds(l.Bounds)
		if p, ok := c.(*PopoverHost); ok {
			if anchor := runtime.FindByID(l, p.AnchorID); anchor != nil {
				p.anchor = anchor.GetBounds()
			}
		}
		c.Render(ctx)
	}
}

type layerElement struct{}

func (layerElement) Tag() string { return "layer" }
func (layerElement) Create(p runtime.Props) core.Component {
	l := &LayerHost{BaseComponent: *core.NewBaseComponent()}
	layerElement{}.Update(l, nil, p)
	return l
}
func (layerElement) Update(c core.Component, _, p runtime.Props) {
	c.(*LayerHost).OnKeyDown, _ = p["onKeyDown"].(func(core.Event) bool)
}

// PopoverHost owns selection, keyboard navigation, dismissal and wheel scrolling.
type PopoverHost struct {
	core.BaseComponent
	AnchorID      string
	Items         []string
	OnSelect      func(string)
	OnClose       func()
	anchor, popup core.Rect
	index, first  int
}

func (p *PopoverHost) Render(ctx core.DrawingContext) {
	if !p.Visible {
		return
	}
	w := min(max(int32(160), p.anchor.W), max(0, p.Bounds.W-16))
	h := int32(min(6, len(p.Items)))*30 + 8
	x := max(p.Bounds.X+8, min(p.anchor.X, p.Bounds.X+p.Bounds.W-w-8))
	y := p.anchor.Y + p.anchor.H + 4
	if y+h > p.Bounds.Y+p.Bounds.H-8 {
		y = p.anchor.Y - h - 4
	}
	p.popup = core.Rect{X: x, Y: max(p.Bounds.Y+8, y), W: w, H: h}
	ctx.SetFont(theme.GetTheme().UIFontFamily, theme.GetTheme().UIFontSize)
	ctx.DrawRoundedRect(x, p.popup.Y, w, h, 8, theme.GetTheme().WindowBg, true)
	ctx.DrawRoundedRect(x, p.popup.Y, w, h, 8, theme.GetTheme().SearchBorder, false)
	for i := 0; i < 6 && p.first+i < len(p.Items); i++ {
		y := p.popup.Y + 4 + int32(i)*30
		if p.first+i == p.index {
			ctx.DrawRoundedRect(x+3, y, w-6, 30, 5, theme.GetTheme().TitleBarBtnHover, true)
		}
		ctx.DrawText(x+10, y+5, fitLabel(ctx, p.Items[p.first+i], w-20), theme.GetTheme().TitleBarFg)
	}
}
func (p *PopoverHost) CaptureEvent(e core.Event) bool {
	if !p.Visible {
		return false
	}
	inside := e.X >= p.popup.X && e.Y >= p.popup.Y && e.X < p.popup.X+p.popup.W && e.Y < p.popup.Y+p.popup.H
	close := func() {
		if p.OnClose != nil {
			p.OnClose()
		}
	}
	selectAt := func(i int) {
		if i >= 0 && i < len(p.Items) && p.OnSelect != nil {
			p.OnSelect(p.Items[i])
		}
	}
	switch e.Type {
	case core.EventMouseDown:
		if e.Button == 0 {
			if inside {
				selectAt(p.first + int((e.Y-p.popup.Y-4)/30))
			} else {
				close()
			}
		}
		return true
	case core.EventMouseUp:
		return true
	case core.EventMouseMove:
		if inside {
			p.index = max(0, min(len(p.Items)-1, p.first+int((e.Y-p.popup.Y-4)/30)))
		}
		return true
	case core.EventMouseWheel:
		p.first = max(0, min(max(0, len(p.Items)-6), p.first-int(e.Delta/120)))
		return true
	case core.EventKeyDown:
		switch e.Key {
		case 27:
			close()
		case core.KeyEnter:
			selectAt(p.index)
		case 38:
			p.index = max(0, p.index-1)
		case 40:
			p.index = min(len(p.Items)-1, p.index+1)
		case core.KeyTab:
			close()
		}
		p.first = max(0, min(p.first, p.index))
		if p.index >= p.first+6 {
			p.first = p.index - 5
		}
		return true
	case core.EventKeyUp, core.EventKeyChar:
		return true
	}
	return false
}

type popoverElement struct{}

func (popoverElement) Tag() string { return "popover" }
func (popoverElement) Create(p runtime.Props) core.Component {
	h := &PopoverHost{BaseComponent: *core.NewBaseComponent()}
	popoverElement{}.Update(h, nil, p)
	return h
}
func (popoverElement) Update(c core.Component, _, p runtime.Props) {
	h := c.(*PopoverHost)
	h.AnchorID = p.Str("anchor", "")
	h.Items, _ = p["items"].([]string)
	h.OnSelect = p.StringFn("onSelect")
	h.OnClose = p.Fn("onClose")
	h.index = max(0, min(h.index, len(h.Items)-1))
	h.first = max(0, min(h.first, max(0, len(h.Items)-6)))
}
func init() { runtime.RegisterElement(layerElement{}); runtime.RegisterElement(popoverElement{}) }
