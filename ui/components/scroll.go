package components

import (
	"sync"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("scroll", renderer.Widget{
		Measure: measureScroll,
		Layout:  layoutScroll,
		Paint:   paintScroll,
	})
	renderer.OverflowOffset = func(id string, horizontal bool) float32 {
		return scrollGetOffset(id, horizontal)
	}
	renderer.StoreOverflow = func(id string, horizontal bool, off, maxOff, view, content float32) {
		scrollSetLayout(id, horizontal, off, scrollExtentEntry{
			maxOff: maxOff, view: view, content: content,
			viewH: view, contentH: content,
		})
	}
	renderer.PaintOverflowBars = paintScrollbar
}

// 滚动状态是包级共享的（按容器 id 索引），碰它的有两类 goroutine：渲染 / 布局那条
// 线，以及业务侧可能从后台线程调用的 ScrollToEnd / ScrollBottomGap（「回复延迟
// 700ms 后贴底」这种定时器）。两个 map 裸奔时 -race 会报并发 mapassign，Go 运行时
// 更会直接以 "concurrent map writes" 结束进程 —— 全部走下面这组带锁入口。
//
// 锁只圈住 map 操作本身，不跨越 layout / paint 里的子节点递归：嵌套滚动容器会重入
// layoutScroll，锁持有太久会自锁。
var (
	scrollMu     sync.RWMutex
	scrollOffset = map[string]float32{}
	// scrollExtent 记录每个容器某一轴的可滚范围（判断是否贴底、滚动条用）。
	// 键带轴：同一 id 可以同时横滚和纵滚。
	scrollExtent = map[string]scrollExtentEntry{}
)

type scrollExtentEntry struct {
	maxOff   float32
	view     float32
	content  float32
	// viewH / contentH 保留给只滚纵向的旧调用（贴底、滚动条）。
	viewH    float32
	contentH float32
}

func scrollKey(id string, horizontal bool) string {
	if horizontal {
		return id + "\x00x"
	}
	return id + "\x00y"
}

// 预览滚动条拖动。grab 是按下时指针相对滑块顶的距离。
var (
	scrollDragID  string
	scrollDragGrab float32
)

func scrollGetOffset(id string, horizontal bool) float32 {
	scrollMu.RLock()
	v := scrollOffset[scrollKey(id, horizontal)]
	scrollMu.RUnlock()
	return v
}

func scrollSetOffset(id string, horizontal bool, v float32) {
	scrollMu.Lock()
	scrollOffset[scrollKey(id, horizontal)] = v
	scrollMu.Unlock()
}

func scrollDelOffset(id string, horizontal bool) {
	scrollMu.Lock()
	delete(scrollOffset, scrollKey(id, horizontal))
	delete(scrollExtent, scrollKey(id, horizontal))
	scrollMu.Unlock()
}

// scrollSetLayout 一次写完「夹过的偏移 + 可滚范围」：两者是同一帧布局的产物，
// 分两次写会让别的 goroutine 读到半新半旧的组合。
func scrollSetLayout(id string, horizontal bool, off float32, e scrollExtentEntry) {
	scrollMu.Lock()
	k := scrollKey(id, horizontal)
	scrollOffset[k] = off
	scrollExtent[k] = e
	scrollMu.Unlock()
}

