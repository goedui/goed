package theme

// Theme 主题定义
type Theme struct {
	Name string

	// Typography
	UIFontFamily     string
	EditorFontFamily string
	UIFontSize       int32
	EditorFontSize   int32
	TitleBarFontSize int32 // 标题栏/菜单栏字体大小
	TabsFontSize     int32 // 标签页字体大小

	// 窗口
	WindowBg     uint32
	WindowBorder uint32

	// 标题栏
	TitleBarBg         uint32
	TitleBarFg         uint32
	TitleBarBtnHover   uint32
	TitleBarBtnPress   uint32
	TitleBarCloseHover uint32
	TitleBarClosePress uint32

	// 按钮
	ButtonBg         uint32
	ButtonFg         uint32
	ButtonHoverBg    uint32
	ButtonPressBg    uint32
	ButtonDisabledBg uint32
	ButtonDisabledFg uint32
	ButtonBorder     uint32

	// 滚动条
	ScrollTrackBg    uint32
	ScrollThumbBg    uint32
	ScrollThumbHover uint32

	// 状态栏
	StatusBarBg uint32
	StatusBarFg uint32

	// 编辑器
	EditorBg           uint32
	EditorFg           uint32
	EditorSelectionBg  uint32
	EditorCursorColor  uint32
	EditorLineNumberBg uint32
	EditorLineNumberFg uint32

	// 文本
	TextPrimary   uint32
	TextSecondary uint32
	TextDisabled  uint32

	// 语法高亮
	SyntaxKeyword  uint32
	SyntaxString   uint32
	SyntaxComment  uint32
	SyntaxNumber   uint32
	SyntaxFunction uint32
	SyntaxType     uint32

	// 标题工具栏
	TitleToolbarBg     uint32
	TitleToolbarFg     uint32
	TitleToolbarHover  uint32
	TitleToolbarActive uint32

	// 菜单（菜单栏 + 下拉/右键菜单，VSCode Dark+ 配色）
	MenuBarBg       uint32 // 菜单栏/标题栏背景
	MenuBarHoverBg  uint32 // 顶级菜单悬停
	MenuFg          uint32 // 菜单文字
	MenuDropdownBg  uint32 // 下拉/右键菜单背景
	MenuBorder      uint32 // 菜单边框
	MenuSeparator   uint32 // 分隔线
	MenuItemHoverBg uint32 // 菜单项悬停高亮
	MenuDisabledFg  uint32 // 禁用项文字
	MenuShortcutFg  uint32 // 快捷键提示文字

	// 对话框
	DialogOverlayBg uint32 // 模态遮罩
	DialogPanelBg   uint32 // 对话框面板

	// Search field
	SearchBg     uint32
	SearchBorder uint32
	SearchFg     uint32

	// 编辑器扩展
	EditorCurrentLineBg uint32
	EditorGutterBg      uint32
	TabStatusModified   uint32
	TabStatusUntracked  uint32
	TabStatusIgnored    uint32
	TabStatusReadOnly   uint32
	TabStatusConflict   uint32
}

// Current 当前主题
var Current *Theme

// themeRegistry 按名字索引的内置主题。新增主题时在 available 中追加即可，
// 标题栏切换按钮会按 available 的顺序循环。
var available = []struct {
	name string
	make func() *Theme
}{
	{"Dark", Dark},
	{"Light", Light},
	{"Tabby", Tabby},
}

// Names 返回内置主题名列表（切换 UI 可直接展示）。
func Names() []string {
	names := make([]string, 0, len(available))
	for _, a := range available {
		names = append(names, a.name)
	}
	return names
}

// ByName 返回指定名字的内置主题；未知名字返回 nil。
func ByName(name string) *Theme {
	for _, a := range available {
		if a.name == name {
			return a.make()
		}
	}
	return nil
}

// Next cycles to the next built-in theme after the current one and applies it.
// Returns the newly active theme. Unknown/absent Current falls back to the
// first entry.
func Next() *Theme {
	current := ""
	if Current != nil {
		current = Current.Name
	}
	for i, a := range available {
		if a.name == current {
			next := available[(i+1)%len(available)]
			Current = next.make()
			return Current
		}
	}
	Current = available[0].make()
	return Current
}

// SetTheme 设置主题
func SetTheme(t *Theme) {
	Current = t
}

// GetTheme 获取当前主题
func GetTheme() *Theme {
	if Current == nil {
		Current = Dark() // 默认暗色主题
	}
	return Current
}

// RGB 创建颜色
func RGB(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

// === 字体阶梯 helper（字体规范.md 第 4 节，唯一取值入口） ===

// FontDisplay 对话框标题档：UIFontSize + 2。
func (t *Theme) FontDisplay() int32 { return t.UIFontSize + 2 }

// FontBody 正文档：UIFontSize（菜单、按钮、文件树、查找面板）。
func (t *Theme) FontBody() int32 { return t.UIFontSize }

// FontSecondary 辅助档：UIFontSize - 1（标签页、状态栏、文件树条目）。
func (t *Theme) FontSecondary() int32 { return t.UIFontSize - 1 }

// FontCaption chrome 档：标题栏/菜单栏文字（TitleBarFontSize，回退 Body）。
func (t *Theme) FontCaption() int32 {
	if t.TitleBarFontSize > 0 {
		return t.TitleBarFontSize
	}
	return t.UIFontSize
}

// FontTabs 标签页档：TabsFontSize（回退 Secondary）。
func (t *Theme) FontTabs() int32 {
	if t.TabsFontSize > 0 {
		return t.TabsFontSize
	}
	return t.UIFontSize - 1
}
