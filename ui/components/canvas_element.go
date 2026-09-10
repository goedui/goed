package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/platform/windows"
	"github.com/goedui/goed/ui/runtime"
)

type canvasElement struct{}

func (canvasElement) Tag() string { return "canvas" }
func (canvasElement) Create(props runtime.Props) core.Component {
	c := NewCanvas()
	c.MouseCapture = props.Bool("mouseCapture", true)
	c.Focusable = props.Bool("focusable", true)
	if draw, ok := props["onDraw"].(func(core.DrawingContext, core.Rect)); ok {
		c.OnDraw = draw
	}
	if event, ok := props["onEvent"].(func(core.Event) bool); ok {
		c.OnEvent = event
	}
	c.Visible = props.Bool("visible", true)
	return &canvasHost{Canvas: c, flex: props.Int("flex", 0), configure: windowFn(props["onWindow"]), drag: dragFn(props["dragRegion"])}
}
func (canvasElement) Update(host core.Component, old, next runtime.Props) {
	h := host.(*canvasHost)
	h.flex = next.Int("flex", 0)
	h.configure = windowFn(next["onWindow"])
	h.drag = dragFn(next["dragRegion"])
	h.Canvas.MouseCapture = next.Bool("mouseCapture", true)
	h.Canvas.Focusable = next.Bool("focusable", true)
	if draw, ok := next["onDraw"].(func(core.DrawingContext, core.Rect)); ok {
		h.Canvas.OnDraw = draw
	}
	if event, ok := next["onEvent"].(func(core.Event) bool); ok {
		h.Canvas.OnEvent = event
	}
	h.Visible = next.Bool("visible", true)
}

type canvasHost struct {
	*Canvas
	flex      int32
	configure func(*windows.FramelessWindow)
	drag      func(int32, int32) bool
}

func (c *canvasHost) Flex() int32 { return c.flex }
func (c *canvasHost) ConfigureWindow(w *windows.FramelessWindow) {
	if c.configure != nil {
		c.configure(w)
	}
}
func (c *canvasHost) IsDragRegion(x, y int32) bool { return c.drag != nil && c.drag(x, y) }
func windowFn(v any) func(*windows.FramelessWindow) {
	f, _ := v.(func(*windows.FramelessWindow))
	return f
}
func dragFn(v any) func(int32, int32) bool { f, _ := v.(func(int32, int32) bool); return f }
func init()                                { runtime.RegisterElement(canvasElement{}) }
