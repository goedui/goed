package runtime

import (
	"github.com/goedui/goed/ui/core"
)

// HostElement 宿主元素接口：由 elements 包实现。
// runtime 通过该接口把 VNode 属性落到具体组件上，
// 避免对任何具体元素类型的依赖（对应 Vue 的 platform modules）。
type HostElement interface {
	// Create 创建宿主组件实例。
	Create(props Props) core.Component
	// Update 将新 props 应用到已有宿主组件（只处理变化的属性）。
	Update(host core.Component, old, new Props)
	// Tag 返回元素标签名。
	Tag() string
}

var hostElements = map[string]HostElement{}

// RegisterElement 注册宿主元素实现（elements 包在 init 中调用）。
func RegisterElement(el HostElement) {
	hostElements[el.Tag()] = el
}

// HasElement 报告指定标签的宿主元素是否已注册。
func HasElement(tag string) bool {
	_, ok := hostElements[tag]
	return ok
}

// ChildrenOf 提取组件的子组件列表。
// 所有嵌入 core.BaseComponent 的宿主（BoxHost/ButtonHost/TextHost...）
// 都能被正确穿透；无法识别的类型返回 nil。
func ChildrenOf(c core.Component) []core.Component {
	switch h := c.(type) {
	case *core.BaseComponent:
		return h.Children
	case interface{ GetChildren() []core.Component }:
		return h.GetChildren()
	}
	return nil
}

// WalkTree 深度优先遍历宿主树（父 → 子）。
// fn 返回 true 时停止遍历。
func WalkTree(root core.Component, fn func(c core.Component) bool) {
	if root == nil || fn(root) {
		return
	}
	for _, child := range ChildrenOf(root) {
		WalkTree(child, fn)
	}
}

// DispatchEvent 向宿主树冒泡分发事件：先问最深的命中组件，
// 再逐层向上，任一层返回 true即停止。
// 鼠标事件按坐标命中；其他事件广播给所有组件。
func DispatchEvent(root core.Component, event core.Event) bool {
	if event.Type == core.EventMouseLeave {
		return dispatchMouseLeave(root, event, true)
	}
	left := false
	if event.Type == core.EventMouseMove {
		// Leaving feedback must be cleared before a sibling consumes the move.
		left = dispatchMouseLeave(root, event, false)
	}
	if captureEvent(root, event) {
		return true
	}
	if isMouse(event.Type) {
		// 自底向上找命中组件链。
		return dispatchMouse(root, root, event) || left
	}
	if isKey(event.Type) {
		if event.Type == core.EventKeyDown && event.Key == core.KeyTab {
			return moveFocus(root, event.Shift)
		}
		if focusedRoot != root || focusedComponent == nil {
			return false
		}
		return focusedComponent.HandleEvent(event)
	}
	handled := false
	WalkTree(root, func(c core.Component) bool {
		if c.HandleEvent(event) {
			handled = true
			return true // 停止
		}
		return false
	})
	return handled
}

func dispatchMouseLeave(root core.Component, event core.Event, all bool) bool {
	handled := false
	event.Type = core.EventMouseLeave
	WalkTree(root, func(c core.Component) bool {
		if all || !contains(c, event.X, event.Y) {
			if c.HandleEvent(event) {
				handled = true
			}
		}
		return false // Every departing component must receive the notification.
	})
	return handled
}

func isMouse(t core.EventType) bool {
	return t >= core.EventMouseDown && t <= core.EventMouseWheel
}

func isKey(t core.EventType) bool {
	return t >= core.EventKeyDown && t <= core.EventKeyChar
}

// Focus is intentionally kept at the runtime boundary: a renderer owns the
// component tree, while the platform only forwards key messages. Goed has one
// UI thread, so a small active-root pointer is sufficient and avoids requiring
// every host component to know about the window implementation.
var (
	focusedRoot      core.Component
	focusedComponent core.Component
)

type focusableHost interface {
	SetFocused(bool)
}

func focusComponent(root, target core.Component) {
	if focusedRoot == root && focusedComponent == target {
		return
	}
	if previous, ok := focusedComponent.(focusableHost); ok {
		previous.SetFocused(false)
	}
	focusedRoot, focusedComponent = root, target
	if next, ok := target.(focusableHost); ok {
		next.SetFocused(true)
	}
	revealFocus(root, target)
}

func clearFocus(root core.Component) {
	if focusedRoot != root {
		return
	}
	if previous, ok := focusedComponent.(focusableHost); ok {
		previous.SetFocused(false)
	}
	focusedRoot, focusedComponent = nil, nil
}

func moveFocus(root core.Component, reverse bool) bool {
	var focusables []core.Component
	WalkTree(root, func(c core.Component) bool {
		if !available(c) {
			return true
		}
		if f, ok := c.(interface{ CanFocus() bool }); ok && !f.CanFocus() {
			return false
		}
		if _, ok := c.(focusableHost); ok {
			focusables = append(focusables, c)
		}
		return false
	})
	if len(focusables) == 0 {
		return false
	}
	index := -1
	for i, candidate := range focusables {
		if candidate == focusedComponent {
			index = i
			break
		}
	}
	if reverse {
		index--
		if index < 0 {
			index = len(focusables) - 1
		}
	} else {
		index++
		if index >= len(focusables) {
			index = 0
		}
	}
	focusComponent(root, focusables[index])
	return true
}

func dispatchMouse(root, c core.Component, event core.Event) bool {
	if c == nil || !available(c) {
		return false
	}
	if clip, ok := c.(interface{ EventClip() core.Rect }); ok {
		b := clip.EventClip()
		if event.X < b.X || event.Y < b.Y || event.X >= b.X+b.W || event.Y >= b.Y+b.H {
			dispatchMouseLeave(c, event, true)
			return false
		}
	}
	// 先给子组件机会（后挂载的在上层）。
	children := ChildrenOf(c)
	for i := len(children) - 1; i >= 0; i-- {
		if dispatchMouse(root, children[i], event) {
			return true
		}
	}
	// 坐标不在自己范围内则不处理（无坐标语义的组件自行决定）。
	if event.Type == core.EventMouseDown && contains(c, event.X, event.Y) {
		if _, ok := c.(focusableHost); ok {
			focusComponent(root, c)
		}
	}
	return c.HandleEvent(event)
}

func contains(c core.Component, x, y int32) bool {
	b := c.GetBounds()
	return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
}
