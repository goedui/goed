package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
	"math"
)

func init() {
	renderer.Register("slider", renderer.Widget{Measure: measureSlider, Paint: paintSlider})
}

// Slider is a controlled numeric range. OnChange receives func(float64).
func Slider(p runtime.Props, min, max, step float64) *runtime.VNode {
	if max <= min {
		max = min + 1
	}
	if step <= 0 {
		step = 1
	}
	n := runtime.H("slider", p)
	n.Attrs["min"], n.Attrs["max"], n.Attrs["step"] = min, max, step
	change := func(value float64) {
		if !renderer.Enabled(n) {
			return
		}
		value = sliderClamp(value, min, max, step)
		n.Attrs["value"] = value
		if fn, ok := n.Attrs["onChange"].(func(float64)); ok && fn != nil {
			fn(value)
		}
	}
	point := func() {
		if !renderer.Enabled(n) {
			return
		}
		runtime.SetFocus(n.ID())
		x, _ := renderer.PointerPos()
		width := n.W - 12
		if width <= 0 {
			return
		}
		change(min + float64((x-n.X-6)/width)*(max-min))
	}
	n.OnClick = point
	n.Attrs["onPointerDown"] = point
	n.Attrs["onPointerDrag"] = point
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		if !e.Down || !renderer.Enabled(n) || !runtime.IsFocused(n) {
			return false
		}
		value := sliderValue(n)
		switch e.Key {
		case runtime.KeyLeft, runtime.KeyDown:
			value -= step
		case runtime.KeyRight, runtime.KeyUp:
			value += step
		case runtime.KeyHome:
			value = min
		case runtime.KeyEnd:
			value = max
		default:
			return false
		}
		change(value)
		return true
	})
	return n
}

func sliderClamp(v, min, max, step float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	v = min + math.Round((v-min)/step)*step
	return math.Min(max, math.Max(min, v))
}

func sliderValue(n *runtime.VNode) float64 {
	switch v := n.Attrs["value"].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	}
	return 0
}

func measureSlider(_ renderer.Context, n *runtime.VNode, maxW, _ float32, _ renderer.TextStyle, _ theme.Theme) (float32, float32) {
	w := n.Style.Width
	if w <= 0 {
		w = maxW
	}
	if w <= 0 {
		w = 160
	}
	return w, 24
}

func paintSlider(c renderer.Context, n *runtime.VNode, _ renderer.TextStyle, th theme.Theme) {
	min, _ := n.Attrs["min"].(float64)
	max, _ := n.Attrs["max"].(float64)
	step, _ := n.Attrs["step"].(float64)
	value := sliderClamp(sliderValue(n), min, max, step)
	progress := float32((value - min) / (max - min))
	bg, fg := renderer.ColorFrom(th.Separator), renderer.ColorFrom(th.Button)
	if n.Style.Color.A > 0 {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	if !renderer.Enabled(n) {
		bg.A *= .45
		fg.A *= .45
	}
	w := n.W - 12
	if w < 0 {
		w = 0
	}
	x, y := n.X+6, n.Y+n.H/2
	c.FillRoundedRect(x, y-2, w, 4, 2, bg)
	c.FillRoundedRect(x, y-2, w*progress, 4, 2, fg)
	c.FillRoundedRect(x+w*progress-6, y-6, 12, 12, 6, fg)
	if n.ID() != "" && runtime.FocusedID() == n.ID() {
		c.DrawRect(n.X, n.Y, n.W, n.H, fg, 1)
	}
}
