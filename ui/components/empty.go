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
	title := renderer.AttrString(n, "title")
	hint := renderer.AttrString(n, "hint")
	iconKind := renderer.AttrString(n, "icon")
	fg := style.Color
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.Foreground)
	}
	muted, iconFg, badge := emptyPalette(th)
	accent := renderer.ColorFrom(th.Button)
	if accent.A == 0 {
		accent = renderer.RGB(47, 107, 242)
	}
	// Accent 不铺实心主色：徽章是底色上的浅染，图标用主色，虚线也带同一色相。
	// 实心圆和灰框会变成两套颜色。缺省仍是灰度。
	decor := false
	if n.Props.Accent || iconKind == "spinner" {
		badge = blendTint(accent, fill, 0.16, th.Dark)
		iconFg = accent
		border = blendTint(accent, border, 0.42, th.Dark)
		decor = n.Props.Accent
	}
	ctx.FillRoundedRect(n.X+1, n.Y+1, n.W-2, n.H-2, radius, fill)
	if renderer.AttrBool(n, "dashed") {
		drawDashedRoundedRect(ctx, n.X+1, n.Y+1, n.W-2, n.H-2, radius, border)
	} else {
		strokeRoundedRect(ctx, n.X+8, n.Y+8, n.W-16, n.H-16, radius-4, 1, border)
	}
	if iconKind == "spinner" {
		kickSpinner()
	}

	_, childH := measureEmptyChildren(ctx, n, style, th)
	cy := emptyContentTop(n, childH)
	cx := n.X + (n.W-emptyBadge)/2
	if decor {
		paintEmptyHalo(ctx, cx, cy, accent)
		paintEmptyDecor(ctx, cx, cy, accent)
	}
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
		// 优先走通用图标集（应用可 RegisterGlyph 覆盖 "image"），没有再回引擎内置。
		if !HasGlyph(kind) || !PaintGlyph(ctx, kind, x, y, size, fg) {
			paintImageGlyph(ctx, x, y, size, fg)
		}
	}
}

// blendTint 把 c 向底色 tint 混合 alpha 份：徽章底是「主色 15% 染在浅底上」。
// 暗色主题把 tint 换成深蓝灰，避免亮蓝块在暗底上刺眼。
func blendTint(c, tint renderer.Color, alpha float32, dark bool) renderer.Color {
	if dark {
		tint = renderer.RGB(24, 34, 52)
	}
	return renderer.Color{
		R: c.R*alpha + tint.R*(1-alpha),
		G: c.G*alpha + tint.G*(1-alpha),
		B: c.B*alpha + tint.B*(1-alpha),
		A: 1,
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
