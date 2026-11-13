//go:build js && wasm

package web

import (
	"syscall/js"

	"github.com/goedui/goed/ui/renderer"
)

// 本文件是 renderer.GradientPainter / ShadowPainter / RadialGradientPainter 的
// Canvas 2D 实现。与 Windows 后端保持同一套降级语义：任一步失败退回纯色，
// 界面仍然可读，只是少了过渡。

// FillGradientRoundedRect 实现 renderer.GradientPainter。
func (c *canvasContext) FillGradientRoundedRect(x, y, w, h, r float32, from, to renderer.Color, vertical bool) {
	if w <= 0 || h <= 0 {
		return
	}
	if max := minf(w, h) / 2; r > max {
		r = max
	}
	var g js.Value
	if vertical {
		g = c.ctx.Call("createLinearGradient", x, y, x, y+h)
	} else {
		g = c.ctx.Call("createLinearGradient", x, y, x+w, y)
	}
	g.Call("addColorStop", 0, cssColor(from))
	g.Call("addColorStop", 1, cssColor(to))
	c.ctx.Set("fillStyle", g)
	if r > 0 {
		roundRectPath(c.ctx, x, y, w, h, r)
		c.ctx.Call("fill")
		return
	}
	c.ctx.Call("fillRect", x, y, w, h)
}

// FillRadialGradient 实现 renderer.RadialGradientPainter。
func (c *canvasContext) FillRadialGradient(g renderer.RadialGradient) {
	if g.W <= 0 || g.H <= 0 || g.Spread <= 0 {
		return
	}
	r := g.Corner
	if max := minf(g.W, g.H) / 2; r > max {
		r = max
	}
	radial := c.ctx.Call("createRadialGradient", g.CenterX, g.CenterY, 0, g.CenterX, g.CenterY, g.Spread)
	radial.Call("addColorStop", 0, cssColor(g.From))
	radial.Call("addColorStop", 1, cssColor(g.To))
	c.ctx.Set("fillStyle", radial)
	if r > 0 {
		roundRectPath(c.ctx, g.X, g.Y, g.W, g.H, r)
		c.ctx.Call("fill")
		return
	}
	c.ctx.Call("fillRect", g.X, g.Y, g.W, g.H)
}

// FillShadowRoundedRect 实现 renderer.ShadowPainter：canvas 的 shadowBlur 就是
// 高斯模糊，直接复用，不必像 Windows 后端那样先光栅化再卷积。
func (c *canvasContext) FillShadowRoundedRect(x, y, w, h, r, blur, offsetY float32, body, shadow renderer.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if max := minf(w, h) / 2; r > max {
		r = max
	}
	if shadow.A > 0 && blur > 0 {
		c.ctx.Set("shadowColor", cssColor(shadow))
		c.ctx.Set("shadowBlur", blur)
		c.ctx.Set("shadowOffsetX", 0)
		c.ctx.Set("shadowOffsetY", offsetY)
		c.ctx.Set("fillStyle", cssColor(body))
		if r > 0 {
			roundRectPath(c.ctx, x, y, w, h, r)
			c.ctx.Call("fill")
		} else {
			c.ctx.Call("fillRect", x, y, w, h)
		}
		c.ctx.Set("shadowColor", "rgba(0,0,0,0)")
		c.ctx.Set("shadowBlur", 0)
		c.ctx.Set("shadowOffsetX", 0)
		c.ctx.Set("shadowOffsetY", 0)
		return
	}
	c.FillRoundedRect(x, y, w, h, r, body)
}
