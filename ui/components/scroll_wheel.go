package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// HandleWheel 把滚轮交给指针下最深、且还能滚的容器。
// 内层到头（或 hidden 只裁不滚）才交给外层，对齐 HTML 的滚轮冒泡。
// 节点自己的 onWheel 若返回 false，也继续向外找。delta 正值向上（一格 = 1）。
func HandleWheel(nodes []*runtime.VNode, x, y, delta float32) {
	chain := wheelChain(nodes, x, y)
	for i := len(chain) - 1; i >= 0; i-- {
		n := chain[i]
		if n.Tag == "dialog" {
			dispatchWheel(n, x, y, delta)
			return
		}
		if dispatchWheel(n, x, y, delta) {
			return
		}
		if wheelScroll(n, delta) {
			return
		}
	}
}

func wheelChain(nodes []*runtime.VNode, x, y float32) []*runtime.VNode {
	var chain []*runtime.VNode
	var walk func([]*runtime.VNode)
	walk = func(list []*runtime.VNode) {
		var hit *runtime.VNode
		for _, n := range list {
			if n != nil && renderer.Contains(n, x, y) {
				hit = n
			}
		}
		if hit == nil {
			return
		}
		chain = append(chain, hit)
		if hit.Tag == "dialog" {
			return
		}
		walk(hit.Children)
	}
	walk(nodes)
	return chain
}

// dispatchWheel 调用节点自己的 onWheel。返回 true 表示已消费，不再冒泡。
// func() bool 返回 false 时继续向外找（文本框到头就用这个）。
func dispatchWheel(n *runtime.VNode, x, y, delta float32) bool {
	if n == nil || n.Attrs == nil {
		return false
	}
	switch fn := n.Attrs["onWheel"].(type) {
	case func(float32) bool:
		return fn(delta)
	case func(float32, float32, float32) bool:
		return fn(x, y, delta)
	case func(float32):
		fn(delta)
		return true
	case func(float32, float32, float32):
		fn(x, y, delta)
		return true
	default:
		return false
	}
}

// wheelScroll 滚这个容器。还能滚就消费；已经到头返回 false，交给外层。
func wheelScroll(n *runtime.VNode, delta float32) bool {
	if n == nil || delta == 0 {
		return false
	}
	id := scrollID(n)
	if id == "" {
		return false
	}
	horizontal := false
	if !runtime.OverflowScrolls(n.Style, false) && runtime.OverflowScrolls(n.Style, true) {
		horizontal = true
	}
	if n.Tag != "scroll" && !runtime.OverflowScrolls(n.Style, horizontal) {
		return false
	}
	if n.Tag == "scroll" {
		horizontal = false
	}
	before := scrollGetOffset(id, horizontal)
	maxOff := scrollMaxOff(id, horizontal)
	if delta < 0 && before >= maxOff-0.5 {
		return false
	}
	if delta > 0 && before <= 0.5 {
		return false
	}
	ScrollBy(id, delta)
	return scrollGetOffset(id, horizontal) != before
}

func scrollMaxOff(id string, horizontal bool) float32 {
	scrollMu.RLock()
	defer scrollMu.RUnlock()
	st, ok := scrollExtent[scrollKey(id, horizontal)]
	if !ok || st.maxOff < 0 {
		return 0
	}
	return st.maxOff
}
