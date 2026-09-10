package components

import (
	"sync"
	"time"

	"github.com/goedui/goed/editor/composables"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/syntax"
	"github.com/goedui/goed/ui/theme"
	"sort"
	"strings"
)

// clipboard 简单的剪贴板实现（跨平台剪贴板需要后续增强）
var clipboard string

// cursorBlinkInterval 编辑器光标闪烁周期（VSCode 默认 530ms）。
const cursorBlinkInterval = 530 * time.Millisecond

// Editor 编辑器核心
type Editor struct {
	// 文本内容
	Lines []string

	// 光标位置
	CursorLine int
	CursorCol  int

	// 滚动位置
	ScrollLine int
	ScrollCol  int
	// ScrollOffset is the pixel offset into the first visible line. It stays in
	// [0, LineHeight) while ScrollLine remains the actual document line.
	ScrollOffset int32
	// ScrollXPx is the horizontal pixel offset used when lines are wider than
	// the text viewport.
	ScrollXPx int32

	// 选择
	Selection     *composables.Selection
	SelectionMode bool // 是否处于选择模式

	// 历史记录
	History *composables.History

	// 文件信息
	FilePath string
	Modified bool

	// 渲染参数
	LineHeight int32
	CharWidth  int32

	// 可见区域
	VisibleLines int
	VisibleCols  int

	// 查找/替换
	FindText    string
	ReplaceText string
	FindResults []composables.Position
	CurrentFind int

	// 鼠标交互状态
	isDragging    bool
	lastClickTime int64 // 上次点击时间（毫秒）
	lastClickLine int
	lastClickCol  int
	clickCount    int // 连续点击次数

	// 多光标
	MultiCursor *composables.MultiCursor

	// 代码折叠
	FoldingManager *composables.FoldingManager
	FoldGutter     *FoldGutter

	// Minimap
	Minimap     *Minimap
	ShowMinimap bool // 是否显示 minimap

	// 右键上下文菜单
	ContextMenu *ContextMenu

	// ClipboardProvider lets platform shells use the editor's native clipboard
	// while keeping the component usable in tests and headless environments.
	clipboardRead  func() string
	clipboardWrite func(string)

	// OnShowInExplorer is called by the context menu for the current document.
	OnShowInExplorer func()

	// Minimap 布局缓存与拖动状态（本地坐标，Render 时刷新）
	minimapX        int32
	minimapH        int32
	minimapDragging bool

	// scrollBar 是编辑器右缘的交互式滚动条，几何状态由 Render 同步。
	scrollBar       *ScrollBar
	scrollBarX      int32 // 滚动条带左缘（编辑器本地坐标）
	scrollBarActive bool  // Render 时依据内容行数决定是否参与命中

	// hScrollBar 是底缘的横向滚动条；长行超出文本视口时启用。
	hScrollBar        *ScrollBar
	hScrollBarX       int32 // 横向条带左缘（编辑器本地坐标）
	hScrollBarY       int32 // 横向条带上缘（编辑器本地坐标）
	hScrollBarActive  bool
	textViewportW     int32 // 文本视口宽度（Render 时刷新，供 EnsureCursorVisible 使用）
	maxLineWidthCache int
	maxLineWidthDirty bool

	// maxLineWidth 增量维护快照：上次测量时的每行内容/宽度，重算时
	// 跳过未变化行；calibratedCharW 变化（字体校准更新）时整体失效。
	widthSnapshotLines  []string
	widthSnapshotWidths []int
	widthSnapshotCharW  int32

	// 查找匹配高亮：宿主（查找替换组件）注入的全部匹配；
	// Render 时全部匹配淡色、当前匹配选区色。
	FindMatches  []composables.SearchResult
	CurrentMatch int

	// calibratedCharW 是 Render 时用真实字体度量校准的半角字符宽。
	// 文本绘制以 GDI MeasureText 推进 X，光标/选区必须用同一度量
	// 才能对准；硬编码 CharWidth 在中英混排时会漂移。
	calibratedCharW int32
	calibrated      bool

	// Render caches. Scrolling only changes the visible range, so tokenization
	// and folded-line mapping should not be repeated for the whole document.
	highlightedLines     [][]syntax.Token
	highlightedPath      string
	highlightedLineCount int
	highlightedDirty     bool
	visibleLineIndices   []int
	visibleLineCount     int
	visibleMapDirty      bool

	// 光标闪烁：cursorVisible 决定 renderCursor 是否绘制；任何光标
	// 移动/输入都会 ResetCursorBlink 复位为可见并重新计时。OnRedraw
	// 由宿主注入，闪烁翻转时触发整窗重绘。
	cursorVisible    bool
	lastCursorBlink  time.Time
	cursorBlinkTimer *time.Timer
	blinkMu          sync.Mutex
	OnRedraw         func()
}

// SetOnRedraw 注入重绘回调并启动闪烁计时。编辑器在会话构造时创建，
// 宿主窗口晚于编辑器就绪，故闪烁随回调注入而启动而非构造时。
func (e *Editor) SetOnRedraw(fn func()) {
	e.blinkMu.Lock()
	e.OnRedraw = fn
	e.blinkMu.Unlock()
	if fn != nil {
		e.ResetCursorBlink()
	}
}

// ResetCursorBlink 光标移动/输入后调用：立即恢复可见并重置闪烁计时。
// 在 BlinkNow 到期翻转前总是先复位，保证打字时光标持续可见（VSCode 行为）。
func (e *Editor) ResetCursorBlink() {
	e.blinkMu.Lock()
	defer e.blinkMu.Unlock()
	e.cursorVisible = true
	e.lastCursorBlink = time.Now()
	if e.cursorBlinkTimer != nil {
		e.cursorBlinkTimer.Stop()
	}
	if e.OnRedraw == nil {
		return // 无宿主（测试/无头环境）：状态仍维护，不触发重绘
	}
	e.cursorBlinkTimer = time.AfterFunc(cursorBlinkInterval, e.BlinkNow)
}

// BlinkNow 定时到期翻转光标可见性并安排下一次翻转；由 AfterFunc 在独立
// goroutine 触发，重绘经 OnRedraw 抛回宿主（Windows 下最终在 UI 线程
// Invalidate），编辑器状态本身按序互斥保护。
func (e *Editor) BlinkNow() {
	e.blinkMu.Lock()
	e.cursorVisible = !e.cursorVisible
	e.lastCursorBlink = time.Now()
	redraw := e.OnRedraw
	e.blinkMu.Unlock()
	if redraw != nil {
		redraw()
	}
	e.blinkMu.Lock()
	if e.cursorBlinkTimer != nil {
		e.cursorBlinkTimer.Stop()
	}
	e.cursorBlinkTimer = time.AfterFunc(cursorBlinkInterval, e.BlinkNow)
	e.blinkMu.Unlock()
}

// CursorVisible 返回光标当前是否应绘制（供 renderCursor 与测试）。
func (e *Editor) CursorVisible() bool {
	e.blinkMu.Lock()
	defer e.blinkMu.Unlock()
	return e.cursorVisible
}

// NewEditor 创建编辑器
func NewEditor() *Editor {
	initialPos := composables.Position{Line: 0, Column: 0}
	foldingMgr := composables.NewFoldingManager()
	minimap := NewMinimap()

	editor := &Editor{
		Lines:             []string{""},
		CursorLine:        0,
		CursorCol:         0,
		ScrollLine:        0,
		ScrollCol:         0,
		ScrollOffset:      0,
		Selection:         nil,
		History:           composables.NewHistory(1000),
		LineHeight:        24,
		CharWidth:         9,
		Modified:          false,
		MultiCursor:       composables.NewMultiCursor(initialPos),
		FoldingManager:    foldingMgr,
		FoldGutter:        NewFoldGutter(foldingMgr),
		Minimap:           minimap,
		ShowMinimap:       true, // 默认显示
		ContextMenu:       NewContextMenu(),
		scrollBar:         NewVScrollBar(0, 0, 100),
		hScrollBar:        NewHScrollBar(0, 0, 100),
		minimapX:          -1, // Render 前不响应命中
		maxLineWidthDirty: true,
		highlightedDirty:  true,
		visibleMapDirty:   true,
		cursorVisible:     true,
		lastCursorBlink:   time.Now(),
	}
	editor.ResetCursorBlink()

	// 设置折叠回调
	editor.FoldGutter.OnToggleFold = func(line int) {
		// 折叠切换后，可能需要调整滚动位置
		editor.visibleMapDirty = true
		editor.EnsureCursorVisible()
	}

	return editor
}

// SetClipboardProvider connects the editor to a platform clipboard.
func (e *Editor) SetClipboardProvider(read func() string, write func(string)) {
	e.clipboardRead = read
	e.clipboardWrite = write
}

func (e *Editor) syncClipboard() {
	if e.clipboardRead != nil {
		clipboard = e.clipboardRead()
	}
}

func (e *Editor) markContentChanged() {
	e.highlightedDirty = true
	e.visibleMapDirty = true
	e.maxLineWidthDirty = true
}

