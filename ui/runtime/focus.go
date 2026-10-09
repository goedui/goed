package runtime

import "sync"

// 当前焦点控件 id。输入框点击后写入，Esc / 点击空白可清空。
var (
	focusedID string
	blurMu    sync.Mutex
	blurHooks = map[string]func(){}
)

// FocusedID 返回当前焦点。
func FocusedID() string { return focusedID }

// SetBlurHook 登记控件失焦回调。id 为空或 fn 为 nil 时清除。
func SetBlurHook(id string, fn func()) {
	if id == "" {
		return
	}
	blurMu.Lock()
	if fn == nil {
		delete(blurHooks, id)
	} else {
		blurHooks[id] = fn
	}
	blurMu.Unlock()
}

// SetFocus 设置焦点。离开上一个控件时触发它的失焦回调。
func SetFocus(id string) {
	prev := focusedID
	if prev == id {
		return
	}
	focusedID = id
	if prev == "" {
		return
	}
	blurMu.Lock()
	fn := blurHooks[prev]
	blurMu.Unlock()
	if fn != nil {
		fn()
	}
}

// IsFocused 报告节点是否拥有焦点。
func IsFocused(n *VNode) bool {
	if n == nil {
		return false
	}
	id := n.ID()
	return id != "" && id == focusedID
}
