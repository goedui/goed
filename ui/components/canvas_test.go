package components

import (
	"testing"

	"github.com/goedui/goed/ui/core"
)

type canvasTestContext struct{ clips, resets, rects, ellipses, polygons int }

func (c *canvasTestContext) DrawRect(int32, int32, int32, int32, uint32, bool)               { c.rects++ }
func (c *canvasTestContext) DrawLine(int32, int32, int32, int32, uint32, int32)              {}
func (c *canvasTestContext) DrawRoundedRect(int32, int32, int32, int32, int32, uint32, bool) {}
func (c *canvasTestContext) DrawEllipse(int32, int32, int32, int32, uint32, bool)            { c.ellipses++ }
func (c *canvasTestContext) DrawPolygon([]core.Point, uint32, bool)                          { c.polygons++ }
func (c *canvasTestContext) DrawText(int32, int32, string, uint32)                           {}
func (c *canvasTestContext) MeasureText(string) (int32, int32)                               { return 0, 0 }
func (c *canvasTestContext) SetFont(string, int32)                                           {}
func (c *canvasTestContext) SetBkMode(bool)                                                  {}
func (c *canvasTestContext) BeginDraw()                                                      {}
func (c *canvasTestContext) EndDraw()                                                        {}
func (c *canvasTestContext) SetClipRect(int32, int32, int32, int32)                          { c.clips++ }
func (c *canvasTestContext) ResetClip()                                                      { c.resets++ }

// rectRecordingContext 记录一次背景填充的矩形，用于验证局部重绘范围。
type rectRecordingContext struct {
	canvasTestContext
	filled core.Rect
}

func (r *rectRecordingContext) DrawRect(x, y, w, h int32, _ uint32, _ bool) {
	r.filled = core.Rect{X: x, Y: y, W: w, H: h}
}

// canvasClipContext 支持嵌套裁剪，用于验证 Canvas 不会破坏外层裁剪区。
type canvasClipContext struct {
	canvasTestContext
	stack []core.Rect
	clip  core.Rect
}

func (c *canvasClipContext) PushClipRect(x, y, w, h int32) {
	next := core.Rect{X: x, Y: y, W: w, H: h}
	if len(c.stack) > 0 {
		next = intersectRect(c.stack[len(c.stack)-1], next)
	}
	c.stack = append(c.stack, next)
	c.clip = next
}

func (c *canvasClipContext) PopClip() {
	if n := len(c.stack); n > 0 {
		c.stack = c.stack[:n-1]
	}
	if n := len(c.stack); n > 0 {
		c.clip = c.stack[n-1]
		return
	}
	c.clip = core.Rect{}
}

func TestCanvasRenderClipsAndDraws(t *testing.T) {
	ctx := &canvasTestContext{}
	called := false
	canvas := NewCanvas(core.Rect{X: 4, Y: 8, W: 100, H: 60}).SetBackground(1)
	canvas.SetDraw(func(_ core.DrawingContext, bounds core.Rect) {
		called = bounds == (core.Rect{X: 4, Y: 8, W: 100, H: 60})
	})
	canvas.Render(ctx)
	if !called || ctx.clips != 1 || ctx.resets != 1 || ctx.rects != 1 {
		t.Fatalf("canvas render = called %v, clips %d, resets %d, rects %d", called, ctx.clips, ctx.resets, ctx.rects)
	}
}

func TestCanvasEventBounds(t *testing.T) {
	canvas := NewCanvas(core.Rect{X: 10, Y: 10, W: 20, H: 20})
	called := false
	canvas.OnEvent = func(event core.Event) bool { called = event.Type == core.EventMouseDown; return true }
	if !canvas.HandleEvent(core.Event{Type: core.EventMouseDown, X: 10, Y: 10}) || !called {
		t.Fatal("expected an in-bounds event to be handled")
	}
	if canvas.HandleEvent(core.Event{Type: core.EventMouseDown, X: 30, Y: 30}) {
		t.Fatal("expected an out-of-bounds event to be ignored")
	}
}

