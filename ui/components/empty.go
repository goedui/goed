package components

import (
	"math"
	"sync"
	"time"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("empty", renderer.Widget{
		Measure: measureEmpty,
		Layout:  layoutEmpty,
		Paint:   paintEmpty,
	})
}

// Empty 空状态占位：圆角底 + 虚线框 + 居中图标/标题/提示，可挂 CTA 子节点。
//
// Attrs:
//
//	title  主文案
//	hint   次文案
//	dashed 画圆角虚线框
//	icon   image（默认）/ history / spinner
func Empty(parts ...any) *runtime.VNode {
	return runtime.H("empty", parts...)
}

const (
	emptyBadge    = float32(72)
	emptyTitleH   = float32(22)
	emptyHintH    = float32(20)
	emptyIconGap  = float32(14)
	emptyTextGap  = float32(6)
	emptyChildGap = float32(16)
	emptyDashOn   = float32(5)
	emptyDashOff  = float32(4)
	emptyRadius   = float32(16)
)

func measureEmpty(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	// 测量阶段父级 HStack 会传 maxW=0。不能回退到窗口全宽，否则 empty
	// 会按 1180 布局，内容看起来偏到右栏右侧。宽度交给 layout 按父级拉伸。
	w := maxW
	if w <= 0 {
		w = 1
	}
	h := maxH
	if h <= 0 {
		h = 180
	}
	_ = ctx
	_ = n
	_ = style
	_ = th
	return w, h
}

func layoutEmpty(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	kids, childH := measureEmptyChildren(ctx, n, style, th)
	cy := emptyContentTop(n, childH)
	cy += emptyTextBlockHeight(n)
	if childH > 0 {
		cy += emptyChildGap
	}
	gap := n.Style.Gap
	if gap <= 0 {
		gap = 8
	}
	for i, c := range kids {
		if i > 0 {
			cy += gap
		}
		cx := x + (w-c.w)/2
		renderer.LayoutNode(ctx, c.n, cx, cy, c.w, c.h, style, th)
		cy += c.h
	}
}

type emptyChild struct {
	n    *runtime.VNode
	w, h float32
}

func measureEmptyChildren(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) ([]emptyChild, float32) {
	if n == nil {
		return nil, 0
	}
	gap := n.Style.Gap
	if gap <= 0 {
		gap = 8
	}
	innerW := n.W - 32
	if innerW < 0 {
		innerW = n.W
	}
	out := make([]emptyChild, 0, len(n.Children))
	var h float32
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, innerW, 0, style, th)
		if len(out) > 0 {
			h += gap
		}
		out = append(out, emptyChild{n: c, w: cw, h: ch})
		h += ch
	}
	return out, h
}

func emptyTextBlockHeight(n *runtime.VNode) float32 {
	h := emptyBadge
	if renderer.AttrString(n, "title") != "" {
		h += emptyIconGap + emptyTitleH
	}
	if renderer.AttrString(n, "hint") != "" {
		h += emptyTextGap + emptyHintH
	}
	return h
}

func emptyContentTop(n *runtime.VNode, childH float32) float32 {
	block := emptyTextBlockHeight(n)
	if childH > 0 {
		block += emptyChildGap + childH
	}
	return n.Y + (n.H-block)/2
}

func paintEmpty(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil || n.W <= 0 || n.H <= 0 {
		return
	}
	border := renderer.ColorFrom(th.Separator)
	if renderer.ColorSet(n.Style.Border) {
		border = renderer.ColorFrom(n.Style.Border)
	}
	radius := n.Style.Radius
	if radius <= 0 {
		radius = emptyRadius
	}
	fill := renderer.RGB(250, 251, 252)
	if th.Dark {
		fill = renderer.RGB(32, 32, 36)
	}
	if renderer.ColorSet(n.Style.Background) {
		fill = renderer.ColorFrom(n.Style.Background)
	}
	ctx.FillRoundedRect(n.X+1, n.Y+1, n.W-2, n.H-2, radius, fill)
	if renderer.AttrBool(n, "dashed") {
		drawDashedRoundedRect(ctx, n.X+1, n.Y+1, n.W-2, n.H-2, radius, border)
	} else {
		strokeRoundedRect(ctx, n.X+8, n.Y+8, n.W-16, n.H-16, radius-4, 1, border)
	}

	title := renderer.AttrString(n, "title")
	hint := renderer.AttrString(n, "hint")
	iconKind := renderer.AttrString(n, "icon")
	fg := style.Color
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	muted, iconFg, badge := emptyPalette(th)
	if iconKind == "spinner" {
		accent := renderer.ColorFrom(th.Button)
		if accent.A == 0 {
			accent = renderer.RGB(59, 166, 235)
		}
		ctx.FillRoundedRect(n.X+radius*0.35, n.Y+1, n.W-radius*0.7, 3, 1.5, accent)
		kickSpinner()
	}

	_, childH := measureEmptyChildren(ctx, n, style, th)
	cy := emptyContentTop(n, childH)
	cx := n.X + (n.W-emptyBadge)/2
	ctx.FillRoundedRect(cx, cy, emptyBadge, emptyBadge, emptyBadge/2, badge)
	iconSize := float32(28)
	paintEmptyGlyph(ctx, iconKind, cx+(emptyBadge-iconSize)/2, cy+(emptyBadge-iconSize)/2, iconSize, iconFg, th)
	cy += emptyBadge + emptyIconGap
	if title != "" {
		st := style
		st.Align = renderer.AlignCenter
		st.VAlign = renderer.AlignCenter
		st.Weight = 600
		st.FontSize = 15
		st.Color = fg
		ctx.DrawText(title, n.X+16, cy, n.W-32, emptyTitleH, st)
		cy += emptyTitleH + emptyTextGap
	}
	if hint != "" {
		st := style
		st.Align = renderer.AlignCenter
		st.VAlign = renderer.AlignCenter
		st.FontSize = 13
		st.Color = muted
		ctx.DrawText(hint, n.X+24, cy, n.W-48, emptyHintH, st)
	}
	renderer.PaintChildren(ctx, n, style, th)
}