// Scroll 垂直滚动容器，等价于 overflow-y: auto 的盒子。
// 偏移与裁剪走 Style overflow，和 VStack/HStack 上写 Overflow 是同一条路径。
func Scroll(parts ...any) *runtime.VNode {
	n := runtime.H("scroll", parts...)
	if n.Style.OverflowY == "" && n.Style.Overflow == "" {
		n.Style.OverflowY = runtime.OverflowAuto
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["onPointerDown"] = func() { beginScrollDrag(n) }
	n.Attrs["onPointerDrag"] = func() { dragScroll(n) }
	return n
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
	maxOff = st.contentH - n.H
	if maxOff < 0 {
		maxOff = 0
	}
	barY = n.Y + pad
	if maxOff > 0 {
		barY = n.Y + pad + (track-barH)*(off/maxOff)
	}
	return barY, barH, maxOff, true
}

func scrollThumbPad(n *runtime.VNode) float32 {
	if renderer.AttrBool(n, "widebar") {
		return 10
	}
	return 8
}

func beginScrollDrag(n *runtime.VNode) {
	x, y, ok := renderer.Pointer()
	if !ok || n == nil {
		return
	}
	barY, barH, _, shown := scrollBarGeom(n)
	inset := float32(18)
	if renderer.AttrBool(n, "widebar") {
		inset = 26
	}
	if !shown || x < n.X+n.W-inset || x > n.X+n.W || y < barY || y > barY+barH {
		return
	}
	scrollDragID = scrollID(n)
	scrollDragGrab = y - barY
}

func dragScroll(n *runtime.VNode) {
	if n == nil || scrollDragID == "" || scrollDragID != scrollID(n) {
		return
	}
	_, y, ok := renderer.Pointer()
	if !ok {
		return
	}
	_, barH, maxOff, shown := scrollBarGeom(n)
	if !shown {
		return
	}
	track := n.H - barH
	if track <= 0 {
		return
	}
	off := (y - scrollDragGrab - n.Y) / track * maxOff
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

// EndScrollDrag 松手时结束滚动条拖动。
func EndScrollDrag() { scrollDragID = "" }

func scrollID(n *runtime.VNode) string {
	if n == nil {
		return ""
	}
	if id := n.ID(); id != "" {
		return id
	}
	return n.Props.Key
}

func measureScroll(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	w := n.Style.Width
	if w <= 0 {
		w = maxW
	}
	h := n.Style.Height
	if h <= 0 {
		if n.Style.Flex > 0 && maxH > 0 {
			h = maxH
		} else {
			_, contentH := contentSize(ctx, n, w, style, th)
			h = contentH
			if maxH > 0 && h > maxH {
				h = maxH
			}
		}
	}
	// Style 上的上下限也要生效：弹层里的滚动列表靠它限高。
	if top := n.Style.MaxHeight; top > 0 && h > top {
		h = top
	}
	if min := n.Style.MinHeight; min > 0 && h < min {
		h = min
	}
	return w, h
}

// scrollBarGutter 是滚动条占掉的内容宽度。宽条（预览）画在内容右缘上，
// 不留出这段的话，表格最后一列会被滑块盖住。
func scrollBarGutter(n *runtime.VNode) float32 {
	if n != nil && renderer.AttrBool(n, "widebar") {
		return 22
	}
	return 0
}

func contentSize(ctx renderer.Context, n *runtime.VNode, maxW float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	var w, h float32
	gap := n.Style.Gap
	pad := n.Style.Padding
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if runtime.IsAbsolute(c) {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, maxW-2*pad-scrollBarGutter(n), 0, style, th)
		cw += 2 * c.Style.Margin
		ch += 2 * c.Style.Margin
		if cw > w {
			w = cw
		}
		if !first {
			h += gap
		}
		first = false
		h += ch
	}
	return w + 2*pad, h + 2*pad
}

func layoutScroll(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	id := scrollID(n)
	offY := scrollGetOffset(id, false)
	innerW := w - 2*n.Style.Padding - scrollBarGutter(n)
	if innerW < 0 {
		innerW = 0
	}
	contentH := placeScrollChildren(ctx, n, innerW, offY, style, th)
	maxOff := contentH - h
	if maxOff < 0 {
		maxOff = 0
	}
	if offY < 0 {
		offY = 0
	}
	clamped := offY > maxOff
	if clamped {
		offY = maxOff
	}
	scrollSetLayout(id, false, offY, scrollExtentEntry{maxOff: maxOff, viewH: h, contentH: contentH})
	// 偏移被夹进范围后，第一次摆的位置是按夹之前算的，要再摆一次。
	if clamped {
		placeScrollChildren(ctx, n, innerW, offY, style, th)
	}
}

// placeScrollChildren 累加子节点高度，并只对视口附近做完整布局。
func placeScrollChildren(ctx renderer.Context, n *runtime.VNode, innerW, off float32, style renderer.TextStyle, th theme.Theme) float32 {
	pad := n.Style.Padding
	gap := n.Style.Gap
	x, y, h := n.X, n.Y, n.H
	// 视口上下各留一屏。滚轮一格仍落在已布局的行上。
	slack := h
	if slack < 48 {
		slack = 48
	}
	viewTop := y - slack
	viewBot := y + h + slack
	cy := y + pad - off
	var contentInner float32
	seen := 0
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if runtime.IsAbsolute(c) {
			continue
		}
		if seen > 0 {
			contentInner += gap
			cy += gap
		}
		seen++
		// 先拿高度再判断相交。只看块顶的话，比视口高的那一块滚过两屏就被当成
		// 完全在外面，整棵子树收成占位，标题坐标全挤到内容顶。
		ch := scrollChildHeight(ctx, c, innerW, true, style, th)
		near := cy+ch > viewTop && cy < viewBot
		if !near {
			ch = scrollChildHeight(ctx, c, innerW, false, style, th)
		}
		m := c.Style.Margin
		if !near {
			placeScrollStub(c, x+pad+m, cy+m, innerW-2*m, ch-2*m)
		} else {
			renderer.LayoutNode(ctx, c, x+pad+m, cy+m, innerW-2*m, ch-2*m, style, th)
		}
		contentInner += ch
		cy += ch
	}
	return contentInner + 2*pad
}

