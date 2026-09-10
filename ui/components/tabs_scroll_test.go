package components

import "testing"

func newScrollTestTabs() *Tabs {
	t := NewTabs()
	for _, p := range []string{"a.go", "b.go", "c.go"} {
		t.Open(p)
	}
	// 3 × 150 = 450 总宽；可用宽度 264 时最大滚动 186。
	return t
}

// TestTabsScrollHitTestAccountsForOffset 验证命中测试跟随滚动偏移。
func TestTabsScrollHitTestAccountsForOffset(t *testing.T) {
	tabs := newScrollTestTabs()
	tabs.clampScrollX(264)
	tabs.ScrollX = 150 // a.go 完全滚出，b.go 从条带左缘开始
	selected := ""
	tabs.OnSelect = func(p string) { selected = p }
	// originX=0: 偏移后 cursor=-150 → b.go 占 [0,150)，点击 10 命中 b.go。
	if !tabs.HandleMouseDown(10, 10, 0, 0, 300) {
		t.Fatal("click not handled")
	}
	if selected != "b.go" {
		t.Fatalf("selected = %q, want b.go", selected)
	}
}

// TestTabsWheelScrollsHorizontally 验证滚轮横向滚动与方向。
func TestTabsWheelScrollsHorizontally(t *testing.T) {
	tabs := newScrollTestTabs()
	tabs.HandleWheel(120) // 滚轮上 → 向左
	if tabs.ScrollX != 0 {
		t.Fatalf("ScrollX = %d, want 0 (clamped at left edge)", tabs.ScrollX)
	}
	tabs.HandleWheel(-240) // 滚轮下两档 → 向右 180px
	if tabs.ScrollX != 180 {
		t.Fatalf("ScrollX = %d, want 180", tabs.ScrollX)
	}
	tabs.HandleWheel(-120) // 再一档 → 270px，超出上界
	tabs.clampScrollX(264) // Render 每帧夹紧上界：450-264=186
	if tabs.ScrollX != 186 {
		t.Fatalf("ScrollX = %d, want 186", tabs.ScrollX)
	}
}

// TestTabsRightClickOnTabShowsMenu 验证标签右键弹出菜单且不切换活动标签。
func TestTabsRightClickOnTabShowsMenu(t *testing.T) {
	tabs := newScrollTestTabs()
	tabs.Active = 2 // 活动标签为 c.go
	var got *Tab
	tabs.OnContextMenu = func(tab *Tab, x, y int32) []MenuItem {
		got = tab
		return []MenuItem{{Label: "关闭", Enabled: true}}
	}
	if !tabs.HandleRightClick(10, 10, 0, 0, 500, 800, 600) {
		t.Fatal("right click on strip not handled")
	}
	if got == nil || got.Path != "a.go" {
		t.Fatalf("context tab = %#v, want a.go", got)
	}
	if tabs.Active != 2 {
		t.Fatalf("active = %d, want 2 (right click must not switch tabs)", tabs.Active)
	}
	if !tabs.ContextMenu.IsVisible {
		t.Fatal("context menu should be visible")
	}
}

// TestTabsRightClickBlankStripNoMenu 验证空白条带右键不弹菜单但消费点击。
func TestTabsRightClickBlankStripNoMenu(t *testing.T) {
	tabs := newScrollTestTabs()
	called := false
	tabs.OnContextMenu = func(tab *Tab, x, y int32) []MenuItem {
		called = true
		return nil
	}
	// 450 总宽，条带宽 520：点击 470 落在"+"按钮区，无标签。
	if !tabs.HandleRightClick(470, 10, 0, 0, 520, 800, 600) {
		t.Fatal("right click on strip should be consumed")
	}
	if called || tabs.ContextMenu.IsVisible {
		t.Fatal("blank strip must not open a context menu")
	}
}

// TestTabsRightClickOutsideStrip 验证条带外右键不被消费。
func TestTabsRightClickOutsideStrip(t *testing.T) {
	tabs := newScrollTestTabs()
	if tabs.HandleRightClick(10, 100, 0, 0, 500, 800, 600) {
		t.Fatal("right click below strip should not be handled")
	}
}
