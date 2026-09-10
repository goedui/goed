package windows

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
	"syscall"
	"unsafe"
)

type penKey struct {
	style, width int32
	color        uint32
}

type fontKey struct {
	name    string
	size    int32
	quality uint32
}

type GDIContext struct {
	hdc         syscall.Handle
	memDC       syscall.Handle
	memBitmap   syscall.Handle
	oldBitmap   syscall.Handle
	oldFont     syscall.Handle
	width       int32
	height      int32
	clearColor  uint32
	fontQuality uint32

	// GDI 对象缓存：DrawRect/DrawLine/SetFont 在 60fps 的游戏场景下每帧
	// 被调用数百次，反复 Create/Delete 会造成显著开销，按颜色/字体复用。
	brushes map[uint32]syscall.Handle
	pens    map[penKey]syscall.Handle
	fonts   map[fontKey]syscall.Handle

	// 裁剪状态：SetClipRect 设定基准裁剪区，PushClipRect/PopClip 维护嵌套层级，
	// 这样 Canvas 之类的子组件不会破坏外层已经设置的裁剪区。
	clipBase    RECT
	clipBaseSet bool
	baseRgn     syscall.Handle
	clipStack   []clipEntry
}

type clipEntry struct {
	rect RECT
	rgn  syscall.Handle
}

func NewGDIContext(hdc syscall.Handle, width, height int32) *GDIContext {
	ctx := &GDIContext{
		hdc:         hdc,
		width:       width,
		height:      height,
		clearColor:  core.RGB(31, 31, 31),
		fontQuality: fontAntialiased,
		brushes:     make(map[uint32]syscall.Handle),
		pens:        make(map[penKey]syscall.Handle),
		fonts:       make(map[fontKey]syscall.Handle),
	}

	// Create memory DC for double buffering
	ctx.memDC = CreateCompatibleDC(hdc)
	ctx.Resize(hdc, width, height)

	// Default font - 使用主题配置
	th := theme.GetTheme()
	ctx.SetFont(th.EditorFontFamily, th.EditorFontSize)

	return ctx
}

func (g *GDIContext) BeginDraw() {
	if g.memDC == 0 || g.width <= 0 || g.height <= 0 {
		return
	}
	// Clear background
	var rect RECT
	rect.Left = 0
	rect.Top = 0
	rect.Right = g.width
	rect.Bottom = g.height
	brush := CreateSolidBrush(g.clearColor)
	FillRect(g.memDC, &rect, brush)
	DeleteObject(brush) // 释放刷子资源
}

func (g *GDIContext) SetBackgroundColor(color uint32) {
	g.clearColor = color
}

func (g *GDIContext) EndDraw() {
	if g.hdc == 0 || g.memDC == 0 || g.width <= 0 || g.height <= 0 {
		return
	}
	// Copy to screen
	BitBlt(g.hdc, 0, 0, g.width, g.height, g.memDC, 0, 0, SRCCOPY)
}

func (g *GDIContext) DrawRect(x, y, w, h int32, color uint32, filled bool) {
	if g.memDC == 0 {
		return
	}
	brush := g.brushFor(color)
	rect := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	if filled {
		FillRect(g.memDC, &rect, brush)
	} else {
		FrameRect(g.memDC, &rect, brush)
	}
}

