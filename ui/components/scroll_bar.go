package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// 滚动条几何 / 拖拽。与 scroll.go 的滚动状态表同包，拆开只为控制单文件长度。

// 预览滚动条拖动。grab 是按下时指针相对滑块顶的距离。
var (
	scrollDragID   string
	scrollDragGrab float32
	scrollDragH    bool
)

// EnsureScrollDrag 给任意可滚容器挂上滚动条拖拽。
//
// 为什么需要：滚动条本身是 paintScrollbar 画在**任意** overflow 盒子上（VStack /
// HStack 写 OverflowY 也会画），但拖拽回调此前只挂在 "scroll" 标签上。于是
// `VStack{OverflowY: auto}` 这类容器「看得到滚动条、却拖不动」。现在渲染管线
// 在布局阶段统一补齐，两种写法行为一致。
//
// 只补缺的：应用自己写了 onPointerDown / onPointerDrag 就不覆盖。
func EnsureScrollDrag(n *runtime.VNode) {
	if n == nil || !scrollableNode(n) {
		return
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	own := false
	if _, ok := n.Attrs["onPointerDown"]; !ok {
		n.Attrs["onPointerDown"] = func() { beginScrollDrag(n) }
		own = true
	}
	if _, ok := n.Attrs["onPointerDrag"]; !ok {
		n.Attrs["onPointerDrag"] = func() { dragScroll(n) }
	}
	// 只标记引擎自己补的那份：编辑框 / 文本域也有 onPointerDown（拖选），
	// 标成 scrollDrag 会让宿主把「拖选文字」误判成「拖滚动条」。
	if own {
		n.Attrs["scrollDrag"] = true
	}
}

// scrollableNode 报告节点是否可能画出纵向滚动条。
func scrollableNode(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	return n.Tag == "scroll" ||
		runtime.OverflowScrolls(n.Style, false) ||
		runtime.OverflowScrolls(n.Style, true)
}

func scrollBarGeom(n *runtime.VNode) (barY, barH, maxOff float32, ok bool) {
	id := scrollID(n)
	scrollMu.RLock()
	st, has := scrollExtent[scrollKey(id, false)]
	off := scrollOffset[scrollKey(id, false)]
	scrollMu.RUnlock()
	if !has || st.contentH <= n.H+1 || n.H <= 0 {
		return 0, 0, 0, false
	}
	pad := scrollThumbPad(n)
	track := n.H - 2*pad
	if track < 8 {
		track = n.H
		pad = 0
	}
	barH = track * n.H / st.contentH
	if barH < 28 {
		barH = 28
	}
	if barH > track {
		barH = track
	}
	// 用布局写入的可滚上限：overflow 盒子的 contentH 不含内边距，
	// 按 contentH-n.H 算会少一截，拖到底时提前停住。
	maxOff = st.maxOff
	if maxOff < 0 {
		maxOff = 0
	}
	barY = n.Y + pad
	if maxOff > 0 {
		barY = n.Y + pad + (track-barH)*(off/maxOff)
	}
	return barY, barH, maxOff, true
}

func scrollBarGeomH(n *runtime.VNode) (barX, barW, maxOff float32, ok bool) {
	id := scrollID(n)
	scrollMu.RLock()
	st, has := scrollExtent[scrollKey(id, true)]
	off := scrollOffset[scrollKey(id, true)]
	scrollMu.RUnlock()
	if !has || st.content <= n.W+1 || n.W <= 0 {
		return 0, 0, 0, false
	}
	pad := scrollThumbPad(n)
	track := n.W - 2*pad
	if track < 8 {
		track = n.W
		pad = 0
	}
	barW = track * n.W / st.content
	if barW < 28 {
		barW = 28
	}
	if barW > track {
		barW = track
	}
	maxOff = st.maxOff
	if maxOff < 0 {
		maxOff = 0
	}
	barX = n.X + pad
	if maxOff > 0 {
		barX = n.X + pad + (track-barW)*(off/maxOff)
	}
	return barX, barW, maxOff, true
}

func scrollThumbPad(n *runtime.VNode) float32 {
	if renderer.AttrBool(n, "widebar") {
		return 10
	}
	return 8
}

// scrollGutter 是「右侧多宽的区域算滚动条」。与 paintScrollbar 画的滑块宽度
// 对齐并留一点余量：滑块 6~8 DIP 宽、内缩 10~18，抓取区给到 18~22 才不至于
// 「看得见点不着」。
func scrollGutter(n *runtime.VNode) float32 {
	if renderer.AttrBool(n, "widebar") {
		return 22
	}
	return 18
}

// scrollbarPainted 与 paintScrollbar 的可见门槛保持一致。
// 内容只溢出一丁点时滑块几乎铺满、看着像分割线，那种情况不画，也不该能拖。
func scrollbarPainted(n *runtime.VNode) bool {
	if n == nil || n.H <= 0 {
		return false
	}
	if contentH := scrollCachedContentH(scrollID(n)); contentH > 0 {
		return contentH > n.H+32
	}
	return true
}

func beginScrollDrag(n *runtime.VNode) {
	x, y, ok := renderer.Pointer()
	if !ok || n == nil {
		return
	}
	// 每次按下都先清掉上一次的拖拽目标：拖拽中途丢 pointer-up 时
	// 残留的 id 会让下一次按在内容区也「粘住」滚动条。
	scrollDragID = ""
	scrollDragH = false
	gutter := scrollGutter(n)
	// Scroll 标签也走纵向分支，否则预览右侧滚动条按不动。
	if runtime.OverflowScrolls(n.Style, false) {
		barY, barH, _, shown := scrollBarGeom(n)
		// 抓取区：整条右侧窄带 + 滑块高度。早先按 y ∈ [barY, barY+barH] 判，
		// 在滑块以外的轨道上按下不会拖动，用户以为「拖不动」。
		if shown && scrollbarPainted(n) &&
			x >= n.X+n.W-gutter && x <= n.X+n.W && y >= n.Y && y <= n.Y+n.H {
			scrollDragID = scrollID(n)
			if y >= barY && y <= barY+barH {
				scrollDragGrab = y - barY
			} else {
				// 点轨道：把滑块中心吸到指针处，手感接近系统滚动条。
				scrollDragGrab = barH / 2
				scrollDragJump(n, y, barH)
			}
			return
		}
	}
	if !runtime.OverflowScrolls(n.Style, true) {
		return
	}
	barX, barW, _, shown := scrollBarGeomH(n)
	if !shown || y < n.Y+n.H-gutter || y > n.Y+n.H || x < n.X || x > n.X+n.W {
		return
	}
	scrollDragID = scrollID(n)
	scrollDragH = true
	if x >= barX && x <= barX+barW {
		scrollDragGrab = x - barX
	} else {
		scrollDragGrab = barW / 2
		scrollDragJumpH(n, x, barW)
	}
}

func dragScroll(n *runtime.VNode) {
	if n == nil || scrollDragID == "" || scrollDragID != scrollID(n) {
		return
	}
	x, y, ok := renderer.Pointer()
	if !ok {
		return
	}
	if scrollDragH {
		_, barW, _, shown := scrollBarGeomH(n)
		if shown {
			scrollDragJumpH(n, x, barW)
		}
		return
	}
	_, barH, _, shown := scrollBarGeom(n)
	if !shown {
		return
	}
	scrollDragJump(n, y, barH)
}

// scrollDragJump 按指针位置把滑块中心摆到指针处，并换算成滚动偏移。
// 拖拽与「点轨道跳转」共用这一段，两者手感才一致。
func scrollDragJump(n *runtime.VNode, y, barH float32) {
	_, _, maxOff, shown := scrollBarGeom(n)
	if !shown || n.H <= 0 {
		return
	}
	pad := scrollThumbPad(n)
	track := n.H - 2*pad
	if track < 8 {
		track = n.H
		pad = 0
	}
	span := track - barH
	if span <= 0 {
		return
	}
	off := (y - scrollDragGrab - (n.Y + pad)) / span * maxOff
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	scrollSetOffset(scrollID(n), false, off)
	if fn := onFloat(n, "onScroll"); fn != nil {
		fn(off)
	}
}

func scrollDragJumpH(n *runtime.VNode, x, barW float32) {
	_, _, maxOff, shown := scrollBarGeomH(n)
	if !shown || n.W <= 0 {
		return
	}
	pad := scrollThumbPad(n)
	track := n.W - 2*pad
	if track < 8 {
		track = n.W
		pad = 0
	}
	span := track - barW
	if span <= 0 {
		return
	}
	off := (x - scrollDragGrab - (n.X + pad)) / span * maxOff
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	scrollSetOffset(scrollID(n), true, off)
}

// EndScrollDrag 松手时结束滚动条拖动。
func EndScrollDrag() { scrollDragID = "" }

// ScrollDragging 报告当前是否真的抓着某个滚动条滑块。
// 宿主用它区分「拖滚动条（只需重排）」与「拖选文字（要重建树）」，
// 也避免在滚动容器空白处按下时每移动一像素就重排一次。
func ScrollDragging() bool { return scrollDragID != "" }
