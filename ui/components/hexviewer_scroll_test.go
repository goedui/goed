package components

import (
	"testing"

	"github.com/goedui/goed/ui/core"
)

func newBigHexFixture(rows int) *HexViewer {
	v := NewHexViewer()
	data := make([]byte, rows*hexBytesPerRow)
	for i := range data {
		data[i] = byte(i % 251)
	}
	v.Load("big.bin", data, int64(len(data)), false)
	v.viewportRows = 5
	v.clampFirstRow()
	return v
}

// TestHexViewerScrollbarGeometry 验证溢出时滚动条激活与几何/位置同步。
func TestHexViewerScrollbarGeometry(t *testing.T) {
	// 视口高度取 5 行：Render 每帧按 h 重算 viewportRows，
	// h = rows*RowHeight + 头部 + 底栏 才能得到 Visible=5。
	const rows = 5
	h := int32(rows*hexRowHeight + hexHeaderH + hexFooterH)
	v := newBigHexFixture(40) // 40 行 > 5 行视口
	v.charW = 8
	v.hexX = 100
	v.asciiX = 500
	v.Render(&nopDrawing{}, 0, 0, 700, h)

	if !v.scrollBarActive {
		t.Fatal("scrollbar should be active for overflowing content")
	}
	if v.scrollBar.Total != 40 || v.scrollBar.Visible != rows {
		t.Fatalf("total=%d visible=%d", v.scrollBar.Total, v.scrollBar.Visible)
	}
	if v.scrollBar.Position != int32(v.FirstRow) {
		t.Fatalf("position %d != firstRow %d", v.scrollBar.Position, v.FirstRow)
	}
	if v.scrollBar.X != 700-12 || v.scrollBar.Height != h-hexHeaderH-hexFooterH {
		t.Fatalf("geometry x=%d h=%d", v.scrollBar.X, v.scrollBar.Height)
	}

	// 内容不溢出：滚动条不激活。
	small := newBigHexFixture(3)
	small.Render(&nopDrawing{}, 0, 0, 700, h)
	if small.scrollBarActive {
		t.Fatal("scrollbar should be inactive when content fits")
	}
}

// TestHexViewerScrollbarDrag 验证拖拽 thumb 更新 FirstRow。
func TestHexViewerScrollbarDrag(t *testing.T) {
	v := newBigHexFixture(40)
	// 模拟 Render 已设置几何后开始拖拽。
	v.scrollBarActive = true
	v.scrollBar.Total = int32(v.totalRows())
	v.scrollBar.Visible = 5
	v.scrollBar.Height = 300 - hexHeaderH - hexFooterH
	v.scrollBar.Position = 0

	v.scrollBar.dragging = true
	v.scrollBar.Position = 20 // HandleMouseMove 按鼠标位置计算；直接置位模拟
	// HandleMouseDown 开始拖拽时会记录 posStart=Position；测试绕过
	// MouseDown 直接置 dragging，需补齐该簿记字段。
	v.scrollBar.posStart = 20
	if !v.HexDragScrollbar(0, 0) {
		t.Fatal("drag should be reported active")
	}
	if v.FirstRow != 20 {
		t.Fatalf("firstRow = %d, want 20", v.FirstRow)
	}
	v.HexScrollbarReleased()
	if v.HexDragScrollbar(0, 0) {
		t.Fatal("drag must end after release")
	}
}

// TestHexViewerMouseDownOnScrollbar 验证滚动条条带点击优先于字节选择。
func TestHexViewerMouseDownOnScrollbar(t *testing.T) {
	v := newBigHexFixture(40)
	v.charW = 8
	v.hexX = 100
	v.asciiX = 500
	v.Render(&nopDrawing{}, 0, 0, 700, 300)

	sbX := v.scrollBar.X
	if !v.HandleMouseDown(sbX+2, 150, 0, 0, 700, 300, false) {
		t.Fatal("scrollbar click should be consumed")
	}
	if v.HasSelection() {
		t.Fatal("scrollbar click must not select bytes")
	}
}

// nopDrawing 空绘图上下文，供无窗口渲染测试。
type nopDrawing struct{}

func (nopDrawing) DrawRect(x, y, w, h int32, color uint32, filled bool)      {}
func (nopDrawing) DrawLine(x1, y1, x2, y2 int32, color uint32, width int32)  {}
func (nopDrawing) DrawRoundedRect(x, y, w, h, radius int32, c uint32, f bool) {}
func (nopDrawing) DrawEllipse(x, y, w, h int32, color uint32, filled bool)   {}
func (nopDrawing) DrawPolygon(points []core.Point, color uint32, filled bool) {}
func (nopDrawing) DrawText(x, y int32, text string, color uint32) {}
func (nopDrawing) MeasureText(text string) (width, height int32)  { return int32(len(text)) * 7, 14 }
func (nopDrawing) SetFont(name string, size int32)                {}
func (nopDrawing) SetBkMode(transparent bool)                     {}
func (nopDrawing) BeginDraw()                                     {}
func (nopDrawing) EndDraw()                                       {}
func (nopDrawing) SetClipRect(x, y, w, h int32)                   {}
func (nopDrawing) ResetClip()                                     {}
