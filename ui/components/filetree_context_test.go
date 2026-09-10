package components

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileTreeRightClickSelectsWithoutOpening 验证右键仅选中并请求菜单项，
// 不触发 OnOpen（与左键语义区分）。
func TestFileTreeRightClickSelectsWithoutOpening(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := NewFileTree(root)
	opened := ""
	tree.OnOpen = func(p string) { opened = p }
	var gotEntry *FileTreeEntry
	tree.OnContextMenu = func(entry *FileTreeEntry, x, y int32) []MenuItem {
		gotEntry = entry
		return []MenuItem{{Label: "打开", Enabled: true}}
	}
	// 行 0 是根目录，行 1 是 main.go（列表起点 y=58）。
	if !tree.HandleRightClick(20, 58+tree.RowHeight, 0, 0, 300, 300, 800, 600) {
		t.Fatal("right click inside tree not handled")
	}
	if opened != "" {
		t.Fatalf("right click must not open files, opened %q", opened)
	}
	if gotEntry == nil || gotEntry.Path != path {
		t.Fatalf("context entry = %#v, want %q", gotEntry, path)
	}
	if tree.Selected != path {
		t.Fatalf("selected = %q, want %q", tree.Selected, path)
	}
	if !tree.ContextMenu.IsVisible || len(tree.ContextMenu.Items) != 1 {
		t.Fatal("context menu should be visible with requested items")
	}
}

// TestFileTreeRightClickBlankArea 验证列表下方空白区域右键 entry 为 nil
// （供"新建文件/文件夹、刷新"菜单使用）。
func TestFileTreeRightClickBlankArea(t *testing.T) {
	root := t.TempDir()
	tree := NewFileTree(root)
	var gotEntry *FileTreeEntry
	called := false
	tree.OnContextMenu = func(entry *FileTreeEntry, x, y int32) []MenuItem {
		called = true
		gotEntry = entry
		return []MenuItem{{Label: "刷新", Enabled: true}}
	}
	if !tree.HandleRightClick(20, 58+tree.RowHeight*10, 0, 0, 300, 400, 800, 600) {
		t.Fatal("right click inside tree not handled")
	}
	if !called || gotEntry != nil {
		t.Fatalf("blank area entry = %#v, want nil (called=%v)", gotEntry, called)
	}
}

// TestFileTreeRightClickOutsideBounds 验证树区域外的右键不被消费。
func TestFileTreeRightClickOutsideBounds(t *testing.T) {
	tree := NewFileTree(t.TempDir())
	tree.OnContextMenu = func(entry *FileTreeEntry, x, y int32) []MenuItem { return nil }
	if tree.HandleRightClick(400, 100, 0, 0, 300, 300, 800, 600) {
		t.Fatal("right click outside tree should not be handled")
	}
}

// TestContextMenuShowClampedKeepsMenuInside 验证靠边弹出时菜单被收纳进窗口。
func TestContextMenuShowClampedKeepsMenuInside(t *testing.T) {
	cm := NewContextMenu()
	items := []MenuItem{{Label: "A", Enabled: true}, {Label: "B", Enabled: true}}
	cm.ShowClamped(780, 570, 800, 600, items)
	menuH := int32(len(items)) * cm.ItemHeight
	if cm.X < 0 || cm.Y < 0 || cm.X+cm.Width > 800 || cm.Y+menuH > 600 {
		t.Fatalf("menu out of bounds: x=%d y=%d w=%d h=%d", cm.X, cm.Y, cm.Width, menuH)
	}
}
