package components

import (
	"strings"
	"testing"
)

func newHexFixture() *HexViewer {
	v := NewHexViewer()
	data := make([]byte, 40) // 2.5 行
	for i := range data {
		data[i] = byte(i)
	}
	v.Load("demo.bin", data, 40, false)
	return v
}

// TestHexViewerSelectionFromPoint 验证三列命中映射与偏移列不选中。
func TestHexViewerSelectionFromPoint(t *testing.T) {
	v := newHexFixture()
	v.charW = 8
	v.hexX = 100
	v.asciiX = 100 + 16*24 + 8 + 8
	viewW, viewH := int32(700), int32(200)

	// 偏移列（x<100）：不选中。
	if got := v.selectionFromPoint(10, hexHeaderH+5, viewW, viewH); got != -1 {
		t.Fatalf("offset col = %d, want -1", got)
	}
	// hex 列第 0 个字节：字节 0。
	if got := v.selectionFromPoint(v.hexX+2, hexHeaderH+5, viewW, viewH); got != 0 {
		t.Fatalf("hex col 0 = %d, want 0", got)
	}
	// hex 列第 9 个字节（跳过组间距）：字节 9。
	 ninth := v.hexX + 9*24 + 8
	if got := v.selectionFromPoint(ninth, hexHeaderH+5, viewW, viewH); got != 9 {
		t.Fatalf("hex col 9 = %d, want 9", got)
	}
	// ascii 列第 3 个字符：字节 3。
	if got := v.selectionFromPoint(v.asciiX+3*8, hexHeaderH+5, viewW, viewH); got != 3 {
		t.Fatalf("ascii col 3 = %d, want 3", got)
	}
	// 头部区域：不选中。
	if got := v.selectionFromPoint(v.hexX+2, 5, viewW, viewH); got != -1 {
		t.Fatalf("header = %d, want -1", got)
	}
}

// TestHexViewerShiftRangeAndCopy 验证 Shift 范围选择与复制文本。
func TestHexViewerShiftRangeAndCopy(t *testing.T) {
	v := newHexFixture()
	v.charW = 8
	v.hexX = 100
	v.asciiX = 300

	if !v.HandleMouseDown(v.hexX+2, hexHeaderH+5, 0, 0, 700, 200, false) {
		t.Fatal("click not handled")
	}
	if v.Selected != 0 || v.SelectedEnd != 0 {
		t.Fatalf("first click = %d..%d", v.Selected, v.SelectedEnd)
	}
	v.HandleMouseDown(v.asciiX+8*8, hexHeaderH+5, 0, 0, 700, 200, true) // ascii 第 8 字符 → 字节 8
	if lo, hi, ok := v.selectionRange(); !ok || lo != 0 || hi != 8 {
		t.Fatalf("range = %d..%d ok=%v", lo, hi, ok)
	}
	got := v.CopyText()
	want := "00 01 02 03 04 05 06 07 08"
	if got != want {
		t.Fatalf("copy = %q, want %q", got, want)
	}
	if !strings.HasPrefix(v.SelectionInfo(), "0x00000000 - 0x00000008") {
		t.Fatalf("info = %q", v.SelectionInfo())
	}
}

// TestHexViewerWheelClamp 验证滚动夹紧。
func TestHexViewerWheelClamp(t *testing.T) {
	v := newHexFixture()
	v.viewportRows = 2
	v.HandleWheel(120)
	if v.FirstRow != 0 {
		t.Fatalf("up at top = %d", v.FirstRow)
	}
	v.HandleWheel(-120 * 100)
	v.clampFirstRow()
	if max := v.totalRows() - 2; v.FirstRow != max {
		t.Fatalf("bottom = %d, want %d", v.FirstRow, max)
	}
}

// TestHexViewerCopyNoSelection 验证无选择时复制/信息为空。
func TestHexViewerCopyNoSelection(t *testing.T) {
	v := newHexFixture()
	if v.CopyText() != "" || v.SelectionInfo() != "" || v.HasSelection() {
		t.Fatal("no selection expected")
	}
}
