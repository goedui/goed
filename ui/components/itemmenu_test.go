package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func testMenuRows() []MenuRow {
	return []MenuRow{
		{ID: "code", Label: "Code", Desc: "Write code"},
		{ID: "ask", Label: "Ask", Desc: "Answer questions", Active: true},
		{ID: "debug", Label: "Debug", Desc: "Find the cause"},
	}
}

// paintItemMenuAgainstTrigger lays the panel out the way the app does: the
// trigger lives in flow content that is laid out first, then the panel resolves
// its anchor through renderer.LayoutRoot().
//
// The trigger must not be absolutely positioned: layoutContent places absolute
// nodes *after* flow ones, so the panel would be laid out before the trigger
// had a rect and would fall back to the canvas origin.
func paintItemMenuAgainstTrigger(t *testing.T, ctx *mockContext, triggerY, triggerW, triggerH float32, props runtime.Props) (*runtime.VNode, *runtime.VNode) {
	t.Helper()
	if props.Anchor == "" {
		props.Anchor = "trigger"
	}
	trigger := runtime.H("button", runtime.Props{
		ID:    props.Anchor,
		Style: runtime.Style{Width: triggerW, Height: triggerH},
	})
	panel := ItemMenu(props)
	if panel == nil {
		t.Fatal("ItemMenu returned nil")
	}
	// A column with a leading spacer pushes the trigger to triggerY, which is
	// how the composer's mode bar ends up at the bottom of the window.
	content := VStack(runtime.Props{
		Style: runtime.Style{Align: runtime.Stretch},
	},
		Box(runtime.Props{Style: runtime.Style{Height: triggerY}}),
		trigger,
	)

	renderer.PaintTree(ctx, []*runtime.VNode{content, panel}, theme.Dark)
	return panel, trigger
}

// TestItemMenuFlipsAboveTrigger 面板长在输入区上方的唯一条件是「装不下就上翻」。
// 少了这条，面板会顶出窗口底部，用户只能看到最后一行。
func TestItemMenuFlipsAboveTrigger(t *testing.T) {
	ctx := &mockContext{w: 1282, h: 858}
	// The composer's mode bar sits at the very bottom of the window.
	panel, trigger := paintItemMenuAgainstTrigger(t, ctx, 828, 74, 26, runtime.Props{
		Items:  testMenuRows(),
		Title:  "说明",
		Footer: "模式",
	})

	cardY := attrFloat(panel, "cardY")
	cardH := attrFloat(panel, "cardH")
	if cardH <= 0 {
		t.Fatal("card height not written by layout")
	}
	if cardY+cardH > trigger.Y {
		t.Errorf("面板没有上翻：卡片底 %.0f 越过触发按钮顶 %.0f", cardY+cardH, trigger.Y)
	}
	if cardY < 0 {
		t.Errorf("面板顶出画布：y=%.0f", cardY)
	}
	if w := attrFloat(panel, "cardW"); w != itemMenuWidth {
		t.Errorf("卡片宽 = %.0f，期望 %.0f", w, itemMenuWidth)
	}
}

// TestItemMenuOpensBelowWhenThereIsRoom 有空间时保持向下展开：底部条只是
// 一个特例，不该让所有面板都朝上长。
func TestItemMenuOpensBelowWhenThereIsRoom(t *testing.T) {
	ctx := &mockContext{w: 1282, h: 858}
	panel, trigger := paintItemMenuAgainstTrigger(t, ctx, 40, 100, 26, runtime.Props{
		Items: testMenuRows(),
	})

	if got, want := attrFloat(panel, "cardY"), trigger.Y+trigger.H+4; got != want {
		t.Errorf("卡片 y = %.0f，期望贴在触发按钮下方 %.0f", got, want)
	}
}

