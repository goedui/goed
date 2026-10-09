package renderer

import (
	"math"

	"github.com/goedui/goed/ui/runtime"
)

func applyStyle(base TextStyle, s runtime.Style) TextStyle {
	if s.FontFamily != "" {
		base.FontFamily = s.FontFamily
	}
	if s.FontSize > 0 {
		base.FontSize = s.FontSize
	}
	if s.Weight > 0 {
		base.Weight = s.Weight
	}
	if colorSet(s.Color) {
		base.Color = colorFrom(s.Color)
	}
	return base
}

func colorFrom(c runtime.Color) Color {
	return Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

func colorSet(c runtime.Color) bool {
	return c.A > 0
}

// TextAlign 把 Style.Align 映射成文本绘制对齐。
//
// 只认节点**自己**写了的 Align，没写就返回 fallback（沿用继承来的值），对现有界面
// 是恒等变换。不能放进 applyStyle 顺着继承传染：容器的 Align 是「子节点往哪边摆」，
// 不是「文字在盒子里往哪边画」，混在一起的话 `VStack{Align: Center}` / Tabs（自己
// 写死了 Center）底下每个撑满的 Text 子节点都会莫名其妙居中。
//
// "stretch" 没有文本对齐语义（是「拉满」不是「靠边」），一律不动。
func TextAlign(s runtime.Style, fallback Align) Align {
	switch s.Align {
	case runtime.Start:
		return AlignStart
	case runtime.Center:
		return AlignCenter
	case runtime.End:
		return AlignEnd
	}
	return fallback
}

func paintBackground(ctx Context, n *runtime.VNode) {
	if ctx == nil || n == nil {
		return
	}
	// 阴影必须先于本体：调用方（可选接口 / 近似叠层）会把本体一并画掉。
	// 有 Gradient 或 Shadow 时走效果路径，路径内部保证本体被填充一次。
	if n.Props.Shadow != nil || n.Props.Gradient != nil {
		paintEffectBackground(ctx, n)
	} else if colorSet(n.Style.Background) {
		c := colorFrom(n.Style.Background)
		if n.Style.Radius > 0 {
			ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, n.Style.Radius, c)
		} else {
			ctx.FillRect(n.X, n.Y, n.W, n.H, c)
		}
	}
	if colorSet(n.Style.Border) {
		StrokeRounded(ctx, n.X, n.Y, n.W, n.H, n.Style.Radius, 1, colorFrom(n.Style.Border))
	}
}

