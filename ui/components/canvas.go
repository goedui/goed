package components

import "github.com/goedui/goed/ui/core"

// clipStack is an optional extension for drawing contexts that support nested
// clipping. Canvas uses it when available: a canvas embedded in an already
// clipped container (scroll area, panel, editor viewport) must not destroy the
// outer clip when it finishes drawing. Contexts that only implement the plain
// DrawingContext keep working through SetClipRect/ResetClip.
type clipStack interface {
	PushClipRect(x, y, w, h int32)
	PopClip()
}

// Canvas is a retained-mode component for custom drawing and input.
// Coordinates passed to OnDraw and OnEvent are in the parent coordinate space,
// which keeps it useful for both small controls and full-window scenes.
type Canvas struct {
	*core.BaseComponent
	Background     uint32
	FillBackground bool

	// Focusable makes the canvas take part in keyboard focus: key events are
	// only delivered while it is focused, and a mouse press inside grabs focus.
	// A canvas that is not focusable keeps the legacy behaviour of receiving
	// every key event, which is what a single full-window scene expects.
	Focusable bool
	// MouseCapture keeps move/up events flowing to the canvas during a drag,
	// even when the pointer leaves the canvas bounds.
	MouseCapture bool

	OnDraw func(ctx core.DrawingContext, bounds core.Rect)
	// OnEvent is the generic handler invoked for any event that no typed
	// hook below claims. It receives keyboard events too.
	OnEvent      func(event core.Event) bool
	OnKeyDown    func(event core.Event) bool
	OnKeyUp      func(event core.Event) bool
	OnKeyChar    func(event core.Event) bool
	OnMouseDown  func(event core.Event) bool
	OnMouseUp    func(event core.Event) bool
	OnMouseMove  func(event core.Event) bool
	OnMouseWheel func(event core.Event) bool
	OnFocus      func(event core.Event) bool
	OnBlur       func(event core.Event) bool

	focused  bool
	capture  bool
	tracking bool
	dirty    bool
	dirtySet bool
	dirtyRct core.Rect
}

// NewCanvas creates a canvas. An optional initial rectangle is accepted so a
// canvas can be used directly or sized later by a layout.
func NewCanvas(bounds ...core.Rect) *Canvas {
	base := core.NewBaseComponent()
	canvas := &Canvas{BaseComponent: base, dirty: true}
	if len(bounds) > 0 {
		base.SetBounds(bounds[0])
	}
	return canvas
}

// SetDraw installs the custom renderer and returns the canvas for fluent setup.
func (c *Canvas) SetDraw(fn func(ctx core.DrawingContext, bounds core.Rect)) *Canvas {
	c.OnDraw = fn
	return c
}

// SetBackground paints a solid background before invoking OnDraw.
func (c *Canvas) SetBackground(color uint32) *Canvas {
	c.Background = color
	c.FillBackground = true
	return c
}

// SetEventHandler installs the generic event handler for fluent setup.
func (c *Canvas) SetEventHandler(fn func(event core.Event) bool) *Canvas {
	c.OnEvent = fn
	return c
}

// SetBounds resizes the canvas and repaints the whole surface, because the
// pixels of the previous size are no longer valid.
func (c *Canvas) SetBounds(rect core.Rect) {
	if c == nil || c.BaseComponent == nil {
		return
	}
	if c.Bounds == rect {
		return
	}
	c.BaseComponent.SetBounds(rect)
	c.Invalidate()
}

// SetDirtyTracking turns the invalidate/repaint protocol on. While enabled a
// Render call without a pending Invalidate is skipped, which lets a host avoid
// redrawing static canvases on every frame.
func (c *Canvas) SetDirtyTracking(enabled bool) *Canvas {
	if c == nil {
		return c
	}
	c.tracking = enabled
	if enabled {
		c.dirty = true
		c.dirtySet = false
	}
	return c
}

// Invalidate marks the whole canvas as needing a repaint.
func (c *Canvas) Invalidate() *Canvas {
	if c == nil {
		return c
	}
	c.dirty = true
	c.dirtySet = false
	return c
}

// InvalidateRect marks a sub-rectangle (in parent coordinates) as needing a
// repaint. Successive calls are merged into one repaint region.
func (c *Canvas) InvalidateRect(rect core.Rect) *Canvas {
	if c == nil {
		return c
	}
	c.dirty = true
	if !c.dirtySet {
		c.dirtyRct, c.dirtySet = rect, true
		return c
	}
	c.dirtyRct = unionRect(c.dirtyRct, rect)
	return c
}

// IsDirty reports whether the canvas needs to be rendered. It is always true
// for canvases that did not opt into dirty tracking.
func (c *Canvas) IsDirty() bool {
	if c == nil {
		return false
	}
	return !c.tracking || c.dirty
}

// SetFocus moves keyboard focus to (or away from) the canvas and fires the
// matching hook.
// SetFocused satisfies the runtime focus host contract.
func (c *Canvas) SetFocused(focused bool) { c.SetFocus(focused) }

func (c *Canvas) SetFocus(focused bool) *Canvas {
	if c == nil {
		return c
	}
	c.applyFocus(focused, core.Event{Type: core.EventFocus})
	return c
}

