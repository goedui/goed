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
}

// 滚动状态是**包级共享**的（按容器 id 索引），而碰它的有两类 goroutine：
// 渲染 / 布局那条线，以及业务侧可能在后台线程调用的 ScrollToEnd / ScrollBottomGap
// （典型的是「回复延迟 700ms 后贴底」这种定时器）。以前两个 map 裸奔，-race 会报
// 「两个 goroutine 同时 mapassign 同一个 map」——那不只是未定义行为：Go 运行时会
// 直接以 "concurrent map writes" 结束进程。所以全部走下面这组带锁入口。
//
// 锁只圈住 map 操作本身，**不**跨越 layout / paint 里的子节点递归
// （嵌套滚动容器会重入 layoutScroll，锁持有太久会自锁）。
var (
	scrollMu     sync.RWMutex
	scrollOffset = map[string]float32{}
	// scrollExtent 记录每个容器的可滚范围与可视高度（判断是否贴底用）。
	scrollExtent = map[string]scrollExtentEntry{}
)

type scrollExtentEntry struct {
	maxOff float32
	viewH  float32
}

func scrollGetOffset(id string) float32 {
	scrollMu.RLock()
	v := scrollOffset[id]
	scrollMu.RUnlock()
	return v
}

func scrollSetOffset(id string, v float32) {
	scrollMu.Lock()
	scrollOffset[id] = v
	scrollMu.Unlock()
}

func scrollDelOffset(id string) {
	scrollMu.Lock()
	delete(scrollOffset, id)
	scrollMu.Unlock()
}

// scrollSetLayout 一次写完「夹过的偏移 + 可滚范围」：两者是同一帧布局的产物，
// 分两次写会让别的 goroutine 读到半新半旧的组合。
func scrollSetLayout(id string, off float32, e scrollExtentEntry) {
	scrollMu.Lock()
	scrollOffset[id] = off
	scrollExtent[id] = e
	scrollMu.Unlock()
}

// Scroll 垂直滚动容器。
func Scroll(parts ...any) *runtime.VNode {
	return runtime.H("scroll", parts...)
}

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

func contentSize(ctx renderer.Context, n *runtime.VNode, maxW float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	var w, h float32
	gap := n.Style.Gap
	pad := n.Style.Padding
	first := true
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, maxW-2*pad, 0, style, th)
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
	_, contentH := contentSize(ctx, n, w, style, th)
	id := scrollID(n)
	off := scrollGetOffset(id)
	maxOff := contentH - h
	if maxOff < 0 {
		maxOff = 0
	}
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	scrollSetLayout(id, off, scrollExtentEntry{maxOff: maxOff, viewH: h})
	pad := n.Style.Padding
	gap := n.Style.Gap
	cy := y + pad - off
	innerW := w - 2*pad
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		cw, ch := renderer.MeasureNode(ctx, c, innerW, 0, style, th)
		renderer.LayoutNode(ctx, c, x+pad, cy, innerW, ch, style, th)
		_ = cw
		cy += ch + gap
	}
}

func paintScroll(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	ctx.PushClip(n.X, n.Y, n.W, n.H)
	renderer.PaintChildren(ctx, n, style, th)
	ctx.PopClip()
	paintScrollbar(ctx, n, style, th)
}

// paintScrollbar 内容超出可视高度时，在右侧画一根细圆角条。
// 没有它，弹层里的长列表看起来就像「只有这几行」，用户不会去滚。
func paintScrollbar(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	_, contentH := contentSize(ctx, n, n.W, style, th)
	if contentH <= n.H+1 || n.H <= 0 {
		return
	}
	off := scrollGetOffset(scrollID(n))
	maxOff := contentH - n.H
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	barH := n.H * n.H / contentH
	if barH < 24 {
		barH = 24
	}
	barY := n.Y
	if maxOff > 0 {
		barY = n.Y + (n.H-barH)*(off/maxOff)
	}
	// 用前景色兑一点透明度：比 Separator 明显，深色浅色主题都看得见。
	bar := renderer.ColorFrom(th.Foreground)
	bar.A = 0.30
	ctx.FillRoundedRect(n.X+n.W-5.5, barY+2, 3.5, barH-4, 1.75, bar)
}

// ScrollBy 按 id 直接滚一段（delta 正值向上，一格 = 1）。
// 弹层常常要「整块区域都能滚」：把面板级的 onWheel 转发到内部列表就靠它。
func ScrollBy(id string, delta float32) {
	if id == "" || delta == 0 {
		return
	}
	// 一格（delta=1，即 120 滚轮单位）≈ 1.7 行，接近系统的「3 行 / 格」手感。
	// 上界不在这里夹：交给 layoutScroll（它才知道容器真实的可滚范围）。
	off := scrollGetOffset(id) - delta*60
	if off < 0 {
		off = 0
	}
	scrollSetOffset(id, off)
}

// ScrollBottomGap 返回距离底部还有多少 DIP（0 表示已到底）。
// 用来判断「用户还在底部吗」——追加流式内容时只在这时贴底，别把翻历史的用户拽回来。
func ScrollBottomGap(id string) float32 {
	// 一次 RLock 里把 extent 与 offset 一起读走：分两次读会读到
	// 「新 extent + 旧 offset」这种不存在的组合。
	scrollMu.RLock()
	defer scrollMu.RUnlock()
	st, ok := scrollExtent[id]
	if !ok {
		return 0
	}
	gap := st.maxOff - scrollOffset[id]
	if gap < 0 {
		gap = 0
	}
	return gap
}

// ResetScroll 丢弃某个滚动容器的偏移（测试与重建界面时用）。
func ResetScroll(id string) {
	scrollDelOffset(id)
}

// ScrollOffsetForTest 返回当前滚动偏移（测试用）。
func ScrollOffsetForTest(id string) float32 {
	return scrollGetOffset(id)
}

// ScrollToEnd 把滚动容器滚到最底（内容不足一屏时等于不动）。
// 偏移量会在 layoutScroll 里被夹到合法范围，所以这里直接给一个大值即可。
func ScrollToEnd(id string) {
	if id == "" {
		return
	}
	scrollSetOffset(id, 1e9)
}

// HandleWheel 把滚轮交给命中的节点：优先 Attrs["onWheel"]，否则交给 scroll。
// delta 正值向上（一格 = 1）。
func HandleWheel(nodes []*runtime.VNode, x, y, delta float32) {
	var scrollTarget, wheelTarget *runtime.VNode
	runtime.Walk(nodes, func(n *runtime.VNode) bool {
		if n == nil || !renderer.Contains(n, x, y) {
			return false
		}
		if n.Tag == "scroll" {
			scrollTarget = n
		}
		if n.Attrs != nil {
			if _, ok := n.Attrs["onWheel"]; ok {
				wheelTarget = n
			}
		}
		return false
	})
	if wheelTarget != nil {
		switch fn := wheelTarget.Attrs["onWheel"].(type) {
		case func(float32):
			fn(delta)
			return
		case func(float32, float32, float32):
			fn(x, y, delta)
			return
		}
	}
	if scrollTarget == nil {
		return
	}
	ScrollBy(scrollID(scrollTarget), delta)
}

// ScrollOffset 测试用：读取滚动偏移。
func ScrollOffset(id string) float32 { return scrollGetOffset(id) }

// SetScrollOffset 测试用。
func SetScrollOffset(id string, v float32) { scrollSetOffset(id, v) }

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