// DrawRectAlpha 以指定不透明度向目标区域混合纯色。实现：构造一张
// 32bpp PREMULTIPLIED-ALPHA 的 DIB 快照（整片源色），经 msimg32!AlphaBlend
// 与 memDC 现有内容做 per-pixel 混合。GDI 的 FillRect 不支持 alpha，
// 这是当前架构下成本最低的真混合路径。
func (g *GDIContext) DrawRectAlpha(x, y, w, h int32, color uint32, alpha byte) {
	if g.memDC == 0 || w <= 0 || h <= 0 {
		return
	}
	// 裁剪到上下文边界，避免越界 DIB。
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > g.width {
		w = g.width - x
	}
	if y+h > g.height {
		h = g.height - y
	}
	if w <= 0 || h <= 0 {
		return
	}

	r := byte(color & 0xFF)
	gr := byte((color >> 8) & 0xFF)
	b := byte((color >> 16) & 0xFF)
	a := uint32(alpha)

	// 预乘 alpha 的 BGRA 源像素。
	srcColor := uint32(b)*a/255 | (uint32(gr)*a/255)<<8 | (uint32(r)*a/255)<<16 | a<<24

	var bi bitmapInfoHeader
	bi.Size = uint32(unsafe.Sizeof(bi))
	bi.Width = w
	bi.Height = -h // top-down
	bi.Planes = 1
	bi.BitCount = 32
	bi.Compression = 0 // BI_RGB

	var bits *byte
	memBitmap := CreateDIBSection(g.memDC, &bi, 0, &bits, 0, 0)
	if memBitmap == 0 || bits == nil {
		if memBitmap != 0 {
			procDeleteObject.Call(uintptr(memBitmap))
		}
		return
	}
	pixels := unsafe.Slice((*uint32)(unsafe.Pointer(bits)), w*h)
	for i := range pixels {
		pixels[i] = srcColor
	}

	srcDC := CreateCompatibleDC(g.memDC)
	oldSrc := SelectObject(srcDC, memBitmap)

	// AlphaBlend 使用 SOURCEOVER 混合：out = src + dst*(1-alpha)。
	blend := blendFunction{
		SourceConstantAlpha: alpha,
		AlphaFormat:         acSrcOver,
	}
	procAlphaBlend.Call(
		uintptr(g.memDC), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(srcDC), 0, 0, uintptr(w), uintptr(h),
		uintptr(unsafe.Pointer(&blend)),
	)

	SelectObject(srcDC, oldSrc)
	DeleteDC(srcDC)
	procDeleteObject.Call(uintptr(memBitmap))
}

// DrawImageBGRA 绘制一张不透明 32bpp 位图（core.ImageDrawingContext）。
// pix 字节序为 B、G、R、保留（Win32 DIB 布局）。
// 统一走"源像素装入 CreateDIBSection → SelectObject → BitBlt"路径：
// 实测（TestDiagPartialAndFix）SetDIBitsToDevice/StretchDIBits 以指针直供
// 位图数据时，真实窗口 DC 上超过约 1300×976 的大块传输会整块静默失败
// （最大化窗口下图片预览空白的根因），而 DIB+BitBlt 组合在所有尺寸下
// 稳定。1:1 时 BitBlt 不拉伸；尺寸不同时经 StretchBlt HALFTONE 缩放。
func (g *GDIContext) DrawImageBGRA(pix []byte, pixW, pixH int32, x, y, w, h int32) {
	if g.memDC == 0 || pix == nil || pixW <= 0 || pixH <= 0 || w <= 0 || h <= 0 {
		return
	}
	if int64(len(pix)) < int64(pixW)*int64(pixH)*4 {
		return
	}
	var bi bitmapInfoHeader
	bi.Size = uint32(unsafe.Sizeof(bi))
	bi.Width = pixW
	bi.Height = -pixH // top-down
	bi.Planes = 1
	bi.BitCount = 32
	bi.Compression = 0 // BI_RGB

	var bits *byte
	srcBmp := CreateDIBSection(g.memDC, &bi, 0, &bits, 0, 0)
	if srcBmp == 0 || bits == nil {
		if srcBmp != 0 {
			procDeleteObject.Call(uintptr(srcBmp))
		}
		return
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), int(pixW)*int(pixH)*4)
	copy(dst, pix)

	srcDC := CreateCompatibleDC(g.memDC)
	oldSrc := SelectObject(srcDC, srcBmp)
	if pixW == w && pixH == h {
		// 1:1 块传输。
		BitBlt(g.memDC, x, y, w, h, srcDC, 0, 0, SRCCOPY)
	} else {
		// 拉伸路径：HALFTONE 提供平滑插值；调用后须重置画刷原点。
		procSetStretchBltMode.Call(uintptr(g.memDC), halftone)
		procSetBrushOrgEx.Call(uintptr(g.memDC), 0, 0, 0)
		procStretchBlt.Call(
			uintptr(g.memDC),
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			uintptr(srcDC), 0, 0, uintptr(pixW), uintptr(pixH),
			SRCCOPY,
		)
	}
	SelectObject(srcDC, oldSrc)
	DeleteDC(srcDC)
	procDeleteObject.Call(uintptr(srcBmp))
}

