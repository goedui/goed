package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// ctxItems 是右键面板的样例菜单：4 项 + 1 条分隔线。
//
// 项数够多是故意的 —— 面板的高度必须明显超出单行，「卡片只盖住第一项」这类
// 高度被夹住的错误才暴露得出来（菜单栏的下拉踩过一模一样的坑）。
func ctxItems(undo, copy func()) []MenuItem {
	return []MenuItem{
		{Label: "撤销", Shortcut: "Ctrl+Z", Action: undo},
		{Label: "复制", Shortcut: "Ctrl+C", Action: copy},
		{Sep: true},
		{Label: "全选", Shortcut: "Ctrl+A"},
		{Label: "查找", Shortcut: "Ctrl+F"},
	}
}

// ctxScene 造一屏「底下一层界面 + 一个右键面板」，并完成布局与绘制。
//
// 用 layer 而不是直接把面板当根：右键面板的常态就是叠在别的东西上，遮罩铺满
// 父级、命中排在最后这些性质都只有在「有东西被它盖住」时才检验得出来。
type ctxScene struct {
	ctx     *mockContext
	root    *runtime.VNode
	menu    *runtime.VNode
	card    *runtime.VNode
	base    *runtime.VNode
	closes  int
	acts    []string
	baseCtx int
}

func newCtxScene(t *testing.T, atX, atY float32, items []MenuItem) *ctxScene {
	t.Helper()
	s := &ctxScene{ctx: &mockContext{w: 800, h: 600}}
	s.base = Box(runtime.Props{
		ID:      "base",
		Style:   runtime.Style{Flex: 1, Background: theme.Dark.Background},
		OnClick: func() {},
		OnContextMenu: func(x, y float32) {
			s.baseCtx++
		},
	})
	s.menu = ContextMenu(runtime.Props{
		ID:    "ctx",
		AtX:   atX,
		AtY:   atY,
		Items: items,
		OnClose: func() {
			s.closes++
		},
		OnSelect: func(v string) {
			s.acts = append(s.acts, v)
		},
	})
	if s.menu == nil {
		t.Fatal("ContextMenu 返回 nil：items 非空时应该造出节点")
	}
	s.root = runtime.H("layer", runtime.Props{Style: runtime.Style{Flex: 1}, ID: "app"},
		s.base, s.menu)
	renderer.PaintTree(s.ctx, []*runtime.VNode{s.root}, theme.Dark)
	s.card = ctxCard(t, s.menu)
	return s
}

func ctxCard(t *testing.T, menu *runtime.VNode) *runtime.VNode {
	t.Helper()
	for _, c := range menu.Children {
		if c != nil && c.Tag == "popover" {
			return c
		}
	}
	t.Fatal("面板里没有 popover 卡片")
	return nil
}

// clickAt 走一遍真实的命中链路：收集命中 → 从后往前找 → 调用。
// 返回 false 表示这个点上没有任何可点的东西。
func clickAt(nodes []*runtime.VNode, x, y float32) bool {
	hits := runtime.CollectHits(nodes)
	fn := runtime.HitTest(hits, x, y)
	if fn == nil {
		return false
	}
	fn()
	return true
}

func ctxSep(t *testing.T, card *runtime.VNode) *runtime.VNode {
	t.Helper()
	for _, c := range card.Children {
		if c != nil && c.Tag == "box" {
			return c
		}
	}
	t.Fatal("卡片里没有分隔线节点")
	return nil
}

// TestContextMenuNeedsItems 钉住「空菜单不占画布」。
//
// 没有菜单项时返回 nil：一块只有边框的空面板比「什么都不弹」更糟。nil 在
// layer 的子节点、命中、绘制里都是合法的「这一项不存在」，所以调用方不需要
// 自己判空 —— 但前提是这里真的返回 nil。
func TestContextMenuNeedsItems(t *testing.T) {
	if n := ContextMenu(runtime.Props{AtX: 10, AtY: 10}); n != nil {
		t.Fatalf("没有 items 时返回了节点：%+v", n)
	}
	if n := ContextMenu(runtime.Props{Items: []MenuItem{}}); n != nil {
		t.Fatal("空切片应该和没有 items 一样返回 nil")
	}
	if n := ContextMenu(runtime.Props{Items: []MenuItem{{Label: "撤销"}}}); n == nil {
		t.Fatal("有菜单项时不该返回 nil")
	}
	// []any 这条形状是 Props.Items 声明成 any 带来的：手写 []any{MenuItem{...}}
	// 也要认得出来，不然调用方得猜该用哪种切片。
	if n := ContextMenu(runtime.Props{Items: []any{MenuItem{Label: "撤销"}}}); n == nil {
		t.Fatal("[]any 形状的菜单项应该同样认得出来")
	}
}

