package components

import (
	"fmt"
	"testing"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
)

// A content panel consuming mouse moves used to prevent titlebar hover cleanup.
type hoverContent struct{ core.BaseComponent }

func (p *hoverContent) HandleEvent(e core.Event) bool {
	return e.Type == core.EventMouseMove && p.Contains(e.X, e.Y)
}

func TestTitleBarHoverClearsWhenPointerLeaves(t *testing.T) {
	for _, width := range []int32{480, 1200} {
		for button := TitleBarMinimize; button <= TitleBarClose; button++ {
			for _, destination := range []string{"content", "caption", "window"} {
				t.Run(fmt.Sprintf("%d/%d/%s", width, button, destination), func(t *testing.T) {
					root := core.NewBaseComponent()
					bar := &TitleBarHost{BaseComponent: *core.NewBaseComponent()}
					bar.SetBounds(core.Rect{W: width, H: TitleBarHeight})
					panel := &hoverContent{BaseComponent: *core.NewBaseComponent()}
					panel.SetBounds(core.Rect{Y: TitleBarHeight, W: width, H: 700})
					root.AddChild(bar)
					root.AddChild(panel)
					x := width - TitleBarButtonWidth*int32(TitleBarClose-button+1) + 20
					move := core.Event{Type: core.EventMouseMove, X: x, Y: 20}
					runtime.DispatchEvent(root, move)
					if !bar.hasHover || bar.hoveredBtn != button {
						t.Fatal("button did not hover")
					}
					switch destination {
					case "content":
						runtime.DispatchEvent(root, core.Event{Type: core.EventMouseMove, X: x, Y: 100})
					case "caption":
						runtime.DispatchEvent(root, core.Event{Type: core.EventMouseMove, X: 80, Y: 20})
					case "window":
						runtime.DispatchEvent(root, core.Event{Type: core.EventMouseLeave})
					}
					if bar.hasHover {
						t.Fatal("hover remained after leaving button")
					}
					runtime.DispatchEvent(root, move)
					if !bar.hasHover || bar.hoveredBtn != button {
						t.Fatal("hover did not return on re-entry")
					}
				})
			}
		}
	}
}

func TestTitleBarLeaveCancelsPressedCommand(t *testing.T) {
	calls := 0
	bar := &TitleBarHost{BaseComponent: *core.NewBaseComponent(), OnClose: func() { calls++ }}
	bar.SetBounds(core.Rect{W: 480, H: TitleBarHeight})
	bar.HandleEvent(core.Event{Type: core.EventMouseMove, X: 460, Y: 20})
	bar.HandleEvent(core.Event{Type: core.EventMouseDown, X: 460, Y: 20})
	bar.HandleEvent(core.Event{Type: core.EventMouseLeave})
	bar.HandleEvent(core.Event{Type: core.EventMouseUp, X: 460, Y: 20})
	if bar.hasHover || bar.hasPress || calls != 0 {
		t.Fatal("leaving retained a pressed command")
	}
}