// maxLineWidth 返回全部行的最大显示宽度（像素），结果在内容变化前缓存。
//
// 采用影子快照增量维护：上次计算时保存每行内容与宽度，重算时仅对
// 内容变化的行重新测量（未变化行退化为字符串比较快速路径）。编辑
// 通常只影响一两行，大文档下从「每字符全文档扫描」降为 O(变更行数)。
// charWidthAt 快照：宽度计算依赖 calibratedCharW，字体度量变化时全量失效。
func (e *Editor) maxLineWidth() int {
	if !e.maxLineWidthDirty {
		return e.maxLineWidthCache
	}
	if e.calibratedCharW != e.widthSnapshotCharW {
		e.widthSnapshotLines = nil
		e.widthSnapshotCharW = e.calibratedCharW
	}
	if len(e.widthSnapshotLines) != len(e.Lines) {
		e.widthSnapshotLines = make([]string, len(e.Lines))
		e.widthSnapshotWidths = make([]int, len(e.Lines))
	}
	max := 0
	for i, line := range e.Lines {
		if e.widthSnapshotLines[i] == line {
			if e.widthSnapshotWidths[i] > max {
				max = e.widthSnapshotWidths[i]
			}
			continue
		}
		w := int(e.CalcDisplayWidth(line, len([]rune(line))))
		e.widthSnapshotLines[i] = line
		e.widthSnapshotWidths[i] = w
		if w > max {
			max = w
		}
	}
	e.maxLineWidthCache = max
	e.maxLineWidthDirty = false
	return max
}

func (e *Editor) visibleLineMap() []int {
	if !e.visibleMapDirty && e.visibleLineCount == len(e.Lines) {
		return e.visibleLineIndices
	}
	visible := make([]int, 0, len(e.Lines))
	for i := range e.Lines {
		if e.FoldingManager == nil || !e.FoldingManager.IsFolded(i) {
			visible = append(visible, i)
		}
	}
	e.visibleLineIndices = visible
	e.visibleLineCount = len(e.Lines)
	e.visibleMapDirty = false
	return visible
}

// VisibleLineWindow returns the actual document lines in the current viewport
// (plus one partially visible row when requested). The returned slice is a
// read-only view used by companion renderers such as the line-number gutter.
func (e *Editor) VisibleLineWindow(rows int) []int {
	visible := e.visibleLineMap()
	if rows <= 0 || len(visible) == 0 {
		return nil
	}
	start := e.clampScrollIndex(e.scrollStartIndex(visible), rows)
	end := start + rows
	if end > len(visible) {
		end = len(visible)
	}
	return visible[start:end]
}

func (e *Editor) highlightedLineTokens() [][]syntax.Token {
	if !e.highlightedDirty && e.highlightedPath == e.FilePath && e.highlightedLineCount == len(e.Lines) {
		return e.highlightedLines
	}
	e.highlightedLines = syntax.Highlight(e.Lines, e.FilePath)
	e.highlightedPath = e.FilePath
	e.highlightedLineCount = len(e.Lines)
	e.highlightedDirty = false
	return e.highlightedLines
}

func (e *Editor) scrollStartIndex(visible []int) int {
	if len(visible) == 0 {
		return 0
	}
	idx := sort.SearchInts(visible, e.ScrollLine)
	if idx >= len(visible) {
		return len(visible) - 1
	}
	return idx
}

func (e *Editor) clampScrollIndex(index, visibleCount int) int {
	maxIndex := len(e.visibleLineMap()) - visibleCount
	if maxIndex < 0 {
		maxIndex = 0
	}
	if index < 0 {
		return 0
	}
	if index > maxIndex {
		return maxIndex
	}
	return index
}

func (e *Editor) setScrollIndex(index int, visible []int) {
	if len(visible) == 0 {
		e.ScrollLine = 0
		e.ScrollOffset = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(visible) {
		index = len(visible) - 1
	}
	e.ScrollLine = visible[index]
}

// === 文本操作 ===

// InsertChar 插入字符
func (e *Editor) InsertChar(ch rune) {
	if e.Selection != nil && !e.Selection.IsEmpty() {
		e.DeleteSelection()
	}

	if e.CursorLine >= len(e.Lines) {
		e.Lines = append(e.Lines, "")
	}

	line := e.Lines[e.CursorLine]
	runes := []rune(line)

	// 使用 rune 切片处理，确保正确处理多字节字符。单次分配插入：
	// 内层 append(...)+外层 append(...) 会对长行产生两次 O(n) 拷贝。
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:e.CursorCol]...)
	newRunes = append(newRunes, ch)
	newRunes = append(newRunes, runes[e.CursorCol:]...)
	newLine := string(newRunes)

	// 记录历史
	e.History.Record(composables.Operation{
		Type:     composables.OpInsert,
		Position: e.LineColToPosition(e.CursorLine, e.CursorCol),
		NewText:  string(ch),
	})

	e.Lines[e.CursorLine] = newLine
	e.CursorCol++
	e.Modified = true
	e.markContentChanged()
	e.Selection = nil
	e.ResetCursorBlink()
}

// DeleteChar 删除字符（Backspace）
func (e *Editor) DeleteChar() {
	if e.Selection != nil && !e.Selection.IsEmpty() {
		e.DeleteSelection()
		return
	}

	if e.CursorCol > 0 {
		line := e.Lines[e.CursorLine]
		runes := []rune(line)

		// 使用 rune 切片处理，确保正确删除完整字符
		oldChar := string(runes[e.CursorCol-1])
		newRunes := append(runes[:e.CursorCol-1], runes[e.CursorCol:]...)
		newLine := string(newRunes)

		// 记录历史
		e.History.Record(composables.Operation{
			Type:     composables.OpDelete,
			Position: e.LineColToPosition(e.CursorLine, e.CursorCol-1),
			OldText:  oldChar,
		})

		e.Lines[e.CursorLine] = newLine
		e.CursorCol--
		e.Modified = true
	} else if e.CursorLine > 0 {
		// 合并到上一行
		prevLine := e.Lines[e.CursorLine-1]
		currentLine := e.Lines[e.CursorLine]
		prevRunes := []rune(prevLine)

		// 记录历史
		e.History.Record(composables.Operation{
			Type:     composables.OpDelete,
			Position: e.LineColToPosition(e.CursorLine-1, len(prevRunes)),
			OldText:  "\n",
		})

		e.Lines[e.CursorLine-1] = prevLine + currentLine
		e.Lines = append(e.Lines[:e.CursorLine], e.Lines[e.CursorLine+1:]...)
		e.CursorLine--
		e.CursorCol = len(prevRunes)
		e.Modified = true
	}
	e.Selection = nil
	e.markContentChanged()
}

// DeleteCharForward 向前删除字符（Delete 键）
func (e *Editor) DeleteCharForward() {
	if e.Selection != nil && !e.Selection.IsEmpty() {
		e.DeleteSelection()
		return
	}

	if e.CursorLine >= len(e.Lines) {
		return
	}

	line := e.Lines[e.CursorLine]
	runes := []rune(line)

	if e.CursorCol < len(runes) {
		// 使用 rune 切片处理，确保正确删除完整字符
		oldChar := string(runes[e.CursorCol])
		newRunes := append(runes[:e.CursorCol], runes[e.CursorCol+1:]...)
		newLine := string(newRunes)

		// 记录历史
		e.History.Record(composables.Operation{
			Type:     composables.OpDelete,
			Position: e.LineColToPosition(e.CursorLine, e.CursorCol),
			OldText:  oldChar,
		})

		e.Lines[e.CursorLine] = newLine
		e.Modified = true
	} else if e.CursorLine < len(e.Lines)-1 {
		// 合并下一行
		currentLine := e.Lines[e.CursorLine]
		nextLine := e.Lines[e.CursorLine+1]
		currentRunes := []rune(currentLine)

		// 记录历史
		e.History.Record(composables.Operation{
			Type:     composables.OpDelete,
			Position: e.LineColToPosition(e.CursorLine, len(currentRunes)),
			OldText:  "\n",
		})

		e.Lines[e.CursorLine] = currentLine + nextLine
		e.Lines = append(e.Lines[:e.CursorLine+1], e.Lines[e.CursorLine+2:]...)
		e.Modified = true
	}
	e.Selection = nil
	e.markContentChanged()
}

// InsertNewLine 插入换行
func (e *Editor) InsertNewLine() {
	if e.Selection != nil && !e.Selection.IsEmpty() {
		e.DeleteSelection()
	}

	if e.CursorLine >= len(e.Lines) {
		e.Lines = append(e.Lines, "")
	}

	line := e.Lines[e.CursorLine]
	runes := []rune(line)

	// 使用 rune 切片处理，确保正确分割字符
	leftPart := string(runes[:e.CursorCol])
	rightPart := string(runes[e.CursorCol:])

	// 记录历史
	e.History.Record(composables.Operation{
		Type:     composables.OpInsert,
		Position: e.LineColToPosition(e.CursorLine, e.CursorCol),
		NewText:  "\n",
	})

	e.Lines[e.CursorLine] = leftPart
	e.Lines = append(e.Lines[:e.CursorLine+1], append([]string{rightPart}, e.Lines[e.CursorLine+1:]...)...)

	e.CursorLine++
	e.CursorCol = 0
	e.Modified = true
	e.markContentChanged()
	e.Selection = nil
}

// === 光标移动 ===

