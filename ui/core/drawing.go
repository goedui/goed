package core

// DrawingContext 绘图上下文接口
type DrawingContext interface {
	// 基础形状
	DrawRect(x, y, w, h int32, color uint32, filled bool)
	DrawLine(x1, y1, x2, y2 int32, color uint32, width int32)
	DrawRoundedRect(x, y, w, h, radius int32, color uint32, filled bool)
	DrawEllipse(x, y, w, h int32, color uint32, filled bool)
	DrawPolygon(points []Point, color uint32, filled bool)

	// 文本
	DrawText(x, y int32, text string, color uint32)
	MeasureText(text string) (width, height int32)

	// 状态
	SetFont(name string, size int32)
	SetBkMode(transparent bool)

	// 双缓冲
	BeginDraw()
	EndDraw()

	// 裁剪
	SetClipRect(x, y, w, h int32)
	ResetClip()
}

// AlphaDrawingContext 是支持半透明矩形填充的绘制上下文扩展能力。
// 平台实现通过接口探测参与：不支持的平台（或旧实现）自然回退到
// 不透明填充，调用方无需分支。
type AlphaDrawingContext interface {
	// DrawRectAlpha 以 0..255 的不透明度向 (x,y,w,h) 混合 color。
	// color 为 COLORREF 布局（R | G<<8 | B<<16）。
	DrawRectAlpha(x, y, w, h int32, color uint32, alpha byte)
}

// ImageDrawingContext 是支持位图绘制的上下文扩展能力（图片预览等）。
// 平台通过接口探测参与；不支持的平台由组件回退为文字提示。
type ImageDrawingContext interface {
	// DrawImageBGRA 绘制一张不透明 32bpp 位图。pix 为行优先像素，
	// 每像素 4 字节，字节序 B、G、R、保留（与 Win32 DIB 布局一致），
	// 长度必须为 pixW*pixH*4。目标矩形 (x,y,w,h) 支持缩放。
	DrawImageBGRA(pix []byte, pixW, pixH int32, x, y, w, h int32)
}

// Color 颜色
type Color struct {
	R, G, B byte
}

// RGB 创建颜色值
func RGB(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

// Rect 矩形
type Rect struct {
	X, Y, W, H int32
}

// Point 点
type Point struct {
	X, Y int32
}
