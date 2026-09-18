package components

import (
	"encoding/binary"
	"math"

	"github.com/goedui/goed/ui/renderer"
)

// Icon 是 github.com/goedui/goed/ui 的矢量图标。绘制走 Direct2D 的 FillRect 几何，不依赖字体字形。
type Icon struct {
	Name  string
	Paint renderer.IconPaint
}

// App 是默认应用图标：细线窗口 + 顶栏，和标题栏最小化 / 最大化同一套几何语言。
var App = Icon{
	Name:  "app",
	Paint: PaintApp,
}

// PaintApp 在 size×size 的 DIP 矩形里画默认应用标记。
func PaintApp(ctx renderer.Context, x, y, size float32, fg, bg renderer.Color) {
	if ctx == nil || size <= 0 {
		return
	}
	_ = bg
	pad := size * 0.125
	if pad < 1 {
		pad = 1
	}
	stroke := size * 0.09
	if stroke < 1.1 {
		stroke = 1.1
	}
	fx := x + pad
	fy := y + pad
	fw := size - pad*2
	fh := size - pad*2
	if fw <= stroke*2 || fh <= stroke*2 {
		ctx.FillRect(x, y, size, size, fg)
		return
	}

	strokeRect(ctx, fx, fy, fw, fh, stroke, fg)

	capH := fh * 0.26
	if capH < stroke*2 {
		capH = stroke * 2
	}
	ctx.FillRect(fx, fy, fw, capH, fg)

	inset := fw * 0.18
	lineW := fw * 0.52
	lineH := stroke
	gap := fh * 0.16
	ly := fy + capH + fh*0.16
	ctx.FillRect(fx+inset, ly, lineW, lineH, fg)
	ctx.FillRect(fx+inset, ly+gap, lineW*0.68, lineH, fg)
}

func strokeRect(ctx renderer.Context, x, y, w, h, s float32, c renderer.Color) {
	if w <= 0 || h <= 0 || s <= 0 {
		return
	}
	ctx.FillRect(x, y, w, s, c)
	ctx.FillRect(x, y+h-s, w, s, c)
	ctx.FillRect(x, y, s, h, c)
	ctx.FillRect(x+w-s, y, s, h, c)
}

// RGBA 把图标栅格化成 size×size 的顶向下 RGBA。标题栏预览用透明底。
func (ic Icon) RGBA(size int, fg, bg renderer.Color) []byte {
	if size <= 0 {
		return nil
	}
	p := ic.Paint
	if p == nil {
		p = PaintApp
	}
	ctx := newPix(size, size)
	p(ctx, 0, 0, float32(size), fg, bg)
	return ctx.pix
}

// ShellRGBA 生成任务栏 / Alt+Tab 用的圆角瓷砖图标：深底浅窗，深浅主题都能认。
func (ic Icon) ShellRGBA(size int) []byte {
	if size <= 0 {
		return nil
	}
	ctx := newPix(size, size)
	tile := renderer.RGB(32, 32, 36)
	fg := renderer.RGB(240, 240, 242)
	fillRounded(ctx, 0, 0, float32(size), float32(size), float32(size)*0.18, tile)
	p := ic.Paint
	if p == nil {
		p = PaintApp
	}
	p(ctx, 0, 0, float32(size), fg, tile)
	return ctx.pix
}

// WindowsIcon 返回 CreateIconFromResourceEx 可用的 BITMAPINFOHEADER + XOR + AND。
func (ic Icon) WindowsIcon(size int) []byte {
	return encodeIconImage(size, ic.ShellRGBA(size))
}

func encodeIconImage(size int, rgba []byte) []byte {
	if size <= 0 || len(rgba) < size*size*4 {
		return nil
	}
	xorSize := size * size * 4
	andRow := ((size + 31) / 32) * 4
	andSize := andRow * size
	buf := make([]byte, 40+xorSize+andSize)
	binary.LittleEndian.PutUint32(buf[0:], 40)
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:], uint32(size*2))
	binary.LittleEndian.PutUint16(buf[12:], 1)
	binary.LittleEndian.PutUint16(buf[14:], 32)
	binary.LittleEndian.PutUint32(buf[20:], uint32(xorSize))
	off := 40
	for y := size - 1; y >= 0; y-- {
		row := y * size * 4
		for x := 0; x < size; x++ {
			i := row + x*4
			buf[off] = rgba[i+2]
			buf[off+1] = rgba[i+1]
			buf[off+2] = rgba[i]
			buf[off+3] = rgba[i+3]
			off += 4
		}
	}
	return buf
}

type pixCtx struct {
	w, h int
	pix  []byte
}