// brushFor 返回按颜色缓存的画刷；GDI 画刷可安全地被多个 DC 共享。
func (g *GDIContext) brushFor(color uint32) syscall.Handle {
	if brush, ok := g.brushes[color]; ok {
		return brush
	}
	brush := CreateSolidBrush(color)
	if brush != 0 {
		g.brushes[color] = brush
	}
	return brush
}

// penFor 返回按样式/宽度/颜色缓存的画笔。
func (g *GDIContext) penFor(style, width int32, color uint32) syscall.Handle {
	if width < 1 {
		width = 1
	}
	key := penKey{style: style, width: width, color: color}
	if pen, ok := g.pens[key]; ok {
		return pen
	}
	pen := CreatePen(style, width, color)
	if pen != 0 {
		g.pens[key] = pen
	}
	return pen
}

func (g *GDIContext) DrawText(x, y int32, text string, color uint32) {
	if g.memDC == 0 {
		return
	}
	SetTextColor(g.memDC, color)
	SetBkMode(g.memDC, TRANSPARENT)
	TextOut(g.memDC, x, y, text)
}

func (g *GDIContext) MeasureText(text string) (width, height int32) {
	if g.memDC == 0 {
		return 0, 0
	}
	var size SIZE
	GetTextExtentPoint32(g.memDC, text, &size)
	// GetTextExtentPoint32 返回字体单元高度（含 internal leading），
	// 直接用于 (H-h)/2 垂直居中会让文字视觉偏下。改用 TEXTMETRIC 的
	// ascent+descent（真实字形高度）作为文本高度。
	return size.Cx, g.textMetrics().tmHeight
}

// textMetrics 查询 memDC 当前字体的度量信息。
func (g *GDIContext) textMetrics() TEXTMETRICW {
	var tm TEXTMETRICW
	GetTextMetrics(g.memDC, &tm)
	return tm
}

func (g *GDIContext) SetFont(name string, size int32) {
	if g.memDC == 0 || size <= 0 {
		return
	}
	key := fontKey{name: name, size: size, quality: g.fontQuality}
	font, ok := g.fonts[key]
	if !ok {
		font = createFontQuality(name, size, g.fontQuality)
		if font == 0 {
			return
		}
		g.fonts[key] = font
	}
	old := SelectObject(g.memDC, font)
	if g.oldFont == 0 {
		g.oldFont = old
	}
}

func (g *GDIContext) SetBkMode(transparent bool) {
	if transparent {
		SetBkMode(g.memDC, TRANSPARENT)
	} else {
		SetBkMode(g.memDC, OPAQUE)
	}
}

func (g *GDIContext) SetClipRect(x, y, w, h int32) {
	if g.memDC == 0 {
		return
	}
	// 硬性设置：丢弃已有的基准裁剪与嵌套层级。
	g.ResetClip()
	if w <= 0 || h <= 0 {
		return
	}
	rect := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	g.clipBase, g.clipBaseSet = rect, true
	g.baseRgn = g.selectRegion(rect)
}

// PushClipRect intersects the requested rectangle with the active clip and
// remembers the previous level, so PopClip can restore it.
func (g *GDIContext) PushClipRect(x, y, w, h int32) {
	if g.memDC == 0 {
		return
	}
	rect := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
	switch {
	case len(g.clipStack) > 0:
		rect = intersectRECT(g.clipStack[len(g.clipStack)-1].rect, rect)
	case g.clipBaseSet:
		rect = intersectRECT(g.clipBase, rect)
	}
	if rect.Left >= rect.Right || rect.Top >= rect.Bottom {
		// 空区域：用画布外的一小块区域实现“全部裁掉”。
		rect = RECT{Left: -2, Top: -2, Right: -1, Bottom: -1}
	}
	g.clipStack = append(g.clipStack, clipEntry{rect: rect, rgn: g.selectRegion(rect)})
}

