package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

var layoutRoot []*runtime.VNode

func layoutContent(ctx Context, nodes []*runtime.VNode, y float32, style TextStyle, th theme.Theme) {
	if ctx == nil {
		return
	}
	layoutRoot = nodes
	w := ctx.Width()
	h := ctx.Height() - y
	if h < 0 {
		h = 0
	}
	// 单个根（Flex 或组件）铺满标题栏以下的客户区。
	// 否则会走「主题 Padding + 自然高度」分支：layer 在 maxH=0 时按整窗
	// 高度度量，再从标题栏下方往下排，状态栏会被顶出窗口。
	if len(nodes) == 1 && fills(nodes[0]) && !runtime.IsAbsolute(nodes[0]) {
		runtime.SetParent(nodes[0], nil)
		layoutNode(ctx, nodes[0], 0, y, w, h, style, th)
		return
	}
	pad := th.Padding
	maxW := w - pad*2
	if maxW < 0 {
		maxW = 0
	}
	placeFlow(ctx, nil, nodes, pad, y+pad, maxW, h, style, th)
	placeAbsolutes(ctx, nil, nodes, 0, y, w, h, style, th)
}

func fills(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	if n.Style.Flex > 0 {
		return true
	}
	// CreateApp 包一层组件时，Flex 在内层根上；外层仍应铺满客户区。
	return n.Kind == runtime.KindComponent
}