// TestContextMenuScrimFillsAndCardSitsAtPointer 是面板的几何契约。
//
// 两件事：
//
//   - 遮罩铺满父级。它得接住画布每一个角落的点击，尺寸小于父级就会在边上漏出
//     一条「点了不关」的缝。
//   - 卡片贴着指针长出来，但不压住指针。压住的话指针就落在第一个菜单项上，
//     菜单一出现第一项就是高亮态，看起来像「已经选中了」。
func TestContextMenuScrimFillsAndCardSitsAtPointer(t *testing.T) {
	s := newCtxScene(t, 300, 200, ctxItems(nil, nil))

	if s.menu.X != 0 || s.menu.Y != 0 || s.menu.W != 800 || s.menu.H != 600 {
		t.Fatalf("遮罩没有铺满父级：x=%.1f y=%.1f w=%.1f h=%.1f，期望 0,0,800,600",
			s.menu.X, s.menu.Y, s.menu.W, s.menu.H)
	}
	if s.card.X != 300 {
		t.Errorf("卡片左边 %.1f，应该贴着指针的 x=300", s.card.X)
	}
	if d := s.card.Y - 200; d < 0 || d > 8 {
		t.Errorf("卡片顶边离指针 %.1f DIP，应该在 0~8 之间（贴着指针但不压住它）", d)
	}
	if s.card.H <= 0 {
		t.Fatalf("卡片高度非正：%.1f", s.card.H)
	}
	// 底色要盖得住所有菜单项：分隔线那一段没有按钮，卡片自己没铺满就会透出底下的界面。
	for _, c := range s.card.Children {
		if c == nil {
			continue
		}
		if c.Y+c.H > s.card.Y+s.card.H+0.5 {
			t.Errorf("子节点底边 %.1f 超出卡片底边 %.1f", c.Y+c.H, s.card.Y+s.card.H)
		}
	}
}

// TestContextMenuClampsToCanvas 钉住靠右下角时的夹取与翻转。
//
// 这两条是 Popover 本来就有的行为，但面板把它「借」过来时很容易弄丢：一旦
// 量卡片的上限传的是面板（而不是画布），夹取就变成拿画布尺寸去夹画布尺寸，
// 等于没有夹 —— 菜单会直接跑到窗口外面去。
func TestContextMenuClampsToCanvas(t *testing.T) {
	s := newCtxScene(t, 790, 590, ctxItems(nil, nil))

	if s.card.X+s.card.W > 800.5 {
		t.Errorf("卡片右边 %.1f 超出画布 800：没有夹回画布", s.card.X+s.card.W)
	}
	if s.card.Y+s.card.H > 590 {
		t.Errorf("卡片底边 %.1f 超过了指针 y=590：没有翻到指针上方", s.card.Y+s.card.H)
	}
}

// TestContextMenuLabelsMatchMenuBar 钉住「菜单项的标签就是 MenuBar 那一套」。
//
// 面板要按标签反查动作（Popover 的回调只给标签），所以标签必须和 MenuBar 用的
// menuItemLabel 逐字一致：带两格前缀、快捷键用制表符分隔。这里变了、那里没变，
// 症状是「点菜单项什么也不发生」—— 反查查不到，而且不会报错。
func TestContextMenuLabelsMatchMenuBar(t *testing.T) {
	s := newCtxScene(t, 10, 10, ctxItems(nil, nil))
	if len(s.card.Children) == 0 {
		t.Fatal("卡片里没有菜单项")
	}
	first := s.card.Children[0]
	if got := renderer.AttrString(first, "label"); got != "  撤销" {
		t.Errorf("第一项标签 %q，期望 %q（menuItemLabel 的两格前缀）", got, "  撤销")
	}
	if got := renderer.AttrString(first, "shortcut"); got != "Ctrl+Z" {
		t.Errorf("第一项快捷键 %q，期望 Ctrl+Z（标签与快捷键用制表符分隔）", got)
	}
}

