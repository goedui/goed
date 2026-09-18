//go:build windows

package windows

import "testing"

// TestDipToPixels 校验 DIP -> 物理像素的换算。
//
// 这条换算错了窗口就会「比预期小一圈」：1350 DIP 在 125% 缩放下应该是
// 1687 像素，少乘一次缩放会得到 1350，界面看起来正好差一个边距。
// 这正是当初排查「窗口缩不小 / 内容被裁」时踩过的坑。
func TestDipToPixels(t *testing.T) {
	cases := []struct {
		name string
		dip  float32
		dpi  float32
		want int32
	}{
		{"100% 不缩放", 1350, 96, 1350},
		{"125% 缩放", 1350, 120, 1688}, // 1350*1.25 = 1687.5 -> 四舍五入 1688
		{"150% 缩放", 1080, 144, 1620},
		{"200% 缩放", 900, 192, 1800},
		{"dpi 为 0 时按 96 处理", 800, 0, 800},
		{"负数 dpi 也按 96 处理", 800, -1, 800},
		{"过小的值夹到 1 而不是 0", 0.1, 96, 1},
		{"零值夹到 1", 0, 96, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dipToPixels(c.dip, c.dpi); got != c.want {
				t.Errorf("dipToPixels(%v, %v) = %d，期望 %d", c.dip, c.dpi, got, c.want)
			}
		})
	}
}

// TestPixelsToDIP 是 dipToPixels 的反向换算，两者必须能来回对上。
func TestPixelsToDIP(t *testing.T) {
	cases := []struct {
		name string
		px   uint32
		dpi  float32
		want int32
	}{
		{"100%", 1350, 96, 1350},
		{"125%", 1688, 120, 1350},
		{"150%", 1620, 144, 1080},
		{"200%", 1800, 192, 900},
		{"dpi 为 0 时按 96 处理", 800, 0, 800},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pixelsToDIP(c.px, c.dpi); got != c.want {
				t.Errorf("pixelsToDIP(%d, %v) = %d，期望 %d", c.px, c.dpi, got, c.want)
			}
		})
	}
}

// TestFitToScreen 校验窗口左上角的夹取：缩小窗口时若不夹，窗口会跑到屏幕外。
func TestFitToScreen(t *testing.T) {
	const screenW, screenH = 1920, 1080

	cases := []struct {
		name         string
		x, y, w, h   int32
		wantX, wantY int32
	}{
		{"完全放得下则不动", 300, 200, 900, 620, 300, 200},
		{"贴住右下角则夹回来", 1500, 900, 900, 620, 1020, 460},
		{"比屏幕还大则夹到 0,0", 100, 100, 2400, 1400, 0, 0},
		{"负坐标拉回 0", -50, -30, 900, 620, 0, 0},
		{"x 越界 y 正常", 1800, 100, 900, 620, 1020, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotX, gotY := fitToScreen(c.x, c.y, c.w, c.h, screenW, screenH)
			if gotX != c.wantX || gotY != c.wantY {
				t.Errorf("fitToScreen(%d,%d,%d,%d) = (%d,%d)，期望 (%d,%d)",
					c.x, c.y, c.w, c.h, gotX, gotY, c.wantX, c.wantY)
			}
		})
	}
}
