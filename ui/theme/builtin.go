package theme

// Light / Dark 对齐 VS Code Default Light+ 与 Default Dark+ 的 workbench 色板。
// 色值来自 vscode colorRegistry 默认值 + theme-defaults 的 light_vs / dark_vs。

// Light 对应 Default Light+。
var Light = Theme{
	Name: "light",
	Dark: false,

	// editor.background / editor.foreground（light_vs.json）
	Background: hex("#FFFFFF"),
	Foreground: hex("#000000"),

	// titleBar.activeBackground / activeForeground
	Titlebar:           hex("#DDDDDD"),
	TitlebarForeground: hex("#333333"),
	TitlebarHover:      hex("#CBCBCB"), // toolbar.hoverBackground #B8B8B850 叠在标题栏上
	TitlebarInactive:   hex("#E7E7E7"),
	CloseHover:         hex("#E81123"), // Windows / VS Code 关闭按钮悬停

	// editorGroup.border
	Separator: hex("#E7E7E7"),

	// button.* ：light 上 hover 是 darken(background, 0.2)
	Button:           hex("#007ACC"),
	ButtonForeground: hex("#FFFFFF"),
	ButtonHover:      hex("#0062A3"),
	ButtonPressed:    hex("#00518A"),

	FontFamily: "Segoe UI",
	FontSize:   13, // workbench 默认字号
	Padding:    16,
}

// Dark 对应 Default Dark+。
var Dark = Theme{
	Name: "dark",
	Dark: true,

	// editor.background / editor.foreground（dark_vs.json）
	Background: hex("#1E1E1E"),
	Foreground: hex("#D4D4D4"),

	// titleBar.activeBackground / activeForeground
	Titlebar:           hex("#3C3C3C"),
	TitlebarForeground: hex("#CCCCCC"),
	TitlebarHover:      hex("#4B4C4D"), // toolbar.hoverBackground #5A5D5E50 叠在标题栏上
	TitlebarInactive:   hex("#2D2D2D"),
	CloseHover:         hex("#E81123"),

	// editorGroup.border
	Separator: hex("#444444"),

	// button.* ：dark 上 hover 是 lighten(background, 0.2)
	Button:           hex("#0E639C"),
	ButtonForeground: hex("#FFFFFF"),
	ButtonHover:      hex("#1177BB"),
	ButtonPressed:    hex("#0C5689"),

	FontFamily: "Segoe UI",
	FontSize:   13,
	Padding:    16,
}