// TestContextMenuClickPaths 走三条点击路径，它们决定「菜单关不关」。
//
// 三条路径的分工是这个控件存在的全部理由：
//
//	菜单项     → 执行动作，关掉
//	卡片空档   → 什么也不做，菜单留着（分隔线那几像素不属于任何一项）
//	其它任何处 → 关掉
//
// 第二条最容易漏：卡片不接住的话，点分隔线会穿到遮罩上，变成「点分隔线把菜单关了」。
func TestContextMenuClickPaths(t *testing.T) {
	ran := []string{}
	s := newCtxScene(t, 300, 200, ctxItems(
		func() { ran = append(ran, "undo") },
		func() { ran = append(ran, "copy") },
	))
	nodes := []*runtime.VNode{s.root}

	// ① 菜单项：第一项的中心。
	item := s.card.Children[0]
	if !clickAt(nodes, item.X+item.W/2, item.Y+item.H/2) {
		t.Fatal("点菜单项没有命中任何东西")
	}
	if len(ran) != 1 || ran[0] != "undo" {
		t.Fatalf("点「撤销」执行了 %v，期望只跑 undo", ran)
	}
	if s.closes != 1 {
		t.Fatalf("点菜单项后关闭回调跑了 %d 次，期望 1 次", s.closes)
	}
	if len(s.acts) != 1 || s.acts[0] != "  撤销\tCtrl+Z" {
		t.Fatalf("onSelect 收到 %v，期望 [  撤销\\tCtrl+Z]（整条标签，含前缀与快捷键）", s.acts)
	}

	// ② 卡片空档：分隔线那条 8 DIP 的带子上。
	s.closes, ran = 0, nil
	sep := ctxSep(t, s.card)
	if !clickAt(nodes, sep.X+2, sep.Y+0.5) {
		t.Fatal("点卡片空档没有命中任何东西：点击会穿到遮罩上")
	}
	if s.closes != 0 {
		t.Fatal("点分隔线的空档把菜单关了：卡片应该自己吃掉这一下")
	}

	// ③ 遮罩：远离卡片的地方。
	if !clickAt(nodes, 700, 550) {
		t.Fatal("点遮罩没有命中任何东西：菜单关不掉")
	}
	if s.closes != 1 {
		t.Fatalf("点空白后关闭回调跑了 %d 次，期望 1 次", s.closes)
	}
}

// TestContextMenuEscCloses 钉住 Esc。
//
// 面板是 OverlayTag，但 DispatchKey 对 dialog / popover 的「先问谁」是硬编码的，
// 面板不在其中就收不到按键 —— 所以它得自己挂 onKeyDown，而 DispatchKey 里也要
// 有 ctxmenu 那一支。这两处少一个，Esc 就静默失灵。
func TestContextMenuEscCloses(t *testing.T) {
	s := newCtxScene(t, 300, 200, ctxItems(nil, nil))
	if !runtime.DispatchKey([]*runtime.VNode{s.root}, runtime.KeyEvent{Key: runtime.KeyEscape, Down: true}) {
		t.Fatal("Esc 没有被消费")
	}
	if s.closes != 1 {
		t.Fatalf("Esc 后关闭回调跑了 %d 次，期望 1 次", s.closes)
	}
}

// TestContextMenuEscBeatsMenuBar 钉住 Esc 的优先级。
//
// 菜单栏的下拉和右键面板可能同时开着。Esc 该关**最后打开的那一层**，也就是
// 右键面板；走 popover 那一支会去关菜单栏，面板留在屏幕上，看起来像「Esc 没反应」。
func TestContextMenuEscBeatsMenuBar(t *testing.T) {
	barClosed := ""
	bar := MenuBar(runtime.Props{
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
			Items: []MenuItem{{Label: "新建"}, {Label: "打开"}},
		}},
		Open:   "menu-file",
		OnOpen: func(id string) { barClosed = id },
	})
	closed := 0
	menu := ContextMenu(runtime.Props{
		AtX: 300, AtY: 200, Items: ctxItems(nil, nil),
		OnClose: func() { closed++ },
	})
	root := runtime.H("layer", runtime.Props{Style: runtime.Style{Flex: 1}},
		runtime.H("vstack", runtime.Props{Style: runtime.Style{Flex: 1}}, bar),
		menu)
	ctx := &mockContext{w: 800, h: 600}
	renderer.PaintTree(ctx, []*runtime.VNode{root}, theme.Dark)

	if !runtime.DispatchKey([]*runtime.VNode{root}, runtime.KeyEvent{Key: runtime.KeyEscape, Down: true}) {
		t.Fatal("Esc 没有被消费")
	}
	if closed != 1 {
		t.Fatalf("Esc 关的是菜单栏（barClosed=%q），右键面板跑了 %d 次", barClosed, closed)
	}
	if barClosed != "" {
		t.Fatalf("Esc 顺手把菜单栏也关了：OnOpen(%q)", barClosed)
	}
}

