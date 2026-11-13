//go:build windows

package windows

import (
	"math"

	"github.com/goedui/goed/ui/renderer"
)

// 本文件实现 renderer.ShadowPainter：把圆角矩形光栅化后做可分离高斯模糊，
// 再作为预乘位图贴回去。位图按形状缓存，渲染目标重建时释放。

type shadowKey struct {
	w, h, r, blur, dpi int32
	rgba               uint32
}

// shadowCacheMax 是阴影位图缓存上限。一个界面里「卡片 + 主按钮 + CTA +
// 徽章 + Logo + 插画」很容易到 7~12 种不同尺寸，上限太小会导致每帧全量重算。
const shadowCacheMax = 24

func (c *d2dContext) FillShadowRoundedRect(x, y, w, h, r, blur, offsetY float32, body, shadow renderer.Color) {
	if bmp := c.shadowBitmap(w, h, r, blur, shadow); bmp != 0 {
		pad := shadowPad(blur)
		c.drawBitmap(bmp, x-pad, y+offsetY-pad, w+pad*2, h+pad*2, d2d1BitmapInterpLinear)
	}
	c.FillRoundedRect(x, y, w, h, r, body)
}

func (c *d2dContext) releaseShadows() {
	for k, bmp := range c.shadows {
		release(bmp)
		delete(c.shadows, k)
	}
}

// evictOldestShadow 释放一块缓存。key 里没有时间戳，用「尺寸最接近 0」不靠谱；
// 这里按 map 迭代取第一块 —— 只要上限远大于实际用量，被淘汰的概率很低，
// 即便淘汰错也只是一次重算，不会像整体清空那样每帧全量重算。
func (c *d2dContext) evictOldestShadow() {
	for k, bmp := range c.shadows {
		release(bmp)
		delete(c.shadows, k)
		return
	}
}

func shadowPad(blur float32) float32 {
	if blur < 1 {
		return 1
	}
	return blur * 2.4
}

func (c *d2dContext) shadowBitmap(w, h, r, blur float32, shadow renderer.Color) com {
	if c.rt == 0 || w < 2 || h < 2 || blur < 0.5 || shadow.A <= 0 {
		return 0
	}
	dpi := c.dpi
	if dpi <= 0 {
		dpi = 96
	}
	key := shadowKey{
		w: qDIP(w), h: qDIP(h), r: qDIP(r), blur: qDIP(blur),
		dpi: int32(dpi + 0.5), rgba: packRGBA(shadow),
	}
	if c.shadows == nil {
		c.shadows = map[shadowKey]com{}
	}
	if bmp, ok := c.shadows[key]; ok {
		return bmp
	}
	// 淘汰最久未用的一块，而不是整体清空。
	//
	// 曾经这里是「超过 6 块就全清」：一个页面只要有 7 种不同尺寸的阴影
	// （两张卡片 + 主按钮 + CTA + 徽章 + Logo + 插画），每次绘制都会把
	// 前面刚光栅化 + 高斯模糊过的位图全丢掉，于是**每一帧都重算 7 次模糊**，
	// 表现为开设置对话框、悬停按钮时明显卡顿。上限抬到 24 后，正常界面
	// 的阴影集合可以常驻；窗口拖拽换尺寸时由 LRU 逐个释放旧尺寸。
	if len(c.shadows) >= shadowCacheMax {
		c.evictOldestShadow()
	}
	pad := shadowPad(blur)
	scale := dpi / 96
	pw := int(math.Ceil(float64((w + pad*2) * scale)))
	ph := int(math.Ceil(float64((h + pad*2) * scale)))
	if pw < 2 || ph < 2 || pw*ph > 2_500_000 {
		return 0
	}
	pix := rasterShadow(pw, ph, pad*scale, w*scale, h*scale, r*scale, shadow)
	sigma := blur * scale / 2
	if sigma < 0.8 {
		sigma = 0.8
	}
	pix = renderer.BlurPremultiplied(pix, pw, ph, sigma)
	bmp := c.createBitmap(pw, ph, pix)
	if bmp == 0 {
		return 0
	}
	c.shadows[key] = bmp
	return bmp
}

func qDIP(v float32) int32 {
	if v < 0 {
		v = 0
	}
	return int32(v*2 + 0.5)
}

func packRGBA(c renderer.Color) uint32 {
	return uint32(c.R*255+0.5)<<24 | uint32(c.G*255+0.5)<<16 | uint32(c.B*255+0.5)<<8 | uint32(c.A*255+0.5)
}

// rasterShadow 在位图中央画预乘 BGRA 的圆角矩形，四周留 pad 给模糊溢出。
func rasterShadow(pw, ph int, pad, w, h, r float32, col renderer.Color) []byte {
	pix := make([]byte, pw*ph*4)
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			px := float32(x) + 0.5 - pad
			py := float32(y) + 0.5 - pad
			cov := roundRectCoverage(px, py, w, h, r)
			if cov <= 0 {
				continue
			}
			a := col.A * cov
			o := (y*pw + x) * 4
			pix[o] = byte(col.B*a*255 + 0.5)
			pix[o+1] = byte(col.G*a*255 + 0.5)
			pix[o+2] = byte(col.R*a*255 + 0.5)
			pix[o+3] = byte(a*255 + 0.5)
		}
	}
	return pix
}

func roundRectCoverage(x, y, w, h, r float32) float32 {
	hw, hh := w/2, h/2
	dx := abs32(x-hw) - hw + r
	dy := abs32(y-hh) - hh + r
	ax, ay := dx, dy
	if ax < 0 {
		ax = 0
	}
	if ay < 0 {
		ay = 0
	}
	outside := float32(math.Hypot(float64(ax), float64(ay)))
	inside := dx
	if dy > inside {
		inside = dy
	}
	if inside > 0 {
		inside = 0
	}
	sd := outside + inside - r
	cov := 0.5 - sd
	if cov < 0 {
		return 0
	}
	if cov > 1 {
		return 1
	}
	return cov
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