// MoveCursor 移动光标
func (e *Editor) MoveCursor(deltaCol, deltaLine int) {
	e.CursorLine += deltaLine
	e.CursorCol += deltaCol

	// 边界检查
	if e.CursorLine < 0 {
		e.CursorLine = 0
	}
	if e.CursorLine >= len(e.Lines) {
		e.CursorLine = len(e.Lines) - 1
	}

	if e.CursorCol < 0 {
		e.CursorCol = 0
	}
	if e.CursorLine < len(e.Lines) {
		currentLine := e.Lines[e.CursorLine]
		currentRunes := []rune(currentLine)
		if e.CursorCol > len(currentRunes) {
			e.CursorCol = len(currentRunes)
		}
	}

	// 滚动跟随
	e.EnsureCursorVisible()
	e.ResetCursorBlink()
}

// MoveCursorToLineStart 移动到行首
func (e *Editor) MoveCursorToLineStart() {
	e.CursorCol = 0
	e.ResetCursorBlink()
}

// MoveCursorToLineEnd 移动到行尾
func (e *Editor) MoveCursorToLineEnd() {
	if e.CursorLine < len(e.Lines) {
		runes := []rune(e.Lines[e.CursorLine])
		e.CursorCol = len(runes)
	}
	e.ResetCursorBlink()
}

// MoveCursorToDocStart 移动到文档开头
func (e *Editor) MoveCursorToDocStart() {
	e.CursorLine = 0
	e.CursorCol = 0
	e.ScrollLine = 0
	e.ScrollOffset = 0
	e.ResetCursorBlink()
}

// MoveCursorToDocEnd 移动到文档末尾
func (e *Editor) MoveCursorToDocEnd() {
	e.CursorLine = len(e.Lines) - 1
	if e.CursorLine >= 0 {
		runes := []rune(e.Lines[e.CursorLine])
		e.CursorCol = len(runes)
	}
	e.EnsureCursorVisible()
	e.ResetCursorBlink()
}

// PageUp 向上翻页
func (e *Editor) PageUp() {
	e.CursorLine -= e.VisibleLines
	if e.CursorLine < 0 {
		e.CursorLine = 0
	}
	e.EnsureCursorVisible()
}

// PageDown 向下翻页
func (e *Editor) PageDown() {
	e.CursorLine += e.VisibleLines
	if e.CursorLine >= len(e.Lines) {
		e.CursorLine = len(e.Lines) - 1
	}
	e.EnsureCursorVisible()
}

// EnsureCursorVisible 确保光标可见
func (e *Editor) EnsureCursorVisible() {
	visible := e.visibleLineMap()
	if len(visible) == 0 {
		e.ScrollLine, e.ScrollOffset = 0, 0
		return
	}
	visibleCount := e.VisibleLines
	if visibleCount < 1 {
		visibleCount = 1
	}
	cursorIndex := sort.SearchInts(visible, e.CursorLine)
	if cursorIndex >= len(visible) {
		cursorIndex = len(visible) - 1
	} else if visible[cursorIndex] != e.CursorLine && cursorIndex > 0 {
		cursorIndex--
	}
	startIndex := e.scrollStartIndex(visible)
	if cursorIndex < startIndex {
		startIndex = cursorIndex
		e.ScrollOffset = 0
	} else if cursorIndex >= startIndex+visibleCount {
		startIndex = cursorIndex - visibleCount + 1
		e.ScrollOffset = 0
	}
	startIndex = e.clampScrollIndex(startIndex, visibleCount)
	e.setScrollIndex(startIndex, visible)

	e.ensureCursorVisibleX()
}

// ensureCursorVisibleX 横向滚动保证光标列可见（按像素列宽估算）。
func (e *Editor) ensureCursorVisibleX() {
	line := ""
	if e.CursorLine >= 0 && e.CursorLine < len(e.Lines) {
		line = e.Lines[e.CursorLine]
	}
	runes := []rune(line)
	col := e.CursorCol
	if col > len(runes) {
		col = len(runes)
	}
	cursorX := e.CalcDisplayWidth(line, col)

	// 光标相对行首的偏移（行首自身缩进 8px 边距）。
	const leftPad = 8
	if e.ScrollXPx > int32(cursorX) {
		e.ScrollXPx = int32(cursorX)
	}
	if int32(cursorX)+e.CharWidth > e.ScrollXPx+e.textViewportW-leftPad {
		e.ScrollXPx = int32(cursorX) + e.CharWidth - e.textViewportW + leftPad
	}
	if e.ScrollXPx < 0 {
		e.ScrollXPx = 0
	}
	maxX := int32(0)
	if e.textViewportW > 0 {
		maxX = int32(e.maxLineWidth()) + 16 - e.textViewportW
		if maxX < 0 {
			maxX = 0
		}
	}
	if e.ScrollXPx > maxX {
		e.ScrollXPx = maxX
	}
}

// === 选择操作 ===

// StartSelection 开始选择
func (e *Editor) StartSelection() {
	e.SelectionMode = true
	e.Selection = composables.NewSelection(
		composables.Position{Line: e.CursorLine, Column: e.CursorCol},
		composables.Position{Line: e.CursorLine, Column: e.CursorCol},
	)
}

// UpdateSelection 更新选择
func (e *Editor) UpdateSelection() {
	if e.SelectionMode && e.Selection != nil {
		e.Selection.End = composables.Position{Line: e.CursorLine, Column: e.CursorCol}
	}
}

// ClearSelection 清除选择
func (e *Editor) ClearSelection() {
	e.Selection = nil
	e.SelectionMode = false
}

// SelectAll 全选
func (e *Editor) SelectAll() {
	lastLineRunes := []rune(e.Lines[len(e.Lines)-1])
	e.Selection = composables.NewSelection(
		composables.Position{Line: 0, Column: 0},
		composables.Position{Line: len(e.Lines) - 1, Column: len(lastLineRunes)},
	)
}

// DeleteSelection 删除选中内容
func (e *Editor) DeleteSelection() {
	if e.Selection == nil || e.Selection.IsEmpty() {
		return
	}
	if len(e.Lines) == 0 {
		e.Selection = nil
		return
	}

	start, end := e.selectionBounds()
	if start.Line != end.Line {
		startRunes := []rune(e.Lines[start.Line])
		endRunes := []rune(e.Lines[end.Line])
		merged := append([]rune{}, startRunes[:start.Column]...)
		merged = append(merged, endRunes[end.Column:]...)
		e.Lines[start.Line] = string(merged)
		copy(e.Lines[start.Line+1:], e.Lines[end.Line+1:])
		e.Lines = e.Lines[:len(e.Lines)-(end.Line-start.Line)]
		e.CursorLine = start.Line
		e.CursorCol = start.Column
		e.Selection = nil
		e.Modified = true
		e.markContentChanged()
		return
	}

	if start.Line == end.Line {
		runes := []rune(e.Lines[start.Line])
		newRunes := append(runes[:start.Column], runes[end.Column:]...)
		e.Lines[start.Line] = string(newRunes)
		e.CursorLine = start.Line
		e.CursorCol = start.Column
	}

	e.Selection = nil
	e.Modified = true
	e.markContentChanged()
}

// GetSelectedText 获取选中文本
func (e *Editor) GetSelectedText() string {
	if e.Selection == nil || e.Selection.IsEmpty() {
		return ""
	}
	if len(e.Lines) == 0 {
		return ""
	}

	start, end := e.selectionBounds()
	if start.Line != end.Line {
		var result strings.Builder
		result.WriteString(string([]rune(e.Lines[start.Line])[start.Column:]))
		for line := start.Line + 1; line < end.Line; line++ {
			result.WriteByte('\n')
			result.WriteString(e.Lines[line])
		}
		result.WriteByte('\n')
		result.WriteString(string([]rune(e.Lines[end.Line])[:end.Column]))
		return result.String()
	}

	if start.Line == end.Line {
		return string([]rune(e.Lines[start.Line])[start.Column:end.Column])
	}

	return ""
}

// === 撤销/重做 ===

// GotoPosition 把光标移动到指定行列并滚动到可见区域；
// selectLen > 0 时选中从该位置起 selectLen 个 rune（查找跳转用）。
func (e *Editor) GotoPosition(line, col int, selectLen int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}
	runes := []rune(e.Lines[line])
	if col > len(runes) {
		col = len(runes)
	}
	if col < 0 {
		col = 0
	}
	endCol := col
	if selectLen > 0 {
		endCol = col + selectLen
		if endCol > len(runes) {
			endCol = len(runes)
		}
	} else if selectLen == 0 {
		endCol = col + 1
		if endCol > len(runes) {
			endCol = len(runes)
		}
	}
	e.CursorLine = line
	e.CursorCol = col
	e.Selection = composables.NewSelection(
		composables.Position{Line: line, Column: col},
		composables.Position{Line: line, Column: endCol},
	)
	e.ensureCursorVisibleX()
	e.CenterLine(line)
}

// CenterLine 滚动视口使 line 位于编辑器中部（查找跳转的 VSCode 行为）。
func (e *Editor) CenterLine(line int) {
	visible := e.visibleLineMap()
	idx := line
	for i, l := range visible {
		if l == line {
			idx = i
			break
		}
	}
	target := idx - e.VisibleLines/2
	if target < 0 {
		target = 0
	}
	e.setScrollIndex(e.clampScrollIndex(target, e.VisibleLines), visible)
}

