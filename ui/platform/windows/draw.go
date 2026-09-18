//go:build windows

package windows

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

const (
	idxCreateBitmap           = 4
	idxDrawRectangle          = 16
	idxFillRoundedRectangle   = 19
	idxDrawBitmap             = 26
	idxPushAxisAlignedClip    = 45
	idxPopAxisAlignedClip     = 46
	d2d1BitmapInterpLinear    = 1
	d2d1BitmapInterpNearest   = 0
	d2d1AntialiasPerPrimitive = 0
)

type d2d1RoundedRect struct {
	rect    d2d1RectF
	radiusX float32
	radiusY float32
}

type d2d1BitmapProperties struct {
	pixelFormat d2d1PixelFormat
	dpiX        float32
	dpiY        float32
}

func (c *d2dContext) FillRoundedRect(x, y, w, h, r float32, col renderer.Color) {
	if c.rt == 0 || c.brush == 0 || w <= 0 || h <= 0 {
		return
	}
	if r <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}
	maxR := w
	if h < maxR {
		maxR = h
	}
	if r > maxR/2 {
		r = maxR / 2
	}
	c.setBrush(col)
	rr := d2d1RoundedRect{
		rect:    d2d1RectF{x, y, x + w, y + h},
		radiusX: r,
		radiusY: r,
	}
	syscall.SyscallN(
		c.rt.slot(idxFillRoundedRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rr)),
		uintptr(c.brush),
	)
	runtime.KeepAlive(rr)
}

func (c *d2dContext) DrawRect(x, y, w, h float32, col renderer.Color, width float32) {
	if c.rt == 0 || c.brush == 0 || w <= 0 || h <= 0 {
		return
	}
	if width <= 0 {
		width = 1
	}
	c.setBrush(col)
	rc := d2d1RectF{x, y, x + w, y + h}
	syscall.SyscallN(
		c.rt.slot(idxDrawRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(c.brush),
		uintptr(math.Float32bits(width)),
		0,
	)
	runtime.KeepAlive(rc)
}

func (c *d2dContext) PushClip(x, y, w, h float32) {
	if c.rt == 0 {
		return
	}
	rc := d2d1RectF{x, y, x + w, y + h}
	syscall.SyscallN(
		c.rt.slot(idxPushAxisAlignedClip),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(d2d1AntialiasPerPrimitive),
	)
	runtime.KeepAlive(rc)
}

func (c *d2dContext) PopClip() {
	if c.rt == 0 {
		return
	}
	syscall.SyscallN(c.rt.slot(idxPopAxisAlignedClip), uintptr(c.rt))
}

func (c *d2dContext) DrawImage(data []byte, x, y, w, h float32) {
	if c.rt == 0 || len(data) == 0 || w <= 0 || h <= 0 {
		return
	}
	bmp := c.bitmap(data)
	if bmp == 0 {
		return
	}
	c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpLinear)
}

// DrawPixels 画原始 NRGBA 像素。每次调用都新建临时位图（内容每帧都变，
// 缓存没有意义），画完即释放。近邻采样，保证像素画面放大后边缘锐利。
func (c *d2dContext) DrawPixels(pix []byte, pw, ph int, x, y, w, h float32) {
	needed := pw * ph * 4
	if c.rt == 0 || pw < 1 || ph < 1 || len(pix) < needed || w <= 0 || h <= 0 {
		return
	}
	if cap(c.scratch) < needed {
		c.scratch = make([]byte, needed)
	}
	bgra := c.scratch[:needed]
	for i := 0; i < needed; i += 4 {
		a := uint32(pix[i+3])
		bgra[i] = byte(uint32(pix[i+2]) * a / 255)
		bgra[i+1] = byte(uint32(pix[i+1]) * a / 255)
		bgra[i+2] = byte(uint32(pix[i]) * a / 255)
		bgra[i+3] = byte(a)
	}
	bmp := c.createBitmap(pw, ph, bgra)
	if bmp == 0 {
		return
	}
	c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpNearest)
	release(bmp)
}

func (c *d2dContext) drawBitmap(bmp com, x, y, w, h float32, interp uintptr) {
	dst := d2d1RectF{x, y, x + w, y + h}
	syscall.SyscallN(
		c.rt.slot(idxDrawBitmap),
		uintptr(c.rt),
		uintptr(bmp),
		uintptr(unsafe.Pointer(&dst)),
		uintptr(math.Float32bits(1)),
		interp,
		0,
	)
	runtime.KeepAlive(dst)
}
