package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// PaintTree 把组件渲染函数产出的 VNode 树画到 Context 上。
// 若根节点是自定义 TitleBar，则钉在窗口顶部；其余节点先布局再绘制。
func PaintTree(ctx Context, nodes []*runtime.VNode, th theme.Theme) {
	if ctx == nil {
		return
	}
	resetCullClip()
	syncMeasureOwner(ctx)
	if th.FontSize <= 0 {
		th = theme.Default
	}
	ctx.Clear(ColorFrom(th.Background))

	style := TextStyle{
		FontFamily: th.FontFamily,
		FontSize:   th.FontSize,
		Color:      ColorFrom(th.Foreground),
		Weight:     400,
	}

	y := float32(0)
	content := nodes
	if len(nodes) > 0 && isTitleBarNode(nodes[0]) {
		tb := unwrapTitleBar(nodes[0])
		if tb == nil {
			tb = nodes[0]
		}
		y = paintFrame(ctx, tb, th)
		content = nodes[1:]
	}

	layoutContent(ctx, content, y, style, th)
	paintLaidOut(ctx, content, style, th)
	paintOverlays(ctx, content, style, th)
}

func paintLaidOut(ctx Context, nodes []*runtime.VNode, style TextStyle, th theme.Theme) {
	var abs, sticky []*runtime.VNode
	for _, n := range nodes {
		if runtime.IsAbsolute(n) {
			abs = append(abs, n)
			continue
		}
		// sticky 后画，盖住滚到它下面的行。
		if runtime.IsSticky(n) {
			sticky = append(sticky, n)
			continue
		}
		paintLaidOutNode(ctx, n, style, th)
	}
	for _, n := range sticky {
		paintLaidOutNode(ctx, n, style, th)
	}
	for _, n := range abs {
		paintLaidOutNode(ctx, n, style, th)
	}
}

func paintLaidOutNode(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme) {
	if n == nil || runtime.OverlayTag(n.Tag) {
		return
	}
	if !nodeInCullClip(n) {
		return
	}
	st := applyStyle(style, n.Style)
	switch n.Kind {
	case runtime.KindText:
		paintBackground(ctx, n)
		if n.Text == "" {
			return
		}
		// 文本节点自己写了 Align 才算数：盒子比文字宽时（显式宽度 / 被撑满 /
		// layoutFlow 给的整行宽），文字按它对齐；否则行为与从前完全一致。
		st.Align = TextAlign(n.Style, st.Align)
		pad := n.Style.Padding
		h := n.H - 2*pad
		if h <= 0 {
			h = st.FontSize
		}
		text := n.Text
		if AttrBool(n, "ellipsis") {
			text = ellipsizeText(ctx, text, n.W-2*pad, st)
		}
		ctx.DrawText(text, n.X+pad, n.Y+pad, n.W-2*pad, h, st)

	case runtime.KindElement:
		if widget, ok := Lookup(n.Tag); ok && widget.Paint != nil {
			widget.Paint(ctx, n, st, th)
			return
		}
		paintBackground(ctx, n)
		PaintChildren(ctx, n, st, th)

	case runtime.KindComponent:
		paintBackground(ctx, n)
		PaintChildren(ctx, n, st, th)
	}
}

// ellipsizeText 把单行文本截到 maxW 内，尾部补省略号。
func ellipsizeText(ctx Context, text string, maxW float32, style TextStyle) string {
	if ctx == nil || text == "" || maxW <= 0 {
		return ""
	}
	style.NoWrap = true
	if w, _ := measureTextCached(ctx, text, 0, style); w <= maxW {
		return text
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		w, _ := measureTextCached(ctx, string(runes[:mid])+"…", 0, style)
		if w <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		if w, _ := measureTextCached(ctx, "…", 0, style); w <= maxW {
			return "…"
		}
		return ""
	}
	return string(runes[:lo]) + "…"
}

func paintOverlays(ctx Context, nodes []*runtime.VNode, style TextStyle, th theme.Theme) {
	runtime.Walk(nodes, func(n *runtime.VNode) bool {
		if n == nil || !runtime.OverlayTag(n.Tag) {
			return false
		}
		st := applyStyle(style, n.Style)
		if widget, ok := Lookup(n.Tag); ok && widget.Paint != nil {
			widget.Paint(ctx, n, st, th)
			return false
		}
		paintBackground(ctx, n)
		paintLaidOut(ctx, n.Children, st, th)
		return false
	})
}
