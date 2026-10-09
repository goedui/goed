package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// applyOverflowScroll 把 auto/scroll 盒子的子树按偏移挪进视口。
// hidden 只裁不滚。visible 不动。scroll 控件有自己的布局，不走这里。
func applyOverflowScroll(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme) {
	if n == nil || n.Tag == "scroll" || len(n.Children) == 0 {
		return
	}
	scrollX := runtime.OverflowScrolls(n.Style, true)
	scrollY := runtime.OverflowScrolls(n.Style, false)
	if !scrollX && !scrollY {
		return
	}
	id := n.ID()
	if id == "" {
		id = n.Props.Key
	}
	if id == "" {
		return
	}
	pad := n.Style.Padding
	innerW := n.W - 2*pad
	innerH := n.H - 2*pad
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	contentW, contentH := overflowContentSize(n, pad)
	offX, maxX := clampOverflow(readOverflow(id, true), contentW, innerW, scrollX)
	offY, maxY := clampOverflow(readOverflow(id, false), contentH, innerH, scrollY)
	// 子节点按未滚动的原点（n.X+pad, n.Y+pad）摆好。滚一段就是整棵子树平移 -off。
	shiftOverflow(n.Children, -offX, -offY)
	_ = ctx
	_ = style
	_ = th
	writeOverflow(id, true, offX, maxX, innerW, contentW)
	writeOverflow(id, false, offY, maxY, innerH, contentH)
}

func readOverflow(id string, horizontal bool) float32 {
	if OverflowOffset == nil {
		return 0
	}
	return OverflowOffset(id, horizontal)
}

func writeOverflow(id string, horizontal bool, off, maxOff, view, content float32) {
	if StoreOverflow == nil {
		return
	}
	StoreOverflow(id, horizontal, off, maxOff, view, content)
}

// OverflowOffset / StoreOverflow 由 components 注入，避免 renderer 反向依赖滚动表。
var (
	OverflowOffset func(id string, horizontal bool) float32
	StoreOverflow  func(id string, horizontal bool, off, maxOff, view, content float32)
)

func overflowContentSize(n *runtime.VNode, pad float32) (float32, float32) {
	var maxX, maxY float32
	originX := n.X + pad
	originY := n.Y + pad
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if r := c.X + c.W - originX; r > maxX {
			maxX = r
		}
		if b := c.Y + c.H - originY; b > maxY {
			maxY = b
		}
	}
	return maxX, maxY
}

func clampOverflow(off, content, view float32, scroll bool) (float32, float32) {
	if !scroll {
		return 0, 0
	}
	maxOff := content - view
	if maxOff < 0 {
		maxOff = 0
	}
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	return off, maxOff
}

func shiftOverflow(nodes []*runtime.VNode, dx, dy float32) {
	if dx == 0 && dy == 0 {
		return
	}
	var walk func([]*runtime.VNode)
	walk = func(list []*runtime.VNode) {
		for _, n := range list {
			if n == nil {
				continue
			}
			n.X += dx
			n.Y += dy
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
}

// PaintChildren 按已布局坐标画子节点。
// overflow 不是 visible 的轴会裁到本节点矩形；auto / scroll 的偏移由布局阶段写入子节点坐标。
func PaintChildren(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme) {
	if n == nil {
		return
	}
	clipX := runtime.OverflowClips(n.Style, true)
	clipY := runtime.OverflowClips(n.Style, false)
	if clipX || clipY {
		ctx.PushClip(n.X, n.Y, n.W, n.H)
		PushCullClip(n.X, n.Y, n.W, n.H)
	}
	paintLaidOut(ctx, n.Children, style, th)
	if clipX || clipY {
		PopCullClip()
		ctx.PopClip()
	}
	if n.Tag != "scroll" && PaintOverflowBars != nil && runtime.OverflowScrolls(n.Style, false) {
		PaintOverflowBars(ctx, n, style, th)
	}
}

// PaintOverflowBars 由滚动实现注册，给 auto/scroll 盒子画滚动条。
var PaintOverflowBars func(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme)