// blendColor 线性混合两个 COLORREF 颜色，t=0 返回 base，t=1 返回 over。
func blendColor(base, over uint32, t float64) uint32 {
	r := int(float64(base&0xFF)*(1-t) + float64(over&0xFF)*t)
	g := int(float64((base>>8)&0xFF)*(1-t) + float64((over>>8)&0xFF)*t)
	b := int(float64((base>>16)&0xFF)*(1-t) + float64((over>>16)&0xFF)*t)
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

// ReplaceRange 用 rep 替换 [line,col] 起 rune 长度 length 的内容，并记录撤销。
// 供查找替换组件逐个替换调用（从后往前替换时位置始终有效）。
func (e *Editor) ReplaceRange(line, col, length int, rep string) {
	if line < 0 || line >= len(e.Lines) {
		return
	}
	runes := []rune(e.Lines[line])
	if col < 0 || col > len(runes) {
		return
	}
	end := col + length
	if end > len(runes) {
		end = len(runes)
	}
	newLine := string(runes[:col]) + rep + string(runes[end:])
	e.History.Record(composables.Operation{
		Type:     composables.OpReplace,
		Position: e.LineColToPosition(line, col),
		OldText:  string(runes[col:end]),
		NewText:  rep,
	})
	e.Lines[line] = newLine
	e.CursorLine = line
	e.CursorCol = col + len([]rune(rep))
	e.Modified = true
	e.markContentChanged()
}

// selectionBounds returns a normalized, line/rune-clamped selection range.
func (e *Editor) selectionBounds() (composables.Position, composables.Position) {
	norm := e.Selection.Normalize()
	clamp := func(pos composables.Position) composables.Position {
		if len(e.Lines) == 0 {
			return composables.Position{}
		}
		if pos.Line < 0 {
			pos.Line = 0
		} else if pos.Line >= len(e.Lines) {
			pos.Line = len(e.Lines) - 1
		}
		lineLen := len([]rune(e.Lines[pos.Line]))
		if pos.Column < 0 {
			pos.Column = 0
		} else if pos.Column > lineLen {
			pos.Column = lineLen
		}
		return pos
	}
	start := clamp(norm.Start)
	end := clamp(norm.End)
	if start.Line > end.Line || (start.Line == end.Line && start.Column > end.Column) {
		start, end = end, start
	}
	return start, end
}

// Undo 撤销
func (e *Editor) Undo() {
	op := e.History.Undo()
	if op == nil {
		return
	}

	// 应用撤销
	switch op.Type {
	case composables.OpInsert:
		// 撤销插入 = 删除
		line, col := e.PositionToLineCol(op.Position)
		e.CursorLine = line
		e.CursorCol = col
		e.DeleteCharForward()
	case composables.OpDelete:
		// 撤销删除 = 插入
		line, col := e.PositionToLineCol(op.Position)
		e.CursorLine = line
		e.CursorCol = col
		for _, ch := range op.OldText {
			e.InsertChar(ch)
		}
	}

	e.Modified = true
}

// Redo 重做
func (e *Editor) Redo() {
	op := e.History.Redo()
	if op == nil {
		return
	}

	// 应用重做
	switch op.Type {
	case composables.OpInsert:
		line, col := e.PositionToLineCol(op.Position)
		e.CursorLine = line
		e.CursorCol = col
		for _, ch := range op.NewText {
			e.InsertChar(ch)
		}
	case composables.OpDelete:
		line, col := e.PositionToLineCol(op.Position)
		e.CursorLine = line
		e.CursorCol = col
		e.DeleteCharForward()
	}

	e.Modified = true
}

// === 渲染 ===

// Render 渲染编辑器
func (e *Editor) Render(ctx core.DrawingContext, x, y, width, height int32) {
	th := theme.GetTheme()
	ctx.SetFont(th.EditorFontFamily, th.EditorFontSize)

	// 校准半角字符宽：测量 20 个 '0' 的总宽再除以 20，消除
	// 单字符 MeasureText 的舍入误差。光标/选区/点击命中与文本
	// 绘制共用同一度量，中英文混排时 X 坐标不漂移。
	if w20, _ := ctx.MeasureText("00000000000000000000"); w20 > 0 {
		e.calibratedCharW = w20 / 20
		e.calibrated = true
	}

	// 背景
	ctx.DrawRect(x, y, width, height, th.EditorBg, true)

	// 计算可见行数。滚动存在像素偏移时，多绘制一行覆盖底部的部分行。
	e.VisibleLines = int(height / e.LineHeight)
	if e.VisibleLines < 1 {
		e.VisibleLines = 1
	}
	e.VisibleCols = int(width / e.CharWidth)

	// 计算布局：折叠侧边栏 | 内容区 | Minimap | 滚动条
	foldGutterWidth := e.FoldGutter.Width
	minimapWidth := int32(0)
	if e.ShowMinimap {
		minimapWidth = e.Minimap.Width
	}
	// 交互式滚动条带：折叠后的内容行数超出视口时占据右缘 16px。
	scrollW := int32(0)
	if e.scrollBar != nil && len(e.visibleLineMap()) > e.VisibleLines {
		scrollW = 16
	}
	// 横向滚动条带：最长行显示宽度超出文本视口时占据底缘 14px。
	hScrollH := int32(0)

	visible := e.visibleLineMap()
	startIndex := e.scrollStartIndex(visible)
	startIndex = e.clampScrollIndex(startIndex, e.VisibleLines)
	e.setScrollIndex(startIndex, visible)
	renderLines := e.VisibleLines
	if e.ScrollOffset > 0 {
		renderLines++
	}
	visibleWindow := visible
	if startIndex < len(visible) {
		end := startIndex + renderLines
		if end > len(visible) {
			end = len(visible)
		}
		visibleWindow = visible[startIndex:end]
	} else {
		visibleWindow = nil
	}

	// 渲染折叠侧边栏
	e.FoldGutter.RenderVisible(ctx, x, y, height, visibleWindow, e.LineHeight, e.ScrollOffset)

	// 横向滚动启用判定：文本视口容不下最长行。
	textW := width - foldGutterWidth - minimapWidth - scrollW
	if e.hScrollBar != nil && int(e.maxLineWidth()) > int(textW) && textW > 0 {
		hScrollH = 14
	}
	e.textViewportW = textW

	// 调整内容区域
	contentX := x + foldGutterWidth
	contentWidth := textW
	highlightedLines := e.highlightedLineTokens()

	// 文本视口裁剪：横向滚动偏移下长行不得溢出到 gutter/minimap/滚动条。
	if clip, ok := ctx.(editorClipStack); ok {
		clip.PushClipRect(contentX, y, contentWidth, height-hScrollH)
		defer clip.PopClip()
	} else {
		ctx.SetClipRect(contentX, y, contentWidth, height-hScrollH)
		defer ctx.ResetClip()
	}

	// 只遍历当前可见窗口，不再从第 0 行跳过折叠和滚动前内容。
	for visibleIdx, lineNum := range visibleWindow {
		line := e.Lines[lineNum]
		lineY := y + int32(visibleIdx)*e.LineHeight - e.ScrollOffset

		// 当前行背景
		if lineNum == e.CursorLine {
			ctx.DrawRect(contentX, lineY, contentWidth, e.LineHeight, th.EditorCurrentLineBg, true)
		}

		// 查找匹配高亮：全部匹配淡色、当前匹配选区色。
		if len(e.FindMatches) > 0 {
			for i, m := range e.FindMatches {
				if m.Position.Line != lineNum {
					continue
				}
				isCurrent := i == e.CurrentMatch
				bg := blendColor(th.EditorBg, th.EditorSelectionBg, 0.35)
				if isCurrent {
					bg = th.EditorSelectionBg
				}
				startX := contentX + 8 - e.ScrollXPx + e.CalcDisplayWidth(line, m.Position.Column)
				matchW := e.CalcDisplayWidth(line, m.Position.Column+m.Length) - e.CalcDisplayWidth(line, m.Position.Column)
				ctx.DrawRect(startX, lineY, matchW, e.LineHeight, bg, true)
			}
		}

		// 渲染所有光标的选择背景
		if e.MultiCursor.Enabled {
			for _, cursor := range e.MultiCursor.GetAll() {
				if cursor.Selection != nil && !cursor.Selection.IsEmpty() {
					e.renderSelection(ctx, cursor.Selection, lineNum, contentX, lineY, line)
				}
			}
		} else {
			// 单光标模式：渲染主选择
			if e.Selection != nil && !e.Selection.IsEmpty() {
				e.renderSelection(ctx, e.Selection, lineNum, contentX, lineY, line)
			}
		}

		// 检查是否是折叠的起始行
		region := e.FoldingManager.GetFoldRegionAt(lineNum)

		// 文本
		var tokens []syntax.Token
		if lineNum >= 0 && lineNum < len(highlightedLines) {
			tokens = highlightedLines[lineNum]
		}
		if region != nil && region.Folded {
			// Do not append to the tokenizer-owned slice: its spare capacity may
			// otherwise mutate the cached line tokens during rendering.
			tokens = append([]syntax.Token(nil), tokens...)
			tokens = append(tokens, syntax.Token{Text: " ...", Kind: syntax.Plain})
		}
		e.renderSyntaxTokens(ctx, contentX+8-e.ScrollXPx, lineY+2, tokens, th)

	}

	// 渲染光标
	if e.MultiCursor.Enabled {
		// 多光标模式：渲染所有光标
		for idx, cursor := range e.MultiCursor.GetAll() {
			isPrimary := (idx == e.MultiCursor.PrimaryIdx)
			e.renderCursor(ctx, cursor.Position.Line, cursor.Position.Column, contentX, y, isPrimary, th)
		}
	} else {
		// 单光标模式：渲染主光标
		e.renderCursor(ctx, e.CursorLine, e.CursorCol, contentX, y, true, th)
	}

	// 结束文本视口裁剪，滚动条/菜单绘制不受影响。
	if clip, ok := ctx.(editorClipStack); ok {
		clip.PopClip()
	} else {
		ctx.ResetClip()
	}

	// 渲染 Minimap（滚动条显示时向左让位）
	if e.ShowMinimap {
		minimapX := x + width - minimapWidth - scrollW
		e.minimapX = width - minimapWidth - scrollW // 本地坐标，供命中测试
		e.minimapH = height
		e.Minimap.Render(ctx, minimapX, y, height, e.Lines, e.ScrollLine, e.VisibleLines, e.ScrollOffset)
	}

	// 渲染交互式滚动条：thumb 位置与可见窗口起始索引同步
	if e.updateScrollBarGeometry(x, y, width, height-hScrollH, startIndex, e.VisibleLines) {
		e.scrollBar.Render(ctx)
	}

	// 渲染横向滚动条：位置为像素偏移 ScrollXPx。
	e.hScrollBarActive = hScrollH > 0 && e.hScrollBar != nil
	if e.hScrollBarActive && e.hScrollBar != nil {
		e.hScrollBarX = foldGutterWidth
		e.hScrollBarY = height - hScrollH
		e.hScrollBar.X = x + e.hScrollBarX
		e.hScrollBar.Y = y + e.hScrollBarY
		e.hScrollBar.Width = width - e.hScrollBarX - scrollW
		e.hScrollBar.Height = hScrollH
		e.hScrollBar.Total = int32(e.maxLineWidth()) + 16 // 右侧留一屏字符余量
		e.hScrollBar.Visible = textW
		e.hScrollBar.Position = e.ScrollXPx
		e.hScrollBar.Render(ctx)
	}

	// 渲染右键上下文菜单（最后渲染，保证在最上层）
	e.ContextMenu.RenderAt(ctx, x, y)
}

// editorClipStack 与 canvas.go 的 clipStack 一致：支持嵌套裁剪的绘制上下文。
type editorClipStack interface {
	PushClipRect(x, y, w, h int32)
	PopClip()
}

// renderSelection 渲染选择区域
func (e *Editor) renderSyntaxTokens(ctx core.DrawingContext, x, y int32, tokens []syntax.Token, th *theme.Theme) {
	if len(tokens) == 0 {
		return
	}
	cursorX := x
	for _, token := range tokens {
		color := th.EditorFg
		switch token.Kind {
		case syntax.Keyword:
			color = th.SyntaxKeyword
		case syntax.String:
			color = th.SyntaxString
		case syntax.Comment:
			color = th.SyntaxComment
		case syntax.Number:
			color = th.SyntaxNumber
		case syntax.Function:
			color = th.SyntaxFunction
		case syntax.Type:
			color = th.SyntaxType
		case syntax.Heading:
			color = th.SyntaxKeyword
		case syntax.Link:
			color = th.SyntaxFunction
		}
		ctx.DrawText(cursorX, y, token.Text, color)
		if measured, _ := ctx.MeasureText(token.Text); measured > 0 {
			cursorX += measured
		} else {
			cursorX += int32(len([]rune(token.Text))) * e.CharWidth
		}
	}
}

func (e *Editor) renderSelection(ctx core.DrawingContext, sel *composables.Selection, lineNum int, x, lineY int32, line string) {
	th := theme.GetTheme()
	norm := sel.Normalize()

	if lineNum >= norm.Start.Line && lineNum <= norm.End.Line {
		runes := []rune(line)
		startCol := 0
		endCol := len(runes)
		if lineNum == norm.Start.Line {
			startCol = norm.Start.Column
		}
		if lineNum == norm.End.Line {
			endCol = norm.End.Column
		}

		// 计算选择区域的实际显示位置和宽度
		selX := x + 8 + e.CalcDisplayWidth(line, startCol)
		selWidth := e.CalcDisplayWidth(line, endCol) - e.CalcDisplayWidth(line, startCol)
		ctx.DrawRect(selX, lineY, selWidth, e.LineHeight, th.EditorSelectionBg, true)
	}
}

// renderCursor 渲染单个光标
func (e *Editor) renderCursor(ctx core.DrawingContext, cursorLine, cursorCol int, x, y int32, isPrimary bool, th *theme.Theme) {
	visible := e.visibleLineMap()
	lineIndex := sort.SearchInts(visible, cursorLine)
	startIndex := e.scrollStartIndex(visible)
	if lineIndex < len(visible) && visible[lineIndex] == cursorLine {
		row := lineIndex - startIndex
		if row >= 0 && row <= e.VisibleLines {
			cursorY := y + int32(row)*e.LineHeight - e.ScrollOffset

			// 计算实际显示宽度（考虑全角字符）
			line := ""
			if cursorLine < len(e.Lines) {
				line = e.Lines[cursorLine]
			}
			displayWidth := e.CalcDisplayWidth(line, cursorCol)
			cursorX := x + 8 + displayWidth

			// 闪烁：翻转到隐藏相位时跳过绘制。
			if !e.CursorVisible() {
				return
			}

			// 主光标用标准颜色，次要光标用半透明
			cursorColor := th.EditorCursorColor
			if !isPrimary {
				// 次要光标：使用稍微透明或不同颜色
				// 这里简化处理，使用相同颜色但宽度不同
				ctx.DrawRect(cursorX, cursorY, 1, e.LineHeight, cursorColor, true)
			} else {
				ctx.DrawRect(cursorX, cursorY, 2, e.LineHeight, cursorColor, true)
			}
		}
	}
}

// === 工具函数 ===

// isWideRune 判断 rune 是否为东亚全角字符（双列宽）。
// 覆盖 CJK 统一表意文字、扩展 A、全角标点/ forms、片假名、
// 谚文、以及 CJK 兼容区；避免单一阈值 >0x2E80 漏判全角标点。
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F: // Hangul Jamo
		return true
	case r >= 0x2E80 && r <= 0x303E: // CJK 部首补充、康熙部首、表意文字描述、CJK 符号标点
		return true
	case r >= 0x3041 && r <= 0x33FF: // 平假名、片假名、注音、兼容文字
		return true
	case r >= 0x3400 && r <= 0x4DBF: // CJK 扩展 A
		return true
	case r >= 0x4E00 && r <= 0x9FFF: // CJK 统一表意文字
		return true
	case r >= 0xA000 && r <= 0xA4CF: // 彝文
		return true
	case r >= 0xAC00 && r <= 0xD7A3: // 谚文音节
		return true
	case r >= 0xF900 && r <= 0xFAFF: // CJK 兼容表意文字
		return true
	case r >= 0xFE30 && r <= 0xFE4F: // CJK 兼容形式
		return true
	case r >= 0xFF00 && r <= 0xFF60: // 全角 ASCII 与全角标点
		return true
	case r >= 0xFFE0 && r <= 0xFFE6: // 全角符号
		return true
	case r >= 0x20000 && r <= 0x3FFFD: // CJK 扩展 B-F
		return true
	}
	return false
}

