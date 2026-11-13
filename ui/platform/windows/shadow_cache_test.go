//go:build windows

package windows

import "testing"

// TestShadowCacheCapAboveTypicalScene 保证阴影缓存上限足够容纳一个常见界面的
// 阴影种类数。
//
// 为什么值得单测：旧实现是 `len > 6 → releaseShadows()`。imgen 美化后一屏有
// 7 种以上不同尺寸的阴影（两张卡片 + 主按钮 + CTA + 徽章 + Logo + 插画），
// 于是**每一帧都把刚模糊好的位图全丢掉重算**，表现为打开设置对话框、悬停按钮
// 时明显卡顿。上限抬到 24 后这些都能常驻。
func TestShadowCacheCapAboveTypicalScene(t *testing.T) {
	const typicalScene = 12
	if shadowCacheMax <= typicalScene {
		t.Fatalf("shadowCacheMax = %d，应明显大于常见界面的 %d 种阴影", shadowCacheMax, typicalScene)
	}
}

// TestShadowEvictKeepsOthers 钉住淘汰策略：只丢一块，不清空。
// 这里只看 map 的条目数，不触碰 COM 句柄（假指针会被 release 解引用）。
func TestShadowEvictKeepsOthers(t *testing.T) {
	c := &d2dContext{shadows: map[shadowKey]com{}}
	for i := 0; i < shadowCacheMax; i++ {
		// 句柄留 0：release(0) 是空操作，可以安全地走淘汰路径。
		c.shadows[shadowKey{w: int32(i + 1), h: int32(i + 1)}] = 0
	}
	before := len(c.shadows)
	if before != shadowCacheMax {
		t.Fatalf("预置条目 = %d，期望 %d", before, shadowCacheMax)
	}
	c.evictOldestShadow()
	if got := len(c.shadows); got != before-1 {
		t.Fatalf("淘汰后条目 = %d，期望 %d（只淘汰一块）", got, before-1)
	}
	if len(c.shadows) == 0 {
		t.Fatal("缓存被整体清空：这正是每帧重算模糊的根因")
	}
}
