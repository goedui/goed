package core

// LayoutSpec describes requested dimensions, separately from arranged bounds.
type LayoutSpec struct {
	Width, Height, Flex int32
}

// MeasureComponent recalculates natural size without retaining stale flex bounds.
// Width-aware containers can wrap or reflow when a window is resized.
func MeasureComponent(ctx DrawingContext, c Component, availableWidth int32) Size {
	if visible, ok := c.(interface{ IsVisible() bool }); ok && !visible.IsVisible() {
		return Size{}
	}
	var spec LayoutSpec
	declared := false
	if p, ok := c.(interface{ LayoutProperties() (LayoutSpec, bool) }); ok {
		spec, declared = p.LayoutProperties()
	}
	if spec.Width > 0 {
		availableWidth = spec.Width
	}
	b := c.GetBounds()
	if m, ok := c.(interface {
		MeasureWidth(DrawingContext, int32) Size
	}); ok {
		size := m.MeasureWidth(ctx, max(0, availableWidth))
		b.W, b.H = size.Width, size.Height
	} else if m, ok := c.(interface{ MeasureContent(DrawingContext) }); ok {
		if declared {
			c.SetBounds(Rect{X: b.X, Y: b.Y})
		}
		m.MeasureContent(ctx)
		b = c.GetBounds()
	}
	if spec.Width > 0 {
		b.W = spec.Width
	}
	if spec.Height > 0 {
		b.H = spec.Height
	}
	c.SetBounds(b)
	return Size{Width: b.W, Height: b.H}
}

// WithClip restores the parent's clip after drawing a nested viewport.
func WithClip(ctx DrawingContext, rect Rect, draw func()) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if stack, ok := ctx.(interface {
		PushClipRect(int32, int32, int32, int32)
		PopClip()
	}); ok {
		stack.PushClipRect(rect.X, rect.Y, rect.W, rect.H)
		defer stack.PopClip()
	} else {
		ctx.SetClipRect(rect.X, rect.Y, rect.W, rect.H)
		defer ctx.ResetClip()
	}
	draw()
}