// charW 返回单个 rune 的显示宽度（以半角宽为单位换算的像素）。
// 优先使用 Render 时校准的真实半角宽 calibratedCharW；测试/无渲染
// 环境回退到估算 CharWidth。
func (e *Editor) charW(r rune) int32 {
	base := e.CharWidth
	if e.calibratedCharW > 0 {
		base = e.calibratedCharW
	}
	if isWideRune(r) {
		return base * 2
	}
	return base
}

// CalcDisplayWidth 计算从行首到指定列的显示宽度（考虑全角字符）
func (e *Editor) CalcDisplayWidth(line string, col int) int32 {
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}

	width := int32(0)
	for i := 0; i < col; i++ {
		width += e.charW(runes[i])
	}
	return width
}

// LineColToPosition 行列转位置
func (e *Editor) LineColToPosition(line, col int) int {
	pos := 0
	for i := 0; i < line && i < len(e.Lines); i++ {
		runes := []rune(e.Lines[i])
		pos += len(runes) + 1 // +1 for newline
	}
	pos += col
	return pos
}

// PositionToLineCol 位置转行列
func (e *Editor) PositionToLineCol(pos int) (int, int) {
	currentPos := 0
	for line, text := range e.Lines {
		runes := []rune(text)
		lineLen := len(runes) + 1 // +1 for newline
		if currentPos+lineLen > pos {
			return line, pos - currentPos
		}
		currentPos += lineLen
	}
	lastLineRunes := []rune(e.Lines[len(e.Lines)-1])
	return len(e.Lines) - 1, len(lastLineRunes)
}

// SetText 设置文本
func (e *Editor) SetText(text string) {
	// 按行分割文本
	e.Lines = splitLines(text)
	if len(e.Lines) == 0 {
		e.Lines = []string{""}
	}
	e.CursorLine = 0
	e.CursorCol = 0
	e.ScrollLine = 0
	e.Modified = false
	e.History.Clear()
	e.markContentChanged()
}

// splitLines 将文本按行分割（处理 \n, \r\n, \r）
func splitLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	// Normalize all supported newline forms before splitting. Splitting the
	// string itself preserves multi-byte UTF-8 characters such as Chinese text.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