func TestCanvasKeyboardEventsBypassHitTest(t *testing.T) {
	canvas := NewCanvas(core.Rect{X: 100, Y: 100, W: 20, H: 20})
	keys := []int32(nil)
	canvas.OnKeyDown = func(event core.Event) bool { keys = append(keys, event.Key); return true }
	// 事件坐标 (0,0) 在画布外，键盘事件仍必须送达
	if !canvas.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A'}) || len(keys) != 1 || keys[0] != 'A' {
		t.Fatal("expected key events to reach the canvas regardless of position")
	}
	// 通用 OnEvent 也能收到键盘事件
	got := false
	canvas2 := NewCanvas(core.Rect{X: 100, Y: 100, W: 20, H: 20})
	canvas2.OnEvent = func(event core.Event) bool { got = event.Type == core.EventKeyUp; return true }
	if !canvas2.HandleEvent(core.Event{Type: core.EventKeyUp, Key: 'B'}) || !got {
		t.Fatal("expected OnEvent to fall back for keyboard events")
	}
}

func TestCanvasTypedHooksTakePrecedence(t *testing.T) {
	canvas := NewCanvas(core.Rect{X: 0, Y: 0, W: 50, H: 50})
	typed, generic := false, false
	canvas.OnMouseDown = func(core.Event) bool { typed = true; return true }
	canvas.OnEvent = func(core.Event) bool { generic = true; return true }
	if !canvas.HandleEvent(core.Event{Type: core.EventMouseDown, X: 5, Y: 5}) || !typed || generic {
		t.Fatal("expected the typed hook to claim the event")
	}
}

func TestCanvasDirtyTrackingSkipsCleanFrames(t *testing.T) {
	ctx := &canvasTestContext{}
	draws := 0
	canvas := NewCanvas(core.Rect{W: 60, H: 40}).SetDirtyTracking(true)
	canvas.SetDraw(func(core.DrawingContext, core.Rect) { draws++ })

	if !canvas.IsDirty() {
		t.Fatal("a freshly tracked canvas should render once")
	}
	canvas.Render(ctx)
	if draws != 1 {
		t.Fatalf("draws = %d, want 1", draws)
	}
	if canvas.IsDirty() {
		t.Fatal("canvas should be clean after a render")
	}
	canvas.Render(ctx)
	if draws != 1 {
		t.Fatalf("draws = %d, want the clean frame to be skipped", draws)
	}
	canvas.Invalidate()
	canvas.Render(ctx)
	if draws != 2 {
		t.Fatalf("draws = %d, want Invalidate to force a repaint", draws)
	}
}

func TestCanvasInvalidateRectRepaintsRegionOnly(t *testing.T) {
	rec := &rectRecordingContext{}
	canvas := NewCanvas(core.Rect{X: 10, Y: 10, W: 100, H: 80}).SetDirtyTracking(true)
	canvas.FillBackground = true
	canvas.SetDraw(func(core.DrawingContext, core.Rect) {})
	canvas.InvalidateRect(core.Rect{X: 40, Y: 20, W: 30, H: 25})
	canvas.Render(rec)
	if rec.filled != (core.Rect{X: 40, Y: 20, W: 30, H: 25}) {
		t.Fatalf("background fill = %+v, want the dirty region only", rec.filled)
	}

	// 脏区超出画布时应被裁剪到画布范围
	rec2 := &rectRecordingContext{}
	canvas2 := NewCanvas(core.Rect{X: 10, Y: 10, W: 100, H: 80}).SetDirtyTracking(true)
	canvas2.FillBackground = true
	canvas2.SetDraw(func(core.DrawingContext, core.Rect) {})
	canvas2.InvalidateRect(core.Rect{X: 90, Y: 60, W: 100, H: 100})
	canvas2.Render(rec2)
	if rec2.filled != (core.Rect{X: 90, Y: 60, W: 20, H: 30}) {
		t.Fatalf("background fill = %+v, want the clipped dirty region", rec2.filled)
	}
}

func TestCanvasSetBoundsInvalidates(t *testing.T) {
	canvas := NewCanvas(core.Rect{W: 50, H: 50}).SetDirtyTracking(true)
	canvas.Render(&canvasTestContext{})
	if canvas.IsDirty() {
		t.Fatal("canvas should be clean after render")
	}
	canvas.SetBounds(core.Rect{W: 80, H: 50})
	if !canvas.IsDirty() {
		t.Fatal("resizing must request a repaint")
	}
	canvas.SetBounds(core.Rect{W: 80, H: 50})
	// 尺寸未变时不应重复触发重绘请求
	canvas.Invalidate()
	canvas.SetBounds(core.Rect{W: 80, H: 50})
	if !canvas.IsDirty() {
		t.Fatal("an unchanged bounds assignment should keep the pending repaint")
	}
}