func emptyPalette(th theme.Theme) (muted, icon, badge renderer.Color) {
	muted = renderer.RGB(122, 130, 142)
	icon = renderer.RGB(168, 174, 184)
	badge = renderer.RGB(238, 240, 244)
	if th.Dark {
		muted = renderer.RGB(160, 166, 176)
		icon = renderer.RGB(140, 146, 156)
		badge = renderer.RGB(44, 46, 52)
	}
	return muted, icon, badge
}

func paintEmptyGlyph(ctx renderer.Context, kind string, x, y, size float32, fg renderer.Color, th theme.Theme) {
	switch kind {
	case "history":
		paintHistoryGlyph(ctx, x, y, size, fg)
	case "spinner":
		accent := renderer.ColorFrom(th.Button)
		if accent.A == 0 {
			accent = fg
		}
		paintSpinnerGlyph(ctx, x, y, size, fg, accent, spinnerPhase())
	default:
		paintImageGlyph(ctx, x, y, size, fg)
	}
}

func paintImageGlyph(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	pad := size * 0.06
	w := size - 2*pad
	h := size * 0.78
	ix := x + pad
	iy := y + (size-h)/2
	r := size * 0.12
	strokeRoundedRect(ctx, ix, iy, w, h, r, 1.6, fg)
	sun := size * 0.16
	ctx.FillRoundedRect(ix+w*0.18, iy+h*0.16, sun, sun, sun/2, fg)
	ctx.FillRoundedRect(ix+w*0.12, iy+h*0.54, w*0.42, h*0.28, r*0.7, fg)
	ctx.FillRoundedRect(ix+w*0.40, iy+h*0.42, w*0.38, h*0.40, r*0.7, fg)
}

func paintHistoryGlyph(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	cx := x + size/2
	cy := y + size/2
	thick := size * 0.08
	if thick < 1.4 {
		thick = 1.4
	}
	tick := size * 0.10
	ctx.FillRoundedRect(cx-thick/2, y+size*0.10, thick, tick, thick/2, fg)
	ctx.FillRoundedRect(cx-thick/2, y+size-size*0.10-tick, thick, tick, thick/2, fg)
	ctx.FillRoundedRect(x+size*0.10, cy-thick/2, tick, thick, thick/2, fg)
	ctx.FillRoundedRect(x+size-size*0.10-tick, cy-thick/2, tick, thick, thick/2, fg)
	ctx.FillRoundedRect(cx-thick/2, cy-size*0.22, thick, size*0.22, thick/2, fg)
	ctx.FillRoundedRect(cx, cy-thick/2, size*0.16, thick, thick/2, fg)
	dot := thick * 1.6
	ctx.FillRoundedRect(cx-dot/2, cy-dot/2, dot, dot, dot/2, fg)
}

func paintSpinnerGlyph(ctx renderer.Context, x, y, size float32, fg, accent renderer.Color, phase float32) {
	cx := x + size/2
	cy := y + size/2
	ring := size * 0.34
	dot := size * 0.11
	if dot < 2 {
		dot = 2
	}
	const n = 8
	for i := 0; i < n; i++ {
		a := (float32(i)/n + phase) * 2 * math.Pi
		t := float32(i) / float32(n-1)
		c := fg
		c.A = 0.22 + 0.78*t
		if t > 0.7 {
			c = accent
			c.A = 0.55 + 0.45*t
		}
		px := cx + float32(math.Cos(float64(a)))*ring
		py := cy + float32(math.Sin(float64(a)))*ring
		ctx.FillRoundedRect(px-dot/2, py-dot/2, dot, dot, dot/2, c)
	}
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

func spinnerPhase() float32 {
	return float32(time.Now().UnixMilli()%1200) / 1200
}

var (
	spinnerMu   sync.Mutex
	spinnerOn   bool
	spinnerUsed time.Time
)

func kickSpinner() {
	spinnerMu.Lock()
	spinnerUsed = time.Now()
	start := !spinnerOn
	spinnerOn = true
	spinnerMu.Unlock()
	if start {
		go spinnerLoop()
	}
}

func spinnerLoop() {
	for {
		time.Sleep(50 * time.Millisecond)
		spinnerMu.Lock()
		if time.Since(spinnerUsed) > 200*time.Millisecond {
			spinnerOn = false
			spinnerMu.Unlock()
			return
		}
		spinnerMu.Unlock()
		// 同上：spinnerLoop 是后台 goroutine，走带锁入口。
		platform.RequestFrame()
	}
}