// GetText 获取文本。strings.Builder 单次分配拼接，避免大文档
// 逐行 += 的 O(n²) 拷贝（保存/查找/stash 都会调用）。
func (e *Editor) GetText() string {
	if len(e.Lines) == 0 {
		return ""
	}
	if len(e.Lines) == 1 {
		return e.Lines[0]
	}
	var b strings.Builder
	total := len(e.Lines) - 1 // 换行符数
	for _, line := range e.Lines {
		total += len(line)
	}
	b.Grow(total)
	for i, line := range e.Lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return b.String()
}

// === 鼠标交互增强 ===

// HandleMouseDown 处理鼠标按下（支持双击、三击、右键）
func (e *Editor) HandleMouseDown(x, y int32, button int32, shift, ctrl bool, currentTime int64) {
	// 检查是否点击了上下文菜单
	if e.ContextMenu.IsVisible {
		if e.ContextMenu.HandleMouseDown(x, y) {
			return
		}
		// 点击菜单外部，关闭菜单
		e.ContextMenu.Hide()
	}

	// 右键点击
	if button == 2 { // 2 表示右键
		e.HandleRightClick(x, y)
		return
	}

	// 折叠侧栏：左键点击图标切换折叠（行号映射与 RenderVisible 几何一致）
	if button == 0 && x < e.FoldGutter.Width && e.LineHeight > 0 {
		if visible := e.visibleLineMap(); len(visible) > 0 {
			e.FoldGutter.HandleClickVisible(visible[e.scrollStartIndex(visible):], y, e.LineHeight, e.ScrollOffset)
		}
		return
	}

	// 交互式滚动条：thumb 拖动起点或轨道点击跳转（几何在 Render 时同步）
	if button == 0 && e.scrollBarActive && e.scrollBar != nil && x >= e.scrollBarX {
		e.scrollBar.X, e.scrollBar.Y = 0, 0 // 命中测试使用编辑器本地坐标系
		if e.scrollBar.HandleMouseDown(x-e.scrollBarX, y) {
			e.scrollToScrollBar()
			return
		}
	}

	// 横向滚动条：条带位于底缘 hScrollBarY 以下。
	if button == 0 && e.hScrollBarActive && e.hScrollBar != nil && y >= e.hScrollBarY {
		e.hScrollBar.X, e.hScrollBar.Y = 0, 0
		if e.hScrollBar.HandleMouseDown(x-e.hScrollBarX, y-e.hScrollBarY) {
			e.ScrollXPx = e.hScrollBar.Position
			return
		}
	}

	// Minimap 区域：左键按下进入视口拖动，不触发光标/选择逻辑
	if button == 0 && e.handleMinimapMouseDown(x, y) {
		return
	}

	line, col := e.PixelToLineCol(x, y)

	// 检测双击/三击（500ms 内）
	timeDiff := currentTime - e.lastClickTime
	if timeDiff < 500 && line == e.lastClickLine && col == e.lastClickCol {
		e.clickCount++
	} else {
		e.clickCount = 1
	}

	e.lastClickTime = currentTime
	e.lastClickLine = line
	e.lastClickCol = col

	switch e.clickCount {
	case 1:
		// 单击：设置光标或扩展选择
		if shift && e.Selection != nil {
			// Shift+点击：扩展选择
			e.Selection.End = composables.Position{Line: line, Column: col}
		} else if ctrl {
			// Ctrl+点击：添加多光标
			e.AddCursorAt(line, col, true)
		} else {
			// 普通点击：移动光标
			e.CursorLine = line
			e.CursorCol = col
			e.ClearSelection()
			e.isDragging = true
			e.ResetCursorBlink()
		}
	case 2:
		// 双击：选中单词
		e.SelectWord(line, col)
		e.clickCount = 0 // 重置避免三击
	case 3:
		// 三击：选中整行
		e.SelectLine(line)
		e.clickCount = 0
	}
}

// HandleRightClick 处理右键点击
func (e *Editor) HandleRightClick(x, y int32) {
	line, col := e.PixelToLineCol(x, y)

	// 如果右键位置不在选择区域内，移动光标到该位置
	if e.Selection == nil || !e.IsInSelection(line, col) {
		e.CursorLine = line
		e.CursorCol = col
		e.ClearSelection()
	}

	// 构建上下文菜单项
	hasSelection := e.Selection != nil && !e.Selection.IsEmpty()
	e.syncClipboard()
	hasClipboard := len(clipboard) > 0

	menuItems := []MenuItem{
		{
			Label:    "剪切",
			Shortcut: "Ctrl+X",
			Enabled:  hasSelection,
			OnClick: func() {
				e.Cut()
			},
		},
		{
			Label:    "复制",
			Shortcut: "Ctrl+C",
			Enabled:  hasSelection,
			OnClick: func() {
				e.Copy()
			},
		},
		{
			Label:    "粘贴",
			Shortcut: "Ctrl+V",
			Enabled:  hasClipboard,
			OnClick: func() {
				e.Paste()
			},
		},
		{
			Label:   "在资源管理器中查看",
			Enabled: e.FilePath != "" && e.OnShowInExplorer != nil,
			OnClick: func() {
				if e.OnShowInExplorer != nil {
					e.OnShowInExplorer()
				}
			},
		},
		{
			Separator: true,
		},
		{
			Label:    "全选",
			Shortcut: "Ctrl+A",
			Enabled:  true,
			OnClick: func() {
				e.SelectAll()
			},
		},
		{
			Separator: true,
		},
		{
			Label:    "撤销",
			Shortcut: "Ctrl+Z",
			Enabled:  e.History.CanUndo(),
			OnClick: func() {
				e.Undo()
			},
		},
		{
			Label:    "重做",
			Shortcut: "Ctrl+Y",
			Enabled:  e.History.CanRedo(),
			OnClick: func() {
				e.Redo()
			},
		},
	}

	// 显示菜单
	e.ContextMenu.Show(x, y, menuItems)
}

// HandleMouseDrag 处理鼠标拖拽选择
func (e *Editor) HandleMouseDrag(x, y int32) {
	// 滚动条 thumb 拖拽独立于文本选择，不依赖 isDragging。
	if e.scrollBar != nil && e.scrollBar.dragging {
		e.scrollBar.X, e.scrollBar.Y = 0, 0
		e.scrollBar.HandleMouseMove(x-e.scrollBarX, y)
		e.scrollToScrollBar()
		return
	}

	// 横向滚动条拖拽。
	if e.hScrollBar != nil && e.hScrollBar.dragging {
		e.hScrollBar.X, e.hScrollBar.Y = 0, 0
		e.hScrollBar.HandleMouseMove(x-e.hScrollBarX, y-e.hScrollBarY)
		e.ScrollXPx = e.hScrollBar.Position
		return
	}

	if !e.isDragging {
		return
	}

	// 如果上下文菜单打开，不处理拖拽
	if e.ContextMenu.IsVisible {
		return
	}

	line, col := e.PixelToLineCol(x, y)

	// 开始选择（如果还没有）
	if e.Selection == nil {
		e.Selection = composables.NewSelection(
			composables.Position{Line: e.CursorLine, Column: e.CursorCol},
			composables.Position{Line: line, Column: col},
		)
	} else {
		// 更新选择终点
		e.Selection.End = composables.Position{Line: line, Column: col}
	}

	// 移动光标到拖拽位置
	e.CursorLine = line
	e.CursorCol = col
	e.EnsureCursorVisible()
}

// IsDragging reports whether a primary-button text drag is active.
func (e *Editor) IsDragging() bool {
	return e.isDragging
}

// HandleMouseUp 处理鼠标释放
func (e *Editor) HandleMouseUp() {
	e.isDragging = false
	e.minimapDragging = false
	if e.scrollBar != nil {
		e.scrollBar.dragging = false
	}
	if e.hScrollBar != nil {
		e.hScrollBar.dragging = false
	}
}

// scrollToScrollBar 把滚动条 Position 应用为编辑器滚动位置（整行对齐）。
func (e *Editor) scrollToScrollBar() {
	e.setScrollIndex(int(e.scrollBar.Position), e.visibleLineMap())
	e.ScrollOffset = 0
}

// updateScrollBarGeometry 在 Render 时同步滚动条几何与位置（不含绘制），
// 返回滚动条是否激活。命中测试与拖拽依赖这份几何状态。
func (e *Editor) updateScrollBarGeometry(x, y, width, height int32, startIndex, visibleCount int) bool {
	total := len(e.visibleLineMap())
	if e.scrollBar == nil || visibleCount < 1 || total <= visibleCount {
		e.scrollBarActive = false
		return false
	}
	e.scrollBarActive = true
	e.scrollBarX = width - 16
	e.scrollBar.X = x + e.scrollBarX
	e.scrollBar.Y = y
	e.scrollBar.Width = 16
	e.scrollBar.Height = height
	e.scrollBar.Total = int32(total)
	e.scrollBar.Visible = int32(visibleCount)
	e.scrollBar.Position = int32(startIndex)
	return true
}

// HandleMouseMoveForMenu 处理上下文菜单的鼠标移动
func (e *Editor) HandleMouseMoveForMenu(x, y int32) bool {
	if e.ContextMenu.IsVisible {
		return e.ContextMenu.HandleMouseMove(x, y)
	}
	return false
}