func (g *GDIContext) PopClip() {
	if g.memDC == 0 || len(g.clipStack) == 0 {
		return
	}
	top := g.clipStack[len(g.clipStack)-1]
	g.clipStack = g.clipStack[:len(g.clipStack)-1]
	if top.rgn != 0 {
		DeleteObject(top.rgn)
	}
	if n := len(g.clipStack); n > 0 {
		SelectClipRgn(g.memDC, g.clipStack[n-1].rgn)
		return
	}
	if g.clipBaseSet {
		SelectClipRgn(g.memDC, g.baseRgn)
		return
	}
	SelectClipRgn(g.memDC, 0)
}

// selectRegion 创建并选中一个矩形裁剪区，返回其句柄（可能为 0）。
func (g *GDIContext) selectRegion(rect RECT) syscall.Handle {
	rgn := CreateRectRgn(rect.Left, rect.Top, rect.Right, rect.Bottom)
	if rgn != 0 {
		SelectClipRgn(g.memDC, rgn)
	}
	return rgn
}

func intersectRECT(a, b RECT) RECT {
	if b.Left > a.Left {
		a.Left = b.Left
	}
	if b.Top > a.Top {
		a.Top = b.Top
	}
	if b.Right < a.Right {
		a.Right = b.Right
	}
	if b.Bottom < a.Bottom {
		a.Bottom = b.Bottom
	}
	return a
}

func (g *GDIContext) ResetClip() {
	if g.memDC == 0 {
		return
	}
	for _, entry := range g.clipStack {
		if entry.rgn != 0 {
			DeleteObject(entry.rgn)
		}
	}
	g.clipStack = nil
	if g.baseRgn != 0 {
		DeleteObject(g.baseRgn)
		g.baseRgn = 0
	}
	g.clipBaseSet = false
	SelectClipRgn(g.memDC, 0)
}

// SetTarget updates the paint DC used by EndDraw. BeginPaint can return a
// different HDC after a resize, while the off-screen surface remains reusable.
func (g *GDIContext) SetTarget(hdc syscall.Handle) {
	if hdc != 0 {
		g.hdc = hdc
	}
}

// Resize recreates only the off-screen bitmap when the client size changes.
func (g *GDIContext) Resize(hdc syscall.Handle, width, height int32) {
	if hdc != 0 {
		g.hdc = hdc
	}
	if g.memDC == 0 || g.hdc == 0 || width <= 0 || height <= 0 {
		return
	}
	if g.memBitmap != 0 && g.width == width && g.height == height {
		return
	}
	if g.memBitmap != 0 {
		SelectObject(g.memDC, g.oldBitmap)
		DeleteObject(g.memBitmap)
		g.memBitmap = 0
	}
	g.memBitmap = CreateCompatibleBitmap(g.hdc, width, height)
	if g.memBitmap == 0 {
		g.width, g.height = 0, 0
		return
	}
	g.oldBitmap = SelectObject(g.memDC, g.memBitmap)
	g.width, g.height = width, height
}

func (g *GDIContext) Cleanup() {
	if g.memDC != 0 && g.oldFont != 0 {
		SelectObject(g.memDC, g.oldFont)
	}
	// 释放所有缓存的 GDI 对象（此时均已不在 DC 中选中）
	for _, brush := range g.brushes {
		DeleteObject(brush)
	}
	g.brushes = nil
	for _, pen := range g.pens {
		DeleteObject(pen)
	}
	g.pens = nil
	for _, font := range g.fonts {
		DeleteObject(font)
	}
	g.fonts = nil
	if g.oldBitmap != 0 {
		SelectObject(g.memDC, g.oldBitmap)
	}
	if g.memBitmap != 0 {
		DeleteObject(g.memBitmap)
	}
	if g.memDC != 0 {
		// 先释放裁剪区域，再销毁 DC。
		g.ResetClip()
		DeleteDC(g.memDC)
	}
	// hdc 由 BeginPaint/EndPaint 管理，不需要 ReleaseDC
}
