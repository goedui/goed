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
	idxCreateBitmap         = 4
	idxDrawRectangle        = 16
	idxDrawRoundedRectangle = 18
	idxFillRoundedRectangle = 19
	idxDrawBitmap           = 26
	idxPushAxisAlignedClip  = 45
	idxPopAxisAlignedClip   = 46
	// 圆角图片用的三个槽位不是猜的：idxCreateSolidColorBrush=8、idxPushAxisAlignedClip=45
	// 已验证，按 d2d1.h 的声明顺序往前 / 往后数出这三个（CreateLayer 隔着
	// CreateGradientStopCollection / CreateLinearGradientBrush / CreateRadialGradientBrush /
	// CreateCompatibleRenderTarget；PushLayer/PopLayer 前有 RestoreDrawingState /
	// SaveDrawingState / Flush 各一）。
	// CreateSolidColorBrush(8) 与 CreateLayer(13) 之间恰是 d2d1.h 声明顺序的
	// CreateGradientStopCollection / CreateLinearGradientBrush / CreateRadialGradientBrush /
	// CreateCompatibleRenderTarget 四项 —— 9/10 即渐变对（上面注释的直接推论）。
	idxCreateGradientStopCollection = 9
	idxCreateLinearGradientBrush    = 10
	idxCreateLayer                  = 13
	idxDrawGeometry                 = 22
	idxPushLayer                    = 40
	idxPopLayer                     = 41
	d2d1BitmapInterpLinear          = 1
	d2d1BitmapInterpNearest         = 0
	d2d1AntialiasPerPrimitive       = 0
	d2d1LayerOptionsNone            = 0

	// 水印旋转用的变换对：SetTransform=30 夹在 DrawGlyphRun(29) 与
	// SetAntialiasMode(32) 之间，GetTransform=31 紧随其后 —— 与上面同一张
	// d2d1.h 序号表，且 GetTransform 能读回做断言（transform_test.go）。
	idxSetTransform = 30
	idxGetTransform = 31
)

// d2d1Matrix3x3F 是 D2D1 的 3x2 仿射矩阵：名字带 3x3 但第三列固定 (0,0,1)，
// 结构体里不存，所以只有 6 个 float。
type d2d1Matrix3x3F struct {
	m11, m12 float32
	m21, m22 float32
	m31, m32 float32
}

var d2d1IdentityMatrix = d2d1Matrix3x3F{m11: 1, m22: 1}

// rotationMatrix 是「绕 (cx, cy) 转 deg 度」的 3x2 矩阵。Direct2D 用行向量约定
// （p' = p·M），屏幕坐标 y 向下，正角顺时针 —— 与 D2D1::Matrix3x2F::Rotation
// 一致（水印的 315° 就是它的读向：左下 → 右上）。
func rotationMatrix(deg, cx, cy float32) d2d1Matrix3x3F {
	s, c := math.Sincos(float64(deg) * math.Pi / 180)
	sn, co := float32(s), float32(c)
	return d2d1Matrix3x3F{
		m11: co, m12: sn,
		m21: -sn, m22: co,
		m31: cx - cx*co + cy*sn,
		m32: cy - cx*sn - cy*co,
	}
}

// RotateAt 把后续绘制绕 (cx, cy) 旋转 deg 度（顺时针），实现
// renderer.TransformPainter。直接**替换**整个变换矩阵：本引擎从不在别处设变换
// （DPI 走渲染目标自己的 DPI 属性，不进矩阵），所以不需要栈式 save/restore，
// ResetTransform 恢复成恒等矩阵即可。
func (c *d2dContext) RotateAt(deg, cx, cy float32) {
	if c.rt == 0 || deg == 0 {
		return
	}
	m := rotationMatrix(deg, cx, cy)
	syscall.SyscallN(
		c.rt.slot(idxSetTransform),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&m)),
	)
	runtime.KeepAlive(m)
}

// ResetTransform 撤销 RotateAt，回到恒等矩阵。
func (c *d2dContext) ResetTransform() {
	if c.rt == 0 {
		return
	}
	m := d2d1IdentityMatrix
	syscall.SyscallN(
		c.rt.slot(idxSetTransform),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&m)),
	)
	runtime.KeepAlive(m)
}

// d2d1LayerParameters 对应 D2D1_LAYER_PARAMETERS：字段顺序与对齐必须和 d2d1.h
// 一致，多一个字节 Direct2D 就读到垃圾（64 位下 72 字节，测试有断言钉住）。
type d2d1LayerParameters struct {
	contentBounds     d2d1RectF
	geometricMask     com
	maskAntialiasMode int32
	maskTransform     d2d1Matrix3x3F
	opacity           float32
	opacityBrush      com
	layerOptions      int32
}

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

