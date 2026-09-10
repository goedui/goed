package components

import (
	"strings"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// Dialog is a modal overlay drawn inside the application window.
type Dialog struct {
	Title   string
	Message string
	Width   int32
	Height  int32
	Visible bool
	Confirm bool // 确认模式：双按钮（确认/取消），Enter 确认、Esc 取消
	Triple  bool // 三按钮模式（Save/Don't Save/Cancel）：保存/不保存并继续/取消

	OnConfirm func() // 确认模式下的确认回调 / 三按钮模式的"保存"回调
	OnDiscard func() // 三按钮模式下的"不保存"回调
	OnCancel  func() // 确认/三按钮模式下的取消回调（X 按钮等同取消）

	onClose func()

	// 通用按钮组件：渲染（含文字居中）与命中委托给 Button，
	// Dialog 只保留模态关闭语义（单事件点击、Enter/Esc、X）。
	saveBtn    *Button
	cancelBtn  *Button
	discardBtn *Button
	saveLabel    string
	discardLabel string
	closeHot     bool
	buttonsInited bool
}

func NewDialog(title, message string) *Dialog {
	return &Dialog{
		Title:        title,
		Message:      message,
		Width:        480,
		Height:       220,
		saveLabel:    "保存",
		discardLabel: "不保存",
	}
}

// syncButtons 把当前弹出的按钮布局（位置/标签/可见性）同步到通用
// Button 组件。Render/事件处理前调用，保证命中与绘制一致。
func (d *Dialog) syncButtons(panel core.Rect) {
	if !d.buttonsInited {
		d.saveBtn = NewButton(d.saveLabel, 0, 0, 72, 30)
		d.cancelBtn = NewButton("取消", 0, 0, 72, 30)
		d.discardBtn = NewButton(d.discardLabel, 0, 0, 72, 30)
		d.buttonsInited = true
	}
	save := d.buttonBounds(panel)
	d.saveBtn.X, d.saveBtn.Y = save.X, save.Y
	d.saveBtn.Width, d.saveBtn.Height = save.W, save.H
	// 主按钮标签随模式变化：普通="确定"、确认="确认"、三按钮=自定义保存标签。
	switch {
	case d.Confirm && d.Triple:
		d.saveBtn.Text = d.saveLabel
	case d.Confirm:
		d.saveBtn.Text = "确认"
	default:
		d.saveBtn.Text = "确定"
	}
	cancel := d.cancelBounds(panel)
	d.cancelBtn.X, d.cancelBtn.Y = cancel.X, cancel.Y
	d.cancelBtn.Width, d.cancelBtn.Height = cancel.W, cancel.H
	discard := d.discardBounds(panel)
	d.discardBtn.X, d.discardBtn.Y = discard.X, discard.Y
	d.discardBtn.Width, d.discardBtn.Height = discard.W, discard.H
	d.discardBtn.Text = d.discardLabel
	d.discardBtn.Enabled = d.Confirm && d.Triple
	d.cancelBtn.Enabled = d.Confirm
}

func (d *Dialog) Show() {
	d.Confirm = false // 普通消息框语义：单按钮
	d.Triple = false
	d.Visible = true
}

// ShowConfirm 以确认/取消双按钮模式弹出；onConfirm 在用户确认后调用。
func (d *Dialog) ShowConfirm(title, message string, onConfirm func()) {
	d.Title = title
	d.Message = message
	d.Confirm = true
	d.Triple = false
	d.OnConfirm = onConfirm
	d.Visible = true
}

// ShowTriple 以三按钮模式弹出（VS Code 保存确认语义：保存 / 不保存 / 取消）。
// onSave 确认保存；onDiscard 放弃修改并继续；X 按钮、面板外点击或 Esc 取消。
func (d *Dialog) ShowTriple(title, message, saveLabel, discardLabel string, onSave, onDiscard func()) {
	d.Title = title
	d.Message = message
	d.Confirm = true
	d.Triple = true
	d.OnConfirm = onSave
	d.OnDiscard = onDiscard
	d.saveLabel = saveLabel
	d.discardLabel = discardLabel
	d.Visible = true
}

func (d *Dialog) Close() {
	if !d.Visible {
		return
	}
	d.Visible = false
	if d.onClose != nil {
		d.onClose()
	}
}

func (d *Dialog) IsOpen() bool {
	return d.Visible
}

func (d *Dialog) SetOnClose(fn func()) {
	d.onClose = fn
}

func (d *Dialog) Bounds(viewWidth, viewHeight int32) core.Rect {
	width, height := d.Width, d.Height
	if width <= 0 {
		width = 440
	}
	if height <= 0 {
		height = 220
	}
	return core.Rect{
		X: (viewWidth - width) / 2,
		Y: (viewHeight - height) / 2,
		W: width,
		H: height,
	}
}

func (d *Dialog) buttonBounds(panel core.Rect) core.Rect {
	return core.Rect{
		X: panel.X + panel.W - 96,
		Y: panel.Y + panel.H - 52,
		W: 72,
		H: 30,
	}
}

// cancelBounds 返回确认/三按钮模式下"取消"按钮的命中区域（最左侧）。
func (d *Dialog) cancelBounds(panel core.Rect) core.Rect {
	if d.Triple {
		// 三按钮：取消 | 不保存 | 保存。
		return core.Rect{
			X: panel.X + panel.W - 260,
			Y: panel.Y + panel.H - 52,
			W: 72,
			H: 30,
		}
	}
	return core.Rect{
		X: panel.X + panel.W - 178,
		Y: panel.Y + panel.H - 52,
		W: 72,
		H: 30,
	}
}

// discardBounds 返回三按钮模式下"不保存"按钮的命中区域（中间）。
func (d *Dialog) discardBounds(panel core.Rect) core.Rect {
	return core.Rect{
		X: panel.X + panel.W - 178,
		Y: panel.Y + panel.H - 52,
		W: 72,
		H: 30,
	}
}

// closeBounds 返回面板右上角 X 按钮的命中区域。
func (d *Dialog) closeBounds(panel core.Rect) core.Rect {
	return core.Rect{X: panel.X + panel.W - 32, Y: panel.Y + 8, W: 24, H: 24}
}

func contains(rect core.Rect, x, y int32) bool {
	return x >= rect.X && x < rect.X+rect.W && y >= rect.Y && y < rect.Y+rect.H
}

func (d *Dialog) HandleMouseMove(x, y, viewWidth, viewHeight int32) bool {
	if !d.Visible {
		return false
	}
	panel := d.Bounds(viewWidth, viewHeight)
	d.syncButtons(panel)
	oldClose := d.closeHot
	d.closeHot = contains(d.closeBounds(panel), x, y)
	d.saveBtn.HandleMouseMove(x, y)
	d.cancelBtn.HandleMouseMove(x, y)
	d.discardBtn.HandleMouseMove(x, y)
	return oldClose != d.closeHot
}

func (d *Dialog) HandleMouseDown(x, y, viewWidth, viewHeight int32) bool {
	if !d.Visible {
		return false
	}
	panel := d.Bounds(viewWidth, viewHeight)
	d.syncButtons(panel)
	// 单事件点击：命中按钮立即触发回调并关闭（模态对话框只收到
	// MouseDown，无完整 Down/Up 流；复用 Button 的命中判定）。
	switch {
	case d.saveBtn.HandleClick(x, y):
		confirmed := d.Confirm
		d.Close()
		if confirmed && d.OnConfirm != nil {
			d.OnConfirm()
		}
	case d.Confirm && d.Triple && d.discardBtn.HandleClick(x, y):
		discard := d.OnDiscard
		d.Close()
		if discard != nil {
			discard()
		}
	case d.Confirm && d.cancelBtn.HandleClick(x, y):
		d.Close()
		if d.OnCancel != nil {
			d.OnCancel()
		}
	case contains(d.closeBounds(panel), x, y):
		cancelled := d.Confirm
		d.Close()
		if cancelled && d.OnCancel != nil {
			d.OnCancel()
		}
	}
	// Consume all clicks while modal, including clicks outside the panel.
	return true
}

func (d *Dialog) HandleKeyDown(key int32) bool {
	if !d.Visible {
		return false
	}
	switch key {
	case 0x1B: // Escape：普通模式关闭；确认/三按钮模式等同取消
		cancelled := d.Confirm
		d.Close()
		if cancelled && d.OnCancel != nil {
			d.OnCancel()
		}
	case 0x0D: // Enter：确认/三按钮模式下提交主按钮（确认/保存）
		confirmed := d.Confirm
		d.Close()
		if confirmed && d.OnConfirm != nil {
			d.OnConfirm()
		}
	}
	return true
}

func (d *Dialog) Render(ctx core.DrawingContext, x, y, width, height int32) {
	if !d.Visible {
		return
	}
	th := theme.GetTheme()
	panel := d.Bounds(width, height)
	panel.X += x
	panel.Y += y

	// 遮罩：支持 alpha 的平台做半透明混合（VS Code 风格，背景可见但被
	// 压暗），否则回退主题定义的不透明遮罩色保持模态语义。
	if alphaCtx, ok := ctx.(core.AlphaDrawingContext); ok {
		alphaCtx.DrawRectAlpha(x, y, width, height, 0x000000, 110)
	} else {
		ctx.DrawRect(x, y, width, height, th.DialogOverlayBg, true)
	}
	ctx.DrawRoundedRect(panel.X, panel.Y, panel.W, panel.H, 6, th.DialogPanelBg, true)
	ctx.DrawRoundedRect(panel.X, panel.Y, panel.W, panel.H, 6, th.MenuBorder, false)

	// 字体阶梯（字体规范.md 第 4 节）：标题 Display 档。
	ctx.SetFont(th.UIFontFamily, th.FontDisplay())

	// 右上角 X 关闭按钮（hover 高亮，点击或 Esc/Enter 均可关闭）。
	closeBtn := d.closeBounds(panel)
	if d.closeHot {
		ctx.DrawRect(closeBtn.X, closeBtn.Y, closeBtn.W, closeBtn.H, th.TitleBarBtnHover, true)
	}
	cx, cy := closeBtn.X+closeBtn.W/2, closeBtn.Y+closeBtn.H/2
	DrawCloseIcon(ctx, cx, cy, th.TextPrimary)
	ctx.DrawText(panel.X+24, panel.Y+24, d.Title, th.TextPrimary)
	ctx.DrawRect(panel.X+24, panel.Y+56, panel.W-48, 1, th.WindowBorder, true)

	// 正文 Secondary 档。
	ctx.SetFont(th.UIFontFamily, th.FontSecondary())
	lineY := panel.Y + 78
	for _, line := range strings.Split(d.Message, "\n") {
		ctx.DrawText(panel.X+24, lineY, line, th.TextSecondary)
		lineY += 24
	}

	d.syncButtons(panel)
	// 三按钮模式：取消 | 不保存 | 保存；确认模式：取消 + 确认；否则单"确定"按钮。
	// 渲染委托通用 Button（含文字居中与悬停态）。
	if d.Confirm {
		d.cancelBtn.Render(ctx)
	}
	if d.Confirm && d.Triple {
		d.discardBtn.Render(ctx)
	}
	d.saveBtn.Render(ctx)
}

// renderPanelButton 以临时 Button 绘制面板内圆角按钮（InputDialog 等
// 无状态场景复用）：文字居中与悬停态由通用 Button 组件负责。
func renderPanelButton(ctx core.DrawingContext, rect core.Rect, label string, hot bool) {
	btn := NewButton(label, rect.X, rect.Y, rect.W, rect.H)
	btn.SetHovered(hot)
	btn.Render(ctx)
}
