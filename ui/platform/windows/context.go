//go:build windows

package windows

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

type formatKey struct {
	family string
	size   float32
	weight int
	align  renderer.Align
	valign renderer.Align
	nowrap bool
}

// d2dContext 把 Direct2D / DirectWrite 接到 renderer.Context。
type d2dContext struct {
	rt           com
	brush        com
	dwrite       com
	locale       []uint16
	formats      map[formatKey]com
	formatOrder  []formatKey
	images       map[uint64]cachedBitmap
	imageOrder   []uint64
	imageBytes   int
	scratch      []byte
	width        float32
	height       float32
	dpi          float32
}

func (c *d2dContext) Width() float32  { return c.width }
func (c *d2dContext) Height() float32 { return c.height }
func (c *d2dContext) DPI() float32    { return c.dpi }

func (c *d2dContext) Clear(col renderer.Color) {
	if c.rt == 0 {
		return
	}
	color := d2d1ColorF{col.R, col.G, col.B, col.A}
	syscall.SyscallN(c.rt.slot(idxClear), uintptr(c.rt), uintptr(unsafe.Pointer(&color)))
	runtime.KeepAlive(color)
}

func (c *d2dContext) FillRect(x, y, w, h float32, col renderer.Color) {
	if c.rt == 0 || c.brush == 0 || w <= 0 || h <= 0 {
		return
	}
	c.setBrush(col)
	rc := d2d1RectF{x, y, x + w, y + h}
	syscall.SyscallN(
		c.rt.slot(idxFillRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(c.brush),
	)
	runtime.KeepAlive(rc)
}

func (c *d2dContext) DrawLine(x1, y1, x2, y2 float32, col renderer.Color, width float32) {
	if c.rt == 0 || c.brush == 0 {
		return
	}
	if width <= 0 {
		width = 1
	}
	c.setBrush(col)
	p0 := uintptr(uint64(math.Float32bits(x1)) | uint64(math.Float32bits(y1))<<32)
	p1 := uintptr(uint64(math.Float32bits(x2)) | uint64(math.Float32bits(y2))<<32)
	syscall.SyscallN(
		c.rt.slot(idxDrawLine),
		uintptr(c.rt),
		p0,
		p1,
		uintptr(c.brush),
		uintptr(math.Float32bits(width)),
		0,
	)
}

func (c *d2dContext) DrawText(text string, x, y, maxW, maxH float32, style renderer.TextStyle) {
	if c.rt == 0 || c.brush == 0 || text == "" {
		return
	}
	format := c.textFormat(style)
	if format == 0 {
		return
	}
	if maxW <= 0 {
		maxW = c.width
	}
	if maxH <= 0 {
		maxH = c.height
	}
	c.setBrush(style.Color)
	buf, n := utf16Buf(text)
	if n == 0 {
		return
	}
	rc := d2d1RectF{x, y, x + maxW, y + maxH}
	opts := uintptr(d2dDrawTextClip | d2dDrawTextEnableColorFont)
	syscall.SyscallN(
		c.rt.slot(idxDrawText),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(n),
		uintptr(format),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(c.brush),
		opts,
		0,
	)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(rc)
}

func (c *d2dContext) MeasureText(text string, maxW float32, style renderer.TextStyle) (float32, float32) {
	if c.dwrite == 0 || text == "" {
		return 0, 0
	}
	format := c.textFormat(style)
	if format == 0 {
		return 0, style.FontSize
	}
	if maxW <= 0 {
		maxW = 1e6
	}
	buf, n := utf16Buf(text)
	if n == 0 {
		return 0, 0
	}
	layout, err := createTextLayout(c.dwrite, &buf[0], n, format, maxW, 1e6)
	if err != nil {
		return float32(n) * style.FontSize * 0.5, style.FontSize * 1.25
	}
	defer release(layout)
	m, err := textLayoutMetrics(layout)
	if err != nil {
		return float32(n) * style.FontSize * 0.5, style.FontSize * 1.25
	}
	return m.widthIncludingTrailingWhitespace, m.height
}

func (c *d2dContext) setBrush(col renderer.Color) {
	color := d2d1ColorF{col.R, col.G, col.B, col.A}
	syscall.SyscallN(c.brush.slot(idxBrushSetColor), uintptr(c.brush), uintptr(unsafe.Pointer(&color)))
	runtime.KeepAlive(color)
}

func (c *d2dContext) textFormat(style renderer.TextStyle) com {
	family := style.FontFamily
	if family == "" {
		family = theme.Default.FontFamily
	}
	size := quantizeFontSize(style.FontSize)
	if size <= 0 {
		size = theme.Default.FontSize
	}
	weight := style.Weight
	if weight <= 0 {
		weight = dwriteFontWeightNormal
	}
	key := formatKey{
		family: family,
		size:   size,
		weight: weight,
		align:  style.Align,
		valign: style.VAlign,
		nowrap: style.NoWrap,
	}
	if c.formats == nil {
		c.formats = make(map[formatKey]com)
	}
	if f, ok := c.formats[key]; ok {
		c.formatOrder = touchKey(c.formatOrder, key)
		return f
	}
	var locale *uint16
	if len(c.locale) > 0 {
		locale = &c.locale[0]
	}
	f, err := createTextFormat(c.dwrite, utf16Ptr(family), size, weight, locale)
	if err != nil && family != theme.Default.FontFamily {
		f, err = createTextFormat(c.dwrite, utf16Ptr(theme.Default.FontFamily), size, weight, locale)
	}
	if err != nil {
		f, err = createTextFormat(c.dwrite, utf16Ptr("Segoe UI"), size, weight, locale)
	}
	if err != nil {
		return 0
	}
	// 三种水平对齐都要落到 IDWriteTextFormat 上。此前只处理了 AlignCenter，
	// 于是 AlignEnd 被静默忽略、文本仍从左边画起 —— 快捷键这类右对齐文本
	// 会直接压在标签上。
	switch style.Align {
	case renderer.AlignCenter:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignCenter))
	case renderer.AlignEnd:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignTrailing))
	default:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignLeading))
	}
	switch style.VAlign {
	case renderer.AlignCenter:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphCenter))
	case renderer.AlignEnd:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphFar))
	default:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphNear))
	}
	if style.NoWrap {
		syscall.SyscallN(f.slot(idxSetWordWrapping), uintptr(f), uintptr(dwriteWordWrappingNoWrap))
	}
	c.rememberFormat(key, f)
	return f
}

func (c *d2dContext) releaseFormats() {
	for k, f := range c.formats {
		release(f)
		delete(c.formats, k)
	}
	c.formatOrder = nil
}
