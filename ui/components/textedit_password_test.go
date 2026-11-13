package components

import (
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func findSelectionFill(fills []mockRound) (mockRound, bool) {
	for _, f := range fills {
		if f.c.A > 0.29 && f.c.A < 0.31 {
			return f, true
		}
	}
	return mockRound{}, false
}

// 密码框全选时，选区高亮必须覆盖全部圆点。以前选区端点用真实值字节偏移去
// 查按圆点建立的布局，6 个字符只画出 2 个圆点的宽度。
func TestPasswordSelectAllCoversMaskedText(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, got := newInput("t-pass-select", "secret", &runtime.Props{Password: true})
	n.X, n.Y, n.W, n.H = 10, 20, 300, 38
	runtime.SetFocus("t-pass-select")
	defer runtime.SetFocus("")

	style := renderer.TextStyle{FontSize: 13}
	paintInput(ctx, n, style, theme.Light)
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyA, Ctrl: true, Down: true})
	paintInput(ctx, n, style, theme.Light)

	lay := stateFor(n).layout
	wantW := lay.pointAtLineEnd(0).x
	if wantW <= 0 {
		t.Fatal("密码布局应有宽度")
	}
	sel, ok := findSelectionFill(ctx.fills)
	if !ok {
		t.Fatalf("没有画出选区高亮：%+v", ctx.fills)
	}
	if absf(sel.w-wantW) > 0.51 {
		t.Fatalf("密码全选高亮宽 %.1f，期望覆盖全部 %d 个圆点的 %.1f", sel.w, len(*got), wantW)
	}
	if absf(sel.x-(n.X+inputPad)) > 0.51 {
		t.Fatalf("选区应从文本起点开始：x=%.1f want %.1f", sel.x, n.X+inputPad)
	}
	cx, _, _, _, ok := FocusedCaretRect()
	if !ok {
		t.Fatal("应记录光标矩形")
	}
	if absf(cx-(n.X+inputPad+wantW)) > 0.51 {
		t.Fatalf("全选后光标应落在文本末尾：cx=%.1f want %.1f", cx, n.X+inputPad+wantW)
	}
}

// 密码框点击落点：布局偏移是圆点字节偏移，必须换回真实值偏移再写入 caret。
func TestPasswordClickLandsOnRealValueOffset(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, got := newInput("t-pass-click", "abcdef", &runtime.Props{Password: true})
	n.X, n.Y, n.W, n.H = 10, 20, 300, 38
	runtime.SetFocus("t-pass-click")
	defer runtime.SetFocus("")

	style := renderer.TextStyle{FontSize: 13}
	paintInput(ctx, n, style, theme.Light)
	bulletW, _ := ctx.MeasureText(passwordMask, 0, style)
	// 第 2、3 个圆点之间偏右：应落在真实值下标 3。
	renderer.SetPointer(n.X+inputPad+2.75*bulletW, n.Y+10, false)
	n.OnClick()
	typeRunes(n, "X")
	if *got != "abcXdef" {
		t.Fatalf("密码框点击落点不对：%q", *got)
	}
}

// 长文本粘贴后光标跟随与横向滚动要按完整显示宽度计算。密码框尤其明显：
// 以前用真实值偏移查圆点布局，滚动只走到约 1/3 处。
func TestLongPasteCaretTracksDisplayWidth(t *testing.T) {
	for _, password := range []bool{false, true} {
		name := "plain"
		if password {
			name = "password"
		}
		t.Run(name, func(t *testing.T) {
			ctx := &mockContext{w: 400, h: 200}
			id := "t-paste-" + name
			n, got := newInput(id, "", &runtime.Props{Password: password})
			n.X, n.Y, n.W, n.H = 10, 20, 200, 38
			runtime.SetFocus(id)
			defer runtime.SetFocus("")

			text := strings.Repeat("x", 120)
			if !runtime.DispatchTextInput([]*runtime.VNode{n}, text) {
				t.Fatal("焦点输入框应接收上屏文本")
			}
			if *got != text {
				t.Fatalf("粘贴文本未完整写入：len=%d", len(*got))
			}
			paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
			st := stateFor(n)
			if st.caret != len(text) {
				t.Fatalf("粘贴后光标应在末尾：caret=%d want %d", st.caret, len(text))
			}
			lay := st.layout
			viewW := n.W - 2*inputPad
			wantScroll := lay.pointAtLineEnd(0).x - viewW
			if wantScroll < 0 {
				wantScroll = 0
			}
			if absf(st.scrollX-wantScroll) > 0.51 {
				t.Fatalf("横向滚动没有跟到文末：scrollX=%.1f want %.1f", st.scrollX, wantScroll)
			}
			cx, _, _, _, ok := FocusedCaretRect()
			if !ok {
				t.Fatal("应记录光标矩形")
			}
			if absf(cx-(n.X+n.W-inputPad)) > 0.51 {
				t.Fatalf("光标应贴住右内边：cx=%.1f want %.1f", cx, n.X+n.W-inputPad)
			}
		})
	}
}
