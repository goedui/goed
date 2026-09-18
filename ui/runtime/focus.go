package runtime

// 当前焦点控件 id。输入框点击后写入，Esc / 点击空白可清空。
var focusedID string

// FocusedID 返回当前焦点。
func FocusedID() string { return focusedID }

// SetFocus 设置焦点。id 为空表示取消。
func SetFocus(id string) { focusedID = id }

// IsFocused 报告节点是否拥有焦点。
func IsFocused(n *VNode) bool {
	if n == nil {
		return false
	}
	id := n.ID()
	return id != "" && id == focusedID
}
