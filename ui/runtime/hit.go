package runtime

// OverlayTag 报告该标签是否是浮层（popover / dialog / ctxmenu）。
// 浮层在绘制和命中时后处理，这样菜单栏内部的下拉不会被后面的编辑区盖住。
//
// ctxmenu（右键面板）也是浮层，但它比另外两个多一层身份：它自带一张**铺满画布
// 的无色遮罩**，点空白处就关。之所以要做成浮层而不是普通子节点，就是为了让
// 「点面板外面」这件事有人接 —— 不然得让 app 层额外通知「刚才那一下点在了空处」，
// 而 app 层并不知道谁开着菜单。
func OverlayTag(tag string) bool {
	return tag == "popover" || tag == "dialog" || tag == "ctxmenu"
}

// Hit 是布局后的可点击区域，坐标单位 DIP。
type Hit struct {
	X, Y, W, H float32
	OnClick    func()
	clips      []hitClip
}

func (h Hit) Contains(x, y float32) bool {
	if x < h.X || y < h.Y || x >= h.X+h.W || y >= h.Y+h.H {
		return false
	}
	return pointInClips(h.clips, x, y)
}

// CollectHits 深度优先收集带 onClick 的节点。后收集的在上层，命中时从后往前找。
// 浮层整棵子树追加在最后，保证下拉 / 对话框盖住下层内容。
func CollectHits(nodes []*VNode) []Hit {
	var base, overlay []Hit
	collectHitsSplit(nodes, &base, &overlay)
	return append(base, overlay...)
}

func collectHitsSplit(nodes []*VNode, base, overlay *[]Hit) {
	collectHitsClipped(nodes, base, overlay, nil)
}

func collectHitsClipped(nodes []*VNode, base, overlay *[]Hit, clips []hitClip) {
	var abs []*VNode
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if IsAbsolute(n) {
			abs = append(abs, n)
			continue
		}
		if OverlayTag(n.Tag) {
			collectHits([]*VNode{n}, overlay)
			continue
		}
		if n.OnClick != nil && n.W > 0 && n.H > 0 && attrEnabled(n) {
			*base = append(*base, Hit{X: n.X, Y: n.Y, W: n.W, H: n.H, OnClick: n.OnClick, clips: copyClips(clips)})
		}
		next := clips
		if OverflowClips(n.Style, true) || OverflowClips(n.Style, false) {
			next = append(append([]hitClip{}, clips...), hitClip{n.X, n.Y, n.X + n.W, n.Y + n.H})
		}
		if len(n.Children) > 0 {
			collectHitsClipped(n.Children, base, overlay, next)
		}
	}
	if len(abs) > 0 {
		collectHitsClipped(abs, base, overlay, clips)
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

// HitNode 返回最上层包含 (x,y) 的可交互节点（OnClick / onPointerDown / onContextMenu）。
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
	var clips []hitClip
	var walk func([]*VNode)
	walk = func(list []*VNode) {
		var abs []*VNode
		for _, n := range list {
			if IsAbsolute(n) {
				abs = append(abs, n)
				continue
			}
			if n == nil || n.W <= 0 || n.H <= 0 || !attrEnabled(n) {
				continue
			}
			if OverlayTag(n.Tag) && (len(nodes) == 0 || n != nodes[0]) {
				continue
			}
			if x < n.X || y < n.Y || x >= n.X+n.W || y >= n.Y+n.H {
				continue
			}
			if !pointInClips(clips, x, y) {
				continue
			}
			if n.OnClick != nil {
				found = n
			}
			if n.Attrs != nil {
				if _, ok := n.Attrs["onPointerDown"].(func()); ok {
					found = n
				} else if _, ok := n.Attrs["onContextMenu"].(func(float32, float32)); ok {
					found = n
				}
			}
			pushed := false
			if OverflowClips(n.Style, true) || OverflowClips(n.Style, false) {
				clips = append(clips, hitClip{n.X, n.Y, n.X + n.W, n.Y + n.H})
				pushed = true
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
			if pushed {
				clips = clips[:len(clips)-1]
			}
		}
		if len(abs) > 0 {
			walk(abs)
		}
	}
	walk(nodes)
	return found
}

type hitClip struct{ x0, y0, x1, y1 float32 }

func copyClips(clips []hitClip) []hitClip {
	if len(clips) == 0 {
		return nil
	}
	return append([]hitClip{}, clips...)
}

func pointInClips(clips []hitClip, x, y float32) bool {
	for _, c := range clips {
		if x < c.x0 || y < c.y0 || x >= c.x1 || y >= c.y1 {
			return false
		}
	}
	return true
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
