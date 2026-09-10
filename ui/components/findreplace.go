package components

import (
	"strconv"
	"strings"

	"github.com/goedui/goed/ui/composables"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// FindReplace 是悬浮在编辑器右上方的查找/替换小组件（VSCode 风格）。
// 定位约定：锚定编辑器视口内右上角，位于文件页签栏之下。宿主在 Render/
// 命中测试前先调用 SetPlacement 声明编辑器几何，组件据此计算自身边界。
type FindReplace struct {
	Visible     bool
	ShowReplace bool // 展开替换输入行（Ctrl+H）

	Query     string // 查找词
	Replace   string // 替换词
	MatchCase bool   // 区分大小写

	// 匹配状态：每次 Render 时基于宿主编辑器重算。
	Matches []composables.SearchResult
	Current int // 当前聚焦的匹配下标；-1 表示无聚焦

	// OnChange 在查询/替换文本或选项变化后回调（宿主重绘高亮）。
	OnChange func()
	// OnNavigate 在用户跳转上/下一个匹配后回调（宿主滚动编辑器）。
	OnNavigate func(line, col, length int)

	// 输入焦点：0=查找框 1=替换框；-1 无
	focus int
	// ReplaceRow 控制替换行的可见性（与 ShowReplace 同步，由切换按钮驱动）
	// 布局缓存（Render 时刷新，供命中测试使用）
	panel            core.Rect
	findBox          core.Rect
	replaceBox       core.Rect
	btnPrev          core.Rect
	btnNext          core.Rect
	btnCase          core.Rect
	btnReplaceToggle core.Rect
	btnClose         core.Rect
	btnReplaceOne    core.Rect
	btnReplaceAll    core.Rect
	hotClose         bool
	hotPrev          bool
	hotNext          bool
	hotCase          bool
	hotReplaceToggle bool
	hotReplaceOne    bool
	hotReplaceAll    bool

	// 宿主注入的编辑器操作（避免组件直接依赖 Editor 的未导出状态）
	getText func() string
	// applyReplace 由宿主实现：替换 [line,col] 起 length 个字符为 rep。
	applyReplace func(line, col, length int, rep string)
	// reveal 由宿主实现：滚动确保匹配可见并按长度选中结果文本。
	reveal func(line, col, length int)

	width int32
}

// FindReplaceWidgetWidth 面板宽度（供宿主预留/测试）。
const FindReplaceWidgetWidth = 460

// NewFindReplace 创建查找替换组件；宿主通过 getter/apply 回调接入编辑器。
func NewFindReplace(getText func() string, applyReplace func(line, col, length int, rep string), reveal func(line, col, length int)) *FindReplace {
	return &FindReplace{
		Current:      -1,
		focus:        0,
		width:        FindReplaceWidgetWidth,
		getText:      getText,
		applyReplace: applyReplace,
		reveal:       reveal,
	}
}

// Show 打开组件；seed 传当前选中文本可作为初始查找词。
func (f *FindReplace) Show(seed string) {
	f.Visible = true
	if seed != "" && !strings.ContainsAny(seed, "\n\r") {
		f.Query = seed
	}
	f.focus = 0
	f.refreshMatches()
}

// Hide 关闭组件并清空匹配高亮状态。
func (f *FindReplace) Hide() {
	f.Visible = false
	f.Matches = nil
	f.Current = -1
	f.hotClose, f.hotPrev, f.hotNext = false, false, false
	if f.OnChange != nil {
		f.OnChange()
	}
}

// Toggle 显示/隐藏。
func (f *FindReplace) Toggle(seed string) {
	if f.Visible {
		f.Hide()
	} else {
		f.Show(seed)
	}
}

// ToggleReplace 展开/收起替换输入行（Ctrl+H）。
func (f *FindReplace) ToggleReplace() {
	f.ShowReplace = !f.ShowReplace
	if f.ShowReplace {
		f.focus = 1
	}
	f.refreshMatches()
}

// IsOpen 报告组件是否可见。
func (f *FindReplace) IsOpen() bool { return f.Visible }

// options 组装搜索选项。
func (f *FindReplace) options() composables.SearchOptions {
	return composables.SearchOptions{CaseSensitive: f.MatchCase}
}

// refreshMatches 重算全部匹配。查询为空或编辑器无内容时清空。
func (f *FindReplace) refreshMatches() {
	if !f.Visible || f.Query == "" || f.getText == nil {
		f.Matches = nil
		f.Current = -1
		return
	}
	f.Matches = composables.Find(f.getText(), f.Query, f.options())
	if len(f.Matches) == 0 {
		f.Current = -1
	} else if f.Current >= len(f.Matches) {
		f.Current = 0
	}
}

// Changed 在输入变化后由输入处理路径调用。
func (f *FindReplace) Changed() {
	f.refreshMatches()
	// 输入新查询时聚焦首个匹配，便于 immediately 定位。
	if len(f.Matches) > 0 {
		f.Current = 0
		f.revealCurrent()
	}
	if f.OnChange != nil {
		f.OnChange()
	}
}

// Next 跳到下一个匹配（循环）。
func (f *FindReplace) Next() {
	if len(f.Matches) == 0 {
		return
	}
	f.Current = (f.Current + 1) % len(f.Matches)
	f.revealCurrent()
}

// Prev 跳到上一个匹配（循环）。
func (f *FindReplace) Prev() {
	if len(f.Matches) == 0 {
		return
	}
	f.Current = (f.Current - 1 + len(f.Matches)) % len(f.Matches)
	f.revealCurrent()
}

func (f *FindReplace) revealCurrent() {
	if f.Current < 0 || f.Current >= len(f.Matches) {
		return
	}
	m := f.Matches[f.Current]
	if f.reveal != nil {
		f.reveal(m.Position.Line, m.Position.Column, m.Length)
	}
	if f.OnNavigate != nil {
		f.OnNavigate(m.Position.Line, m.Position.Column, m.Length)
	}
}

// ReplaceCurrent 替换当前聚焦匹配并跳到下一个。
func (f *FindReplace) ReplaceCurrent() {
	if f.Current < 0 || f.Current >= len(f.Matches) || f.applyReplace == nil {
		return
	}
	m := f.Matches[f.Current]
	f.applyReplace(m.Position.Line, m.Position.Column, m.Length, f.Replace)
	// 重新搜索后保持相对位置。
	f.refreshMatches()
	if len(f.Matches) > 0 {
		f.Current = f.Current % len(f.Matches)
		f.revealCurrent()
	}
	if f.OnChange != nil {
		f.OnChange()
	}
}

// ReplaceAll 替换全部匹配，返回替换数量。
func (f *FindReplace) ReplaceAll() int {
	if f.applyReplace == nil || f.Query == "" {
		return 0
	}
	// 倒序逐个替换，位置始终有效。
	n := 0
	for i := len(f.Matches) - 1; i >= 0; i-- {
		m := f.Matches[i]
		f.applyReplace(m.Position.Line, m.Position.Column, m.Length, f.Replace)
		n++
	}
	f.refreshMatches()
	f.Current = -1
	if f.OnChange != nil {
		f.OnChange()
	}
	return n
}

// StatusText 返回 "n of m" 风格的匹配状态（无结果时提示）。
func (f *FindReplace) StatusText() string {
	if f.Query == "" {
		return ""
	}
	if len(f.Matches) == 0 {
		return "无结果"
	}
	if f.Current < 0 {
		return strconv.Itoa(len(f.Matches)) + " 个结果"
	}
	return strconv.Itoa(f.Current+1) + "/" + strconv.Itoa(len(f.Matches))
}

// --- 布局 ---

// panelHeight 高度依赖替换行是否展开：单行 40px，两行 76px。
func (f *FindReplace) panelHeight() int32 {
	h := int32(40) // 单行面板（查找行 8 + 26 + 6 余量）
	if f.ShowReplace {
		h = int32(76) // 8 + 26 + 8 + 26 + 8
	}
	return h
}

// SetPlacement 在 Render/命中测试前声明编辑器几何（全局坐标）。
// 组件锚定编辑器视口右上角、向下偏移 8px；右距预留 minimap
// 宽度 + 安全间距，避免遮挡缩略图。
//
// 布局（VSCode 同款）：
//   [⌄] [查找输入......] [n/m] ↑ ↓ Aa  ✕
//   [⌄] [替换输入......]    ⇥  ⇤
// chevron 位于面板最左侧，点击在单行/两行间切换（仅两行时收起为单行）。
func (f *FindReplace) SetPlacement(editorX, editorY, editorW, editorH int32) {
	w := f.width
	if w > editorW-16 {
		w = editorW - 16
	}
	if w < 240 {
		w = 240
	}
	const minimapGap = 110 // minimap 宽度 + 安全间距（再向左偏移一些）
	f.panel = core.Rect{X: editorX + editorW - w - 12 - minimapGap, Y: editorY + 8, W: w, H: f.panelHeight()}

	// 左列：chevron 展开钮（宽 26），垂直居中于面板。
	chevW := int32(26)
	f.btnReplaceToggle = core.Rect{X: f.panel.X + 6, Y: f.panel.Y + (f.panel.H-22)/2, W: chevW - 4, H: 22}

	// 右列起点：chevron 之后。
	colX := f.panel.X + 6 + chevW
	row1Y := f.panel.Y + 8
	f.findBox = core.Rect{X: colX, Y: row1Y, W: f.panel.W - (chevW + 12) - 132, H: 26}
	f.btnPrev = core.Rect{X: f.findBox.X + f.findBox.W + 6, Y: row1Y + 3, W: 24, H: 20}
	f.btnNext = core.Rect{X: f.btnPrev.X + 26, Y: row1Y + 3, W: 24, H: 20}
	f.btnCase = core.Rect{X: f.btnNext.X + 26, Y: row1Y + 3, W: 26, H: 20}
	f.btnClose = core.Rect{X: f.panel.X + f.panel.W - 30, Y: row1Y + 3, W: 24, H: 20}

	// 第二行（展开时）：替换输入与查找框等宽；两个替换按钮在输入右侧。
	f.replaceBox = core.Rect{X: colX, Y: row1Y + 34, W: f.findBox.W, H: 26}
	f.btnReplaceOne = core.Rect{X: f.replaceBox.X + f.replaceBox.W + 6, Y: f.replaceBox.Y + 3, W: 30, H: 20}
	f.btnReplaceAll = core.Rect{X: f.btnReplaceOne.X + 32, Y: f.replaceBox.Y + 3, W: 30, H: 20}
}

// Panel 返回当前面板边界（宿主可据此决定事件拦截）。
func (f *FindReplace) Panel() core.Rect { return f.panel }

// ContainsHit 报告全局坐标是否落在面板内。
func (f *FindReplace) ContainsHit(x, y int32) bool {
	return f.Visible && contains(f.panel, x, y)
}

// --- 输入处理 ---

// HandleChar 处理可打印字符输入；返回是否消费。
func (f *FindReplace) HandleChar(ch rune) bool {
	if !f.Visible || f.focus < 0 {
		return false
	}
	if f.focus == 0 {
		f.Query += string(ch)
	} else {
		f.Replace += string(ch)
	}
	f.Changed()
	return true
}

// HandleKeyDown 处理按键；返回是否消费。
func (f *FindReplace) HandleKeyDown(key int32) bool {
	if !f.Visible || f.focus < 0 {
		return false
	}
	switch key {
	case core.KeyBackspace:
		if f.focus == 0 {
			if r := []rune(f.Query); len(r) > 0 {
				f.Query = string(r[:len(r)-1])
			}
		} else {
			if r := []rune(f.Replace); len(r) > 0 {
				f.Replace = string(r[:len(r)-1])
			}
		}
		f.Changed()
		return true
	case 0x0D: // Enter：跳到下一个匹配
		f.Next()
		return true
	case 0x1B: // Esc
		f.Hide()
		return true
	case 0x09: // Tab 在查找/替换框间切换
		if f.ShowReplace {
			f.focus = (f.focus + 1) % 2
		}
		return true
	}
	return false
}

// HandleMouseDown 处理面板内点击；返回是否消费。点击面板外时自动失焦。
func (f *FindReplace) HandleMouseDown(x, y int32) bool {
	if !f.Visible {
		return false
	}
	if !contains(f.panel, x, y) {
		f.focus = -1
		return false
	}
	switch {
	case contains(f.findBox, x, y):
		f.focus = 0
	case f.ShowReplace && contains(f.replaceBox, x, y):
		f.focus = 1
	case contains(f.btnPrev, x, y):
		f.hotPrev = true
		f.Prev()
	case contains(f.btnNext, x, y):
		f.hotNext = true
		f.Next()
	case contains(f.btnCase, x, y):
		f.hotCase = true
		f.MatchCase = !f.MatchCase
		f.Changed()
	case contains(f.btnClose, x, y):
		f.hotClose = true
		f.Hide()
	case contains(f.btnReplaceToggle, x, y):
		// chevron：单行时展开为两行，两行时收起。
		f.hotReplaceToggle = true
		f.ToggleReplace()
	case f.ShowReplace && contains(f.btnReplaceOne, x, y):
		f.hotReplaceOne = true
		f.ReplaceCurrent()
	case f.ShowReplace && contains(f.btnReplaceAll, x, y):
		f.hotReplaceAll = true
		f.ReplaceAll()
	default:
		f.focus = -1
	}
	return true
}

// HandleMouseMove 更新悬停态，返回是否变化（宿主决定是否重绘）。
func (f *FindReplace) HandleMouseMove(x, y int32) bool {
	if !f.Visible {
		return false
	}
	old := [7]bool{f.hotClose, f.hotPrev, f.hotNext, f.hotCase, f.hotReplaceToggle, f.hotReplaceOne, f.hotReplaceAll}
	f.hotClose = contains(f.btnClose, x, y)
	f.hotPrev = contains(f.btnPrev, x, y)
	f.hotNext = contains(f.btnNext, x, y)
	f.hotCase = contains(f.btnCase, x, y)
	f.hotReplaceToggle = contains(f.btnReplaceToggle, x, y)
	f.hotReplaceOne = contains(f.btnReplaceOne, x, y)
	f.hotReplaceAll = contains(f.btnReplaceAll, x, y)
	now := [7]bool{f.hotClose, f.hotPrev, f.hotNext, f.hotCase, f.hotReplaceToggle, f.hotReplaceOne, f.hotReplaceAll}
	return old != now
}

// --- 渲染 ---

// Render 绘制面板（全局坐标）。需先调用 SetPlacement。
func (f *FindReplace) Render(ctx core.DrawingContext) {
	if !f.Visible {
		return
	}
	f.refreshMatches()
	th := theme.GetTheme()

	// 面板：纯色不透明背景（主题 DialogPanelBg）。
	ctx.DrawRect(f.panel.X, f.panel.Y, f.panel.W, f.panel.H, th.DialogPanelBg, true)
	// 边框
	ctx.DrawRect(f.panel.X, f.panel.Y, f.panel.W, 1, th.SearchBorder, true)
	ctx.DrawRect(f.panel.X, f.panel.Y+f.panel.H-1, f.panel.W, 1, th.SearchBorder, true)
	ctx.DrawRect(f.panel.X, f.panel.Y, 1, f.panel.H, th.SearchBorder, true)
	ctx.DrawRect(f.panel.X+f.panel.W-1, f.panel.Y, 1, f.panel.H, th.SearchBorder, true)

	// 字体阶梯：Body 档（字体规范.md 第 4 节）。
	ctx.SetFont(th.UIFontFamily, th.FontBody())

	// 查找行
	f.drawInput(ctx, f.findBox, f.Query, f.focus == 0, "查找", th)

	// 匹配状态文本（查找框右侧、按钮左侧）
	status := f.StatusText()
	if status != "" {
		sw, _ := ctx.MeasureText(status)
		statusColor := th.SearchFg
		if len(f.Matches) == 0 {
			statusColor = th.ButtonDisabledFg
		}
		ctx.DrawText(f.btnClose.X-8-sw, f.findBox.Y+6, status, statusColor)
	}

	// 图标按钮：全部使用 icons.go 几何绘制（图标资源.md 第 4 节）。
	// Aa 是文字标签，是唯一允许的 DrawText "图标"。
	f.drawIconBtn(ctx, f.btnPrev, f.hotPrev, func(cx, cy int32, c uint32) { DrawNavUpIcon(ctx, cx, cy, c) }, th)
	f.drawIconBtn(ctx, f.btnNext, f.hotNext, func(cx, cy int32, c uint32) { DrawNavDownIcon(ctx, cx, cy, c) }, th)
	f.drawIconBtnToggle(ctx, f.btnCase, f.hotCase, f.MatchCase, func(cx, cy int32, c uint32) {
		ctx.DrawText(cx-8, cy-7, "Aa", c)
	}, th)
	f.drawIconBtn(ctx, f.btnClose, f.hotClose, func(cx, cy int32, c uint32) { DrawCloseIcon(ctx, cx, cy, c) }, th)

	// 替换行：chevron 始终绘制在面板最左（方向指示展开状态）。
	if f.ShowReplace {
		f.drawInput(ctx, f.replaceBox, f.Replace, f.focus == 1, "替换", th)
		f.drawIconBtn(ctx, f.btnReplaceOne, f.hotReplaceOne, func(cx, cy int32, c uint32) { DrawReplaceOneIcon(ctx, cx, cy, c) }, th)
		f.drawIconBtn(ctx, f.btnReplaceAll, f.hotReplaceAll, func(cx, cy int32, c uint32) { DrawReplaceAllIcon(ctx, cx, cy, c) }, th)
	}
	chevDir := ChevronRight
	if f.ShowReplace {
		chevDir = ChevronDown
	}
	f.drawIconBtnToggle(ctx, f.btnReplaceToggle, f.hotReplaceToggle, f.ShowReplace, func(cx, cy int32, c uint32) {
		DrawChevron(ctx, cx, cy, chevDir, c)
	}, th)
}

// drawInput 绘制单个输入框：边框 + 内容或占位符 + 焦点高亮。
func (f *FindReplace) drawInput(ctx core.DrawingContext, box core.Rect, text string, focused bool, placeholder string, th *theme.Theme) {
	border := th.SearchBorder
	if focused {
		border = th.MenuItemHoverBg
	}
	ctx.DrawRect(box.X, box.Y, box.W, box.H, th.SearchBg, true)
	ctx.DrawRect(box.X, box.Y, box.W, 1, border, true)
	ctx.DrawRect(box.X, box.Y+box.H-1, box.W, 1, border, true)
	ctx.DrawRect(box.X, box.Y, 1, box.H, border, true)
	ctx.DrawRect(box.X+box.W-1, box.Y, 1, box.H, border, true)

	content := text
	color := th.SearchFg
	if content == "" {
		if focused {
			content = ""
		} else {
			content = placeholder
			color = th.ButtonDisabledFg
		}
	}
	if content != "" {
		// 裁剪超宽文本
		maxW := box.W - 12
		if tw, _ := ctx.MeasureText(content); tw > maxW {
			runes := []rune(content)
			for len(runes) > 1 {
				runes = runes[:len(runes)-1]
				if tw, _ = ctx.MeasureText(string(runes)); tw <= maxW {
					content = string(runes) + "…"
					break
				}
			}
		}
		ctx.DrawText(box.X+6, box.Y+6, content, color)
	}
	// 焦点光标（简单竖线，跟随文本尾部）
	if focused {
		tw, _ := ctx.MeasureText(text)
		curX := box.X + 6 + tw
		if curX < box.X+box.W-4 {
			ctx.DrawRect(curX+1, box.Y+5, 1, box.H-10, th.EditorCursorColor, true)
		}
	}
}

// iconPainter 在 (cx, cy) 中心以颜色 c 绘制图标本体。
type iconPainter func(cx, cy int32, c uint32)

// drawIconBtn 绘制小图标按钮（背景 + 居中图标）。
func (f *FindReplace) drawIconBtn(ctx core.DrawingContext, box core.Rect, hot bool, paint iconPainter, th *theme.Theme) {
	f.drawIconBtnState(ctx, box, hot, false, paint, th)
}

// drawIconBtnToggle 绘制可保持按下状态的开关按钮。
func (f *FindReplace) drawIconBtnToggle(ctx core.DrawingContext, box core.Rect, hot, active bool, paint iconPainter, th *theme.Theme) {
	f.drawIconBtnState(ctx, box, hot, active, paint, th)
}

func (f *FindReplace) drawIconBtnState(ctx core.DrawingContext, box core.Rect, hot, active bool, paint iconPainter, th *theme.Theme) {
	bg := th.DialogPanelBg
	if active {
		bg = th.MenuItemHoverBg
	} else if hot {
		bg = th.MenuBarHoverBg
	}
	if bg != th.DialogPanelBg {
		ctx.DrawRect(box.X-2, box.Y-2, box.W+4, box.H+4, bg, true)
	}
	fg := th.SearchFg
	if active {
		fg = th.TextPrimary
	} else if hot {
		fg = th.TextPrimary
	}
	paint(box.X+box.W/2, box.Y+box.H/2, fg)
}
