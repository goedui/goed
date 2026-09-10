//go:build windows

package windows

import (
	"syscall"
	"testing"

	"github.com/goedui/goed/ui/theme"
)

// TestMeasureTextHeightIsGlyphHeight 验证 MeasureText 返回的高度是真实
// 字形高度（ascent+descent，TEXTMETRIC.tmHeight），而非含 internal leading
// 的字体单元高度。Button/Dialog 等组件用 (H-h)/2 做垂直居中，若 h 含
// leading 文字会视觉偏下（About 对话框"确定"按钮文字偏移 bug）。
func TestMeasureTextHeightIsGlyphHeight(t *testing.T) {
	th := theme.Dark()
	theme.SetTheme(th)

	// 屏幕 DC：测试环境始终可用，无需真实窗口。
	hdc := GetDC(0)
	if hdc == 0 {
		t.Skip("cannot acquire screen DC")
	}
	defer ReleaseDC(0, hdc)

	g := NewGDIContext(hdc, 200, 100)
	defer g.Cleanup()

	g.SetFont(th.UIFontFamily, th.UIFontSize)

	width, height := g.MeasureText("确定")
	if width <= 0 || height <= 0 {
		t.Fatalf("MeasureText = %d, %d; want positive extents", width, height)
	}

	// 字形高度必须 ≲ 字号 em 高度上限（14px 字号下通常 ~18-19px）。
	// 含 leading 的 cell 高度会明显超过该上限，此断言捕获回归。
	if height > th.UIFontSize+6 {
		t.Fatalf("MeasureText height = %d, want glyph height ≲ %d; font cell height leaked?",
			height, th.UIFontSize+6)
	}

	// 一致性：高度必须与 TEXTMETRIC 的 tmHeight（ascent+descent）一致。
	tm := g.textMetrics()
	if height != tm.tmHeight {
		t.Fatalf("MeasureText height %d != TEXTMETRIC tmHeight %d", height, tm.tmHeight)
	}
	_ = syscall.Handle(0)
}
