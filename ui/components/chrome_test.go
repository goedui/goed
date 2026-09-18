package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestDividerSizes(t *testing.T) {
	h := Divider()
	v := Divider(true)
	ctx := &mockContext{w: 400, h: 200}
	hw, hh := measureDivider(ctx, h, 400, 200, renderer.TextStyle{}, theme.Light)
	if hw != 400 || hh != 1 {
		t.Fatalf("水平分隔线尺寸 = %.1fx%.1f", hw, hh)
	}
	vw, vh := measureDivider(ctx, v, 400, 200, renderer.TextStyle{}, theme.Light)
	if vw != 1 || vh != 200 {
		t.Fatalf("竖直分隔线尺寸 = %.1fx%.1f", vw, vh)
	}
}

func TestSelectOpensPopoverAndChooses(t *testing.T) {
	SetSelectOpen("size", false)
	t.Cleanup(func() { SetSelectOpen("size", false) })

	var got string
	n := Select(runtime.Props{
		Style:    runtime.Style{Width: 160, Height: 34},
		ID:       "size",
		Value:    "1024×1024",
		Items:    []string{"1024×1024", "1536×1024", "auto"},
		OnChange: func(v string) { got = v },
	})
	ctx := &mockContext{w: 640, h: 480}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	if runtime.FindByID([]*runtime.VNode{n}, "size") == nil {
		t.Fatal("缺少 select")
	}
	n.OnClick()
	n = Select(runtime.Props{
		Style:    runtime.Style{Width: 160, Height: 34},
		ID:       "size",
		Value:    "1024×1024",
		Items:    []string{"1024×1024", "1536×1024", "auto"},
		OnChange: func(v string) { got = v },
	})
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	var pop *runtime.VNode
	runtime.Walk([]*runtime.VNode{n}, func(x *runtime.VNode) bool {
		if x != nil && x.Tag == "popover" {
			pop = x
			return true
		}
		return false
	})
	if pop == nil || pop.W < 80 {
		t.Fatalf("展开后应有 popover: %+v", pop)
	}
	if !runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyDown, Down: true}) {
		t.Fatal("Down 应由 popover 消费")
	}
	if got != "1536×1024" {
		t.Fatalf("键盘选择 = %q", got)
	}
}

func TestMenuBarOpensAndRunsAction(t *testing.T) {
	var ran bool
	open := ""
	build := func() *runtime.VNode {
		return MenuBar(runtime.Props{
			Style: runtime.Style{Height: 30, Flex: 1},
			ID:    "bar",
			Open:  open,
			OnOpen: func(id string) { open = id },
			Menus: []Menu{
				{
					ID:    "menu-file",
					Label: "文件(F)",
					Items: []MenuItem{
						{Label: "新建", Shortcut: "Ctrl+N", Action: func() { ran = true }},
						{Sep: true},
						{Label: "退出"},
					},
				},
			},
		})
	}
	ctx := &mockContext{w: 800, h: 400}
	n := build()
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	btn := runtime.FindByID([]*runtime.VNode{n}, "menu-file")
	if btn == nil || btn.OnClick == nil || btn.W < 20 {
		t.Fatalf("菜单按钮不可点: %+v", btn)
	}
	btn.OnClick()
	n = build()
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	if open != "menu-file" {
		t.Fatalf("open = %q", open)
	}
	pop := runtime.FindByID([]*runtime.VNode{n}, "bar-popup")
	if pop == nil || pop.W < 80 {
		t.Fatalf("下拉缺失: %+v", pop)
	}
	if !runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyEnter, Down: true}) {
		t.Fatal("Enter 应选第一项")
	}
	if !ran {
		t.Fatal("应执行新建")
	}
}

func TestTabsSelectsAndCloses(t *testing.T) {
	var got, closed string
	n := Tabs(runtime.Props{
		Style: runtime.Style{Height: 32, Flex: 1},
		ID:    "bar",
		Value: "a",
		Items: []Tab{
			{ID: "a", Label: "结果"},
			{ID: "b", Label: "历史", Closable: true, CloseID: "b-close"},
		},
		OnChange: func(id string) { got = id },
		OnClose:  func(id string) { closed = id },
	})
	ctx := &mockContext{w: 640, h: 80}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	b := runtime.FindByID([]*runtime.VNode{n}, "b")
	if b == nil || b.OnClick == nil || b.W < 20 {
		t.Fatalf("标签不可点: %+v", b)
	}
	b.OnClick()
	if got != "b" {
		t.Fatalf("onChange = %q", got)
	}
	x := runtime.FindByID([]*runtime.VNode{n}, "b-close")
	if x == nil || x.OnClick == nil {
		t.Fatal("缺少关闭按钮")
	}
	x.OnClick()
	if closed != "b" {
		t.Fatalf("onClose = %q", closed)
	}
}

func TestTabsPillTrailing(t *testing.T) {
	n := Tabs(runtime.Props{
		Style:   runtime.Style{Height: 34, Gap: 8},
		Variant: "pill",
		Value:   "one",
		Items:   []Tab{{ID: "one", Label: "配置一"}, {ID: "two", Label: "配置二"}},
	}, Button(runtime.Props{ID: "add", Label: "+"}, "+"))
	ctx := &mockContext{w: 800, h: 80}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	if runtime.FindByID([]*runtime.VNode{n}, "add") == nil {
		t.Fatal("trailing 子节点应保留")
	}
	one := runtime.FindByID([]*runtime.VNode{n}, "one")
	if one == nil || !renderer.AttrBool(one, "selected") {
		t.Fatal("选中项应带 selected")
	}
}

func TestOverlayHitsAboveContent(t *testing.T) {
	var hit string
	editor := runtime.H("button", runtime.Props{OnClick: func() { hit = "editor" }})
	editor.X, editor.Y, editor.W, editor.H = 0, 40, 400, 200
	pop := runtime.H("popover", runtime.Props{
		OnClick: func() { hit = "pop" },
	})
	pop.X, pop.Y, pop.W, pop.H = 10, 50, 120, 80
	tree := []*runtime.VNode{
		runtime.H("vstack", editor, pop),
	}
	hits := runtime.CollectHits(tree)
	fn := runtime.HitTest(hits, 20, 60)
	if fn == nil {
		t.Fatal("应命中浮层")
	}
	fn()
	if hit != "pop" {
		t.Fatalf("浮层应盖住编辑区，got %q", hit)
	}
}
