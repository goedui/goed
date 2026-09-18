package runtime

// OverlayTag 报告该标签是否是浮层（popover / dialog）。
// 浮层在绘制和命中时后处理，这样菜单栏内部的下拉不会被后面的编辑区盖住。
func OverlayTag(tag string) bool {
	return tag == "popover" || tag == "dialog"
}

// Hit 是布局后的可点击区域，坐标单位 DIP。
type Hit struct {
	X, Y, W, H float32
	OnClick    func()
}

func (h Hit) Contains(x, y float32) bool {
	return x >= h.X && y >= h.Y && x < h.X+h.W && y < h.Y+h.H
}

// CollectHits 深度优先收集带 onClick 的节点。后收集的在上层，命中时从后往前找。
// 浮层整棵子树追加在最后，保证下拉 / 对话框盖住下层内容。
func CollectHits(nodes []*VNode) []Hit {
	var base, overlay []Hit
	collectHitsSplit(nodes, &base, &overlay)
	return append(base, overlay...)
}

func collectHitsSplit(nodes []*VNode, base, overlay *[]Hit) {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if OverlayTag(n.Tag) {
			collectHits([]*VNode{n}, overlay)
			continue
		}
		if n.OnClick != nil && n.W > 0 && n.H > 0 && attrEnabled(n) {
			*base = append(*base, Hit{X: n.X, Y: n.Y, W: n.W, H: n.H, OnClick: n.OnClick})
		}
		if len(n.Children) > 0 {
			collectHitsSplit(n.Children, base, overlay)
		}
	}
}

func collectHits(nodes []*VNode, out *[]Hit) {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.OnClick != nil && n.W > 0 && n.H > 0 && attrEnabled(n) {
			*out = append(*out, Hit{X: n.X, Y: n.Y, W: n.W, H: n.H, OnClick: n.OnClick})
		}
		if len(n.Children) > 0 {
			collectHits(n.Children, out)
		}
	}
}

func attrEnabled(n *VNode) bool {
	if n == nil || n.Attrs == nil {
		return true
	}
	if v, ok := n.Attrs["enabled"].(bool); ok {
		return v
	}
	if v, ok := n.Attrs["disabled"].(bool); ok {
		return !v
	}
	return true
}

// HitNode 返回最上层包含 (x,y) 的可交互节点（OnClick 或 onPointerDown）。
func HitNode(nodes []*VNode, x, y float32) *VNode {
	var overlays []*VNode
	Walk(nodes, func(n *VNode) bool {
		if n != nil && OverlayTag(n.Tag) {
			overlays = append(overlays, n)
		}
		return false
	})
	for i := len(overlays) - 1; i >= 0; i-- {
		if n := hitNodeWalk([]*VNode{overlays[i]}, x, y); n != nil {
			return n
		}
	}
	return hitNodeWalk(nodes, x, y)
}

func hitNodeWalk(nodes []*VNode, x, y float32) *VNode {
	var found *VNode
	Walk(nodes, func(n *VNode) bool {
		if n == nil || n.W <= 0 || n.H <= 0 || !attrEnabled(n) {
			return false
		}
		if OverlayTag(n.Tag) && n != nodes[0] {
			return false
		}
		if x < n.X || y < n.Y || x >= n.X+n.W || y >= n.Y+n.H {
			return false
		}
		if n.OnClick != nil {
			found = n
			return false
		}
		if n.Attrs != nil {
			if _, ok := n.Attrs["onPointerDown"].(func()); ok {
				found = n
			}
		}
		return false
	})
	return found
}

// HitTest 返回最上层包含 (x,y) 的点击回调。
func HitTest(hits []Hit, x, y float32) func() {
	for i := len(hits) - 1; i >= 0; i-- {
		if hits[i].Contains(x, y) {
			return hits[i].OnClick
		}
	}
	return nil
}

// HoverChanged 报告指针是否进入或离开了可点击区域。
// 用于按需重绘：只有 hover 目标变化时才重画按钮高亮。
func HoverChanged(hits []Hit, x1, y1, x2, y2 float32) bool {
	return hoverIndex(hits, x1, y1) != hoverIndex(hits, x2, y2)
}

func hoverIndex(hits []Hit, x, y float32) int {
	for i := len(hits) - 1; i >= 0; i-- {
		if hits[i].Contains(x, y) {
			return i
		}
	}
	return -1
}
