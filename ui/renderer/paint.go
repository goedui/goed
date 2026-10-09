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
	var abs []*runtime.VNode
	for _, n := range nodes {
		if runtime.IsAbsolute(n) {
			abs = append(abs, n)
			continue
		}
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
		ctx.DrawText(n.Text, n.X+pad, n.Y+pad, n.W-2*pad, h, st)

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