// HandleMouseWheel 处理鼠标滚轮；shift 为 true 时横向滚动。
func (e *Editor) HandleMouseWheel(delta int32, shift bool) {
	// WM_MOUSEWHEEL reports multiples of 120. Convert the wheel delta to a
	// pixel distance so one event can land between two text rows.
	if delta == 0 {
		return
	}
	// Use a 1:1 delta-to-pixel ratio so a standard 120-unit wheel notch
	// advances five 24px editor rows while preserving high-resolution deltas.
	// 分母越小滚动越快
	// Shift+滚轮横向滚动（约每档 3 个字符）。
	if shift {
		e.ScrollXPx += -delta / 3
		e.clampScrollX()
		return
	}

	pixels := -delta / 1
	if pixels == 0 {
		pixels = -1
	}
	e.ScrollByPixels(pixels)
}

// clampScrollX 把横向像素偏移限制在 [0, 最长行宽+边距-视口宽]。
func (e *Editor) clampScrollX() {
	maxX := int32(0)
	if e.textViewportW > 0 {
		maxX = int32(e.maxLineWidth()) + 16 - e.textViewportW
		if maxX < 0 {
			maxX = 0
		}
	}
	if e.ScrollXPx < 0 {
		e.ScrollXPx = 0
	}
	if e.ScrollXPx > maxX {
		e.ScrollXPx = maxX
	}
}

// ScrollByPixels updates the scroll position using the folded visible-line
// map. ScrollLine remains the actual document line for existing callers while
// ScrollOffset carries the fractional row displacement.
func (e *Editor) ScrollByPixels(pixels int32) {
	if pixels == 0 || e.LineHeight <= 0 {
		return
	}
	visible := e.visibleLineMap()
	if len(visible) == 0 {
		return
	}
	visibleCount := e.VisibleLines
	if visibleCount < 1 {
		visibleCount = 1
	}
	startIndex := e.scrollStartIndex(visible)
	offset := e.ScrollOffset + pixels
	for offset >= e.LineHeight {
		offset -= e.LineHeight
		startIndex++
	}
	for offset < 0 {
		offset += e.LineHeight
		startIndex--
	}
	maxIndex := len(visible) - visibleCount
	if maxIndex < 0 {
		maxIndex = 0
	}
	if startIndex < 0 {
		startIndex = 0
		offset = 0
	} else if startIndex > maxIndex {
		startIndex = maxIndex
		offset = 0
	} else if startIndex == maxIndex && pixels > 0 {
		// Do not scroll beyond the final complete viewport.
		offset = 0
	}
	e.setScrollIndex(startIndex, visible)
	e.ScrollOffset = offset
}

// PixelToLineCol 像素坐标转行列
func (e *Editor) PixelToLineCol(x, y int32) (int, int) {
	if e.LineHeight <= 0 {
		return 0, 0
	}
	visible := e.visibleLineMap()
	if len(visible) == 0 {
		return 0, 0
	}
	startIndex := e.scrollStartIndex(visible)
	row := int((y + e.ScrollOffset) / e.LineHeight)
	if row < 0 {
		row = 0
	}
	lineIndex := startIndex + row
	if lineIndex >= len(visible) {
		lineIndex = len(visible) - 1
	}
	line := visible[lineIndex]

	// 计算列（考虑左边距8px和全角字符）
	if line >= len(e.Lines) {
		return line, 0
	}

	gutterWidth := int32(0)
	if e.FoldGutter != nil {
		gutterWidth = e.FoldGutter.Width
	}
	targetX := x - gutterWidth - 8 + e.ScrollXPx
	if targetX < 0 {
		return line, 0
	}

	lineText := e.Lines[line]
	runes := []rune(lineText)

	// 遍历字符，累加宽度直到超过目标位置（与光标绘制同一度量）。
	// 点击点落在全角字符左半边时定位到该字符，右半边则到下一列。
	currentX := int32(0)
	for col := 0; col < len(runes); col++ {
		w := e.charW(runes[col])
		if targetX < currentX+w/2 {
			return line, col
		}
		currentX += w
	}
	if targetX >= currentX {
		return line, len(runes)
	}
	return line, len(runes)
}

// SelectWord 选中光标位置的单词
func (e *Editor) SelectWord(line, col int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}

	text := e.Lines[line]
	if len(text) == 0 {
		return
	}

	// 确保col在范围内
	if col >= len(text) {
		col = len(text) - 1
	}

	// 向左找单词边界
	start := col
	for start > 0 && isWordChar(rune(text[start-1])) {
		start--
	}

	// 向右找单词边界
	end := col
	for end < len(text) && isWordChar(rune(text[end])) {
		end++
	}

	// 创建选择
	e.Selection = composables.NewSelection(
		composables.Position{Line: line, Column: start},
		composables.Position{Line: line, Column: end},
	)
	e.CursorLine = line
	e.CursorCol = end
}

// SelectLine 选中整行
func (e *Editor) SelectLine(line int) {
	if line < 0 || line >= len(e.Lines) {
		return
	}

	runes := []rune(e.Lines[line])
	e.Selection = composables.NewSelection(
		composables.Position{Line: line, Column: 0},
		composables.Position{Line: line, Column: len(runes)},
	)
	e.CursorLine = line
	e.CursorCol = len(runes)
}

// isWordChar 判断是否为单词字符
func isWordChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// === 按词移动 ===

// MoveByWord 按词移动光标
func (e *Editor) MoveByWord(forward bool) {
	if e.CursorLine < 0 || e.CursorLine >= len(e.Lines) {
		return
	}

	text := e.Lines[e.CursorLine]

	if forward {
		// 向右移动
		if e.CursorCol >= len(text) {
			// 移动到下一行开头
			if e.CursorLine < len(e.Lines)-1 {
				e.CursorLine++
				e.CursorCol = 0
			}
			return
		}

		// 跳过当前单词
		for e.CursorCol < len(text) && isWordChar(rune(text[e.CursorCol])) {
			e.CursorCol++
		}
		// 跳过空白
		for e.CursorCol < len(text) && !isWordChar(rune(text[e.CursorCol])) {
			e.CursorCol++
		}
	} else {
		// 向左移动
		if e.CursorCol <= 0 {
			// 移动到上一行末尾
			if e.CursorLine > 0 {
				e.CursorLine--
				runes := []rune(e.Lines[e.CursorLine])
				e.CursorCol = len(runes)
			}
			return
		}

		// 向左一步
		e.CursorCol--

		// 跳过空白
		for e.CursorCol > 0 && !isWordChar(rune(text[e.CursorCol])) {
			e.CursorCol--
		}
		// 跳到单词开头
		for e.CursorCol > 0 && isWordChar(rune(text[e.CursorCol-1])) {
			e.CursorCol--
		}
	}

	e.EnsureCursorVisible()
}

// === 剪贴板操作 ===

// Copy 复制选中文本到剪贴板
func (e *Editor) Copy() {
	if e.Selection == nil || e.Selection.IsEmpty() {
		return
	}
	if len(e.Lines) == 0 {
		return
	}

	start, end := e.selectionBounds()

	// 多行复制
	if start.Line == end.Line {
		// 单行
		line := []rune(e.Lines[start.Line])
		clipboard = string(line[start.Column:end.Column])
	} else {
		// 多行
		var result strings.Builder
		for i := start.Line; i <= end.Line; i++ {
			if i > start.Line {
				result.WriteString("\n")
			}

			line := []rune(e.Lines[i])
			if i == start.Line {
				result.WriteString(string(line[start.Column:]))
			} else if i == end.Line {
				result.WriteString(string(line[:end.Column]))
			} else {
				result.WriteString(string(line))
			}
		}
		clipboard = result.String()
	}
	if e.clipboardWrite != nil {
		e.clipboardWrite(clipboard)
	}
}

// Cut 剪切选中文本到剪贴板
func (e *Editor) Cut() {
	e.Copy()
	e.DeleteSelection()
}

// Paste 从剪贴板粘贴文本
func (e *Editor) Paste() {
	e.syncClipboard()
	if len(clipboard) == 0 {
		return
	}

	// 如果有选择，先删除
	if e.Selection != nil && !e.Selection.IsEmpty() {
		e.DeleteSelection()
	}

	// 插入剪贴板内容
	lines := strings.Split(clipboard, "\n")

	if len(lines) == 1 {
		// 单行粘贴
		line := []rune(e.Lines[e.CursorLine])
		pasteRunes := []rune(clipboard)
		newLine := string(line[:e.CursorCol]) + clipboard + string(line[e.CursorCol:])
		e.Lines[e.CursorLine] = newLine
		e.CursorCol += len(pasteRunes)
	} else {
		// 多行粘贴
		currentLine := []rune(e.Lines[e.CursorLine])
		leftPart := string(currentLine[:e.CursorCol])
		rightPart := string(currentLine[e.CursorCol:])

		// 第一行
		e.Lines[e.CursorLine] = leftPart + lines[0]

		// 中间行
		for i := 1; i < len(lines)-1; i++ {
			e.Lines = append(e.Lines[:e.CursorLine+i], append([]string{lines[i]}, e.Lines[e.CursorLine+i:]...)...)
		}

		// 最后一行
		lastLine := lines[len(lines)-1] + rightPart
		e.Lines = append(e.Lines[:e.CursorLine+len(lines)-1], append([]string{lastLine}, e.Lines[e.CursorLine+len(lines)-1:]...)...)

		// 更新光标位置
		e.CursorLine += len(lines) - 1
		e.CursorCol = len(lines[len(lines)-1])
	}

	e.Modified = true
	e.markContentChanged()
	e.EnsureCursorVisible()
}