func newPix(w, h int) *pixCtx {
	return &pixCtx{w: w, h: h, pix: make([]byte, w*h*4)}
}

func (p *pixCtx) Width() float32  { return float32(p.w) }
func (p *pixCtx) Height() float32 { return float32(p.h) }
func (p *pixCtx) DPI() float32    { return 96 }
func (p *pixCtx) Clear(c renderer.Color) {
	p.FillRect(0, 0, float32(p.w), float32(p.h), c)
}
func (p *pixCtx) DrawLine(float32, float32, float32, float32, renderer.Color, float32) {}
func (p *pixCtx) DrawRect(x, y, w, h float32, c renderer.Color, width float32) {
	if width <= 0 {
		width = 1
	}
	p.FillRect(x, y, w, width, c)
	p.FillRect(x, y+h-width, w, width, c)
	p.FillRect(x, y, width, h, c)
	p.FillRect(x+w-width, y, width, h, c)
}
func (p *pixCtx) FillRoundedRect(x, y, w, h, r float32, c renderer.Color) {
	_ = r
	p.FillRect(x, y, w, h, c)
}
func (p *pixCtx) DrawText(string, float32, float32, float32, float32, renderer.TextStyle) {
}
func (p *pixCtx) MeasureText(string, float32, renderer.TextStyle) (float32, float32) {
	return 0, 0
}
func (p *pixCtx) DrawImage([]byte, float32, float32, float32, float32) {}
func (p *pixCtx) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (p *pixCtx) PushClip(float32, float32, float32, float32)          {}
func (p *pixCtx) PopClip()                                             {}

func (p *pixCtx) FillRect(x, y, w, h float32, c renderer.Color) {
	if p == nil || w <= 0 || h <= 0 || c.A <= 0 {
		return
	}
	x0 := int(math.Floor(float64(x)))
	y0 := int(math.Floor(float64(y)))
	x1 := int(math.Ceil(float64(x + w)))
	y1 := int(math.Ceil(float64(y + h)))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > p.w {
		x1 = p.w
	}
	if y1 > p.h {
		y1 = p.h
	}
	sr, sg, sb, sa := to8(c.R), to8(c.G), to8(c.B), to8(c.A)
	for yy := y0; yy < y1; yy++ {
		row := yy * p.w * 4
		for xx := x0; xx < x1; xx++ {
			i := row + xx*4
			if sa == 255 {
				p.pix[i] = sr
				p.pix[i+1] = sg
				p.pix[i+2] = sb
				p.pix[i+3] = 255
				continue
			}
			inv := uint16(255 - sa)
			p.pix[i] = uint8((uint16(sr)*uint16(sa) + uint16(p.pix[i])*inv) / 255)
			p.pix[i+1] = uint8((uint16(sg)*uint16(sa) + uint16(p.pix[i+1])*inv) / 255)
			p.pix[i+2] = uint8((uint16(sb)*uint16(sa) + uint16(p.pix[i+2])*inv) / 255)
			a := uint16(p.pix[i+3])
			na := uint16(sa) + a*inv/255
			if na > 255 {
				na = 255
			}
			p.pix[i+3] = uint8(na)
		}
	}
}

func fillRounded(p *pixCtx, x, y, w, h, r float32, c renderer.Color) {
	if p == nil || w <= 0 || h <= 0 {
		return
	}
	if r < 1 {
		p.FillRect(x, y, w, h, c)
		return
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	x0 := int(math.Floor(float64(x)))
	y0 := int(math.Floor(float64(y)))
	x1 := int(math.Ceil(float64(x + w)))
	y1 := int(math.Ceil(float64(y + h)))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > p.w {
		x1 = p.w
	}
	if y1 > p.h {
		y1 = p.h
	}
	cx0 := x + r
	cy0 := y + r
	cx1 := x + w - r
	cy1 := y + h - r
	rr := r * r
	sr, sg, sb, sa := to8(c.R), to8(c.G), to8(c.B), to8(c.A)
	for yy := y0; yy < y1; yy++ {
		py := float32(yy) + 0.5
		row := yy * p.w * 4
		for xx := x0; xx < x1; xx++ {
			px := float32(xx) + 0.5
			dx, dy := float32(0), float32(0)
			if px < cx0 {
				dx = cx0 - px
			} else if px > cx1 {
				dx = px - cx1
			}
			if py < cy0 {
				dy = cy0 - py
			} else if py > cy1 {
				dy = py - cy1
			}
			if dx*dx+dy*dy > rr {
				continue
			}
			i := row + xx*4
			p.pix[i] = sr
			p.pix[i+1] = sg
			p.pix[i+2] = sb
			p.pix[i+3] = sa
		}
	}
}

func to8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}