// HasFocus reports whether the canvas currently owns keyboard focus.
func (c *Canvas) HasFocus() bool { return c != nil && c.focused }

// ReleaseCapture ends an active pointer capture started by a mouse press.
func (c *Canvas) ReleaseCapture() {
	if c != nil {
		c.capture = false
	}
}

// ToLocal translates parent coordinates into canvas-local coordinates.
func (c *Canvas) ToLocal(x, y int32) (int32, int32) {
	if c == nil || c.BaseComponent == nil {
		return x, y
	}
	bounds := c.GetBounds()
	return x - bounds.X, y - bounds.Y
}

// LocalEvent returns a copy of the event with coordinates translated into
// canvas-local space, which is what most OnDraw/OnEvent pairs expect.
func (c *Canvas) LocalEvent(event core.Event) core.Event {
	event.X, event.Y = c.ToLocal(event.X, event.Y)
	return event
}

func (c *Canvas) Render(ctx core.DrawingContext) {
	if c == nil || c.BaseComponent == nil || !c.Visible {
		return
	}
	bounds := c.GetBounds()
	if bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	if !c.IsDirty() {
		return
	}
	clip := c.paintRect(bounds)
	if clip.W <= 0 || clip.H <= 0 {
		c.markClean()
		return
	}
	if c.FillBackground {
		ctx.DrawRect(clip.X, clip.Y, clip.W, clip.H, c.Background, true)
	}
	c.withClip(ctx, clip, func() {
		if c.OnDraw != nil {
			c.OnDraw(ctx, bounds)
		}
	})
	c.markClean()
}

func (c *Canvas) HandleEvent(event core.Event) bool {
	if c == nil || c.BaseComponent == nil || !c.Enabled || !c.Visible {
		return false
	}
	if event.Type == core.EventFocus || event.Type == core.EventBlur {
		return c.applyFocus(event.Type == core.EventFocus, event)
	}
	// 鼠标事件要求落在画布范围内（捕获期间除外）；键盘事件没有坐标语义，
	// 按焦点放行，否则游戏画布这类全屏组件永远收不到按键。
	if isMouseEvent(event.Type) {
		if !c.capture && !c.Contains(event.X, event.Y) {
			return false
		}
		switch event.Type {
		case core.EventMouseDown:
			if c.MouseCapture {
				c.capture = true
			}
			if c.Focusable {
				c.applyFocus(true, event)
			}
		case core.EventMouseUp:
			c.capture = false
		}
	}
	if isKeyEvent(event.Type) && c.Focusable && !c.focused {
		return false
	}
	if handler := c.typedHandler(event.Type); handler != nil {
		return handler(event)
	}
	if c.OnEvent != nil {
		return c.OnEvent(event)
	}
	return false
}

func (c *Canvas) typedHandler(t core.EventType) func(core.Event) bool {
	switch t {
	case core.EventKeyDown:
		return c.OnKeyDown
	case core.EventKeyUp:
		return c.OnKeyUp
	case core.EventKeyChar:
		return c.OnKeyChar
	case core.EventMouseDown:
		return c.OnMouseDown
	case core.EventMouseUp:
		return c.OnMouseUp
	case core.EventMouseMove:
		return c.OnMouseMove
	case core.EventMouseWheel:
		return c.OnMouseWheel
	case core.EventFocus:
		return c.OnFocus
	case core.EventBlur:
		return c.OnBlur
	}
	return nil
}

// applyFocus updates the focus state and notifies the hooks exactly once per
// change. Focus events coming from the event bus carry the original event,
// programmatic calls get a synthetic one.
func (c *Canvas) applyFocus(focused bool, event core.Event) bool {
	changed := c.focused != focused
	c.focused = focused
	if !focused {
		c.capture = false
	}
	if !changed {
		return false
	}
	if focused {
		if c.OnFocus != nil {
			return c.OnFocus(event)
		}
		return false
	}
	if c.OnBlur != nil {
		return c.OnBlur(event)
	}
	return false
}

// paintRect is the region that has to be repainted this frame.
func (c *Canvas) paintRect(bounds core.Rect) core.Rect {
	if !c.dirtySet {
		return bounds
	}
	return intersectRect(bounds, c.dirtyRct)
}

func (c *Canvas) markClean() {
	c.dirty = false
	c.dirtySet = false
}

// withClip keeps the clip region balanced even if the custom renderer panics.
func (c *Canvas) withClip(ctx core.DrawingContext, r core.Rect, draw func()) {
	if ctx == nil {
		return
	}
	if stack, ok := ctx.(clipStack); ok {
		stack.PushClipRect(r.X, r.Y, r.W, r.H)
		defer stack.PopClip()
	} else {
		ctx.SetClipRect(r.X, r.Y, r.W, r.H)
		defer ctx.ResetClip()
	}
	draw()
}

func isMouseEvent(t core.EventType) bool {
	return t >= core.EventMouseDown && t <= core.EventMouseWheel
}

func isKeyEvent(t core.EventType) bool {
	return t == core.EventKeyDown || t == core.EventKeyUp || t == core.EventKeyChar
}

func intersectRect(a, b core.Rect) core.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return core.Rect{}
	}
	return core.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func unionRect(a, b core.Rect) core.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return core.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
