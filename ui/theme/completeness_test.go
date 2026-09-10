package theme

import (
	"fmt"
	"testing"
)

// TestBuiltinThemesHaveNoZeroColors 防止内置主题遗漏字段：uint32 颜色零值
// 是黑色，遗漏会在特定主题下把面板染黑（如 Light 模式资源管理器）。
func TestBuiltinThemesHaveNoZeroColors(t *testing.T) {
	builders := map[string]func() *Theme{
		"Dark":  Dark,
		"Light": Light,
		"Tabby": Tabby,
	}

	for name, build := range builders {
		th := build()
		if th.Name != name {
			t.Errorf("%s(): Name = %q, want %q", name, th.Name, name)
		}
		// 仅检查背景类字段：黑色 (0,0,0) 是 Light 等明亮主题的合法前景色，
		// 前景字段无法用零值区分"未设置"与"真黑色"。
		checks := []struct {
			field string
			value uint32
		}{
			{"WindowBg", th.WindowBg},
			{"TitleBarBg", th.TitleBarBg},
			{"MenuBarBg", th.MenuBarBg},
			{"EditorBg", th.EditorBg},
			{"EditorCurrentLineBg", th.EditorCurrentLineBg},
			{"EditorGutterBg", th.EditorGutterBg},
			{"EditorLineNumberBg", th.EditorLineNumberBg},
			{"StatusBarBg", th.StatusBarBg},
			{"DialogPanelBg", th.DialogPanelBg},
			{"ScrollTrackBg", th.ScrollTrackBg},
			{"MenuDropdownBg", th.MenuDropdownBg},
		}
		for _, c := range checks {
			if c.value == 0 {
				t.Errorf("%s theme has zero (black) %s", name, c.field)
			}
		}
		if th.TitleBarFontSize <= 0 || th.TabsFontSize <= 0 {
			t.Errorf("%s theme has unset font sizes (TitleBar=%d, Tabs=%d)",
				name, th.TitleBarFontSize, th.TabsFontSize)
		}
	}
}

// TestTypographyConsistency 固化字体规范.md 第 6 节：主题切换只换
// 颜色不换排版——所有主题的字号与字体族必须一致。
func TestTypographyConsistency(t *testing.T) {
	base := Dark()
	for _, th := range []struct {
		name string
		get  func() *Theme
	}{{"Light", Light}, {"Tabby", Tabby}} {
		other := th.get()
		if other.UIFontSize != base.UIFontSize {
			t.Errorf("%s UIFontSize = %d, want %d (unified across themes)", th.name, other.UIFontSize, base.UIFontSize)
		}
		if other.EditorFontSize != base.EditorFontSize {
			t.Errorf("%s EditorFontSize = %d, want %d", th.name, other.EditorFontSize, base.EditorFontSize)
		}
		if other.TitleBarFontSize != base.TitleBarFontSize {
			t.Errorf("%s TitleBarFontSize = %d, want %d", th.name, other.TitleBarFontSize, base.TitleBarFontSize)
		}
		if other.TabsFontSize != base.TabsFontSize {
			t.Errorf("%s TabsFontSize = %d, want %d", th.name, other.TabsFontSize, base.TabsFontSize)
		}
		if other.EditorFontFamily != base.EditorFontFamily {
			t.Errorf("%s EditorFontFamily = %q, want %q", th.name, other.EditorFontFamily, base.EditorFontFamily)
		}
		if other.UIFontFamily != base.UIFontFamily {
			t.Errorf("%s UIFontFamily = %q, want %q", th.name, other.UIFontFamily, base.UIFontFamily)
		}
		// 最小可读字号约束（规范第 4 节）。
		if base.UIFontSize-1 < 13 {
			t.Errorf("Secondary tier (%d) below 13px minimum", base.UIFontSize-1)
		}
		// 行高比例约束（规范第 5 节）：LineHeight 由编辑器维护，
		// 此处验证字号满足 1.5 比例下 24px 行高的推导基准。
		if base.EditorFontSize+6 > 24 {
			t.Errorf("EditorFontSize %d violates LineHeight >= size+6", base.EditorFontSize)
		}
	}
}

// ExampleNames 展示内置主题名。
func ExampleNames() {
	fmt.Println(len(Names()) >= 3)
	// Output: true
}