// StrokeRounded 画圆角描边。线宽沿中心线均匀，四角和直边同色同粗。
// 以前四角用小方块拼接，抗锯齿叠色后比直边深一档。
func StrokeRounded(ctx Context, x, y, w, h, r, s float32, c Color) {
	if ctx == nil || w <= 0 || h <= 0 || s <= 0 {
		return
	}
	if r <= 0 {
		ctx.DrawRect(x, y, w, h, c, s)
		return
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	if rs, ok := ctx.(RoundedRectStroker); ok && rs != nil {
		rs.DrawRoundedRect(x, y, w, h, r, c, s)
		return
	}
	if DrawRoundStroke(ctx, roundedStroke(x, y, w, h, r, s), s, c, true) {
		return
	}
	strokeRoundedFallback(ctx, x, y, w, h, r, s, c)
}

// roundedStroke 是描边中心线：向内收半个线宽，圆角半径同步减去。
func roundedStroke(x, y, w, h, r, s float32) []StrokePoint {
	inset := s / 2
	left, top := x+inset, y+inset
	right, bottom := x+w-inset, y+h-inset
	cr := r - inset
	if cr < 0 {
		cr = 0
	}
	if cr > (right-left)/2 {
		cr = (right - left) / 2
	}
	if cr > (bottom-top)/2 {
		cr = (bottom - top) / 2
	}
	pts := []StrokePoint{{X: left + cr, Y: top}}
	pts = append(pts, StrokePoint{X: right - cr, Y: top})
	pts = append(pts, quarterArc(right-cr, top+cr, cr, -math.Pi/2, 0)...)
	pts = append(pts, StrokePoint{X: right, Y: bottom - cr})
	pts = append(pts, quarterArc(right-cr, bottom-cr, cr, 0, math.Pi/2)...)
	pts = append(pts, StrokePoint{X: left + cr, Y: bottom})
	pts = append(pts, quarterArc(left+cr, bottom-cr, cr, math.Pi/2, math.Pi)...)
	pts = append(pts, StrokePoint{X: left, Y: top + cr})
	pts = append(pts, quarterArc(left+cr, top+cr, cr, math.Pi, math.Pi*3/2)...)
	return pts
}

// quarterArc 用一段三次贝塞尔逼近 90° 圆弧，第一点由上一段提供。
func quarterArc(cx, cy, r, a0, a1 float32) []StrokePoint {
	k := float32(math.Tan(float64(a1-a0)/2)) * 4 / 3
	cos0, sin0 := float32(math.Cos(float64(a0))), float32(math.Sin(float64(a0)))
	cos1, sin1 := float32(math.Cos(float64(a1))), float32(math.Sin(float64(a1)))
	return []StrokePoint{{
		Curve: true,
		C1X:   cx + r*(cos0-k*sin0),
		C1Y:   cy + r*(sin0+k*cos0),
		C2X:   cx + r*(cos1+k*sin1),
		C2Y:   cy + r*(sin1-k*cos1),
		X:     cx + r*cos1,
		Y:     cy + r*sin1,
	}}
}

func strokeRoundedFallback(ctx Context, x, y, w, h, r, s float32, c Color) {
	ctx.FillRect(x+r, y, w-2*r, s, c)
	ctx.FillRect(x+r, y+h-s, w-2*r, s, c)
	ctx.FillRect(x, y+r, s, h-2*r, c)
	ctx.FillRect(x+w-s, y+r, s, h-2*r, c)
	strokeArcDots(ctx, x+r, y+r, r-s/2, s, math.Pi, math.Pi*1.5, c)
	strokeArcDots(ctx, x+w-r, y+r, r-s/2, s, -math.Pi/2, 0, c)
	strokeArcDots(ctx, x+w-r, y+h-r, r-s/2, s, 0, math.Pi/2, c)
	strokeArcDots(ctx, x+r, y+h-r, r-s/2, s, math.Pi/2, math.Pi, c)
}

func strokeArcDots(ctx Context, cx, cy, r, s, a0, a1 float32, c Color) {
	if r < 0 {
		r = 0
	}
	steps := int(r)
	if steps < 5 {
		steps = 5
	}
	for i := 1; i < steps; i++ {
		t := float32(i) / float32(steps)
		a := a0 + (a1-a0)*t
		px := cx + float32(math.Cos(float64(a)))*r
		py := cy + float32(math.Sin(float64(a)))*r
		ctx.FillRoundedRect(px-s/2, py-s/2, s, s, s/2, c)
	}
}

// paintEffectBackground 画「投影 + 渐变/纯色本体」的组合背景。
// 没有 Shadow 只有 Gradient 时本体由 FillGradientRoundedRect 画（降级为 from 纯色）；
// 有 Shadow 时 FillShadowRoundedRect 负责画阴影 + 本体，此时 Gradient 作为本体的
// 近似不叠加 —— 投影按钮用渐变本体是主场景，直接走下面的优先级判断。
func paintEffectBackground(ctx Context, n *runtime.VNode) {
	bgSet := colorSet(n.Style.Background)
	var c Color
	if bgSet {
		c = colorFrom(n.Style.Background)
	} else {
		c = RGB(255, 255, 255)
	}
	grad := n.Props.Gradient
	if grad != nil {
		from := c
		to := c
		if colorSet(grad.From) {
			from = colorFrom(grad.From)
		}
		if colorSet(grad.To) {
			to = colorFrom(grad.To)
		} else {
			to = from
		}
		if n.Props.Shadow == nil {
			FillGradientRoundedRect(ctx, n.X, n.Y, n.W, n.H, n.Style.Radius, from, to, grad.Vertical)
			return
		}
		// 有阴影：先画阴影（本体纯色），再在本体上叠一层同形状渐变，两者用同一
		// 组端点色。近似的接缝在圆角内不可见 —— 渐变两端与纯色端点一致。
	}
	if n.Props.Shadow != nil {
		blur := n.Props.Shadow.Blur
		if blur <= 0 {
			blur = 8
		}
		offY := n.Props.Shadow.OffsetY
		shadowC := RGB(15, 23, 42)
		if colorSet(n.Props.Shadow.Color) {
			shadowC = colorFrom(n.Props.Shadow.Color)
		}
		FillShadowRoundedRect(ctx, n.X, n.Y, n.W, n.H, n.Style.Radius, blur, offY, c, shadowC)
		if grad != nil && n.Props.Shadow != nil {
			from, to := c, c
			if colorSet(grad.From) {
				from = colorFrom(grad.From)
			}
			if colorSet(grad.To) {
				to = colorFrom(grad.To)
			}
			FillGradientRoundedRect(ctx, n.X, n.Y, n.W, n.H, n.Style.Radius, from, to, grad.Vertical)
		}
		return
	}
}
