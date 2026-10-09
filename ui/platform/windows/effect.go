//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

// 本文件是 renderer.GradientPainter 的 Windows 实现（D2D 线性渐变）。
// 阴影在 shadow.go：圆角矩形光栅化后走可分离高斯模糊，再贴回预乘位图。

const (
	d2d1Gamma22         = 1 // D2D1_GAMMA_2_2，sRGB 感知均匀，与 CSS 渐变观感一致
	d2d1ExtendModeClamp = 0 // D2D1_EXTEND_MODE_CLAMP
)

// d2d1GradientStop 对应 D2D1_GRADIENT_STOP：位置 + 颜色（4 + 16 字节，4 对齐）。
type d2d1GradientStop struct {
	position float32
	color    d2d1ColorF
}

// d2d1LinearGradientBrushProperties 对应 D2D1_LINEAR_GRADIENT_BRUSH_PROPERTIES。
type d2d1LinearGradientBrushProperties struct {
	startPoint d2d1Point2F
	endPoint   d2d1Point2F
}

// d2d1BrushProperties 对应 D2D1_BRUSH_PROPERTIES：不透明度 + 6 元素变换矩阵。
type d2d1BrushProperties struct {
	opacity   float32
	transform d2d1Matrix3x3F
}

// FillGradientRoundedRect 实现 renderer.GradientPainter：建两点渐变刷，
// 用 FillRoundedRectangle（槽位与纯色填充同一个）着色。任一步失败退回 from 纯色。
func (c *d2dContext) FillGradientRoundedRect(x, y, w, h, r float32, from, to renderer.Color, vertical bool) {
	if c.rt == 0 || w <= 0 || h <= 0 {
		return
	}
	maxR := w
	if h < maxR {
		maxR = h
	}
	if r > maxR/2 {
		r = maxR / 2
	}
	if r < 0 {
		r = 0
	}
	stops := [2]d2d1GradientStop{
		{position: 0, color: d2d1ColorF{from.R, from.G, from.B, from.A}},
		{position: 1, color: d2d1ColorF{to.R, to.G, to.B, to.A}},
	}
	var coll com
	hr, _, _ := syscall.SyscallN(
		c.rt.slot(idxCreateGradientStopCollection),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&stops[0])),
		uintptr(len(stops)),
		uintptr(d2d1Gamma22),
		uintptr(d2d1ExtendModeClamp),
		uintptr(unsafe.Pointer(&coll)),
	)
	runtime.KeepAlive(stops)
	if hresult(hr).failed() || coll == 0 {
		c.FillRoundedRect(x, y, w, h, r, from)
		return
	}
	defer release(coll)

	end := d2d1Point2F{x + w, y}
	if vertical {
		end = d2d1Point2F{x, y + h}
	}
	props := d2d1LinearGradientBrushProperties{
		startPoint: d2d1Point2F{x, y},
		endPoint:   end,
	}
	bp := d2d1BrushProperties{opacity: 1, transform: d2d1IdentityMatrix}
	var brush com
	hr, _, _ = syscall.SyscallN(
		c.rt.slot(idxCreateLinearGradientBrush),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&props)),
		uintptr(unsafe.Pointer(&bp)),
		uintptr(coll),
		uintptr(unsafe.Pointer(&brush)),
	)
	runtime.KeepAlive(props)
	runtime.KeepAlive(bp)
	if hresult(hr).failed() || brush == 0 {
		c.FillRoundedRect(x, y, w, h, r, from)
		return
	}
	defer release(brush)

	if r <= 0 {
		rc := d2d1RectF{x, y, x + w, y + h}
		syscall.SyscallN(
			c.rt.slot(idxFillRectangle),
			uintptr(c.rt),
			uintptr(unsafe.Pointer(&rc)),
			uintptr(brush),
		)
		runtime.KeepAlive(rc)
		return
	}
	rr := d2d1RoundedRect{
		rect:    d2d1RectF{x, y, x + w, y + h},
		radiusX: r,
		radiusY: r,
	}
	syscall.SyscallN(
		c.rt.slot(idxFillRoundedRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rr)),
		uintptr(brush),
	)
	runtime.KeepAlive(rr)
}
