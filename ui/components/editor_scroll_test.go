package components

import (
	"reflect"
	"testing"

	"github.com/goedui/goed/ui/composables"
)

func TestEditorWheelScrollUsesConfiguredSensitivity(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4", "5"}
	e.VisibleLines = 3
	e.visibleMapDirty = true

	e.HandleMouseWheel(-120, false)
	if e.ScrollLine != 3 || e.ScrollOffset != 0 {
		t.Fatalf("downward wheel = line %d offset %d, want line 3 offset 0", e.ScrollLine, e.ScrollOffset)
	}
	e.HandleMouseWheel(120, false)
	if e.ScrollLine != 0 || e.ScrollOffset != 0 {
		t.Fatalf("upward wheel = line %d offset %d, want line 0 offset 0", e.ScrollLine, e.ScrollOffset)
	}
}

func TestEditorScrollClampsToLastVisibleViewport(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4"}
	e.VisibleLines = 3
	e.visibleMapDirty = true

	e.ScrollByPixels(1000)
	if e.ScrollLine != 2 || e.ScrollOffset != 0 {
		t.Fatalf("bottom scroll = line %d offset %d, want line 2 offset 0", e.ScrollLine, e.ScrollOffset)
	}
}

func TestEditorScrollBarSyncsToVisibleWindow(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4", "5"}
	e.VisibleLines = 3
	e.visibleMapDirty = true
	e.HandleMouseWheel(-120, false) // 滚到第 3 行（可见窗口起始索引 3）

	// Render 时调用的几何同步：编辑器宽 400、高 72（3 行）。
	if !e.updateScrollBarGeometry(0, 0, 400, 72, 3, e.VisibleLines) {
		t.Fatal("scrollbar should be active when content exceeds viewport")
	}
	if got := e.scrollBar.Position; got != 3 {
		t.Fatalf("scrollbar position = %d, want 3", got)
	}
	if e.scrollBar.Total != 6 || e.scrollBar.Visible != 3 {
		t.Fatalf("scrollbar total/visible = %d/%d, want 6/3", e.scrollBar.Total, e.scrollBar.Visible)
	}
}

func TestEditorScrollBarDragScrollsEditor(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4", "5"}
	e.VisibleLines = 3
	e.visibleMapDirty = true

	// Render 同步后的几何：编辑器宽 400、高 120。
	if !e.updateScrollBarGeometry(0, 0, 400, 120, 0, 3) {
		t.Fatal("scrollbar should be active")
	}

	// 点击滚动条带（本地 x >= 384）底部：轨道跳转到最大位置（6-3=3）。
	e.HandleMouseDown(392, 119, 0, false, false, 0)
	if e.ScrollLine != 3 || e.ScrollOffset != 0 {
		t.Fatalf("track jump = line %d offset %d, want line 3 offset 0", e.ScrollLine, e.ScrollOffset)
	}

	// 释放后拖拽状态清除。
	e.HandleMouseUp()
	if e.scrollBar.dragging {
		t.Fatal("dragging should clear on mouse up")
	}
}

func TestEditorFoldGutterClickTogglesFold(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4", "5", "6"}
	e.FoldingManager.Regions = []composables.FoldRegion{{StartLine: 1, EndLine: 4, Folded: false}}
	e.visibleMapDirty = true

	// gutter 宽 20px，点击第 1 行图标（可见窗口从第 0 行开始）。
	e.HandleMouseDown(10, 1*24+12, 0, false, false, 0)
	if !e.FoldingManager.Regions[0].Folded {
		t.Fatal("gutter click should fold region at line 1")
	}

	// 再次点击展开。
	e.HandleMouseDown(10, 1*24+12, 0, false, false, 0)
	if e.FoldingManager.Regions[0].Folded {
		t.Fatal("second gutter click should unfold region")
	}
}

func TestEditorHorizontalScrollClampsToLongestLine(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"short", "a much much much longer line of text"}
	e.VisibleLines = 3
	e.visibleMapDirty = true
	e.textViewportW = 100 // 只能容纳约 11 个字符

	e.ScrollXPx = 10000
	e.clampScrollX()
	max := int32(e.maxLineWidth()) + 16 - e.textViewportW
	if e.ScrollXPx != max {
		t.Fatalf("clamped scrollX = %d, want %d", e.ScrollXPx, max)
	}
	e.ScrollXPx = -50
	e.clampScrollX()
	if e.ScrollXPx != 0 {
		t.Fatalf("negative scrollX = %d, want 0", e.ScrollXPx)
	}
}

func TestEditorCursorFollowsHorizontalScroll(t *testing.T) {
	e := NewEditor()
	long := "x"
	for i := 0; i < 60; i++ {
		long += "x"
	}
	e.Lines = []string{long}
	e.visibleMapDirty = true
	e.textViewportW = 9 * 10 // 10 列视口

	// 光标移动到行尾（60 列）后横向滚动应跟随，末端留一列可见。
	e.CursorCol = 61
	e.ensureCursorVisibleX()
	if e.ScrollXPx <= 0 {
		t.Fatalf("scrollX = %d, want > 0 after cursor jump to EOL", e.ScrollXPx)
	}
	// 光标回到行首后横向偏移应归零。
	e.CursorCol = 0
	e.ensureCursorVisibleX()
	if e.ScrollXPx != 0 {
		t.Fatalf("scrollX = %d, want 0 after cursor jump to BOL", e.ScrollXPx)
	}
}

func TestEditorVisibleLineMapSkipsFoldedRanges(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"0", "1", "2", "3", "4", "5", "6"}
	e.FoldingManager.Regions = []composables.FoldRegion{{StartLine: 1, EndLine: 4, Folded: true}}
	e.visibleMapDirty = true

	want := []int{0, 1, 5, 6}
	if got := e.visibleLineMap(); !reflect.DeepEqual(got, want) {
		t.Fatalf("visible line map = %v, want %v", got, want)
	}
	if got := e.MapVisibleToActual(2); got != 5 {
		t.Fatalf("visible line 2 maps to %d, want 5", got)
	}
}