// TestContextMenuScrimShieldsRightClick 钉住「菜单开着时右键点空白不会再弹一层」。
//
// 右键找的是 HitNode 的最上层节点，遮罩（面板根）带着 OnClick，所以它会被
// 选中 —— 而它没有 onContextMenu，于是这一下右键被吃掉。底下的界面收不到，
// 菜单不会被第二层面板盖住。
func TestContextMenuScrimShieldsRightClick(t *testing.T) {
	s := newCtxScene(t, 300, 200, ctxItems(nil, nil))
	n := runtime.HitNode([]*runtime.VNode{s.root}, 700, 550)
	if n == nil {
		t.Fatal("遮罩没有被 HitNode 选中：右键会穿到底下的界面")
	}
	if n.Attrs != nil {
		if _, ok := n.Attrs["onContextMenu"]; ok {
			t.Fatal("遮罩自己带 onContextMenu：右键点空白会再弹一层菜单")
		}
	}
	if s.baseCtx != 0 {
		t.Fatalf("底下的界面收到了 %d 次右键", s.baseCtx)
	}
}

// TestContextMenuPaintsCardOnce 钉住「卡片只画一遍」。
//
// 面板和卡片都是浮层，paintOverlays 会顺着树各画一次。面板要是什么都不画，
// 这里就只该有一次圆角填充 + 一条描边；一旦面板顺手把子节点也画一遍，
// 圆角矩形会翻倍 —— 半透明底色上叠两遍就是明显更深的一块。
func TestContextMenuPaintsCardOnce(t *testing.T) {
	s := newCtxScene(t, 300, 200, ctxItems(nil, nil))
	if len(s.ctx.rounds) != 2 {
		t.Fatalf("圆角填充 %d 次，期望 1 次（卡片底色）", len(s.ctx.rounds))
	}
	r := s.ctx.rounds[1]
	if r.x != s.card.X || r.y != s.card.Y || r.w != s.card.W || r.h != s.card.H {
		t.Errorf("卡片底色画在 (%.1f,%.1f,%.1f,%.1f)，卡片本身是 (%.1f,%.1f,%.1f,%.1f)",
			r.x, r.y, r.w, r.h, s.card.X, s.card.Y, s.card.W, s.card.H)
	}
	if len(s.ctx.strokes) != 1 {
		t.Fatalf("描边 %d 次，期望 1 次（卡片边框）", len(s.ctx.strokes))
	}
}

// TestHitNodeFindsContextOnlyNode 钉住「只写了右键的节点也能收到右键」。
//
// HitNode 原先只认 OnClick / onPointerDown。一个节点如果只声明了
// onContextMenu（比如一块只提供右键的空白区），右键会直接穿过去落到它下面的
// 兄弟节点上 —— 而它自己永远收不到。命中判定和「谁有右键」必须对得上。
func TestHitNodeFindsContextOnlyNode(t *testing.T) {
	fired := 0
	hot := runtime.H("box", runtime.Props{
		ID:            "hot",
		Style:         runtime.Style{Width: 100, Height: 40},
		OnContextMenu: func(x, y float32) { fired++ },
	})
	root := runtime.H("vstack", runtime.Props{Style: runtime.Style{Flex: 1}}, hot)
	ctx := &mockContext{w: 800, h: 600}
	renderer.PaintTree(ctx, []*runtime.VNode{root}, theme.Dark)

	n := runtime.HitNode([]*runtime.VNode{root}, 50, 20)
	if n == nil {
		t.Fatal("只带 onContextMenu 的节点没被 HitNode 选中")
	}
	fn, ok := n.Attrs["onContextMenu"].(func(float32, float32))
	if !ok || fn == nil {
		t.Fatalf("选中的节点不是那个热区：%+v", n.Attrs)
	}
	fn(50, 20)
	if fired != 1 {
		t.Fatalf("右键回调跑了 %d 次，期望 1 次", fired)
	}
}
