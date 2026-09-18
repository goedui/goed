package runtime

// Color 是行内 style 的颜色。通道 0..1，和 Direct2D 一致。
type Color struct {
	R, G, B, A float32
}

// RGB 从不透明 sRGB 构造颜色。
func RGB(r, g, b uint8) Color {
	return Color{R: float32(r) / 255, G: float32(g) / 255, B: float32(b) / 255, A: 1}
}

// RGBA 从带透明度的 sRGB 构造颜色。
func RGBA(r, g, b, a uint8) Color {
	return Color{R: float32(r) / 255, G: float32(g) / 255, B: float32(b) / 255, A: float32(a) / 255}
}

// Style 对应 Vue / HTML 的行内 style。零值表示未设置，走主题默认。
// 每个 VNode 都可以带一份，不限于某个控件。
type Style struct {
	Flex       float32
	Width      float32
	Height     float32
	MinWidth   float32
	MinHeight  float32
	MaxWidth   float32
	MaxHeight  float32
	Padding    float32
	Gap        float32
	FontSize   float32
	FontFamily string
	Weight     int
	Align      string
	Justify    string
	Color      Color
	Background Color
	Border     Color
	Radius     float32
}
