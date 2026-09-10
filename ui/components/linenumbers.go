package components

import (
	"fmt"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// LineNumbers 行号列组件
type LineNumbers struct {
	Width        int32
	StartLine    int
	VisibleLines int
	CurrentLine  int
	ScrollOffset int32
	LineIndices  []int
}

func NewLineNumbers() *LineNumbers {
	return &LineNumbers{
		Width:        60,
		StartLine:    1,
		VisibleLines: 20,
		CurrentLine:  1,
	}
}

func (ln *LineNumbers) Render(ctx core.DrawingContext, x, y, height int32) {
	th := theme.GetTheme()
	ctx.SetFont(th.EditorFontFamily, th.EditorFontSize)

	// 背景
	ctx.DrawRect(x, y, ln.Width, height, th.EditorLineNumberBg, true)

	// 右边框
	ctx.DrawLine(x+ln.Width-1, y, x+ln.Width-1, y+height, th.WindowBorder, 1)

	// 行号
	lineHeight := int32(24)
	rows := ln.VisibleLines
	if ln.ScrollOffset > 0 {
		rows++
	}
	for i := 0; i < rows; i++ {
		lineNum := ln.StartLine + i
		if i < len(ln.LineIndices) {
			lineNum = ln.LineIndices[i] + 1
		}
		lineY := y + int32(i)*lineHeight - ln.ScrollOffset

		// 当前行高亮
		if lineNum == ln.CurrentLine {
			ctx.DrawRect(x, lineY, ln.Width, lineHeight, th.EditorCurrentLineBg, true)
		}

		// 行号文字（右对齐）
		lineText := fmt.Sprintf("%d", lineNum)
		textWidth, _ := ctx.MeasureText(lineText)
		ctx.DrawText(x+ln.Width-textWidth-12, lineY+2, lineText, th.EditorLineNumberFg)
	}
}

// SetScroll 设置滚动位置
func (ln *LineNumbers) SetScroll(startLine int) {
	ln.StartLine = startLine
	ln.LineIndices = nil
}

// SetLineIndices keeps the gutter aligned with the editor's folded visible
// line map. Passing nil restores the traditional sequential StartLine mode.
func (ln *LineNumbers) SetLineIndices(indices []int) {
	ln.LineIndices = indices
}

// SetCurrentLine 设置当前行
func (ln *LineNumbers) SetCurrentLine(line int) {
	ln.CurrentLine = line
}

// SetVisibleLines 设置可见行数
func (ln *LineNumbers) SetVisibleLines(count int) {
	ln.VisibleLines = count
}
