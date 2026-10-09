package components

import (
	"testing"
	"time"

	"github.com/goedui/goed/ui/caret"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// 输入框光标的闪烁。
//
// 相位本体搬去了 ui/caret（输入框和 editor 的代码编辑区共用同一份）。这里钉两件事：
// blinkVisible 真的接上了那份相位，以及它真的管住了那一笔 FillRect —— 只断言
// 「调用了 blinkVisible」是没用的，一个永远返回 true 的实现照样全绿。

// caretFills 挑出光标那一笔填充。
//
// caretWidth(1.6) 全仓库只有光标在用，按宽度认最省事，也不用去猜颜色。
func caretFills(ctx *mockContext) []mockRound {
	var out []mockRound
	for _, r := range ctx.fills {
		if r.w == caretWidth {
			out = append(out, r)
		}
	}
	return out
}

// waitBlinkOff 等相位翻到灭相。调用前要先把半周期压小（caret.SetPeriod）。
func waitBlinkOff(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for caret.Visible() {
		if time.Now().After(deadline) {
			t.Fatal("闪烁相位一直没翻到灭相")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestInputCaretRespectsBlinkPhase：分两相各断言一次。
//
// 半周期压到 50ms：要在 Reset 之后读一次相位，半周期太小的话后台循环可能抢在
// 前面又把它翻回灭相，测试就成了随机。
func TestInputCaretRespectsBlinkPhase(t *testing.T) {
	old := caret.SetPeriod(50 * time.Millisecond)
	t.Cleanup(func() { caret.SetPeriod(old) })

	n, _ := newInput("t-blink", "", nil)
	runtime.SetFocus("t-blink")
	defer runtime.SetFocus("")

	waitBlinkOff(t)
	ctx := &mockContext{w: 300, h: 40}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if got := caretFills(ctx); len(got) != 0 {
		t.Fatalf("灭相还是画了 %d 根光标：输入框没接上闪烁相位", len(got))
	}

	caret.Reset()
	ctx = &mockContext{w: 300, h: 40}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if got := caretFills(ctx); len(got) != 1 {
		t.Fatalf("亮相画了 %d 根光标，期望 1 根", len(got))
	}
}

// TestInputTypingResetsBlink：打字要把光标点亮。
//
// 不重置的话，连打的时候光标可能正好落在灭相上，看上去像光标丢了。
// 相位本体搬走之后，这条同时也在守「resetBlink 有没有接上 ui/caret」。
func TestInputTypingResetsBlink(t *testing.T) {
	old := caret.SetPeriod(50 * time.Millisecond)
	t.Cleanup(func() { caret.SetPeriod(old) })

	n, _ := newInput("t-blink-reset", "", nil)
	runtime.SetFocus("t-blink-reset")
	defer runtime.SetFocus("")

	waitBlinkOff(t)
	typeRunes(n, "a")
	if !caret.Visible() {
		t.Fatal("打字之后相位还是灭的：resetBlink 没接上 ui/caret")
	}
}
