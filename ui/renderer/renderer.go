package renderer

// Color 使用 0..1 的预乘友好线性通道。Direct2D 的 D2D1_COLOR_F 与此一致。
type Color struct {
	R, G, B, A float32
}

// RGB 从 8-bit sRGB 构造不透明颜色。
func RGB(r, g, b uint8) Color {
	return Color{
		R: float32(r) / 255,
		G: float32(g) / 255,
		B: float32(b) / 255,
		A: 1,
	}
}

// RGBA 从 8-bit sRGB 构造带透明度的颜色。
func RGBA(r, g, b, a uint8) Color {
	return Color{
		R: float32(r) / 255,
		G: float32(g) / 255,
		B: float32(b) / 255,
		A: float32(a) / 255,
	}
}

// Align 是文本在布局矩形内的对齐方式。
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
)

// TextStyle 描述一段文本的绘制属性。尺寸单位是 DIP（1/96 inch）。
type TextStyle struct {
	FontFamily string
	FontSize   float32
	Color      Color
	Weight     int
	Align      Align
	VAlign     Align
	NoWrap     bool
}

// Context 是平台无关的 2D 绘制接口。Windows 后端用 Direct2D + DirectWrite 实现。
//
// 选择 Direct2D 而不是 GDI 的原因：
//   - GPU 加速，后续动画、半透明、复杂场景不会被 CPU blit 卡住
//   - DirectWrite 的字形、ClearType/灰度抗锯齿、度量远好于 GDI TextOut
//   - 坐标是 DIP，高 DPI / 每监视器感知是一等公民
//   - Fill、裁剪、变换、几何体都是原生能力，GDI 需要自己堆双缓冲和 AlphaBlend
//
// GDI 只适合极老的兼容层。本引擎的目标平台是 Windows 10+，没有这条包袱。
type Context interface {
	// Width / Height 返回绘制表面尺寸，单位 DIP。
	Width() float32
	Height() float32
	DPI() float32

	Clear(c Color)
	FillRect(x, y, w, h float32, c Color)
	FillRoundedRect(x, y, w, h, r float32, c Color)
	DrawRect(x, y, w, h float32, c Color, width float32)
	DrawLine(x1, y1, x2, y2 float32, c Color, width float32)
	DrawText(text string, x, y, maxW, maxH float32, style TextStyle)
	MeasureText(text string, maxW float32, style TextStyle) (w, h float32)
	DrawImage(data []byte, x, y, w, h float32)
	// DrawPixels 把一块原始 NRGBA 像素近邻缩放到 (x,y,w,h)。pix 长度须为
	// pw*ph*4，按 R,G,B,A 字节序，颜色不做预乘。与 DrawImage 不同，它不解码、
	// 不进位图缓存——专为每帧都变的像素缓冲（游戏画面、实时可视化）设计。
	DrawPixels(pix []byte, pw, ph int, x, y, w, h float32)
	PushClip(x, y, w, h float32)
	PopClip()
}

// TextPoint 是一段文本在布局里的一个插入点：UTF-8 字节偏移、可视行、水平位置。
// 选区、光标、点击落点都用它，必须和 DrawText 的折行一致。
type TextPoint struct {
	Off  int
	Line int
	X    float32
}

// TextLayout 是 DrawText 使用的同一套折行结果。
type TextLayout struct {
	LineH     float32
	LineCount int
	Points    []TextPoint
}

// TextLayouter 由能报告「和 DrawText 相同折行」的后端实现（Windows 上是 DirectWrite）。
// 未实现时编辑框退回按字符宽度折行。
type TextLayouter interface {
	LayoutText(text string, maxW float32, style TextStyle) TextLayout
}