// DrawRoundedRect 用 DrawRoundedRectangle 描边。线骑在矩形边上，和 FillRoundedRect
// 同一套圆角，直边和四角同粗。路径几何那条路槽位对不上时边框整段消失。
func (c *d2dContext) DrawRoundedRect(x, y, w, h, r float32, col renderer.Color, width float32) {
	if c.rt == 0 || c.brush == 0 || w <= 0 || h <= 0 {
		return
	}
	if r <= 0 {
		c.DrawRect(x, y, w, h, col, width)
		return
	}
	if width <= 0 {
		width = 1
	}
	limit := w
	if h < limit {
		limit = h
	}
	if r > limit/2 {
		r = limit / 2
	}
	c.setBrush(col)
	rr := d2d1RoundedRect{
		rect:    d2d1RectF{x, y, x + w, y + h},
		radiusX: r,
		radiusY: r,
	}
	syscall.SyscallN(
		c.rt.slot(idxDrawRoundedRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rr)),
		uintptr(c.brush),
		uintptr(math.Float32bits(width)),
		0,
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

// DrawImageRounded 把图片画进 (x,y,w,h) 并按半径 r 切圆角，实现
// renderer.ImagePainter。
//
// 做法是「几何遮罩 + 图层」：工厂建圆角矩形几何体当遮罩，PushLayer 后画图再
// PopLayer。不用「先画图、再拿背景色补四个角」的土办法 —— 那要求调用方知道底下
// 的颜色，而正文底可能是卡片或渐变，补出来比直角还难看。
//
// 任何一步失败都退回直角画法：圆角是锦上添花，什么都看不见才不行。
func (c *d2dContext) DrawImageRounded(data []byte, x, y, w, h, r float32) {
	if c.rt == 0 || len(data) == 0 || w <= 0 || h <= 0 {
		return
	}
	bmp := c.bitmap(data)
	if bmp == 0 {
		return
	}
	// 半径超过一半、或工厂指针没接上（测试里直接构造的 d2dContext）都走直角画法。
	if r <= 0 || r*2 > w || r*2 > h || c.d2d == 0 {
		c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpLinear)
		return
	}
	geom := c.roundedRectGeometry(x, y, w, h, r)
	layer := com(0)
	if geom != 0 {
		layer = c.createLayer()
	}
	if geom == 0 || layer == 0 {
		if geom != 0 {
			release(geom)
		}
		if layer != 0 {
			release(layer)
		}
		c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpLinear)
		return
	}
	params := d2d1LayerParameters{
		// contentBounds 显式给成图片矩形：不给就是无界图层，Direct2D 得按整个
		// 表面准备，白付一遍代价。
		contentBounds:     d2d1RectF{x, y, x + w, y + h},
		geometricMask:     geom,
		maskAntialiasMode: d2d1AntialiasPerPrimitive,
		maskTransform:     d2d1IdentityMatrix,
		opacity:           1,
		layerOptions:      d2d1LayerOptionsNone,
	}
	hr, _, _ := syscall.SyscallN(
		c.rt.slot(idxPushLayer),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&params)),
		uintptr(layer),
	)
	runtime.KeepAlive(params)
	if hresult(hr).failed() {
		release(layer)
		release(geom)
		c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpLinear)
		return
	}
	c.drawBitmap(bmp, x, y, w, h, d2d1BitmapInterpLinear)
	syscall.SyscallN(c.rt.slot(idxPopLayer), uintptr(c.rt))
	release(layer)
	release(geom)
}

// roundedRectGeometry 建一个圆角矩形几何体当图层遮罩。失败返回 0。
func (c *d2dContext) roundedRectGeometry(x, y, w, h, r float32) com {
	rr := d2d1RoundedRect{
		rect:    d2d1RectF{x, y, x + w, y + h},
		radiusX: r,
		radiusY: r,
	}
	var geom com
	hr, _, _ := syscall.SyscallN(
		c.d2d.slot(idxFactoryCreateRoundedRectGeometry),
		uintptr(c.d2d),
		uintptr(unsafe.Pointer(&rr)),
		uintptr(unsafe.Pointer(&geom)),
	)
	runtime.KeepAlive(rr)
	if hresult(hr).failed() || geom == 0 {
		return 0
	}
	return geom
}

// createLayer 建一个图层对象。每个圆角图片一个，画完即释放 —— 图层本身很轻
// （贵的是几何体），缓存反而要管生命周期。
func (c *d2dContext) createLayer() com {
	var layer com
	hr, _, _ := syscall.SyscallN(
		c.rt.slot(idxCreateLayer),
		uintptr(c.rt),
		0,
		uintptr(unsafe.Pointer(&layer)),
	)
	if hresult(hr).failed() || layer == 0 {
		return 0
	}
	return layer
}

// DrawPixels 画原始 NRGBA 像素，每次调用新建临时位图（内容每帧都变，缓存没意义），
// 画完即释放。近邻采样保证放大后边缘锐利。
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
