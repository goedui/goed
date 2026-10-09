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
	// Margin 是四边相同的外边距，参与流式占位。可为负。绝对定位不占父级空间。
	Margin     float32
	Gap        float32
	FontSize   float32
	FontFamily string
	Weight     int
	Align      string
	Justify    string
	// Overflow 同时作用于两轴，对应 CSS overflow。空字符串是 visible。
	// OverflowX / OverflowY 非空时盖过这一轴。visible | hidden | auto | scroll。
	Overflow  string
	OverflowX string
	OverflowY string
	// Position 空字符串是 static。relative 仍占位，再按 Top/Left 平移。
	// absolute 脱离文档流，相对最近的 relative/absolute 祖先（没有则相对内容区）的 padding 边。
	Position string
	// Top/Right/Bottom/Left 为 0 时默认「没写」。要钉在 0 上，把对应位写进 Inset。
	// 绝对定位时对边都写了且没写 Width/Height，该轴被拉满。
	Top, Right, Bottom, Left float32
	Inset                    uint8
	Color                    Color
	Background Color
	Border     Color
	Radius     float32
}

// GradientFill 是线性渐变填充参数：From→To 两端颜色，Vertical 决定方向
// （true 从上到下，false 从左到右）。A==0 的端点由绘制端回退。
type GradientFill struct {
	From, To Color
	Vertical bool
}

// ShadowStyle 是圆角矩形投影：Blur 模糊半径，OffsetY 向下偏移，Color 为阴影色
// （A==0 时绘制端按黑色低透明度画）。与 CSS box-shadow 的 0 offset X 形态对应。
type ShadowStyle struct {
	Blur, OffsetY float32
	Color         Color
}
