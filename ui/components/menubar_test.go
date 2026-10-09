package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// sampleMenuBar 造一个展开着「文件」菜单的菜单栏，用来验证下拉的几何。
//
// 用 4 项（含 1 条分隔线）而不是 1 项：只有项数够多，弹出层的高度才会明显
// 超出菜单栏自己的高度，「被菜单栏夹住」这个错误才暴露得出来。
func sampleMenuBar(open string) *runtime.VNode {
	return MenuBar(runtime.Props{
		ID: "menubar",
		Style: runtime.Style{
			Height: 32, Padding: 3, Gap: 2,
			Background: theme.Dark.Titlebar,
			Color:      theme.Dark.Foreground,
			FontSize:   13,
		},
		Menus: []Menu{{
			ID:    "menu-file",
			Label: "文件(F)",
			Items: []MenuItem{
				{Label: "新建", Shortcut: "Ctrl+N"},
				{Label: "打开", Shortcut: "Ctrl+O"},
				{Sep: true},
				{Label: "保存", Shortcut: "Ctrl+S"},
			},
		}},
		Open:   open,
		OnOpen: func(string) {},
	})
}

func popupOf(t *testing.T, bar *runtime.VNode) *runtime.VNode {
	t.Helper()
	pop := runtime.FindByID([]*runtime.VNode{bar}, "menubar-popup")
	if pop == nil {
		t.Fatal("没找到 popover 节点 menubar-popup")
	}
	return pop
}

// TestMenuBarPopupIsLaidOut 钉住「下拉菜单的底色只盖住第一项」这个 bug。
//
// 根因在度量而不在绘制：layoutMenuBar 拿**菜单栏自己的高度**（32 DIP）当 maxH
// 去量 popover，而 renderer.applySize 会把子节点高度夹到 maxH。于是弹出层被
// 夹成 32 DIP 高 —— paintPopover 的 FillRoundedRect 只盖住第一项，分隔线那
// 8 DIP 的空档直接透出底下的界面（编辑器里就是侧栏的文件名）。
//
// 症状难发现的另一个原因：菜单项是 ghost 按钮，底色恰好也是 th.Titlebar，
// 和 paintPopover 给弹出层铺的底色**完全同色**，所以除分隔线外看不出异常。
//
// 判据用「每一项都落在弹出层矩形内」，而不是断言某个具体高度：高度取决于
// 字体行高，写死数字会随字体变脆；而「底色能不能盖住所有项」正是这个 bug
// 的充要条件。
func TestMenuBarPopupIsLaidOut(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	bar := sampleMenuBar("menu-file")
	renderer.PaintTree(ctx, []*runtime.VNode{bar}, theme.Dark)

	pop := popupOf(t, bar)
	if pop.W <= 0 || pop.H <= 0 {
		t.Fatalf("弹出层尺寸非正：%.1fx%.1f", pop.W, pop.H)
	}
	if pop.H <= 32 {
		t.Fatalf("弹出层高 %.1f，被菜单栏的 32 DIP 夹住了；底色盖不住下面的菜单项", pop.H)
	}
	for _, c := range pop.Children {
		if c == nil {
			continue
		}
		if c.Y+c.H > pop.Y+pop.H+0.5 {
			t.Errorf("子节点底边 %.1f 超出弹出层底边 %.1f（弹出层 y=%.1f h=%.1f）",
				c.Y+c.H, pop.Y+pop.H, pop.Y, pop.H)
		}
	}
}

// TestMenuBarPopupPaintsBackground 是上一条的绘制侧补充：弹出层必须**自己**
// 铺一层底。
//
// 为什么不能只靠菜单项按钮的底色：分隔线那一段没有按钮（只有一个 1 DIP 高的
// 分隔块），没人给它铺底。所以 paintPopover 自己那次 FillRoundedRect 是必需的，
// 而 paintMenuBar 是**跳过** popover 子节点的（它由 renderer 的 overlay 通道
// 单独绘制），这条要钉住「跳过之后仍然有人画」。
func TestMenuBarPopupPaintsBackground(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	bar := sampleMenuBar("menu-file")
	renderer.PaintTree(ctx, []*runtime.VNode{bar}, theme.Dark)

	pop := popupOf(t, bar)
	// paintPopover 画的是「整块弹出层矩形」的圆角填充，几何与节点一致。
	found := false
	for _, r := range ctx.rounds {
		if approx(r.x, pop.X) && approx(r.y, pop.Y) &&
			approx(r.w, pop.W) && approx(r.h, pop.H) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("没有以弹出层矩形 %.1f,%.1f %.1fx%.1f 为几何的填充；"+
			"分隔线那段会透出底下的界面。已画圆角矩形 %d 个", pop.X, pop.Y, pop.W, pop.H, len(ctx.rounds))
	}
}

// TestMenuBarPopupAnchorsUnderItsMenu 确认下拉挂在被点菜单项的正下方。
//
// 与上面两条互补：高度对了、底色也铺了，但位置飘到别处同样不可用。
func TestMenuBarPopupAnchorsUnderItsMenu(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	bar := sampleMenuBar("menu-file")
	renderer.PaintTree(ctx, []*runtime.VNode{bar}, theme.Dark)

	pop := popupOf(t, bar)
	anchor := runtime.FindByID([]*runtime.VNode{bar}, "menu-file")
	if anchor == nil {
		t.Fatal("没找到菜单项按钮 menu-file")
	}
	if !approx(pop.X, anchor.X) {
		t.Errorf("弹出层左边缘 %.1f 与菜单项左边缘 %.1f 不一致（alignX=start）", pop.X, anchor.X)
	}
	if pop.Y < anchor.Y+anchor.H {
		t.Errorf("弹出层顶边 %.1f 压在菜单项（底边 %.1f）上面", pop.Y, anchor.Y+anchor.H)
	}
}

// TestMenuBarClosedPaintsNoPopup 确认收起时不留残余：没有 popover 子节点，
// 也就不会有底色或菜单项文字漏在界面上。
func TestMenuBarClosedPaintsNoPopup(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	bar := sampleMenuBar("")
	renderer.PaintTree(ctx, []*runtime.VNode{bar}, theme.Dark)

	if pop := runtime.FindByID([]*runtime.VNode{bar}, "menubar-popup"); pop != nil {
		t.Fatalf("菜单未展开却存在 popover 节点")
	}
	for _, s := range ctx.texts {
		if s == "新建" || s == "保存" {
			t.Fatalf("菜单未展开却画出了菜单项 %q", s)
		}
	}
}

func approx(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.5
}
