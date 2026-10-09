package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("hstack", renderer.Widget{
		Measure: measureHStack,
		Layout:  layoutHStack,
		Paint:   paintHStack,
	})
}

// HStack 水平堆叠。对应 h('hstack', props?, children...)。
func HStack(parts ...any) *runtime.VNode {
	return runtime.H("hstack", parts...)
}

func measureHStack(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	pad := n.Style.Padding
	gap := n.Style.Gap

	// 第一遍：按「不受约束的自然宽度」量每一列。
	//
	// 宽度**不能**改成传可用宽度：贪婪的子节点（vstack 只要 maxW > 0 就报满 maxW）
	// 会各自都报全宽，两列并排时第二列直接被推到行外——实测 HStack(Width 660) 里
	// 放两个无宽度无 flex 的 VStack，两列各报 660，第二列右缘跑到 1364（行右缘 696）。
	var (
		kept    []*runtime.VNode
		natural []float32 // 自然宽度
		flex    []float32 // flex 权重
		heights []float32 // 按自然宽度量出来的高度
	)
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if runtime.IsAbsolute(c) {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, 0, maxH-2*pad, style, th)
		m := 2 * c.Style.Margin
		kept = append(kept, c)
		natural = append(natural, cw+m)
		flex = append(flex, c.Style.Flex)
		heights = append(heights, ch+m)
	}

	// 自己最终会拿到的宽度。这里先算出来，一是它就是下面 return 出去的那个 w
	// （applySize 随后只会做同样的收尾），二是**它同时是分配的依据**：
	// Style.Width 是在本控件的 Measure 之后才由 applySize 覆盖的，用父级给的
	// maxW 当依据会偏大，弹性列就被按一个偏大的宽度量了高。
	w := 2 * pad
	for i, cw := range natural {
		if i > 0 {
			w += gap
		}
		w += cw
	}
	if n.Style.Flex > 0 && maxW > 0 {
		w = maxW
	}
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}

	// 第二遍：把宽度按同一份规则分下去，弹性列按**分到的**宽度重新量高度。
	//
	// 高度不能跟着自然宽度走：后端把 maxW <= 0 当成不限宽（Windows 的 MeasureText
	// 里 maxW <= 0 → 1e6），换行文本只会报一行高，布局时才按真实宽度画出多行，
	// 多出来的部分被祖先容器的 PushClip 裁掉。非弹性列拿到的就是自己的自然宽度，
	// 高度已经是对的，不用重量。
	innerW := w - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	widths, _ := renderer.Distribute(natural, flex, gap, innerW)
	var contentH float32
	for i, c := range kept {
		ch := heights[i]
		if flex[i] > 0 {
			if _, h2 := renderer.MeasureNode(ctx, c, widths[i], maxH-2*pad, style, th); h2 > ch {
				ch = h2
			}
		}
		if ch > contentH {
			contentH = ch
		}
	}

	h := contentH + 2*pad
	if n.Style.Flex > 0 && maxH > 0 && h < maxH {
		h = maxH
	}
	return w, h
}

func layoutHStack(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	pad := n.Style.Padding
	gap := n.Style.Gap
	align := n.Style.Align
	justify := n.Style.Justify
	innerW := w - 2*pad
	innerH := h - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	// 同 measureHStack：先按自然宽度量每一列，再用**同一份**分配规则分宽度。
	// 两处共用 renderer.Distribute，测量与布局才会对「每一列多宽」得出同一个答案。
	var (
		kept    []*runtime.VNode
		natural []float32
		flex    []float32
		heights []float32
	)
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if runtime.IsAbsolute(c) {
			continue
		}
		iw, ih := renderer.MeasureNode(ctx, c, 0, innerH, style, th)
		m := 2 * c.Style.Margin
		kept = append(kept, c)
		natural = append(natural, iw+m)
		flex = append(flex, c.Style.Flex)
		heights = append(heights, ih+m)
	}
	widths, leftover := renderer.Distribute(natural, flex, gap, innerW)

	cx := x + pad
	if leftover > 0 {
		switch justify {
		case runtime.Center:
			cx += leftover / 2
		case runtime.End:
			cx += leftover
		}
	}
	for i, c := range kept {
		cw := widths[i]
		ch := heights[i]
		// 弹性列分到的宽度不等于自己的自然宽度（可能被撑宽、也可能被压窄），
		// 高度得按分到的那一版重量：换行文本被压窄后会多出几行，按自然宽度
		// 量出来的高度装不下。
		if flex[i] > 0 {
			if _, h2 := renderer.MeasureNode(ctx, c, cw, innerH, style, th); h2 > ch {
				ch = h2
			}
		}
		cy := y + pad
		switch align {
		case runtime.Center:
			cy += (innerH - ch) / 2
		case runtime.End:
			cy += innerH - ch
		case runtime.Stretch:
			ch = innerH
		}
		m := c.Style.Margin
		renderer.LayoutNode(ctx, c, cx+m, cy+m, cw-2*m, ch-2*m, style, th)
		cx += cw + gap
	}
}

func paintHStack(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if n.OnClick != nil && renderer.Enabled(n) && renderer.PointerIn(n.X, n.Y, n.W, n.H) && renderer.ColorSet(n.Style.Background) {
		paintHStackHover(ctx, n)
	} else {
		renderer.PaintBackground(ctx, n)
	}
	renderer.PaintChildren(ctx, n, style, th)
}

// paintHStackHover 可点行（历史卡）悬停时略压暗底色，让「整行可点」看得见。
func paintHStackHover(ctx renderer.Context, n *runtime.VNode) {
	c := renderer.ColorFrom(n.Style.Background)
	c.R *= 0.97
	c.G *= 0.975
	c.B *= 0.98
	r := n.Style.Radius
	if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, c)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, c)
	}
	if renderer.ColorSet(n.Style.Border) {
		renderer.StrokeRounded(ctx, n.X, n.Y, n.W, n.H, n.Style.Radius, 1, renderer.ColorFrom(n.Style.Border))
	}
}
