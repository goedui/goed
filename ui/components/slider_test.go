package components

import (
	"reflect"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestSliderPointerKeyboardBoundsAndDisabled(t *testing.T) {
	t.Cleanup(func() { renderer.SetPointer(0, 0, false); runtime.SetFocus("") })
	var values []float64
	n := Slider(runtime.Props{ID: "range-interaction", Value: 30.0, Style: runtime.Style{Width: 200},
		OnChange: func(v float64) { values = append(values, v) }}, 10, 100, 5)
	renderer.PaintTree(&mockContext{w: 300, h: 120}, []*runtime.VNode{n}, theme.Dark)
	key := n.Attrs["onKeyDown"].(runtime.KeyHandler)
	if key(runtime.KeyEvent{Key: runtime.KeyRight, Down: true}) {
		t.Fatal("unfocused slider consumed an arrow key")
	}
	renderer.SetPointer(n.X+n.W/2, n.Y+12, true)
	n.Attrs["onPointerDown"].(func())()
	if runtime.FocusedID() != n.ID() {
		t.Fatal("pointer must focus the slider")
	}
	renderer.SetPointer(n.X+n.W+100, n.Y+12, true)
	n.Attrs["onPointerDrag"].(func())()
	key(runtime.KeyEvent{Key: runtime.KeyLeft, Down: true})
	key(runtime.KeyEvent{Key: runtime.KeyHome, Down: true})
	key(runtime.KeyEvent{Key: runtime.KeyEnd, Down: true})
	if !reflect.DeepEqual(values, []float64{55, 100, 95, 10, 100}) {
		t.Fatalf("interaction values = %v", values)
	}
	n.Attrs["disabled"] = true
	n.OnClick()
	n.Attrs["onPointerDrag"].(func())()
	if key(runtime.KeyEvent{Key: runtime.KeyLeft, Down: true}) || len(values) != 5 {
		t.Fatal("disabled slider changed its value")
	}
}
