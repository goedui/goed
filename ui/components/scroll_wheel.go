package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// HandleWheel 把滚轮交给指针下最深、且还能滚的容器。
// 内层到头（或 hidden 只裁不滚）才交给外层，对齐 HTML 的滚轮冒泡。
// 节点自己的 onWheel 若返回 false，也继续向外找。delta 正值向上（一格 = 1）。
//
// dialog 是模态面。这里必须先于「找最深可滚节点」处理它，原因是两者的搜索
// 范围不同：
//
//   - dialog 与正文是兄弟时，指针落在卡片上，但同一坐标也落在下面的滚动区里。
//     wheelChain 走正文那条分支，碰到第一个可滚容器就返回，dialog 压根没进链，
//     滚轮于是滚了遮罩背后的内容。
//   - dialog 嵌在滚动区内部时，链里只有子树，外层页面反而可能被当成「更外层」
//     继续冒泡。
//
// 所以：先在整棵树里找覆盖该点的最上层 dialog，找到就把搜索范围收窄到它的
// 子树。dialog 内部能滚就滚自己，滚到头也到此为止。
func HandleWheel(nodes []*runtime.VNode, x, y, delta float32) {
	if dlg := topmostDialogAt(nodes, x, y); dlg != nil {
		nodes = []*runtime.VNode{dlg}
	}
	chain := wheelChain(nodes, x, y)
	for i := len(chain) - 1; i >= 0; i-- {
		n := chain[i]
		if dispatchWheel(n, x, y, delta) {
			return
		}
		if wheelScroll(n, delta) {
			return
		}
		if n.Tag == "dialog" {
			// At the modal surface with nothing inside willing to scroll:
			// absorb the event instead of letting it reach the page.
			return
		}
	}
}

// topmostDialogAt 返回覆盖 (x,y) 的、最后绘制的那个 dialog。
//
// 先递归子节点再判断自己，且按兄弟顺序遍历，于是后绘制的覆盖先绘制的 ——
// 与渲染顺序一致，叠放时拿到的是真正压在最上面的那层。
func topmostDialogAt(nodes []*runtime.VNode, x, y float32) *runtime.VNode {
	var found *runtime.VNode
	var walk func([]*runtime.VNode)
	walk = func(list []*runtime.VNode) {
		for _, n := range list {
			if n == nil || n.W <= 0 || n.H <= 0 || !renderer.Contains(n, x, y) {
				continue
			}
			walk(n.Children)
			if n.Tag == "dialog" {
				found = n
			}
		}
	}
	walk(nodes)
	return found
}

// wheelChain 收集指针下「从外到内」的候选节点。
//
// 早先命中第一个包含该点的节点就直接 return（只走一条路径）。问题是任何铺满
// 父级的**纯装饰层**（应用自绘的页面背景、光斑 Canvas）都会把后面的兄弟节点
// 整条链挡掉，表现为「滚轮完全没反应」——而它自己既不可滚、也没有 onWheel。
//
// 现在改成：一条分支走到底后，若整条分支里没有任何能处理滚轮的节点，就把自己
// 从链上摘掉、继续试下一个兄弟。语义等于 CSS 的 pointer-events:none ——
// 装饰层不再吞事件，能滚的容器仍然只有最深那一个。
func wheelChain(nodes []*runtime.VNode, x, y float32) []*runtime.VNode {
	var chain []*runtime.VNode
	var clips []wheelClip
	var walk func([]*runtime.VNode) bool
	walk = func(list []*runtime.VNode) bool {
		for _, n := range list {
			if n == nil || n.W <= 0 || n.H <= 0 || !renderer.Contains(n, x, y) {
				continue
			}
			if !wheelInClips(clips, x, y) {
				continue
			}
			chain = append(chain, n)
			pushed := false
			if runtime.OverflowClips(n.Style, true) || runtime.OverflowClips(n.Style, false) {
				clips = append(clips, wheelClip{n.X, n.Y, n.X + n.W, n.Y + n.H})
				pushed = true
			}
			handled := walk(n.Children)
			if pushed {
				clips = clips[:len(clips)-1]
			}
			if handled || wheelCapable(n) {
				return true
			}
			// 这条分支只是装饰：摘掉自己，继续找兄弟。
			chain = chain[:len(chain)-1]
		}
		return false
	}
	walk(nodes)
	return chain
}

// wheelCapable 报告节点自己能不能处理滚轮：有 onWheel 回调，或本身就是可滚容器。
// 不能只看 OverflowScrolls —— 输入框 / 下拉的 onWheel 也要拦住事件。
func wheelCapable(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	if n.Attrs != nil {
		if _, ok := n.Attrs["onWheel"]; ok {
			return true
		}
	}
	return n.Tag == "scroll" || runtime.OverflowScrolls(n.Style, true) || runtime.OverflowScrolls(n.Style, false)
}

type wheelClip struct{ x0, y0, x1, y1 float32 }

func wheelInClips(clips []wheelClip, x, y float32) bool {
	for _, c := range clips {
		if x < c.x0 || y < c.y0 || x >= c.x1 || y >= c.y1 {
			return false
		}
	}
	return true
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
	if horizontal && renderer.AttrBool(n, "noWheelX") {
		return false
	}
	if n.Tag != "scroll" && !runtime.OverflowScrolls(n.Style, horizontal) {
		return false
	}
	if n.Tag == "scroll" {
		horizontal = false
	}
	before := scrollGetOffset(id, horizontal)
	maxOff := scrollMaxOff(id, horizontal)
	// 横向溢出的内容宽在子节点上，布局表还没记下上限时按子节点右缘估。
	if horizontal && maxOff <= 0.5 {
		if wide := wheelContentRight(n) - n.W; wide > maxOff {
			maxOff = wide
		}
	}
	if delta < 0 && before >= maxOff-0.5 {
		return false
	}
	if delta > 0 && before <= 0.5 {
		return false
	}
	ScrollAxis(id, delta, horizontal)
	return scrollGetOffset(id, horizontal) != before
}

func wheelContentRight(n *runtime.VNode) float32 {
	var right float32
	var walk func([]*runtime.VNode)
	walk = func(list []*runtime.VNode) {
		for _, c := range list {
			if c == nil {
				continue
			}
			if c.X+c.W > right {
				right = c.X + c.W
			}
			walk(c.Children)
		}
	}
	walk(n.Children)
	return right - n.X
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