// IsInSelection 判断位置是否在选择区域内
func (e *Editor) IsInSelection(line, col int) bool {
	if e.Selection == nil || e.Selection.IsEmpty() {
		return false
	}

	norm := e.Selection.Normalize()

	if line < norm.Start.Line || line > norm.End.Line {
		return false
	}

	if line == norm.Start.Line && line == norm.End.Line {
		return col >= norm.Start.Column && col < norm.End.Column
	}

	if line == norm.Start.Line {
		return col >= norm.Start.Column
	}

	if line == norm.End.Line {
		return col < norm.End.Column
	}

	return true
}

// === 多光标功能 ===

// AddCursorAt 在指定位置添加光标（Ctrl+Click）
func (e *Editor) AddCursorAt(line, col int, ctrl bool) {
	if !ctrl {
		return
	}

	pos := composables.Position{Line: line, Column: col}
	e.MultiCursor.AddCursor(pos)
}

// SelectNextOccurrence 选中下一个相同的词（Ctrl+D）
func (e *Editor) SelectNextOccurrence() {
	// 获取当前选中的文本或当前词
	selectedText := e.GetSelectedText()
	if selectedText == "" {
		// 没有选择，选中当前词
		e.SelectWord(e.CursorLine, e.CursorCol)
		selectedText = e.GetSelectedText()
		if selectedText == "" {
			return
		}
	}

	// 从当前位置开始查找下一个匹配
	for i := e.CursorLine; i < len(e.Lines); i++ {
		line := e.Lines[i]
		startCol := 0
		if i == e.CursorLine {
			startCol = e.CursorCol + 1
		}

		for col := startCol; col <= len(line)-len(selectedText); col++ {
			if line[col:col+len(selectedText)] == selectedText {
				// 找到匹配，添加光标
				e.MultiCursor.AddCursor(composables.Position{
					Line:   i,
					Column: col,
				})

				// 同时添加选择
				lastIdx := len(e.MultiCursor.Cursors) - 1
				e.MultiCursor.UpdateSelection(lastIdx, &composables.Selection{
					Start: composables.Position{Line: i, Column: col},
					End:   composables.Position{Line: i, Column: col + len(selectedText)},
				})

				return
			}
		}
	}
}

// SelectAllOccurrences 在所有匹配处添加光标（Ctrl+Shift+L）
func (e *Editor) SelectAllOccurrences() {
	selectedText := e.GetSelectedText()
	if selectedText == "" {
		return
	}

	// 清除现有多光标
	e.MultiCursor.Clear()

	// 遍历所有行查找匹配
	for i, line := range e.Lines {
		for col := 0; col <= len(line)-len(selectedText); col++ {
			if line[col:col+len(selectedText)] == selectedText {
				pos := composables.Position{Line: i, Column: col}
				e.MultiCursor.AddCursor(pos)

				// 添加选择
				lastIdx := len(e.MultiCursor.Cursors) - 1
				e.MultiCursor.UpdateSelection(lastIdx, &composables.Selection{
					Start: composables.Position{Line: i, Column: col},
					End:   composables.Position{Line: i, Column: col + len(selectedText)},
				})
			}
		}
	}
}

// ExitMultiCursorMode 退出多光标模式（Esc）
func (e *Editor) ExitMultiCursorMode() {
	e.MultiCursor.Clear()
	e.MultiCursor.Enabled = false
}

// InsertCharMulti 多光标插入字符
func (e *Editor) InsertCharMulti(ch rune) {
	if !e.MultiCursor.Enabled {
		e.InsertChar(ch)
		return
	}

	// 从后往前处理，避免位置变化影响
	e.MultiCursor.Sort()

	for i := len(e.MultiCursor.Cursors) - 1; i >= 0; i-- {
		cursor := e.MultiCursor.Cursors[i]

		// 删除选择（如果有）
		if cursor.Selection != nil && !cursor.Selection.IsEmpty() {
			e.deleteSelectionAt(cursor.Selection)
			cursor.Selection = nil
		}

		// 插入字符
		line := cursor.Position.Line
		col := cursor.Position.Column

		if line >= 0 && line < len(e.Lines) {
			lineText := e.Lines[line]
			newLine := lineText[:col] + string(ch) + lineText[col:]
			e.Lines[line] = newLine

			// 更新光标位置
			e.MultiCursor.UpdateCursor(i, composables.Position{
				Line:   line,
				Column: col + 1,
			})
		}
	}

	e.Modified = true
	e.markContentChanged()
}

// DeleteCharMulti 多光标删除字符
func (e *Editor) DeleteCharMulti() {
	if !e.MultiCursor.Enabled {
		e.DeleteChar()
		return
	}

	// 从后往前处理
	e.MultiCursor.Sort()

	for i := len(e.MultiCursor.Cursors) - 1; i >= 0; i-- {
		cursor := e.MultiCursor.Cursors[i]

		// 如果有选择，删除选择
		if cursor.Selection != nil && !cursor.Selection.IsEmpty() {
			e.deleteSelectionAt(cursor.Selection)
			cursor.Selection = nil
			continue
		}

		// 否则删除前一个字符
		line := cursor.Position.Line
		col := cursor.Position.Column

		if col > 0 {
			lineText := e.Lines[line]
			newLine := lineText[:col-1] + lineText[col:]
			e.Lines[line] = newLine

			e.MultiCursor.UpdateCursor(i, composables.Position{
				Line:   line,
				Column: col - 1,
			})
		}
	}

	e.Modified = true
	e.markContentChanged()
}

// deleteSelectionAt 删除指定选择区域的文本
func (e *Editor) deleteSelectionAt(sel *composables.Selection) {
	if sel == nil || sel.IsEmpty() {
		return
	}

	norm := sel.Normalize()

	// 简化版本：仅支持单行删除
	if norm.Start.Line == norm.End.Line {
		line := e.Lines[norm.Start.Line]
		newLine := line[:norm.Start.Column] + line[norm.End.Column:]
		e.Lines[norm.Start.Line] = newLine
	}
}

// === 代码折叠功能 ===

// UpdateFoldRegions 更新折叠区域（在文本变化后调用）
func (e *Editor) UpdateFoldRegions(mode string) {
	e.visibleMapDirty = true
	switch mode {
	case "indent":
		e.FoldingManager.DetectFoldRegions(e.Lines)
	case "brace":
		e.FoldingManager.DetectFoldRegionsBrace(e.Lines)
	default:
		// 自动检测：优先花括号，否则缩进
		e.FoldingManager.DetectFoldRegionsBrace(e.Lines)
		if len(e.FoldingManager.Regions) == 0 {
			e.FoldingManager.DetectFoldRegions(e.Lines)
		}
	}
}

// ToggleFoldAtCursor 切换当前行的折叠状态
func (e *Editor) ToggleFoldAtCursor() {
	e.FoldingManager.ToggleFold(e.CursorLine)
	e.visibleMapDirty = true
	e.ScrollOffset = 0
}

// FoldAll 折叠所有区域
func (e *Editor) FoldAll() {
	e.FoldingManager.FoldAll()
	e.visibleMapDirty = true
	e.ScrollOffset = 0
}

// UnfoldAll 展开所有区域
func (e *Editor) UnfoldAll() {
	e.FoldingManager.UnfoldAll()
	e.visibleMapDirty = true
	e.ScrollOffset = 0
}

// GetVisibleLineCount 获取可见行数（考虑折叠）
func (e *Editor) GetVisibleLineCount() int {
	return len(e.visibleLineMap())
}

// MapVisibleToActual 将可见行号映射到实际行号
func (e *Editor) MapVisibleToActual(visibleLine int) int {
	visible := e.visibleLineMap()
	if visibleLine >= 0 && visibleLine < len(visible) {
		return visible[visibleLine]
	}
	return len(e.Lines) - 1
}

// MapActualToVisible 将实际行号映射到可见行号
func (e *Editor) MapActualToVisible(actualLine int) int {
	visible := e.visibleLineMap()
	if len(visible) == 0 {
		return 0
	}
	return sort.SearchInts(visible, actualLine)
}

// === Minimap 功能 ===

// ToggleMinimap 切换 Minimap 显示
func (e *Editor) ToggleMinimap() {
	e.ShowMinimap = !e.ShowMinimap
}

// handleMinimapMouseDown 命中 minimap 区域时开始纵向拖动（视口外点击先居中跳转），
// 返回是否消费事件。
func (e *Editor) handleMinimapMouseDown(relX, relY int32) bool {
	if !e.ShowMinimap || e.minimapX < 0 || relX < e.minimapX || relX >= e.minimapX+e.Minimap.Width {
		return false
	}
	e.ScrollLine = e.Minimap.HandleMouseDown(relY, e.minimapH, len(e.Lines), e.VisibleLines, e.ScrollLine)
	e.ScrollOffset = 0
	e.minimapDragging = true
	return true
}

// HandleMinimapMouseMove 处理 minimap 纵向拖动（由窗口 mousemove 转发），
// 返回是否消费事件。
func (e *Editor) HandleMinimapMouseMove(relY int32) bool {
	if !e.minimapDragging {
		return false
	}
	e.ScrollLine = e.Minimap.HandleDragMove(relY, e.minimapH, len(e.Lines), e.VisibleLines)
	e.ScrollOffset = 0
	return true
}