// scrollChildHeight 视口外优先用显式高度或上一帧测量。
// 两者都没有才 MeasureNode，避免长列表每帧量完全部行。
//
// 视口外不能把高度收成 0：滚动同步靠子节点的 Y 找标题，高度一塌后面的锚点
// 全挤到内容顶，编辑区滚过两屏预览就会被拉回开头。
func scrollChildHeight(ctx renderer.Context, n *runtime.VNode, maxW float32, near bool, style renderer.TextStyle, th theme.Theme) float32 {
	if n == nil {
		return 0
	}
	if _, h, ok := n.LookupMeasure(maxW, 0, style.FontFamily, style.FontSize, style.Weight); ok {
		return h
	}
	if !near {
		if n.Style.Height > 0 {
			return n.Style.Height
		}
		if h := measuredChildHeight(n); h > 0 {
			return h
		}
	}
	_, h := renderer.MeasureNode(ctx, n, maxW, 0, style, th)
	return h
}

// measuredChildHeight 沿用上一帧量出来的高度。
// n.H 在被收成占位（0）之后就不可信，要往下找还留着高度的子孙。
func measuredChildHeight(n *runtime.VNode) float32 {
	if n == nil {
		return 0
	}
	if n.H > 0 && len(n.Children) == 0 {
		return n.H
	}
	var sum float32
	gap := n.Style.Gap
	seen := 0
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		ch := measuredChildHeight(c)
		if ch <= 0 {
			return 0
		}
		if seen > 0 {
			sum += gap
		}
		seen++
		sum += ch
	}
	if seen == 0 {
		return n.H
	}
	return sum + 2*n.Style.Padding
}

// placeScrollStub 给视口外的子树写盒子，并清掉子孙上一次的坐标。
// 不走 LayoutNode，避免再量一遍并摆整棵子树。
func placeScrollStub(n *runtime.VNode, x, y, w, h float32) {
	if n == nil {
		return
	}
	n.X, n.Y, n.W, n.H = x, y, w, h
	for _, c := range n.Children {
		placeScrollStub(c, x, y, 0, 0)
	}
}

func paintScroll(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	renderer.PaintChildren(ctx, n, style, th)
	paintScrollbar(ctx, n, style, th)
}

func scrollCachedContentH(id string) float32 {
	scrollMu.RLock()
	st, ok := scrollExtent[scrollKey(id, false)]
	scrollMu.RUnlock()
	if ok && st.contentH > 0 {
		return st.contentH
	}
	return 0
}

