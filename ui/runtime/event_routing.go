package runtime

import "github.com/goedui/goed/ui/core"

func available(c core.Component) bool {
	if v, ok := c.(interface{ IsVisible() bool }); ok && !v.IsVisible() {
		return false
	}
	if e, ok := c.(interface{ IsEnabled() bool }); ok && !e.IsEnabled() {
		return false
	}
	return true
}
func captureEvent(c core.Component, e core.Event) bool {
	if c == nil || !available(c) {
		return false
	}
	children := ChildrenOf(c)
	for i := len(children) - 1; i >= 0; i-- {
		if captureEvent(children[i], e) {
			return true
		}
	}
	if capture, ok := c.(interface{ CaptureEvent(core.Event) bool }); ok {
		return capture.CaptureEvent(e)
	}
	return false
}
func revealFocus(root, target core.Component) {
	WalkTree(root, func(c core.Component) bool {
		if viewport, ok := c.(interface{ Reveal(core.Component) }); ok {
			viewport.Reveal(target)
		}
		return false
	})
}
