package theme

import "testing"

// webHex 把 #RRGGBB 形式的十六进制颜色转成 COLORREF 布局（R | G<<8 | B<<16）
func webHex(hex uint32) uint32 {
	return ((hex & 0xFF) << 16) | (hex & 0x00FF00) | ((hex >> 16) & 0xFF)
}

// TestDarkThemeMatchesVSCodeDarkPlus 校验暗色主题的关键色值精确对应
// VSCode Dark+ 经典配色，防止后续改动悄悄偏离主题。
func TestDarkThemeMatchesVSCodeDarkPlus(t *testing.T) {
	th := Dark()

	cases := []struct {
		name string
		got  uint32
		hex  uint32 // #RRGGBB
	}{
		// 编辑器
		{"编辑器背景", th.EditorBg, 0x1F1F1F},
		{"编辑器文本", th.EditorFg, 0xD4D4D4},
		{"选区背景", th.EditorSelectionBg, 0x264F78},
		{"当前行高亮", th.EditorCurrentLineBg, 0x282828},
		{"行号前景", th.EditorLineNumberFg, 0x858585},

		// 标题栏 / 菜单栏
		{"标题栏背景", th.TitleBarBg, 0x1F1F1F},
		{"标题栏文字", th.TitleBarFg, 0xCCCCCC},
		{"菜单栏背景", th.MenuBarBg, 0x1F1F1F},
		{"菜单栏悬停", th.MenuBarHoverBg, 0x4A4A4A},

		// 下拉/右键菜单
		{"菜单下拉背景", th.MenuDropdownBg, 0x252526},
		{"菜单边框", th.MenuBorder, 0x454545},
		{"菜单项悬停", th.MenuItemHoverBg, 0x094771},
		{"菜单禁用文字", th.MenuDisabledFg, 0x6C6C6C},

		// 状态栏（Dark+ 经典蓝）
		{"状态栏背景", th.StatusBarBg, 0x007ACC},
		{"状态栏文字", th.StatusBarFg, 0xFFFFFF},

		// 按钮
		{"按钮背景", th.ButtonBg, 0x0E639C},
		{"按钮悬停", th.ButtonHoverBg, 0x1177BB},

		// 语法高亮
		{"关键字", th.SyntaxKeyword, 0x569CD6},
		{"字符串", th.SyntaxString, 0xCE9178},
		{"注释", th.SyntaxComment, 0x6A9955},
		{"数字", th.SyntaxNumber, 0xB5CEA8},
		{"函数", th.SyntaxFunction, 0xDCDCAA},
		{"类型", th.SyntaxType, 0x4EC9B0},

		// 对话框
		{"对话框面板", th.DialogPanelBg, 0x252526},
	}

	for _, c := range cases {
		if want := webHex(c.hex); c.got != want {
			t.Errorf("%s = 0x%06X, want 0x%06X (VSCode Dark+ #%06X)", c.name, c.got, want, c.hex)
		}
	}
}

// TestThemesDefineCompleteMenuPalette 确保所有主题都填了完整的菜单/对话框字段，
// 避免新增主题时漏字段导致渲染出黑色块。
func TestThemesDefineCompleteMenuPalette(t *testing.T) {
	for _, th := range []*Theme{Dark(), Light()} {
		fields := []struct {
			name  string
			value uint32
		}{
			{"MenuBarBg", th.MenuBarBg},
			{"MenuBarHoverBg", th.MenuBarHoverBg},
			{"MenuFg", th.MenuFg},
			{"MenuDropdownBg", th.MenuDropdownBg},
			{"MenuBorder", th.MenuBorder},
			{"MenuSeparator", th.MenuSeparator},
			{"MenuItemHoverBg", th.MenuItemHoverBg},
			{"MenuDisabledFg", th.MenuDisabledFg},
			{"MenuShortcutFg", th.MenuShortcutFg},
			{"DialogOverlayBg", th.DialogOverlayBg},
			{"DialogPanelBg", th.DialogPanelBg},
		}
		for _, f := range fields {
			if f.value == 0 {
				t.Errorf("theme %s: field %s is unset (would render as black)", th.Name, f.name)
			}
		}
	}
}
