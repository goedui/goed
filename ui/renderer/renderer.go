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
	// Italic 走字体的 italic 面（没有则后端合成倾斜）。
	//
	// 它不是「把正体斜过来」那么单纯：字体有 italic 面就换一套字形，步进会跟着
	// 变几个百分点。所以量宽与画字必须用同一个 TextStyle —— 后端按 TextStyle 缓存
	// 文本格式，同一条样式量出来的宽度与画出来的宽度才会一致。
	Italic bool
	Align  Align
	VAlign Align
	NoWrap bool
	// LineSpacing 是强制行距（DIP）。0 表示用字体自己的行距。
	// 编辑器把它收成整数：文字和光标共用这一格，换行不会越积越偏。
	LineSpacing float32
}

// Context 是平台无关的 2D 绘制接口。Windows 后端是 Direct2D + DirectWrite：
// GPU 加速、ClearType 抗锯齿、DIP 坐标与原生裁剪/几何都是现成的，GDI 得自己堆
// 双缓冲和 AlphaBlend，只适合极老的兼容层（本引擎目标是 Windows 10+）。
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

// FontMetrics 是字体在给定字号下的纵向度量，单位 DIP，原点在**行盒顶**。
//
// 为什么必须由后端给：行内文字的落笔位置由基线决定，而基线是取「上伸部」
// （ascent）。这个数只有拿到真实字体度量才知道 —— Arial 的上伸是 0.905em、
// Microsoft YaHei UI 是 1.06em，猜一个常数（比如 0.85em）会让所有文字整体
// 偏上或偏下几个像素，而且不同字体的偏差还不一样。
type FontMetrics struct {
	// Ascent 是基线到行盒顶的距离（DIP，正数）。
	Ascent float32
	// Descent 是基线到行盒底的距离（DIP，正数）。
	Descent float32
	// LineGap 是字体自带的额外行距（DIP，通常为 0）。
	LineGap float32
}

// LineHeight 返回字体推荐的行高（ascent + descent + lineGap）。
func (m FontMetrics) LineHeight() float32 { return m.Ascent + m.Descent + m.LineGap }

// FontMetricsProvider 由能报告真实字体度量的后端实现（Windows 上是 DirectWrite）。
//
// 未实现时排版退回按字号估算基线（见 layout 包的 textMetricsOf），画面仍能出来，
// 但文字纵向会错几个像素。
type FontMetricsProvider interface {
	FontMetrics(style TextStyle) (FontMetrics, bool)
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

// ImagePainter 由能自己画圆角图片的后端实现。
//
// 为什么要单独一个接口、而不是给 Context 加方法：Context 是公开接口，加一个方法
// 就把所有外部实现（以及测试里的假后端）全打断了。圆角图片属于「有更好、没有也能
// 看」的能力，做成可选接口，后端按自己的条件实现，调用方走 DrawImageRounded
// 自动降级。
type ImagePainter interface {
	// DrawImageRounded 把图片解码后画进 (x,y,w,h)，四个角按半径 r 切圆角。
	// r <= 0 时等价于 DrawImage。
	DrawImageRounded(data []byte, x, y, w, h, r float32)
}

// DrawImageRounded 画一张圆角图片：后端实现了 ImagePainter 就走它，
// 否则退回 DrawImage 画直角——不会因为后端不支持就什么都不画。
//
// 圆角是「占位框 → 真图」这段过渡里唯一会跳变的视觉特征：占位框一直是圆角的
// （FillRoundedRect + 描边），图片要是直角，图一到位的瞬间四个角会方一下。
func DrawImageRounded(ctx Context, data []byte, x, y, w, h, r float32) {
	if ctx == nil || len(data) == 0 || w <= 0 || h <= 0 {
		return
	}
	if r > 0 {
		if ip, ok := ctx.(ImagePainter); ok {
			ip.DrawImageRounded(data, x, y, w, h, r)
			return
		}
	}
	ctx.DrawImage(data, x, y, w, h)
}
