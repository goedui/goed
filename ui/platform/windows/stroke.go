//go:build windows

package windows

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

// 圆头描边走 ID2D1PathGeometry + ID2D1StrokeStyle。槽位按 d2d1.h 声明顺序，
// 与已验证的 CreateRoundedRectangleGeometry=6、CreateHwndRenderTarget=14 同一基准。
const (
	idxPathOpen = 17 // ID2D1PathGeometry::Open（IUnknown 0-2 + ID2D1Geometry 3-16）

	// ID2D1SimplifiedGeometrySink：3 SetFillMode、4 SetSegmentFlags、5 BeginFigure、
	// 6 AddLines、7 AddBeziers、8 EndFigure、9 Close。
	// ID2D1GeometrySink 在其后：10 AddLine、11 AddBezier。
	// 单点必须走 AddLine/AddBezier；AddLines 要数组加个数，传错会把设备画进错误状态。
	idxSinkBeginFigure = 5
	idxSinkEndFigure   = 8
	idxSinkClose       = 9
	idxSinkAddLine     = 10
	idxSinkAddBezier   = 11

	d2d1FillModeWinding   = 1
	d2d1FigureBeginFilled = 0
	d2d1FigureEndOpen     = 0
	d2d1CapStyleRound     = 2
	d2d1LineJoinRound     = 1
)

// d2d1BezierSegment 对应 D2D1_BEZIER_SEGMENT：三个点，24 字节。
type d2d1BezierSegment struct {
	point1, point2, point3 d2d1Point2F
}

// d2d1StrokeStyleProperties 对应 D2D1_STROKE_STYLE_PROPERTIES。
// 头文件顺序是 start/end/dash cap、lineJoin、miterLimit、dashStyle、dashOffset。
type d2d1StrokeStyleProperties struct {
	startCap, endCap, dashCap int32
	lineJoin                  int32
	miterLimit                float32
	dashStyle                 int32
	dashOffset                float32
}

func (c *d2dContext) ensureStroke() {
	if c.stroke != 0 || c.d2d == 0 {
		return
	}
	props := d2d1StrokeStyleProperties{
		startCap:   d2d1CapStyleRound,
		endCap:     d2d1CapStyleRound,
		dashCap:    d2d1CapStyleRound,
		lineJoin:   d2d1LineJoinRound,
		miterLimit: 10,
	}
	var style com
	hr, _, _ := syscall.SyscallN(
		c.d2d.slot(idxFactoryCreateStrokeStyle),
		uintptr(c.d2d),
		uintptr(unsafe.Pointer(&props)),
		0,
		0,
		uintptr(unsafe.Pointer(&style)),
	)
	runtime.KeepAlive(props)
	if hresult(hr).failed() || style == 0 {
		return
	}
	c.stroke = style
}

func (c *d2dContext) releaseStroke() {
	if c.stroke != 0 {
		release(c.stroke)
		c.stroke = 0
	}
}

// DrawRoundStroke 实现 renderer.RoundStrokePainter：一条圆头折线，可含三次贝塞尔。
// 工厂或 stroke style 建不出来时返回 false，调用方退回折线。
func (c *d2dContext) DrawRoundStroke(pts []renderer.StrokePoint, width float32, col renderer.Color, closed bool) bool {
	if c.rt == 0 || c.brush == 0 || c.d2d == 0 || len(pts) < 2 || width <= 0 {
		return false
	}
	c.ensureStroke()
	if c.stroke == 0 {
		return false
	}
	var path com
	hr, _, _ := syscall.SyscallN(
		c.d2d.slot(idxFactoryCreatePathGeometry),
		uintptr(c.d2d),
		uintptr(unsafe.Pointer(&path)),
	)
	if hresult(hr).failed() || path == 0 {
		return false
	}
	defer release(path)

	var sink com
	hr, _, _ = syscall.SyscallN(path.slot(idxPathOpen), uintptr(path), uintptr(unsafe.Pointer(&sink)))
	if hresult(hr).failed() || sink == 0 {
		return false
	}
	ok := fillSink(sink, pts, closed)
	syscall.SyscallN(sink.slot(idxSinkClose), uintptr(sink))
	release(sink)
	if !ok {
		return false
	}

	c.setBrush(col)
	syscall.SyscallN(
		c.rt.slot(idxDrawGeometry),
		uintptr(c.rt),
		uintptr(path),
		uintptr(c.brush),
		uintptr(math.Float32bits(width)),
		uintptr(c.stroke),
	)
	return true
}

func fillSink(sink com, pts []renderer.StrokePoint, closed bool) bool {
	start := d2d1Point2F{pts[0].X, pts[0].Y}
	// D2D1_POINT_2F 是 8 字节，x64 上按值进寄存器，不能传指针。
	syscall.SyscallN(
		sink.slot(idxSinkBeginFigure),
		uintptr(sink),
		packPoint(start),
		uintptr(d2d1FigureBeginFilled),
	)
	for i := 1; i < len(pts); i++ {
		p := pts[i]
		if p.Curve {
			seg := d2d1BezierSegment{
				point1: d2d1Point2F{p.C1X, p.C1Y},
				point2: d2d1Point2F{p.C2X, p.C2Y},
				point3: d2d1Point2F{p.X, p.Y},
			}
			syscall.SyscallN(sink.slot(idxSinkAddBezier), uintptr(sink), uintptr(unsafe.Pointer(&seg)))
			runtime.KeepAlive(seg)
			continue
		}
		syscall.SyscallN(sink.slot(idxSinkAddLine), uintptr(sink), packPoint(d2d1Point2F{p.X, p.Y}))
	}
	end := uintptr(d2d1FigureEndOpen)
	if closed {
		end = 1 // D2D1_FIGURE_END_CLOSED
	}
	syscall.SyscallN(sink.slot(idxSinkEndFigure), uintptr(sink), end)
	return true
}

func packPoint(p d2d1Point2F) uintptr {
	return uintptr(uint64(math.Float32bits(p.x)) | uint64(math.Float32bits(p.y))<<32)
}