// paintScrollbar 内容超出可视高度时，在右侧画一根细圆角条。
// 没有它，弹层里的长列表看起来就像「只有这几行」，用户不会去滚。
func paintScrollbar(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	id := scrollID(n)
	contentH := scrollCachedContentH(id)
	if contentH <= 0 {
		_, contentH = contentSize(ctx, n, n.W, style, th)
	}
	// 只高出一点点时滑块几乎铺满，看起来像卡片右缘的分割线。
	if contentH <= n.H+32 || n.H <= 0 {
		return
	}
	off := scrollGetOffset(id, false)
	maxOff := contentH - n.H
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	barY, barH, _, shown := scrollBarGeom(n)
	if !shown {
		return
	}
	// 胶囊滑块：离开右缘和上下圆角，不再贴边画成一条缝。
	bar := renderer.ColorFrom(th.Foreground)
	bar.A = 0.18
	wide := renderer.AttrBool(n, "widebar")
	bw, inset := float32(5), float32(12)
	if wide {
		bw, inset = 8, 18
	}
	ctx.FillRoundedRect(n.X+n.W-inset, barY, bw, barH, bw/2, bar)
}

// ScrollBy 按 id 直接滚一段（delta 正值向上，一格 = 1）。
// 弹层常常要「整块区域都能滚」：把面板级的 onWheel 转发到内部列表就靠它。
func ScrollBy(id string, delta float32) {
	if id == "" || delta == 0 {
		return
	}
	// 一格（delta=1，即 120 滚轮单位）≈ 1.7 行，接近系统的「3 行 / 格」手感。
	// 上界不在这里夹：交给 layoutScroll（它才知道容器真实的可滚范围）。
	off := scrollGetOffset(id, false) - delta*60
	if off < 0 {
		off = 0
	}
	scrollSetOffset(id, false, off)
}

// ScrollBottomGap 返回距离底部还有多少 DIP（0 表示已到底）。
// 用来判断「用户还在底部吗」——追加流式内容时只在这时贴底，别把翻历史的用户拽回来。
func ScrollBottomGap(id string) float32 {
	// 一次 RLock 里把 extent 与 offset 一起读走：分两次读会读到
	// 「新 extent + 旧 offset」这种不存在的组合。
	scrollMu.RLock()
	defer scrollMu.RUnlock()
	st, ok := scrollExtent[scrollKey(id, false)]
	if !ok {
		return 0
	}
	gap := st.maxOff - scrollOffset[scrollKey(id, false)]
	if gap < 0 {
		gap = 0
	}
	return gap
}

// ResetScroll 丢弃某个滚动容器的偏移（测试与重建界面时用）。
func ResetScroll(id string) {
	scrollDelOffset(id, false)
	scrollDelOffset(id, true)
}

// ScrollOffsetForTest 返回当前纵向滚动偏移（测试用）。
func ScrollOffsetForTest(id string) float32 {
	return scrollGetOffset(id, false)
}

// ScrollToEnd 把滚动容器滚到最底（内容不足一屏时等于不动）。
// 偏移量会在 layoutScroll 里被夹到合法范围，所以这里直接给一个大值即可。
func ScrollToEnd(id string) {
	if id == "" {
		return
	}
	scrollSetOffset(id, false, 1e9)
}

// ScrollOffset 测试用：读取纵向滚动偏移。
func ScrollOffset(id string) float32 { return scrollGetOffset(id, false) }

// SetScrollOffset 测试用。
func SetScrollOffset(id string, v float32) { scrollSetOffset(id, false, v) }

// ContentHeight 返回滚动内容相对容器顶部的高度。
func ContentHeight(n *runtime.VNode) float32 {
	if n == nil {
		return 0
	}
	var bottom float32
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		b := c.Y + c.H
		if b > bottom {
			bottom = b
		}
	}
	if bottom <= n.Y {
		return 0
	}
	return bottom - n.Y + n.Style.Padding
}
