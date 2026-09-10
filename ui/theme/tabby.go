package theme

// Tabby returns the graphite/cyan palette used by the terminal surface. It
// keeps the same Theme contract as Dark, while separating terminal chrome
// from the editor's VS Code Dark+ colors.
func Tabby() *Theme {
	return &Theme{
		Name:             "Tabby",
		UIFontFamily:     "Segoe UI",
		EditorFontFamily: "Cascadia Code",
		UIFontSize:       15, // 与 Dark/Light 统一：主题切换只换颜色不换排版
		EditorFontSize:   16,
		TitleBarFontSize: 12, // 三主题统一（字体规范.md 第 6 节）
		TabsFontSize:     14,
		WindowBg:         RGB(24, 26, 29),
		WindowBorder:     RGB(52, 57, 62),
		TitleBarBg:       RGB(27, 29, 32),
		TitleBarFg:       RGB(224, 228, 232),
		TitleBarBtnHover: RGB(51, 56, 62), TitleBarBtnPress: RGB(38, 42, 47),
		TitleBarCloseHover: RGB(196, 64, 72), TitleBarClosePress: RGB(151, 45, 53),
		ButtonBg: RGB(76, 157, 178), ButtonFg: RGB(247, 250, 252),
		ButtonHoverBg: RGB(96, 181, 201), ButtonPressBg: RGB(54, 119, 138),
		ButtonDisabledBg: RGB(43, 47, 52), ButtonDisabledFg: RGB(117, 124, 131), ButtonBorder: RGB(64, 71, 78),
		ScrollTrackBg: RGB(18, 20, 22), ScrollThumbBg: RGB(64, 71, 78), ScrollThumbHover: RGB(91, 102, 111),
		StatusBarBg: RGB(32, 36, 40), StatusBarFg: RGB(183, 192, 200),
		EditorBg: RGB(18, 20, 22), EditorFg: RGB(216, 222, 228), EditorSelectionBg: RGB(48, 76, 88),
		EditorCursorColor: RGB(139, 224, 245), EditorLineNumberBg: RGB(18, 20, 22), EditorLineNumberFg: RGB(112, 121, 129),
		TextPrimary: RGB(216, 222, 228), TextSecondary: RGB(158, 168, 177), TextDisabled: RGB(92, 100, 108),
		SyntaxKeyword: RGB(130, 190, 230), SyntaxString: RGB(193, 215, 157), SyntaxComment: RGB(115, 143, 119),
		SyntaxNumber: RGB(216, 180, 127), SyntaxFunction: RGB(139, 224, 245), SyntaxType: RGB(194, 160, 255),
		TitleToolbarBg: RGB(27, 29, 32), TitleToolbarFg: RGB(224, 228, 232), TitleToolbarHover: RGB(51, 56, 62), TitleToolbarActive: RGB(38, 42, 47),
		MenuBarBg: RGB(27, 29, 32), MenuBarHoverBg: RGB(51, 56, 62), MenuFg: RGB(224, 228, 232),
		MenuDropdownBg: RGB(37, 41, 46), MenuBorder: RGB(64, 71, 78), MenuSeparator: RGB(58, 64, 70),
		MenuItemHoverBg: RGB(48, 76, 88), MenuDisabledFg: RGB(111, 120, 128), MenuShortcutFg: RGB(143, 153, 162),
		DialogOverlayBg: RGB(14, 16, 18), DialogPanelBg: RGB(37, 41, 46),
		SearchBg: RGB(37, 41, 46), SearchBorder: RGB(64, 71, 78), SearchFg: RGB(224, 228, 232),
		EditorCurrentLineBg: RGB(30, 36, 40), EditorGutterBg: RGB(18, 20, 22),
		TabStatusModified: RGB(242, 190, 92), TabStatusUntracked: RGB(126, 206, 151),
		TabStatusIgnored: RGB(118, 127, 136), TabStatusReadOnly: RGB(239, 183, 82), TabStatusConflict: RGB(235, 104, 112),
	}
}
