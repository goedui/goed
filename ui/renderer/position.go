package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// measureIgnoreAbsolute 让绝对定位节点在自己摆放时仍能量出边框盒。
// 流式测量里它必须报 0，否则会把脱离文档流的尺寸算进父级。
var measureIgnoreAbsolute bool

// placeFlow 按文档流纵向摆放。绝对定位不占位，留给 placeAbsolutes。
func placeFlow(ctx Context, parent *runtime.VNode, nodes []*runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	var flow []*runtime.VNode
	for _, c := range nodes {
		if c != nil && !runtime.IsAbsolute(c) {
			flow = append(flow, c)
		}
	}
	if h > 0 && len(flow) == 1 && fills(flow[0]) {
		m := flow[0].Style.Margin
		placeChild(ctx, parent, flow[0], x+m, y+m, w-2*m, h-2*m, style, th)
		return
	}
	cy := y
	for _, c := range flow {
		_, ch := measureNode(ctx, c, w, 0, style, th)
		m := c.Style.Margin
		placeChild(ctx, parent, c, x+m, cy+m, w-2*m, ch, style, th)
		cy += ch + 2*m
	}
}

func placeChild(ctx Context, parent, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	if !runtime.IsAbsolute(n) {
		runtime.SetParent(n, parent)
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	layoutNode(ctx, n, x, y, w, h, style, th)
}

// placeAbsolutes 把直接子节点里的 absolute 摆到包含块上。
// 包含块是最近的 relative/absolute 祖先；parent 自己就是时用它的 padding 边。
func placeAbsolutes(ctx Context, parent *runtime.VNode, nodes []*runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	for _, c := range nodes {
		if c == nil || !runtime.IsAbsolute(c) {
			continue
		}
		runtime.SetParent(c, parent)
		ax, ay, aw, ah := absoluteBox(ctx, c, x, y, w, h, style, th)
		placeChild(ctx, parent, c, ax, ay, aw, ah, style, th)
	}
}

func absoluteBox(ctx Context, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) (float32, float32, float32, float32) {
	ax, ay, aw, ah, ok := runtime.ContainingBlock(n)
	if !ok {
		ax, ay, aw, ah = x, y, w, h
	}
	if aw < 0 {
		aw = 0
	}
	if ah < 0 {
		ah = 0
	}
	s := n.Style
	m := s.Margin
	left := runtime.InsetSpecified(s, runtime.InsetLeft, s.Left)
	right := runtime.InsetSpecified(s, runtime.InsetRight, s.Right)
	top := runtime.InsetSpecified(s, runtime.InsetTop, s.Top)
	bottom := runtime.InsetSpecified(s, runtime.InsetBottom, s.Bottom)

	maxW := aw - 2*m
	if left {
		maxW -= s.Left
	}
	if right {
		maxW -= s.Right
	}
	if maxW < 0 {
		maxW = 0
	}
	measureIgnoreAbsolute = true
	bw, bh := measureNode(ctx, n, maxW, 0, style, th)
	measureIgnoreAbsolute = false

	boxW := bw
	if s.Width > 0 {
		boxW = s.Width
	} else if left && right {
		boxW = maxW
	}
	boxH := bh
	if s.Height > 0 {
		boxH = s.Height
	} else if top && bottom {
		boxH = ah - s.Top - s.Bottom - 2*m
	}
	if boxW < 0 {
		boxW = 0
	}
	if boxH < 0 {
		boxH = 0
	}
	boxX := ax + m
	if left {
		boxX += s.Left
	} else if right {
		boxX = ax + aw - m - s.Right - boxW
	}
	boxY := ay + m
	if top {
		boxY += s.Top
	} else if bottom {
		boxY = ay + ah - m - s.Bottom - boxH
	}
	return boxX, boxY, boxW, boxH
}

func relativeShift(n *runtime.VNode) (float32, float32) {
	if n == nil || n.Style.Position != runtime.PositionRelative {
		return 0, 0
	}
	s := n.Style
	var dx, dy float32
	if runtime.InsetSpecified(s, runtime.InsetLeft, s.Left) {
		dx = s.Left
	} else if runtime.InsetSpecified(s, runtime.InsetRight, s.Right) {
		dx = -s.Right
	}
	if runtime.InsetSpecified(s, runtime.InsetTop, s.Top) {
		dy = s.Top
	} else if runtime.InsetSpecified(s, runtime.InsetBottom, s.Bottom) {
		dy = -s.Bottom
	}
	return dx, dy
}

func shiftTree(n *runtime.VNode, dx, dy float32) {
	if n == nil || (dx == 0 && dy == 0) {
		return
	}
	n.X += dx
	n.Y += dy
	shiftOverflow(n.Children, dx, dy)
}
