package theme

// Dark VSCode Dark+ 风格暗色主题（经典 Dark+ 精确配色）
func Dark() *Theme {
	t := &Theme{
		Name:             "Dark",
		UIFontFamily:     "Segoe UI",
		EditorFontFamily: "Cascadia Code", // VSCode 默认字体，fallback 到 Consolas
		UIFontSize:       15,
		EditorFontSize:   16,
		TitleBarFontSize: 12, // 标题栏字体更小更精致
		TabsFontSize:     14, // 标签页字体大小

		// 窗口 - VSCode 编辑器背景
		WindowBg:     RGB(31, 31, 31), // #1F1F1F
		WindowBorder: RGB(62, 62, 64), // #3E3E40

		// 标题栏 - VSCode Dark+ 标题栏 #3C3C3C
		TitleBarBg:         RGB(31, 31, 31),    // #1F1F1F
		TitleBarFg:         RGB(204, 204, 204), // #CCCCCC
		TitleBarBtnHover:   RGB(74, 74, 74),    // #4A4A4A (悬停微亮，类似 20% 白色叠加)
		TitleBarBtnPress:   RGB(40, 40, 40),    // #282828
		TitleBarCloseHover: RGB(232, 17, 35),   // #E81123 (VSCode 红色)
		TitleBarClosePress: RGB(196, 43, 28),   // #C42B1C (按压深红)

		// 按钮 - VSCode 按钮配色
		ButtonBg:         RGB(14, 99, 156),   // #0E639C
		ButtonFg:         RGB(255, 255, 255), // #FFFFFF
		ButtonHoverBg:    RGB(17, 119, 187),  // #1177BB
		ButtonPressBg:    RGB(9, 71, 113),    // #094771
		ButtonDisabledBg: RGB(45, 45, 45),    // #2D2D2D
		ButtonDisabledFg: RGB(127, 127, 127), // #7F7F7F
		ButtonBorder:     RGB(62, 62, 64),    // #3E3E40

		// 滚动条 - VSCode 滚动条
		ScrollTrackBg:    RGB(31, 31, 31),    // #1F1F1F
		ScrollThumbBg:    RGB(79, 79, 79),    // #4F4F4F
		ScrollThumbHover: RGB(104, 104, 104), // #686868

		// 状态栏 - VSCode Dark+ 经典蓝色状态栏 #007ACC
		StatusBarBg: RGB(0, 122, 204),   // #007ACC
		StatusBarFg: RGB(255, 255, 255), // #FFFFFF

		// 编辑器 - VSCode 编辑器配色
		EditorBg:           RGB(31, 31, 31),    // #1F1F1F
		EditorFg:           RGB(212, 212, 212), // #D4D4D4 (VSCode 默认文本)
		EditorSelectionBg:  RGB(38, 79, 120),   // #264F78 (选中背景)
		EditorCursorColor:  RGB(255, 255, 255), // #FFFFFF
		EditorLineNumberBg: RGB(31, 31, 31),    // #1F1F1F
		EditorLineNumberFg: RGB(133, 133, 133), // #858585 (行号灰色)

		// 文本
		TextPrimary:   RGB(212, 212, 212), // #D4D4D4
		TextSecondary: RGB(180, 180, 180), // #B4B4B4
		TextDisabled:  RGB(108, 108, 108), // #6C6C6C

		// 语法高亮 - VSCode Dark+ 配色方案
		SyntaxKeyword:  RGB(86, 156, 214),  // #569CD6 (关键字蓝)
		SyntaxString:   RGB(206, 145, 120), // #CE9178 (字符串橙)
		SyntaxComment:  RGB(106, 153, 85),  // #6A9955 (注释绿)
		SyntaxNumber:   RGB(181, 206, 168), // #B5CEA8 (数字浅绿)
		SyntaxFunction: RGB(220, 220, 170), // #DCDCAA (函数黄)
		SyntaxType:     RGB(78, 201, 176),  // #4EC9B0 (类型青)

		// 标题工具栏 - 与标题栏一致 #3C3C3C
		TitleToolbarBg:     RGB(31, 31, 31),    // #1F1F1F
		TitleToolbarFg:     RGB(204, 204, 204), // #CCCCCC
		TitleToolbarHover:  RGB(74, 74, 74),    // #4A4A4A
		TitleToolbarActive: RGB(40, 40, 40),    // #282828

		// 菜单 - VSCode Dark+ 菜单配色
		MenuBarBg:       RGB(31, 31, 31),    // #1F1F1F
		MenuBarHoverBg:  RGB(74, 74, 74),    // #4A4A4A (悬停微亮)
		MenuFg:          RGB(204, 204, 204), // #CCCCCC
		MenuDropdownBg:  RGB(37, 37, 38),    // #252526 (下拉/右键菜单背景)
		MenuBorder:      RGB(69, 69, 69),    // #454545 (菜单边框)
		MenuSeparator:   RGB(69, 69, 69),    // #454545 (分隔线)
		MenuItemHoverBg: RGB(9, 71, 113),    // #094771 (悬停高亮蓝)
		MenuDisabledFg:  RGB(108, 108, 108), // #6C6C6C (禁用)
		MenuShortcutFg:  RGB(148, 148, 148), // #949494 (快捷键提示)

		// 对话框
		DialogOverlayBg: RGB(18, 18, 18), // #121212 (模态遮罩)
		DialogPanelBg:   RGB(37, 37, 38), // #252526 (面板背景)

		// 搜索框 - VSCode 输入框配色
		SearchBg:     RGB(60, 60, 60),    // #3C3C3C
		SearchBorder: RGB(62, 62, 64),    // #3E3E40
		SearchFg:     RGB(204, 204, 204), // #CCCCCC

		// 编辑器扩展
		EditorCurrentLineBg: RGB(40, 40, 40), // #282828 (当前行高亮)
		EditorGutterBg:      RGB(31, 31, 31), // #1F1F1F
		TabStatusModified:   RGB(224, 175, 104),
		TabStatusUntracked:  RGB(115, 190, 139),
		TabStatusIgnored:    RGB(127, 127, 127),
		TabStatusReadOnly:   RGB(220, 170, 92),
		TabStatusConflict:   RGB(232, 101, 101),
	}
	t.MenuBarBg = RGB(31, 31, 31)
	t.EditorGutterBg = RGB(31, 31, 31)
	return t
}