func measureNode(ctx Context, n *runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (float32, float32) {
	if n == nil || (runtime.IsAbsolute(n) && !measureIgnoreAbsolute) {
		return 0, 0
	}
	if w, h, ok := n.LookupMeasure(maxW, maxH, style.FontFamily, style.FontSize, style.Weight); ok {
		return w, h
	}
	st := applyStyle(style, n.Style)
	var w, h float32
	switch n.Kind {
	case runtime.KindText:
		w, h = measureText(ctx, n, maxW, st)
	case runtime.KindComponent:
		expandComponent(n)
		w, h = measureFlow(ctx, n.Children, InnerWidth(n, maxW), 0, st, th)
		w, h = addPadding(n, w, h)
	case runtime.KindElement:
		if widget, ok := Lookup(n.Tag); ok && widget.Measure != nil {
			w, h = widget.Measure(ctx, n, maxW, maxH, st, th)
		} else {
			w, h = measureFlow(ctx, n.Children, InnerWidth(n, maxW), 0, st, th)
			w, h = addPadding(n, w, h)
		}
	}
	w, h = applySize(n, w, h, maxW, maxH)
	n.StoreMeasure(maxW, maxH, style.FontFamily, style.FontSize, style.Weight, w, h)
	return w, h
}

func measureText(ctx Context, n *runtime.VNode, maxW float32, style TextStyle) (float32, float32) {
	if n.Text == "" {
		return 0, 0
	}
	pad := n.Style.Padding
	limit := maxW
	if limit > 2*pad {
		limit -= 2 * pad
	}
	// 走跨帧测量缓存：重建帧里绝大部分文本跟上帧一样，命中就不必再问后端。
	w, h := measureTextCached(ctx, n.Text, limit, style)
	return w + 2*pad, h + 2*pad
}

func measureFlow(ctx Context, nodes []*runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (float32, float32) {
	var w, h float32
	for _, c := range nodes {
		if c == nil {
			continue
		}
		if runtime.IsAbsolute(c) {
			continue
		}
		cw, ch := measureNode(ctx, c, maxW, 0, style, th)
		cw += 2 * c.Style.Margin
		ch += 2 * c.Style.Margin
		if cw > w {
			w = cw
		}
		h += ch
	}
	if maxW > 0 {
		w = maxW
	}
	_ = maxH
	return w, h
}

// Distribute 把可用宽度 innerW 按 flex 权重分给一组列，返回每列的最终宽度，以及
// 「没有弹性列时剩下的空白」（justify 用）。natural / flex 长度须一致。
//
// 窄了按权重分给弹性列、宽了只由弹性列承担缺口（每列保底 1），没有弹性列时既不
// 分配也不强缩（交给溢出裁剪兜底）。
//
// 测量与布局必须共用这一套分配规则：测量靠它决定「按多宽量高」（换行文本的高度
// 取决于给了多宽），布局靠它决定「摆在哪、摆多宽」。两边各算各的就会量一行画两行。
func Distribute(natural, flex []float32, gap, innerW float32) ([]float32, float32) {
	n := len(natural)
	if n == 0 {
		return nil, innerW
	}
	var contentW, flexSum float32
	for i := 0; i < n; i++ {
		if i > 0 {
			contentW += gap
		}
		contentW += natural[i]
		if i < len(flex) {
			flexSum += flex[i]
		}
	}
	extra := innerW - contentW
	shrink := float32(0)
	if extra < 0 {
		shrink, extra = -extra, 0
	}
	widths := make([]float32, n)
	for i := 0; i < n; i++ {
		cw := natural[i]
		f := float32(0)
		if i < len(flex) {
			f = flex[i]
		}
		if flexSum > 0 && f > 0 {
			if extra > 0 {
				cw += extra * (f / flexSum)
			}
			if shrink > 0 {
				cw -= shrink * (f / flexSum)
				if cw < 1 {
					cw = 1
				}
			}
		}
		widths[i] = cw
	}
	if flexSum > 0 {
		extra = 0 // 弹性列把剩余宽度吃掉了，justify 没得分
	}
	return widths, extra
}

// InnerWidth 返回节点「最终会拿到」的内容宽度。
//
// 为什么要单独一个函数：Style.Width 是控件自己的 Measure 跑完之后，由 applySize
// 覆盖上去的（见 measureNode 的收尾）。所以控件在度量子节点时不能直接拿父级给的
// maxW —— 那个值可能比自己最终的宽度大得多，子节点会按一个偏大的宽度量高：
// 换行文本量出偏少的行数，布局时才按真实宽度画出更多行，多出来的部分溢出。
func InnerWidth(n *runtime.VNode, maxW float32) float32 {
	w := maxW
	if n != nil && n.Style.Width > 0 {
		w = n.Style.Width
	}
	var pad float32
	if n != nil {
		pad = 2 * n.Style.Padding
	}
	if w < pad {
		return 0
	}
	return w - pad
}

func addPadding(n *runtime.VNode, w, h float32) (float32, float32) {
	p := n.Style.Padding
	return w + 2*p, h + 2*p
}

func applySize(n *runtime.VNode, w, h, maxW, maxH float32) (float32, float32) {
	if n.Style.Width > 0 {
		w = n.Style.Width
	}
	if n.Style.Height > 0 {
		h = n.Style.Height
	}
	if n.Style.MinWidth > 0 && w < n.Style.MinWidth {
		w = n.Style.MinWidth
	}
	if n.Style.MinHeight > 0 && h < n.Style.MinHeight {
		h = n.Style.MinHeight
	}
	if n.Style.MaxWidth > 0 && w > n.Style.MaxWidth {
		w = n.Style.MaxWidth
	}
	if n.Style.MaxHeight > 0 && h > n.Style.MaxHeight {
		h = n.Style.MaxHeight
	}
	// 父级给的 max 是可用空间，不是 CSS 的 max-width。
	// overflow:visible（默认）允许内容画出 padding box，不能在这里夹掉。
	// hidden / auto / scroll 的视口就是这个盒子，超出部分裁或滚，尺寸停在视口。
	// 绝对定位的 max 是包含块扣掉 inset 后的剩余，不是父级流的可用宽，不能夹。
	if maxW > 0 && w > maxW && !runtime.OverflowVisibleAxis(n, true) && !runtime.IsAbsolute(n) {
		w = maxW
	}
	if maxH > 0 && h > maxH && !runtime.OverflowVisibleAxis(n, false) && !runtime.IsAbsolute(n) {
		h = maxH
	}
	return w, h
}

func layoutNode(ctx Context, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	if n == nil {
		return
	}
	n.X, n.Y, n.W, n.H = x, y, w, h
	st := applyStyle(style, n.Style)
	switch n.Kind {
	case runtime.KindText:
		return
	case runtime.KindComponent:
		expandComponent(n)
		layoutPadded(ctx, n, st, th)
	case runtime.KindElement:
		if widget, ok := Lookup(n.Tag); ok && widget.Layout != nil {
			widget.Layout(ctx, n, x, y, w, h, st, th)
		} else {
			layoutPadded(ctx, n, st, th)
		}
	}
	// scroll 控件自己摆偏移。其余盒子在 auto/scroll 时把子树挪进视口。
	// 必须在控件 Layout 返回之后做：vstack 的 Layout 会提前 return。
	if n.Tag != "scroll" {
		applyOverflowScroll(ctx, n, style, th)
	}
	placeAbsolutes(ctx, n, n.Children, n.X, n.Y, n.W, n.H, style, th)
	if dx, dy := relativeShift(n); dx != 0 || dy != 0 {
		shiftTree(n, dx, dy)
	}
}

func layoutPadded(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme) {
	p := n.Style.Padding
	layoutFlow(ctx, n.Children, n.X+p, n.Y+p, n.W-2*p, n.H-2*p, style, th)
}

func layoutFlow(ctx Context, nodes []*runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	var visible []*runtime.VNode
	for _, c := range nodes {
		if c != nil {
			visible = append(visible, c)
		}
	}
	placeFlow(ctx, nil, visible, x, y, w, h, style, th)
}

func expandComponent(n *runtime.VNode) {
	runtime.ResolveComponent(n)
}

// LayoutRoot 返回本帧正在布局的根节点，供 popover 找锚点。
func LayoutRoot() []*runtime.VNode { return layoutRoot }