// TestItemMenuSelectAndClose 点行只回调 onSelect，点遮罩才回调 onClose ——
// 两者搞反会让「选模式」变成「关掉面板」。
func TestItemMenuSelectAndClose(t *testing.T) {
	var picked string
	closed := 0
	ctx := &mockContext{w: 1282, h: 858}
	panel, _ := paintItemMenuAgainstTrigger(t, ctx, 828, 74, 26, runtime.Props{
		Items:    testMenuRows(),
		OnSelect: func(id string) { picked = id },
		OnClose:  func() { closed++ },
	})

	var row *runtime.VNode
	for _, c := range panel.Children {
		if c.Tag == "hstack" {
			row = c
			break
		}
	}
	if row == nil || row.OnClick == nil {
		t.Fatal("面板里没有可点的行")
	}
	row.OnClick()
	if picked != "code" {
		t.Errorf("onSelect 收到 %q，期望 code", picked)
	}
	if closed != 0 {
		t.Errorf("点行不该关面板，onClose 调了 %d 次", closed)
	}

	panel.OnClick()
	if closed != 1 {
		t.Errorf("点遮罩应关面板，onClose 调了 %d 次", closed)
	}
}

// TestItemMenuKeyboardMovesSelection 上下键把「当前项的相邻项」交给 onSelect。
//
// 每次按键都重建面板：行的 Active 是构建时定下来的，真实应用里选中项一变就
// 会重渲染。复用同一棵树会让第二次按键仍从旧的当前项出发。
func TestItemMenuKeyboardMovesSelection(t *testing.T) {
	rowsWithActive := func(active string) []MenuRow {
		rows := testMenuRows()
		for i := range rows {
			rows[i].Active = rows[i].ID == active
		}
		return rows
	}

	press := func(active string, key int) string {
		var picked string
		ctx := &mockContext{w: 1282, h: 858}
		panel, _ := paintItemMenuAgainstTrigger(t, ctx, 828, 74, 26, runtime.Props{
			Items:    rowsWithActive(active),
			OnSelect: func(id string) { picked = id },
		})
		runtime.DispatchKey([]*runtime.VNode{panel}, runtime.KeyEvent{Key: key, Down: true})
		return picked
	}

	// 当前项 ask（中间），↓ 到 debug。
	if got := press("ask", runtime.KeyDown); got != "debug" {
		t.Errorf("↓ 从 ask 选中 %q，期望 debug", got)
	}
	// 当前项 ask，↑ 到 code。
	if got := press("ask", runtime.KeyUp); got != "code" {
		t.Errorf("↑ 从 ask 选中 %q，期望 code", got)
	}
	// 当前项 debug（末项），↓ 回绕到 code。
	if got := press("debug", runtime.KeyDown); got != "code" {
		t.Errorf("↓ 从 debug 回绕到 %q，期望 code", got)
	}
	// 当前项 code（首项），↑ 回绕到 debug。
	if got := press("code", runtime.KeyUp); got != "debug" {
		t.Errorf("↑ 从 code 回绕到 %q，期望 debug", got)
	}
}

// TestItemMenuEscapeCloses Esc 关面板。
func TestItemMenuEscapeCloses(t *testing.T) {
	closed := 0
	ctx := &mockContext{w: 1282, h: 858}
	panel, _ := paintItemMenuAgainstTrigger(t, ctx, 828, 74, 26, runtime.Props{
		Items:   testMenuRows(),
		OnClose: func() { closed++ },
	})
	runtime.DispatchKey([]*runtime.VNode{panel}, runtime.KeyEvent{
		Key: runtime.KeyEscape, Down: true,
	})
	if closed != 1 {
		t.Errorf("Esc 应关面板，onClose 调了 %d 次", closed)
	}
}

// TestItemMenuIsOverlay 面板必须是浮层：否则后画的输入区会盖住它、吞掉点击。
func TestItemMenuIsOverlay(t *testing.T) {
	if !runtime.OverlayTag("itemmenu") {
		t.Fatal("itemmenu 必须登记为浮层")
	}
}

// TestItemMenuEmptyReturnsNil items 为空返回 nil，layer / 命中 / 绘制都认这个约定。
func TestItemMenuEmptyReturnsNil(t *testing.T) {
	if n := ItemMenu(runtime.Props{Anchor: "trigger"}); n != nil {
		t.Error("空 items 应返回 nil")
	}
}
