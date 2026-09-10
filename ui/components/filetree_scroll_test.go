package components

import (
	"fmt"
	"testing"
)

func TestFileTreeWheelScrollClamps(t *testing.T) {
	tree := NewFileTree(".")
	// 造 20 个条目，视口 5 行。
	tree.Entries = make([]FileTreeEntry, 20)
	tree.viewportRows = 5

	tree.HandleWheel(-120 * 3) // 向下滚 3 档 = 3 行
	if tree.ScrollRow != 3 {
		t.Fatalf("ScrollRow = %d, want 3", tree.ScrollRow)
	}
	tree.HandleWheel(-120 * 100) // 远超底部
	if tree.ScrollRow != 15 {    // 20 - 5
		t.Fatalf("ScrollRow = %d, want 15", tree.ScrollRow)
	}
	tree.HandleWheel(120 * 100) // 远超顶部
	if tree.ScrollRow != 0 {
		t.Fatalf("ScrollRow = %d, want 0", tree.ScrollRow)
	}
}

func TestFileTreeMouseDownAccountsForScroll(t *testing.T) {
	tree := NewFileTree(".")
	tree.Entries = make([]FileTreeEntry, 20)
	for i := range tree.Entries {
		tree.Entries[i].Path = fmt.Sprintf("/item%d", i)
	}
	tree.viewportRows = 5
	tree.ScrollRow = 4

	var clickedPath string
	tree.OnSelect = func(path string) { clickedPath = path }
	// 点击列表区第 2 行（originY=40，列表头 58px，RowHeight=28）：
	// 可见第 2 行 = Entries[4+1]。
	tree.HandleMouseDown(100, 40+58+28+10, 0, 40, 240, 600)
	if want := "/item5"; clickedPath != want {
		t.Fatalf("clicked path = %q, want %q (ScrollRow 4 + row 1)", clickedPath, want)
	}
}

func TestFileTreeScrollbarDrag(t *testing.T) {
	tree := NewFileTree(".")
	tree.Entries = make([]FileTreeEntry, 20)
	tree.viewportRows = 5
	tree.ScrollRow = 0

	// 模拟 Render 同步后的滚动条几何（激活态）。
	tree.scrollBarActive = true
	tree.scrollBar.X, tree.scrollBar.Y = 228, 98
	tree.scrollBar.Width, tree.scrollBar.Height = 12, 500
	tree.scrollBar.Total, tree.scrollBar.Visible = 20, 5

	// 点击轨道底部（边界左闭右开，取 590）→ 跳到最大行（20-5=15）。
	if !tree.HandleMouseDown(234, 590, 0, 40, 240, 600) {
		t.Fatal("scrollbar click should be consumed")
	}
	if tree.ScrollRow != 15 {
		t.Fatalf("ScrollRow = %d, want 15", tree.ScrollRow)
	}
	// 拖拽释放。
	if !tree.TreeDragScrollbar(234, 300) {
		t.Fatal("expected active drag after track click")
	}
	tree.TreeScrollbarReleased()
	if tree.TreeDragScrollbar(234, 300) {
		t.Fatal("drag should end after release")
	}
}
