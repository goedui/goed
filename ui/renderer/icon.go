package renderer

// IconPaint 把图标画进 DIP 矩形。fg 是笔画，bg 是衬底（标题栏上通常等于 Titlebar）。
type IconPaint func(ctx Context, x, y, size float32, fg, bg Color)

const (
	// TitleBarIconSize 是自定义标题栏左侧应用图标的边长，单位 DIP。
	TitleBarIconSize = 16
	// TitleBarIconGap 是图标与标题文字的间距。
	TitleBarIconGap = 8
)
