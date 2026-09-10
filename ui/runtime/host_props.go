package runtime

import "github.com/goedui/goed/ui/core"

func applyHostProps(host core.Component, p Props) {
	if h, ok := host.(interface{ SetLayoutSpec(core.LayoutSpec) }); ok {
		h.SetLayoutSpec(core.LayoutSpec{Width: p.Int("width", 0), Height: p.Int("height", 0), Flex: p.Int("flex", 0)})
	}
	if h, ok := host.(interface{ SetElementState(string, bool, bool) }); ok {
		h.SetElementState(p.Str("id", ""), p.Bool("visible", true), p.Bool("enabled", true))
	}
}

func hostRef(p Props, c core.Component) {
	if ref, ok := p["ref"].(func(core.Component)); ok {
		ref(c)
	}
}

// FindByID finds a mounted element without exposing application-specific hosts.
func FindByID(root core.Component, id string) core.Component {
	var found core.Component
	WalkTree(root, func(c core.Component) bool {
		if h, ok := c.(interface{ ElementID() string }); ok && h.ElementID() == id {
			found = c
		}
		return found != nil
	})
	return found
}
