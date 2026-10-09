//go:build windows

package windows

import "testing"

// 滚轮坐标换算：窗口不在屏幕原点时，屏幕坐标必须减掉客户区原点，
// 否则命中判定会落到别处（症状：滚十几下才响应一次）。
func TestWheelClientDIP(t *testing.T) {
	// 窗口在 (400,250)，滚轮打在客户区 (500,150)px：屏幕坐标 (900,400)，
	// DPI 120 → 客户区 DIP = px / 1.25
	x, y := wheelClientDIP(900, 400, 400, 250, 120)
	if x != 400 || y != 120 {
		t.Fatalf("换算结果应为 (400,120)，实际 (%.1f,%.1f)", x, y)
	}
	// 窗口贴在原点时两者一致（这也是老代码「有时能用」的原因）
	x2, y2 := wheelClientDIP(500, 150, 0, 0, 120)
	if x2 != 400 || y2 != 120 {
		t.Fatalf("原点窗口换算结果应为 (400,120)，实际 (%.1f,%.1f)", x2, y2)
	}
	// dpi 兜底
	x3, _ := wheelClientDIP(500, 150, 0, 0, 0)
	if x3 != 500 {
		t.Fatalf("dpi<=0 应按 96 处理，实际 %.1f", x3)
	}
}