func TestCanvasFocusGatesKeyboard(t *testing.T) {
	canvas := NewCanvas(core.Rect{W: 40, H: 40})
	canvas.Focusable = true
	got := 0
	canvas.OnKeyDown = func(core.Event) bool { got++; return true }

	if canvas.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A'}) {
		t.Fatal("a focusable canvas must not receive keys while unfocused")
	}
	canvas.HandleEvent(core.Event{Type: core.EventFocus})
	if !canvas.HasFocus() {
		t.Fatal("focus event should mark the canvas as focused")
	}
	if !canvas.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A'}) || got != 1 {
		t.Fatal("focused canvas should receive the key")
	}
	canvas.HandleEvent(core.Event{Type: core.EventBlur})
	if canvas.HasFocus() || canvas.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A'}) {
		t.Fatal("blur must stop keyboard delivery")
	}
}

func TestCanvasMouseDownGrabsFocusAndCapture(t *testing.T) {
	canvas := NewCanvas(core.Rect{X: 20, Y: 20, W: 40, H: 40})
	canvas.Focusable = true
	canvas.MouseCapture = true
	moves := 0
	canvas.OnMouseMove = func(core.Event) bool { moves++; return true }

	canvas.HandleEvent(core.Event{Type: core.EventMouseDown, X: 25, Y: 25})
	if !canvas.HasFocus() {
		t.Fatal("a press inside a focusable canvas should grab focus")
	}
	// 拖出画布后仍应收到移动事件
	if !canvas.HandleEvent(core.Event{Type: core.EventMouseMove, X: 200, Y: 200}) || moves != 1 {
		t.Fatal("captured drags must keep receiving move events outside the canvas")
	}
	canvas.HandleEvent(core.Event{Type: core.EventMouseUp, X: 200, Y: 200})
	if canvas.HandleEvent(core.Event{Type: core.EventMouseMove, X: 200, Y: 200}) {
		t.Fatal("the capture must be released on mouse up")
	}
}

func TestCanvasLocalCoordinates(t *testing.T) {
	canvas := NewCanvas(core.Rect{X: 30, Y: 40, W: 100, H: 50})
	lx, ly := canvas.ToLocal(35, 42)
	if lx != 5 || ly != 2 {
		t.Fatalf("local = (%d,%d), want (5,2)", lx, ly)
	}
	local := canvas.LocalEvent(core.Event{Type: core.EventMouseDown, X: 60, Y: 90})
	if local.X != 30 || local.Y != 50 {
		t.Fatalf("local event = (%d,%d), want (30,50)", local.X, local.Y)
	}
}

func TestCanvasNestedClipPreservesOuterClip(t *testing.T) {
	ctx := &canvasClipContext{}
	// 外层容器先把裁剪限制到 (0,0,50,50)
	ctx.PushClipRect(0, 0, 50, 50)
	var inner core.Rect
	canvas := NewCanvas(core.Rect{X: 10, Y: 10, W: 100, H: 100})
	canvas.SetDraw(func(_ core.DrawingContext, _ core.Rect) { inner = ctx.clip })
	canvas.Render(ctx)
	if inner != (core.Rect{X: 10, Y: 10, W: 40, H: 40}) {
		t.Fatalf("clip inside the canvas = %+v, want the intersection with the outer clip", inner)
	}
	// 画布画完后外层裁剪必须原样恢复
	if ctx.clip != (core.Rect{X: 0, Y: 0, W: 50, H: 50}) || len(ctx.stack) != 1 {
		t.Fatalf("outer clip = %+v, stack = %d, want the outer clip restored", ctx.clip, len(ctx.stack))
	}
}

func TestCanvasRestoresClipWhenDrawPanics(t *testing.T) {
	ctx := &canvasTestContext{}
	canvas := NewCanvas(core.Rect{W: 20, H: 20}).SetDraw(func(core.DrawingContext, core.Rect) {
		panic("draw failed")
	})
	func() {
		defer func() { _ = recover() }()
		canvas.Render(ctx)
	}()
	if ctx.resets != 1 || ctx.clips != 1 {
		t.Fatalf("clips = %d, resets = %d, want the clip to be balanced after a panic", ctx.clips, ctx.resets)
	}
}
