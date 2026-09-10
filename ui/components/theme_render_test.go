package components

import (
	"testing"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// recordingContext 记录绘制用到的所有颜色，用于验证组件渲染确实取自主题
// 而不是散落的硬编码色值。
type recordingContext struct {
	colors map[uint32]bool
}

func newRecordingContext() *recordingContext {
	return &recordingContext{colors: make(map[uint32]bool)}
}

func (r *recordingContext) DrawRect(x, y, w, h int32, color uint32, filled bool) {
	r.colors[color] = true
}
func (r *recordingContext) DrawLine(x1, y1, x2, y2 int32, color uint32, width int32) {
	r.colors[color] = true
}
func (r *recordingContext) DrawRoundedRect(x, y, w, h, radius int32, color uint32, filled bool) {
	r.colors[color] = true
}
func (r *recordingContext) DrawEllipse(x, y, w, h int32, color uint32, filled bool) {
	r.colors[color] = true
}
func (r *recordingContext) DrawPolygon(points []core.Point, color uint32, filled bool) {
	r.colors[color] = true
}
func (r *recordingContext) DrawText(x, y int32, text string, color uint32) {
	r.colors[color] = true
}
func (r *recordingContext) MeasureText(text string) (int32, int32) {
	return int32(len(text) * 7), 14
}
func (r *recordingContext) SetFont(name string, size int32) {}
func (r *recordingContext) SetBkMode(transparent bool)      {}
func (r *recordingContext) BeginDraw()                      {}
func (r *recordingContext) EndDraw()                        {}
func (r *recordingContext) SetClipRect(x, y, w, h int32)    {}
func (r *recordingContext) ResetClip()                      {}

func (r *recordingContext) used(color uint32) bool { return r.colors[color] }

// themeColors 汇总当前主题中所有色值，用于判断渲染色是否来自主题。
func themeColors() map[uint32]bool {
	th := theme.GetTheme()
	all := []uint32{
		th.WindowBg, th.WindowBorder,
		th.TitleBarBg, th.TitleBarFg, th.TitleBarBtnHover, th.TitleBarBtnPress,
		th.TitleBarCloseHover, th.TitleBarClosePress,
		th.MenuBarBg, th.MenuBarHoverBg, th.MenuFg, th.MenuDropdownBg,
		th.MenuBorder, th.MenuSeparator, th.MenuItemHoverBg, th.MenuDisabledFg, th.MenuShortcutFg,
		th.StatusBarBg, th.StatusBarFg,
		th.EditorBg, th.EditorFg, th.EditorSelectionBg, th.EditorCursorColor,
		th.EditorLineNumberBg, th.EditorLineNumberFg, th.EditorCurrentLineBg,
		th.SyntaxKeyword, th.SyntaxString, th.SyntaxComment, th.SyntaxNumber, th.SyntaxFunction, th.SyntaxType,
		th.TabStatusModified, th.TabStatusUntracked, th.TabStatusIgnored, th.TabStatusReadOnly, th.TabStatusConflict,
		th.TextPrimary, th.TextSecondary, th.TextDisabled,
		th.ButtonBg, th.ButtonFg, th.ButtonHoverBg, th.ButtonPressBg,
		th.DialogOverlayBg, th.DialogPanelBg,
	}
	set := make(map[uint32]bool, len(all))
	for _, c := range all {
		set[c] = true
	}
	return set
}

// assertAllColorsFromTheme 断言渲染中出现的每种颜色都能在主题里找到出处。
func assertAllColorsFromTheme(t *testing.T, name string, ctx *recordingContext) {
	t.Helper()
	themed := themeColors()
	for color := range ctx.colors {
		if !themed[color] {
			t.Errorf("%s 渲染使用了主题外的硬编码颜色 0x%06X", name, color)
		}
	}
}

func TestContextMenuRendersWithThemeColors(t *testing.T) {
	theme.SetTheme(theme.Dark())
	menu := NewContextMenu()
	menu.Show(10, 10, []MenuItem{
		{Label: "复制", Shortcut: "Ctrl+C", Enabled: true, OnClick: func() {}},
		{Separator: true},
		{Label: "粘贴", Enabled: false},
	})

	ctx := newRecordingContext()
	menu.Render(ctx)

	th := theme.GetTheme()
	if !ctx.used(th.MenuDropdownBg) {
		t.Error("右键菜单背景应使用主题 MenuDropdownBg")
	}
	if !ctx.used(th.MenuBorder) {
		t.Error("右键菜单边框应使用主题 MenuBorder")
	}
	assertAllColorsFromTheme(t, "右键菜单", ctx)
}

func TestStatusBarRendersWithDarkPlusBlue(t *testing.T) {
	theme.SetTheme(theme.Dark())
	bar := NewStatusBar()
	bar.SetLeft("Ln 1, Col 1")
	bar.SetRight("UTF-8")

	ctx := newRecordingContext()
	bar.Render(ctx, 0, 100, 800)

	th := theme.GetTheme()
	if !ctx.used(th.StatusBarBg) {
		t.Errorf("状态栏背景应使用主题 StatusBarBg (#007ACC Dark+ 蓝)")
	}
	assertAllColorsFromTheme(t, "状态栏", ctx)
}

func TestDialogRendersWithThemeColors(t *testing.T) {
	theme.SetTheme(theme.Dark())
	dialog := NewDialog("关于", "Goed v0.2")
	dialog.Show()

	ctx := newRecordingContext()
	dialog.Render(ctx, 0, 0, 800, 600)

	th := theme.GetTheme()
	if !ctx.used(th.DialogOverlayBg) {
		t.Error("对话框遮罩应使用主题 DialogOverlayBg")
	}
	if !ctx.used(th.DialogPanelBg) {
		t.Error("对话框面板应使用主题 DialogPanelBg")
	}
	assertAllColorsFromTheme(t, "对话框", ctx)
}
