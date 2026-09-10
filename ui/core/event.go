package core

// EventType 事件类型
type EventType int

const (
	EventMouseDown EventType = iota
	EventMouseUp
	EventMouseMove
	EventMouseWheel
	EventKeyDown
	EventKeyUp
	EventKeyChar
	EventFocus
	EventBlur
	// EventMouseLeave clears pointer feedback when a component or window is left.
	EventMouseLeave
)

// Common virtual-key values used by keyboard events.
const (
	KeyBackspace int32 = 8
	KeyTab       int32 = 9
	KeyEnter     int32 = 13
	KeyDelete    int32 = 46
	KeyHome      int32 = 36
	KeyEnd       int32 = 35
	KeyLeft      int32 = 37
	KeyRight     int32 = 39
)

// Event 事件
type Event struct {
	Type    EventType
	X, Y    int32
	Button  int32
	Delta   int32
	Key     int32
	Char    rune
	Ctrl    bool
	Shift   bool
	Alt     bool
	Handled bool
	Target  Component
}

// EventHandler 事件处理器
type EventHandler func(event *Event)

// EventBus 事件总线
type EventBus struct {
	root     Component
	focused  Component
	handlers map[EventType][]EventHandler
}

func NewEventBus(root Component) *EventBus {
	return &EventBus{
		root:     root,
		handlers: make(map[EventType][]EventHandler),
	}
}

// DispatchEvent 分发事件
func (bus *EventBus) DispatchEvent(event *Event) bool {
	// 调用全局处理器
	if handlers, ok := bus.handlers[event.Type]; ok {
		for _, handler := range handlers {
			handler(event)
			if event.Handled {
				return true
			}
		}
	}

	// 根据事件类型分发
	switch event.Type {
	case EventMouseDown, EventMouseUp, EventMouseMove:
		return bus.dispatchMouseEvent(event)
	case EventKeyDown, EventKeyUp, EventKeyChar:
		return bus.dispatchKeyEvent(event)
	}

	return false
}

// 分发鼠标事件（冒泡）
func (bus *EventBus) dispatchMouseEvent(event *Event) bool {
	// 查找最深的组件
	target := bus.findTarget(bus.root, event.X, event.Y)
	if target == nil {
		return false
	}

	event.Target = target

	// 从目标向上冒泡
	current := target
	for current != nil {
		if current.HandleEvent(*event) {
			event.Handled = true
			return true
		}
		if base, ok := current.(*BaseComponent); ok {
			current = base.Parent
		} else {
			break
		}
	}

	return false
}

// 分发键盘事件（发送到焦点组件）
func (bus *EventBus) dispatchKeyEvent(event *Event) bool {
	if bus.focused == nil {
		return false
	}

	event.Target = bus.focused
	return bus.focused.HandleEvent(*event)
}

// 查找目标组件
func (bus *EventBus) findTarget(comp Component, x, y int32) Component {
	bounds := comp.GetBounds()
	if x < bounds.X || x >= bounds.X+bounds.W ||
		y < bounds.Y || y >= bounds.Y+bounds.H {
		return nil
	}

	// 检查子组件（从后往前，因为后面的在上层）
	if base, ok := comp.(*BaseComponent); ok {
		for i := len(base.Children) - 1; i >= 0; i-- {
			if child := bus.findTarget(base.Children[i], x, y); child != nil {
				return child
			}
		}
	}

	return comp
}

// SetFocus 设置焦点
func (bus *EventBus) SetFocus(comp Component) {
	if bus.focused == comp {
		return
	}

	// 发送失焦事件
	if bus.focused != nil {
		event := &Event{Type: EventBlur, Target: bus.focused}
		bus.focused.HandleEvent(*event)
	}

	bus.focused = comp

	// 发送获焦事件
	if comp != nil {
		event := &Event{Type: EventFocus, Target: comp}
		comp.HandleEvent(*event)
	}
}

// GetFocused 获取焦点组件
func (bus *EventBus) GetFocused() Component {
	return bus.focused
}

// On 注册事件处理器
func (bus *EventBus) On(eventType EventType, handler EventHandler) {
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}
