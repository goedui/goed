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

// TestDividerFallsBackToPxWhenNoSpace 可用空间也是 0 时退到 1px：
// 否则会量出 0 宽 / 0 高的线，1px 分隔线直接变成不可见。
func TestDividerFallsBackToPxWhenNoSpace(t *testing.T) {
	ctx := &mockContext{w: 0, h: 0}
	st := renderer.TextStyle{}

	h := Divider()
	if w, hh := measureDivider(ctx, h, 0, 0, st, theme.Light); w != 1 || hh != 1 {
		t.Fatalf("水平线在零空间下 = %gx%g，期望 1x1", w, hh)
	}
	v := Divider(true)
	if w, hh := measureDivider(ctx, v, 0, 0, st, theme.Light); w != 1 || hh != 1 {
		t.Fatalf("竖直线在零空间下 = %gx%g，期望 1x1", w, hh)
	}
}

// TestDividerVerticalHeuristic 竖直判定有两条来源：显式 Divider(true) /
// Attrs["vertical"]，以及「只给了 Width、没给 Height」这条启发式
// （注释里写的「Divider(props) 用 Style.Width/Height」指的就是它）。
// 启发式只认 Height **恰好为 0**，宽高都给时仍是水平线。
func TestDividerVerticalHeuristic(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	st := renderer.TextStyle{}

	// 只给宽 → 当竖直：宽 1，高吃满可用高度。
	byStyle := Divider(runtime.Props{Style: runtime.Style{Width: 1}})
	if w, h := measureDivider(ctx, byStyle, 400, 200, st, theme.Light); w != 1 || h != 200 {
		t.Fatalf("只给 Width 应判为竖直：%gx%g，期望 1x200", w, h)
	}

	// 宽高都给 → 启发式不触发，按水平线走（宽 1、高 50）。
	both := Divider(runtime.Props{Style: runtime.Style{Width: 1, Height: 50}})
	if w, h := measureDivider(ctx, both, 400, 200, st, theme.Light); w != 1 || h != 50 {
		t.Fatalf("宽高都给应判为水平：%gx%g，期望 1x50", w, h)
	}
}

// TestDividerLayoutWritesRect layout 只做一件事：把矩形写进节点。
func TestDividerLayoutWritesRect(t *testing.T) {
	n := Divider()
	layoutDivider(&mockContext{w: 400, h: 200}, n, 4, 8, 392, 1, renderer.TextStyle{}, theme.Light)
	if n.X != 4 || n.Y != 8 || n.W != 392 || n.H != 1 {
		t.Fatalf("布局后矩形 (%g,%g,%g,%g)，期望 (4,8,392,1)", n.X, n.Y, n.W, n.H)
	}
}

// TestDividerPaintColorLadder 取色阶梯：主题分隔色 → Style.Background → Style.Color
// → A==0 时兜底浅灰。四级缺一不可 —— 1px 画成透明就等于这条线看不见。
func TestDividerPaintColorLadder(t *testing.T) {
	cases := []struct {
		name  string
		style runtime.Style
		th    theme.Theme
		want  renderer.Color
	}{
		{"默认用主题分隔色", runtime.Style{}, theme.Dark,
			renderer.ColorFrom(theme.Dark.Separator)},
		{"Style.Background 压过主题色", runtime.Style{Background: runtime.RGB(1, 2, 3)}, theme.Dark,
			renderer.ColorFrom(runtime.RGB(1, 2, 3))},
		{"没有 Background 时 Style.Color 次之", runtime.Style{Color: runtime.RGB(4, 5, 6)}, theme.Dark,
			renderer.ColorFrom(runtime.RGB(4, 5, 6))},
		{"主题分隔色透明 → 兜底浅灰", runtime.Style{}, theme.Theme{},
			renderer.RGB(228, 231, 236)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := Divider(runtime.Props{Style: c.style})
			layoutDivider(nil, n, 0, 0, 100, 1, renderer.TextStyle{}, c.th)
			ctx := &mockContext{w: 400, h: 200}
			paintDivider(ctx, n, renderer.TextStyle{}, c.th)

			if len(ctx.fills) != 1 {
				t.Fatalf("应画 1 条线，实际 %d 条", len(ctx.fills))
			}
			if got := ctx.fills[0].c; got != c.want {
				t.Fatalf("颜色 %+v，期望 %+v", got, c.want)
			}
			if f := ctx.fills[0]; f.w != 100 || f.h != 1 {
				t.Fatalf("线几何 %gx%g，期望 100x1", f.w, f.h)
			}
		})
	}
}

// TestDividerPaintNilGuardsAreLoadBearing paint 开头的守卫是承重的：
// 去掉之后 paintDivider(ctx, nil, …) 会在 renderer.ColorSet(n.Style.Background)
// 上解引用空指针（ColorFrom(th.Separator) 那一步不碰 n，挡不住它）。
func TestDividerPaintNilGuardsAreLoadBearing(t *testing.T) {
	n := Divider()
	layoutDivider(nil, n, 0, 0, 100, 1, renderer.TextStyle{}, theme.Dark)

	// ctx 为 nil：静默返回，不该把 nil 上下文递给下游。
	paintDivider(nil, n, renderer.TextStyle{}, theme.Dark)
	// n 为 nil：不能 panic（ColorSet(n.Style.Background) 会解引用它）。
	paintDivider(&mockContext{w: 400, h: 200}, nil, renderer.TextStyle{}, theme.Dark)
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
