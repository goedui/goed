package renderer

import (
	"fmt"
	"sync"
)

// 跨帧文本测量缓存。
//
// 布局阶段每个文本节点都要向后端问一次尺寸（DirectWrite 要取 TextFormat 再量一次），
// 而重建帧里绝大多数文本跟上一帧一字不差——md-previewer 那种「改一个字整棵树重建」
// 的场景，2000 次测量里真正新的只有几十次。键含文本、限宽与所有影响结果的字体字段。
//
// 按后端隔离：PaintTree 每帧比对 Context 的类型与 DPI，换了就整体清空 —— 同一个
// 测试进程会造多个假 Context，各自测量公式不同，不能互相串。

type measureKey struct {
	text   string
	limit  float32
	family string
	size   float32
	weight int
	italic bool
	nowrap bool
}

var (
	measureMu    sync.Mutex
	measureCache = map[measureKey][2]float32{}
	measureOrder []measureKey
	measureOwner string
)

// measureCacheCap 是缓存条数上限。超了丢最早的四分之一：文本测量是幂等的，
// 丢了只是重算一次，不会算错。
const measureCacheCap = 4096

// ResetMeasureCache 清空文本测量缓存。PaintTree 换后端时会自动调用；
// 测试想从冷缓存开始也可以自己调。
func ResetMeasureCache() {
	measureMu.Lock()
	measureCache = map[measureKey][2]float32{}
	measureOrder = measureOrder[:0]
	measureMu.Unlock()
}

// syncMeasureOwner 在每帧布局前对齐缓存归属的后端。
func syncMeasureOwner(ctx Context) {
	owner := ""
	if ctx != nil {
		// %T 区分后端实现（D2D / 软件 / 各种假 Context），DPI 区分缩放。
		owner = fmt.Sprintf("%T|%.2f", ctx, ctx.DPI())
	}
	measureMu.Lock()
	changed := owner != measureOwner
	measureOwner = owner
	measureMu.Unlock()
	if changed {
		ResetMeasureCache()
	}
}

// measureTextCached 是布局用的测量入口。命中缓存时不碰后端。
func measureTextCached(ctx Context, text string, limit float32, style TextStyle) (float32, float32) {
	if ctx == nil {
		return 0, 0
	}
	k := measureKey{
		text:   text,
		limit:  limit,
		family: style.FontFamily,
		size:   style.FontSize,
		weight: style.Weight,
		italic: style.Italic,
		nowrap: style.NoWrap,
	}
	measureMu.Lock()
	if v, ok := measureCache[k]; ok {
		measureMu.Unlock()
		return v[0], v[1]
	}
	measureMu.Unlock()

	// 后端调用可能很慢（DirectWrite 要建 TextFormat），不要在持锁时做。
	w, h := ctx.MeasureText(text, limit, style)

	measureMu.Lock()
	if _, ok := measureCache[k]; !ok {
		if len(measureOrder) >= measureCacheCap {
			drop := measureCacheCap / 4
			for _, old := range measureOrder[:drop] {
				delete(measureCache, old)
			}
			measureOrder = append(measureOrder[:0], measureOrder[drop:]...)
		}
		measureCache[k] = [2]float32{w, h}
		measureOrder = append(measureOrder, k)
	}
	measureMu.Unlock()
	return w, h
}
