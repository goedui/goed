package components

import (
	"math"

	"github.com/goedui/goed/ui/renderer"
)

// paintEmptyHalo 在徽章背后铺一圈淡色光晕，设计图里徽章不是孤零零的圆。
func paintEmptyHalo(ctx renderer.Context, bx, by float32, accent renderer.Color) {
	for i, grow := range []float32{34, 20, 10} {
		c := accent
		c.A = 0.035 + float32(i)*0.018
		ctx.FillRoundedRect(bx-grow, by-grow, emptyBadge+grow*2, emptyBadge+grow*2, (emptyBadge+grow*2)/2, c)
	}
}

// paintFourStar 用十字渐收的方块拼一颗四角星。空态装饰不走图标网格。
func paintFourStar(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	c := size / 2
	core := size * 0.28
	ctx.FillRoundedRect(x+c-core/2, y+c-core/2, core, core, core*0.35, fg)
	for i := float32(0); i < 4; i++ {
		t := i / 4
		wi := size * 0.16 * (1 - t*0.65)
		seg := size * 0.34 / 4
		ctx.FillRect(x+c-wi/2, y+t*size*0.34, wi, seg+0.6, fg)
		ctx.FillRect(x+c-wi/2, y+size-(t+1)*seg, wi, seg+0.6, fg)
		ctx.FillRect(x+t*size*0.34, y+c-wi/2, seg+0.6, wi, fg)
		ctx.FillRect(x+size-(t+1)*seg, y+c-wi/2, seg+0.6, wi, fg)
	}
}

// paintEmptyDecor 在徽章四角画星点和短斜线，只装饰、不参与布局。
func paintEmptyDecor(ctx renderer.Context, bx, by float32, accent renderer.Color) {
	soft := accent
	soft.A = 0.42
	faint := accent
	faint.A = 0.26
	paintFourStar(ctx, bx-emptyBadge*0.46, by-emptyBadge*0.18, emptyBadge*0.28, soft)
	paintFourStar(ctx, bx+emptyBadge*1.12, by+emptyBadge*0.48, emptyBadge*0.18, faint)
	ctx.DrawLine(bx+emptyBadge*1.08, by-emptyBadge*0.08, bx+emptyBadge*1.34, by+emptyBadge*0.16, faint, 1.6)
	ctx.DrawLine(bx-emptyBadge*0.28, by+emptyBadge*0.92, bx-emptyBadge*0.04, by+emptyBadge*1.14, soft, 1.6)
}

func strokeRoundedRect(ctx renderer.Context, x, y, w, h, r, s float32, c renderer.Color) {
	if ctx == nil || w <= 0 || h <= 0 || s <= 0 {
		return
	}
	if r < s {
		r = s
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	ctx.FillRect(x+r, y, w-2*r, s, c)
	ctx.FillRect(x+r, y+h-s, w-2*r, s, c)
	ctx.FillRect(x, y+r, s, h-2*r, c)
	ctx.FillRect(x+w-s, y+r, s, h-2*r, c)
	paintCornerArc(ctx, x+r, y+r, r, s, math.Pi, math.Pi*1.5, c)
	paintCornerArc(ctx, x+w-r, y+r, r, s, -math.Pi/2, 0, c)
	paintCornerArc(ctx, x+w-r, y+h-r, r, s, 0, math.Pi/2, c)
	paintCornerArc(ctx, x+r, y+h-r, r, s, math.Pi/2, math.Pi, c)
}

func paintCornerArc(ctx renderer.Context, cx, cy, r, s, a0, a1 float32, c renderer.Color) {
	steps := int(r)
	if steps < 5 {
		steps = 5
	}
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		a := a0 + (a1-a0)*t
		px := cx + float32(math.Cos(float64(a)))*r
		py := cy + float32(math.Sin(float64(a)))*r
		ctx.FillRoundedRect(px-s/2, py-s/2, s, s, s/2, c)
	}
}

// drawDashedRoundedRect 沿圆角矩形周长画 1px 虚线。
func drawDashedRoundedRect(ctx renderer.Context, x, y, w, h, r float32, c renderer.Color) {
	if w <= 2 || h <= 2 {
		ctx.DrawRect(x, y, w, h, c, 1)
		return
	}
	if r < 1 {
		drawDashedRect(ctx, x, y, w, h, c)
		return
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	pen := dashPen{ctx: ctx, c: c, on: emptyDashOn, off: emptyDashOff}
	pen.line(x+r, y, x+w-r, y)
	pen.arc(x+w-r, y+r, r, -math.Pi/2, 0)
	pen.line(x+w-1, y+r, x+w-1, y+h-r)
	pen.arc(x+w-r, y+h-r, r, 0, math.Pi/2)
	pen.line(x+w-r, y+h-1, x+r, y+h-1)
	pen.arc(x+r, y+h-r, r, math.Pi/2, math.Pi)
	pen.line(x, y+h-r, x, y+r)
	pen.arc(x+r, y+r, r, math.Pi, math.Pi*1.5)
}

// drawDashedRect 直角虚线，给半径为 0 的回退路径用。
func drawDashedRect(ctx renderer.Context, x, y, w, h float32, c renderer.Color) {
	pen := dashPen{ctx: ctx, c: c, on: emptyDashOn, off: emptyDashOff}
	pen.line(x, y, x+w, y)
	pen.line(x+w-1, y, x+w-1, y+h)
	pen.line(x+w, y+h-1, x, y+h-1)
	pen.line(x, y+h, x, y)
}

type dashPen struct {
	ctx     renderer.Context
	c       renderer.Color
	on, off float32
	phase   float32
}

func (p *dashPen) plot(x, y float32) {
	if p.phase < p.on {
		p.ctx.FillRect(x, y, 1, 1, p.c)
	}
	p.phase++
	if p.phase >= p.on+p.off {
		p.phase = 0
	}
}

func (p *dashPen) line(x0, y0, x1, y1 float32) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
	if steps < 1 {
		p.plot(x0, y0)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		p.plot(x0+dx*t, y0+dy*t)
	}
}

func (p *dashPen) arc(cx, cy, r, a0, a1 float32) {
	length := math.Abs(float64(a1-a0)) * float64(r)
	steps := int(length)
	if steps < 4 {
		steps = 4
	}
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		a := a0 + (a1-a0)*t
		p.plot(cx+r*float32(math.Cos(float64(a))), cy+r*float32(math.Sin(float64(a))))
	}
}
