package components

import (
	"strings"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// InputDialog 单行文本输入对话框（重命名、新建文件/文件夹等场景）。
// Enter 或"确认"按钮提交非空值；Esc 或点击面板外取消。
type InputDialog struct {
	Title   string
	Prompt  string
	Value   []rune
	Visible bool

	OnConfirm func(value string)

	confirmHot bool
	cancelHot  bool
}

func NewInputDialog() *InputDialog {
	return &InputDialog{}
}

// Show 弹出输入框；initial 为初始文本（重命名场景填旧名称便于直接修改）。
func (d *InputDialog) Show(title, prompt, initial string) {
	d.Title = title
	d.Prompt = prompt
	d.Value = []rune(initial)
	d.Visible = true
}

func (d *InputDialog) Hide() {
	d.Visible = false
}

func (d *InputDialog) IsOpen() bool {
	return d.Visible
}

func (d *InputDialog) panelBounds(viewWidth, viewHeight int32) core.Rect {
	width, height := int32(440), int32(190)
	if width > viewWidth {
		width = viewWidth
	}
	return core.Rect{
		X: (viewWidth - width) / 2,
		Y: (viewHeight - height) / 2,
		W: width,
		H: height,
	}
}

func (d *InputDialog) inputBounds(panel core.Rect) core.Rect {
	return core.Rect{X: panel.X + 24, Y: panel.Y + 88, W: panel.W - 48, H: 30}
}

func (d *InputDialog) confirmBounds(panel core.Rect) core.Rect {
	return core.Rect{X: panel.X + panel.W - 96, Y: panel.Y + panel.H - 52, W: 72, H: 30}
}

func (d *InputDialog) cancelBounds(panel core.Rect) core.Rect {
	return core.Rect{X: panel.X + panel.W - 178, Y: panel.Y + panel.H - 52, W: 72, H: 30}
}

func (d *InputDialog) HandleMouseMove(x, y, viewWidth, viewHeight int32) bool {
	if !d.Visible {
		return false
	}
	panel := d.panelBounds(viewWidth, viewHeight)
	oldConfirm, oldCancel := d.confirmHot, d.cancelHot
	d.confirmHot = contains(d.confirmBounds(panel), x, y)
	d.cancelHot = contains(d.cancelBounds(panel), x, y)
	return oldConfirm != d.confirmHot || oldCancel != d.cancelHot
}

// HandleMouseDown 模态消费所有点击：确认/取消按钮、面板外点击（视为取消）。
func (d *InputDialog) HandleMouseDown(x, y, viewWidth, viewHeight int32) bool {
	if !d.Visible {
		return false
	}
	panel := d.panelBounds(viewWidth, viewHeight)
	switch {
	case contains(d.confirmBounds(panel), x, y):
		d.submit()
	case contains(d.cancelBounds(panel), x, y):
		d.Hide()
	case !contains(panel, x, y):
		d.Hide()
	}
	return true
}

// HandleChar 追加可打印字符；控制字符交由 HandleKeyDown 处理。
func (d *InputDialog) HandleChar(c rune) bool {
	if !d.Visible || c < 0x20 {
		return d.Visible
	}
	d.Value = append(d.Value, c)
	return true
}

// HandleKeyDown 处理编辑键：Enter 确认、Esc 取消、Backspace 删除末字符。
func (d *InputDialog) HandleKeyDown(key int32) bool {
	if !d.Visible {
		return false
	}
	switch key {
	case 0x1B: // Escape 取消
		d.Hide()
	case 0x0D: // Enter 确认
		d.submit()
	case 0x08: // Backspace
		if len(d.Value) > 0 {
			d.Value = d.Value[:len(d.Value)-1]
		}
	}
	return true
}

// submit 提交非空值（去首尾空白）；空值视为无效输入不关闭。
func (d *InputDialog) submit() {
	value := strings.TrimSpace(string(d.Value))
	if value == "" {
		return
	}
	d.Hide()
	if d.OnConfirm != nil {
		d.OnConfirm(value)
	}
}

func (d *InputDialog) Render(ctx core.DrawingContext, viewWidth, viewHeight int32) {
	if !d.Visible {
		return
	}
	th := theme.GetTheme()
	panel := d.panelBounds(viewWidth, viewHeight)

	// 遮罩：与 Dialog 相同的模态语义（支持 alpha 平台半透明压暗）。
	if alphaCtx, ok := ctx.(core.AlphaDrawingContext); ok {
		alphaCtx.DrawRectAlpha(0, 0, viewWidth, viewHeight, 0x000000, 110)
	} else {
		ctx.DrawRect(0, 0, viewWidth, viewHeight, th.DialogOverlayBg, true)
	}
	ctx.DrawRoundedRect(panel.X, panel.Y, panel.W, panel.H, 6, th.DialogPanelBg, true)
	ctx.DrawRoundedRect(panel.X, panel.Y, panel.W, panel.H, 6, th.MenuBorder, false)

	// 标题 Display 档、说明 Secondary 档（字体规范.md 第 4 节）。
	ctx.SetFont(th.UIFontFamily, th.FontDisplay())
	ctx.DrawText(panel.X+24, panel.Y+24, d.Title, th.TextPrimary)
	ctx.SetFont(th.UIFontFamily, th.FontSecondary())
	ctx.DrawText(panel.X+24, panel.Y+58, d.Prompt, th.TextSecondary)

	// 输入框：文本末尾常驻光标（单行简化语义）。
	box := d.inputBounds(panel)
	ctx.DrawRect(box.X, box.Y, box.W, box.H, th.EditorBg, true)
	ctx.DrawRect(box.X, box.Y, box.W, box.H, th.MenuBorder, false)
	ctx.SetFont(th.UIFontFamily, th.UIFontSize)
	value := string(d.Value)
	ctx.DrawText(box.X+8, box.Y+8, value, th.TextPrimary)
	if width, _ := ctx.MeasureText(value); width+16 <= box.W-4 {
		ctx.DrawRect(box.X+8+width+1, box.Y+7, 1, box.H-14, th.TextPrimary, true)
	}

	renderPanelButton(ctx, d.cancelBounds(panel), "取消", d.cancelHot)
	renderPanelButton(ctx, d.confirmBounds(panel), "确认", d.confirmHot)
}
