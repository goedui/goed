package runtime

// 虚拟键，数值对齐 Windows VK_*，其它平台按同样编号解释。
const (
	KeyBackspace = 0x08
	KeyTab       = 0x09
	KeyEnter     = 0x0D
	KeyShift     = 0x10
	KeyControl   = 0x11
	KeyEscape    = 0x1B
	KeySpace     = 0x20
	KeyEnd       = 0x23
	KeyHome      = 0x24
	KeyLeft      = 0x25
	KeyUp        = 0x26
	KeyRight     = 0x27
	KeyDown      = 0x28
	KeyPageUp    = 0x21
	KeyPageDown  = 0x22
	KeyDelete    = 0x2E
	KeyA         = 0x41
	KeyC         = 0x43
	KeyF         = 0x46
	KeyL         = 0x4C
	KeyN         = 0x4E
	KeyO         = 0x4F
	KeyP         = 0x50
	KeyS         = 0x53
	KeyT         = 0x54
	KeyV         = 0x56
	KeyW         = 0x57
	KeyX         = 0x58
	KeyY         = 0x59
	KeyZ         = 0x5A
	KeyF1        = 0x70
	KeyF11       = 0x7A
)

// KeyEvent 是一次按键。Char 非 0 表示 WM_CHAR 文本输入。
type KeyEvent struct {
	Key   int
	Char  rune
	Ctrl  bool
	Shift bool
	Alt   bool
	Down  bool
}

// KeyHandler 处理按键。返回 true 表示已消费。
type KeyHandler func(KeyEvent) bool

// DispatchKey 把按键交给树：先 dialog / ctxmenu / popover / 焦点控件的 onKeyDown，再根节点。
func DispatchKey(nodes []*VNode, e KeyEvent) bool {
	if handle := findHandler(nodes, "dialog"); handle != nil {
		if handle(e) {
			return true
		}
	}
	// ctxmenu（右键面板）排在 popover 前面。
	//
	// 面板里的菜单项是 Popover 在画、上下键也归它，但**关闭的语义属于面板**：
	// Esc 要关掉整张菜单，而不是「把内层下拉收起来」。排在前面还顺手解决一个
	// 交叉情况 —— 菜单栏的下拉和右键面板同时开着时，Esc 应该先关右键面板
	// （它是最后打开的那一层），走 popover 那一支会去关菜单栏。
	if handle := findHandler(nodes, "ctxmenu"); handle != nil {
		if handle(e) {
			return true
		}
	}
	if handle := findHandler(nodes, "popover"); handle != nil {
		if handle(e) {
			return true
		}
	}
	if handle := focusedKeyHandler(nodes); handle != nil {
		if handle(e) {
			return true
		}
	}
	var handled bool
	Walk(nodes, func(n *VNode) bool {
		if n == nil || n.Attrs == nil {
			return false
		}
		fn, _ := n.Attrs["onKeyDown"].(KeyHandler)
		if fn == nil {
			if f, ok := n.Attrs["onKeyDown"].(func(KeyEvent) bool); ok {
				fn = f
			}
		}
		if fn != nil && fn(e) {
			handled = true
			return true
		}
		return false
	})
	return handled
}

func findHandler(nodes []*VNode, tag string) KeyHandler {
	var fn KeyHandler
	Walk(nodes, func(n *VNode) bool {
		if n != nil && n.Tag == tag && n.Attrs != nil {
			if h, ok := n.Attrs["onKeyDown"].(KeyHandler); ok {
				fn = h
				return true
			}
			if h, ok := n.Attrs["onKeyDown"].(func(KeyEvent) bool); ok {
				fn = h
				return true
			}
		}
		return false
	})
	return fn
}

func focusedKeyHandler(nodes []*VNode) KeyHandler {
	id := FocusedID()
	if id == "" {
		return nil
	}
	n := FindByID(nodes, id)
	if n == nil || n.Attrs == nil {
		return nil
	}
	if h, ok := n.Attrs["onKeyDown"].(KeyHandler); ok {
		return h
	}
	if h, ok := n.Attrs["onKeyDown"].(func(KeyEvent) bool); ok {
		return h
	}
	return nil
}

// DispatchKeyUp 分发按键抬起。与 DispatchKey 不同：它不先走 popover / 焦点
// 控件，只做全树查找——游戏、快捷键这类「按住状态」场景把 onKeyUp 挂在根上。
// 返回 true 表示已消费。
func DispatchKeyUp(nodes []*VNode, e KeyEvent) bool {
	e.Down = false
	var handled bool
	Walk(nodes, func(n *VNode) bool {
		if n == nil || n.Attrs == nil {
			return false
		}
		fn, _ := n.Attrs["onKeyUp"].(KeyHandler)
		if fn == nil {
			if f, ok := n.Attrs["onKeyUp"].(func(KeyEvent) bool); ok {
				fn = f
			}
		}
		if fn != nil && fn(e) {
			handled = true
			return true
		}
		return false
	})
	return handled
}

// CompositionHandler 处理输入法组合串（预编辑）：text 是组合中的文本，
// cursor 是组合串内的字符偏移。返回 true 表示已消费（组合串由控件自己绘制）。
type CompositionHandler func(text string, cursor int) bool

// TextInputHandler 处理一段「上屏」文本（输入法提交、宿主直接注入等）。
type TextInputHandler func(text string) bool

// DispatchComposition 把输入法组合串交给焦点控件；没人消费时返回 false，
// 宿主可以据此决定要不要交回系统处理。
func DispatchComposition(nodes []*VNode, text string, cursor int) bool {
	id := FocusedID()
	if id == "" {
		return false
	}
	n := FindByID(nodes, id)
	if n == nil || n.Attrs == nil {
		return false
	}
	if h, ok := n.Attrs["onComposition"].(CompositionHandler); ok {
		return h(text, cursor)
	}
	if h, ok := n.Attrs["onComposition"].(func(string, int) bool); ok {
		return h(text, cursor)
	}
	return false
}

// DispatchTextInput 把上屏文本交给焦点控件。
func DispatchTextInput(nodes []*VNode, text string) bool {
	id := FocusedID()
	if id == "" || text == "" {
		return false
	}
	n := FindByID(nodes, id)
	if n == nil || n.Attrs == nil {
		return false
	}
	if h, ok := n.Attrs["onTextInput"].(TextInputHandler); ok {
		return h(text)
	}
	if h, ok := n.Attrs["onTextInput"].(func(string) bool); ok {
		return h(text)
	}
	return false
}
