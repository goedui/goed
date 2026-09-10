package components

// MenuItem / Menu 基础类型。MenuBar 与 EditorTitleBar 的实现已迁移到
// editor/components（编辑器专属、随编辑器会话演进）；本包仍保留菜单
// 数据类型，供 ContextMenu、FileTree、Tabs、Editor 等通用组件的
// 右键菜单/条目回调继续使用。

// MenuItem 菜单项
type MenuItem struct {
	Label     string
	Shortcut  string // 快捷键提示，如 "Ctrl+S"
	Icon      string // 可选图标（暂不实现）
	Enabled   bool
	Separator bool // 是否为分隔线
	OnClick   func()
	SubItems  []MenuItem // 子菜单（可选，用于多级菜单）
}

// Menu 顶级菜单
type Menu struct {
	Label   string
	Items   []MenuItem
	OnClick func() // Optional direct action for top-level command buttons.
}
