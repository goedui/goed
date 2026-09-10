package theme

// Light 明亮主题
func Light() *Theme {
	return &Theme{
		Name:             "Light",
		UIFontFamily:     "Segoe UI",
		EditorFontFamily: "Cascadia Code", // 与 Dark/Tabby 统一（字体规范.md 第 6 节）
		UIFontSize:       15,
		EditorFontSize:   16,
		TitleBarFontSize: 12, // 标题栏字体更小更精致
		TabsFontSize:     14, // 标签页字体大小

		// 窗口
		WindowBg:     RGB(255, 255, 255),
		WindowBorder: RGB(200, 200, 200),

		// 标题栏
		TitleBarBg:         RGB(240, 240, 240),
		TitleBarFg:         RGB(0, 0, 0),
		TitleBarBtnHover:   RGB(220, 220, 220),
		TitleBarBtnPress:   RGB(200, 200, 200),
		TitleBarCloseHover: RGB(232, 17, 35),
		TitleBarClosePress: RGB(196, 13, 29),

		// 按钮
		ButtonBg:         RGB(0, 120, 212),
		ButtonFg:         RGB(255, 255, 255),
		ButtonHoverBg:    RGB(0, 136, 240),
		ButtonPressBg:    RGB(0, 104, 184),
		ButtonDisabledBg: RGB(220, 220, 220),
		ButtonDisabledFg: RGB(160, 160, 160),
		ButtonBorder:     RGB(180, 180, 180),

		// 滚动条
		ScrollTrackBg:    RGB(240, 240, 240),
		ScrollThumbBg:    RGB(160, 160, 160),
		ScrollThumbHover: RGB(120, 120, 120),

		// 状态栏
		StatusBarBg: RGB(0, 122, 204),
		StatusBarFg: RGB(255, 255, 255),

		// 编辑器
		EditorBg:           RGB(255, 255, 255),
		EditorFg:           RGB(0, 0, 0),
		EditorSelectionBg:  RGB(173, 214, 255),
		EditorCursorColor:  RGB(0, 0, 0),
		EditorLineNumberBg: RGB(245, 245, 245),
		EditorLineNumberFg: RGB(120, 120, 120),

		// 文本
		TextPrimary:   RGB(0, 0, 0),
		TextSecondary: RGB(80, 80, 80),
		TextDisabled:  RGB(160, 160, 160),

		// 语法高亮（Markdown）
		SyntaxKeyword:  RGB(0, 0, 255),   // Blue
		SyntaxString:   RGB(163, 21, 21), // Red
		SyntaxComment:  RGB(0, 128, 0),   // Green
		SyntaxNumber:   RGB(9, 134, 88),  // Dark Green
		SyntaxFunction: RGB(121, 94, 38), // Brown
		SyntaxType:     RGB(0, 128, 128), // Teal

		TitleToolbarBg:     RGB(240, 240, 240),
		TitleToolbarFg:     RGB(32, 32, 32),
		TitleToolbarHover:  RGB(220, 220, 220),
		TitleToolbarActive: RGB(200, 200, 200),

		// 菜单（VSCode Light 风格）
		MenuBarBg:       RGB(243, 243, 243), // #F3F3F3
		MenuBarHoverBg:  RGB(224, 224, 224), // #E0E0E0
		MenuFg:          RGB(51, 51, 51),    // #333333
		MenuDropdownBg:  RGB(255, 255, 255), // #FFFFFF
		MenuBorder:      RGB(200, 200, 200), // #C8C8C8
		MenuSeparator:   RGB(200, 200, 200), // #C8C8C8
		MenuItemHoverBg: RGB(0, 96, 192),    // #0060C0 (悬停高亮蓝)
		MenuDisabledFg:  RGB(160, 160, 160), // #A0A0A0
		MenuShortcutFg:  RGB(120, 120, 120), // #787878

		// 对话框
		DialogOverlayBg: RGB(179, 179, 179), // 30% 黑色遮罩
		DialogPanelBg:   RGB(255, 255, 255), // #FFFFFF

		SearchBg:           RGB(255, 255, 255),
		SearchBorder:       RGB(200, 200, 200),
		SearchFg:           RGB(100, 100, 100),

		// 编辑器扩展（缺失时零值为黑色，会把侧栏/当前行染黑）
		EditorCurrentLineBg: RGB(232, 240, 251), // #E8F0FB (当前行高亮，VSCode Light)
		EditorGutterBg:      RGB(243, 243, 243), // #F3F3F3 (资源管理器/行号区背景)
		TabStatusModified:  RGB(194, 126, 28),
		TabStatusUntracked: RGB(46, 138, 78),
		TabStatusIgnored:   RGB(130, 130, 130),
		TabStatusReadOnly:  RGB(176, 116, 20),
		TabStatusConflict:  RGB(198, 55, 55),
	}
}
