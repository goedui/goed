package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"testing"
)

func TestKeyedConditionalChildrenKeepOrderAndInputIdentity(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	defer r.Unmount()
	show := runtime.Ref(false)
	reverse := runtime.Ref(false)
	r.Mount(func() []*runtime.VNode {
		a := runtime.Input(runtime.Props{"id": "a", "value": "draft"}).WithKey("a")
		b := runtime.H("button", runtime.Props{"id": "b", "label": "B"}).WithKey("b")
		var added *runtime.VNode
		if show.Get() {
			added = runtime.H("button", runtime.Props{"id": "added", "label": "Added"}).WithKey("added")
		}
		children := []*runtime.VNode{a, added, b}
		if reverse.Get() {
			children = []*runtime.VNode{b, added, a}
		}
		return []*runtime.VNode{runtime.H("vstack", runtime.Props{"gap": 8, "padding": 12}, children...)}
	})
	render := func() { root.Children[0].SetBounds(core.Rect{W: 320, H: 240}); root.Render(&variableCtx{}) }
	render()
	a := runtime.FindByID(root, "a")
	b := runtime.FindByID(root, "b")
	oldY := b.GetBounds().Y
	show.Set(true)
	render()
	if b.GetBounds().Y <= oldY {
		t.Fatal("conditional child did not reflow stack")
	}
	reverse.Set(true)
	render()
	if runtime.FindByID(root, "a") != a || runtime.FindByID(root, "b") != b {
		t.Fatal("keyed move replaced a stable host")
	}
	if b.GetBounds().Y >= a.GetBounds().Y {
		t.Fatal("keyed order did not change")
	}
	show.Set(false)
	render()
	if runtime.FindByID(root, "added") != nil {
		t.Fatal("conditional host not removed")
	}
}

func TestScrollClipsPointerEventsAndRevealsKeyboardFocus(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	defer r.Unmount()
	clicks := 0
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.H("scroll", runtime.Props{"id": "scroll"}, runtime.H("vstack", runtime.Props{"gap": 8},
			runtime.H("button", runtime.Props{"id": "first", "label": "First", "height": 40, "onClick": func() { clicks++ }}),
			runtime.H("button", runtime.Props{"id": "disabled", "label": "Disabled", "enabled": false, "height": 40}),
			runtime.Input(runtime.Props{"id": "last", "height": 40}),
		))}
	})
	render := func() { root.Children[0].SetBounds(core.Rect{Y: 40, W: 240, H: 60}); root.Render(&variableCtx{}) }
	render()
	viewport := root.Children[0].(*ScrollHost)
	first := runtime.FindByID(root, "first")
	viewport.Offset = 50
	render()
	b := first.GetBounds()
	runtime.DispatchEvent(root, core.Event{Type: core.EventMouseDown, X: 10, Y: b.Y + 10})
	runtime.DispatchEvent(root, core.Event{Type: core.EventMouseUp, X: 10, Y: b.Y + 10})
	if clicks != 0 {
		t.Fatal("offscreen button received click")
	}
	runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: core.KeyTab})
	render()
	if viewport.Offset != 0 {
		t.Fatal("focus did not reveal first button")
	}
	runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: core.KeyTab})
	render()
	last := runtime.FindByID(root, "last").(*InputHost)
	if !last.Focused() || viewport.Offset == 0 {
		t.Fatal("tab did not skip disabled field or reveal input")
	}
}
