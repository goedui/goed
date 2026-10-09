// 编辑框按键与上屏文本的常驻回归。
//
// 这几处此前「有调用点、却一行都没被跑过」（覆盖率 0%），属于纯缺口而不是死代码：
//
//	moveWord           ← textedit_keys.go 的 Ctrl+Left / Ctrl+Right 分派
//	movePage           ← 同上的 PageUp / PageDown 分派
//	handleTextInput    ← textarea.go / input.go 的 onTextInput 回调
//	viewportH          ← movePage 的翻页行数来源（可视文本高度）
//	SelectRange/SelectAll ← textedit_api.go 的导出入口
//
// 断言尽量写成不变量而不是魔数：翻页步长与 int(viewportH/lineH) 对齐、列位置
// 保持不变、按词移动落在词边界上。这样以后调字体或调内边距不会假红，而一旦
// 翻页公式被改错（漏掉 int() 截断、用了 n.H 而不是可视高度、步长写死），
// 或者按词移动把 CJK 当成词字符，就会立刻变红。
//
// 翻页有两条实现路径，样本要各喂一个：
//   - 全量 points 布局（普通文档）→ 用 lineH 与 points 算目标行；
//   - 稀疏分支（large：nowrap + ≥256KB；visual：> maxLayoutRun 个 rune）
//     → 用 lineForOffset / lineStart 定位。两条路径的语义必须一致。
package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// focusArea 造一个受控多行文本域并聚焦；测试结束自动清掉全局焦点与编辑状态，
// 避免焦点残留把后面用例的 DispatchKey / DispatchTextInput 引到别的节点上。
func focusArea(t *testing.T, id, value string, extra *runtime.Props) *runtime.VNode {
	t.Helper()
	n, _ := newArea(t, id, value, extra)
	runtime.SetFocus(id)
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState(id)
	})
	return n
}

// paintArea 在 focusArea 基础上走一遍真实绘制。
// 翻页依赖 viewportH(n)（= n.H 减去上下内边距），而 n.H 由 measure 决定、
// 只有绘制之后才落定；布局（st.layout）同样是绘制时建立的。
func paintArea(t *testing.T, id, value string) *runtime.VNode {
	t.Helper()
	n := focusArea(t, id, value, nil)
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)
	if st := stateFor(n); st.layout == nil {
		t.Fatalf("绘制后应建立布局，否则翻页 / 按词都无从谈起")
	}
	return n
}

// assertCaret 断言光标落在 0 基的 (行, 列)，列按 rune 计。
func assertCaret(t *testing.T, n *runtime.VNode, line, col int) {
	t.Helper()
	if gotL, gotC := CaretLineCol(n); gotL != line || gotC != col {
		t.Fatalf("光标在 (%d,%d)，期望 (%d,%d)", gotL, gotC, line, col)
	}
}

// ctrlKey 派发一次带 Ctrl 的按键（shift 决定是否扩展选区）。
func ctrlKey(n *runtime.VNode, key int, shift bool) {
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: key, Ctrl: true, Shift: shift, Down: true})
}

// numberedLines 造 n 行 "line0\n"…"line{n-1}\n"，用于翻页（行数够多才有页可翻）。
func numberedLines(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "line%d\n", i)
	}
	return b.String()
}

// pageStep 是「每页几行」，与 movePage 用的公式一致。
// 用例一律通过它来断言，不写死行数：换字体 / 换内边距不会假红，而公式被改错就会红。
func pageStep(t *testing.T, n *runtime.VNode) int {
	t.Helper()
	st := stateFor(n)
	step := int(viewportH(n) / st.layout.lineH)
	if step < 2 {
		t.Fatalf("每页 %d 行，样本太小、用例没意义（viewportH=%.1f lineH=%.2f）",
			step, viewportH(n), st.layout.lineH)
	}
	return step
}

// TestCtrlArrowMovesByWord 按词移动：连续的字词字符（含下划线）算一段，
// 向右会连带吃掉紧跟的一个非词字符，到末尾后夹住不动。
func TestCtrlArrowMovesByWord(t *testing.T) {
	n := focusArea(t, "tk-word", "foo_bar baz", nil)

	MoveCaret(n, 0, 0)
	assertCaret(t, n, 0, 0)

	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 8) // 吃掉整段 "foo_bar" 及其后的空格

	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 11) // 走到 "baz" 末尾

	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 11) // 已在末尾，夹住

	ctrlKey(n, runtime.KeyLeft, false)
	assertCaret(t, n, 0, 7) // 回退过空格，停在 "foo_bar" 之后

	ctrlKey(n, runtime.KeyLeft, false)
	assertCaret(t, n, 0, 0) // 一路吃掉 "foo_bar"
}

// TestCtrlArrowTreatsCJKRuneAsOneWord CJK 一字一词：0x2E80 以上的 rune 不算词字符，
// 所以每个汉字自成一段（若把 CJK 当词字符，第一次右移会直接跳到行尾）。
func TestCtrlArrowTreatsCJKRuneAsOneWord(t *testing.T) {
	n := focusArea(t, "tk-cjk", "你好abc", nil)

	MoveCaret(n, 0, 0)
	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 1) // 「你」一字
	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 2) // 「好」一字
	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 5) // "abc" 整段
	ctrlKey(n, runtime.KeyRight, false)
	assertCaret(t, n, 0, 5) // 末尾夹住
}

// TestCtrlArrowCollapsesSelectionBeforeMoving 不扩展的 Ctrl+方向键要先把选区
// 塌缩到对应端、再走一步。用例特意挑「塌缩点」与「不塌缩的起点」按词落点不同的
// 选区：若塌缩被删掉，向右会停在 6（而不是 8）、向左会停在 5（而不是 2）。
func TestCtrlArrowCollapsesSelectionBeforeMoving(t *testing.T) {
	const value = "aa bb cc"

	t.Run("向右先塌缩到右端", func(t *testing.T) {
		n := focusArea(t, "tk-sel-r", value, nil)
		SelectRange(n, 3, 6)
		ctrlKey(n, runtime.KeyRight, false)
		assertCaret(t, n, 0, 8)
		if _, _, has := SelectedRange(n); has {
			t.Fatal("不扩展的 Ctrl+→ 之后不应还留着选区")
		}
	})

	t.Run("向左先塌缩到左端", func(t *testing.T) {
		n := focusArea(t, "tk-sel-l", value, nil)
		SelectRange(n, 3, 6)
		ctrlKey(n, runtime.KeyLeft, false)
		assertCaret(t, n, 0, 2)
	})
}

// TestCtrlShiftArrowExtendsSelection 带 Shift 的按词移动扩展选区，锚点不动。
func TestCtrlShiftArrowExtendsSelection(t *testing.T) {
	n := focusArea(t, "tk-ext", "foo_bar baz", nil)

	MoveCaret(n, 0, 0)
	ctrlKey(n, runtime.KeyRight, true)

	from, to, has := SelectedRange(n)
	if !has || from != 0 || to != 8 {
		t.Fatalf("Ctrl+Shift+→ 选区 = [%d,%d) has=%v，期望 [0,8)", from, to, has)
	}
	if got := stateFor(n).selectedText(); got != "foo_bar " {
		t.Fatalf("选中文本 = %q，期望 %q", got, "foo_bar ")
	}

	ctrlKey(n, runtime.KeyRight, true)
	if from, to, has = SelectedRange(n); !has || from != 0 || to != 11 {
		t.Fatalf("第二次 Ctrl+Shift+→ 选区 = [%d,%d) has=%v，期望 [0,11)", from, to, has)
	}
}

// TestPageDownStepsByViewportHeight 普通文档（全量 points 布局）的翻页：
// 步长 = 可视高度能放下的行数，并且保持列位置。
func TestPageDownStepsByViewportHeight(t *testing.T) {
	n := paintArea(t, "tk-page", numberedLines(30))
	st := stateFor(n)

	if st.layout.sparse() || len(st.layout.points) == 0 {
		t.Fatalf("样本应走全量 points 布局：sparse=%v points=%d",
			st.layout.sparse(), len(st.layout.points))
	}
	if got, want := viewportH(n), n.H-2*inputTopPad; got != want {
		t.Fatalf("viewportH = %.1f，期望 n.H-2*inputTopPad = %.1f（n.H=%.1f）", got, want, n.H)
	}
	step := pageStep(t, n)

	MoveCaret(n, 0, 3)
	assertCaret(t, n, 0, 3)

	press(n, runtime.KeyPageDown)
	assertCaret(t, n, step, 3) // 步长 = 每页行数，列保持
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, 2*step, 3) // 连续翻页是累加，不是每次都跳到底
	press(n, runtime.KeyPageUp)
	assertCaret(t, n, step, 3) // PageUp 与 PageDown 对称
}

// TestPageUpAtFirstPageGoesToDocumentStart 首页再上翻：目标行为负，落到文档开头。
func TestPageUpAtFirstPageGoesToDocumentStart(t *testing.T) {
	n := paintArea(t, "tk-page-top", numberedLines(30))

	MoveCaret(n, 0, 3)
	press(n, runtime.KeyPageUp)
	if off := stateFor(n).caret; off != 0 {
		t.Fatalf("首页 PageUp 后光标偏移 = %d，期望 0", off)
	}
	assertCaret(t, n, 0, 0)
}

// TestPageDownAtLastLineGoesToDocumentEnd 末页再下翻：落到文档末尾（不是留在原地）。
func TestPageDownAtLastLineGoesToDocumentEnd(t *testing.T) {
	n := paintArea(t, "tk-page-end", numberedLines(30))
	st := stateFor(n)

	MoveCaret(n, 29, 1)
	press(n, runtime.KeyPageDown)
	if st.caret != len(st.value) {
		t.Fatalf("末行 PageDown 后光标偏移 = %d，期望文档末尾 %d", st.caret, len(st.value))
	}
}

// TestPageDownOnVisualLayoutStepsByViewportHeight 超过 maxLayoutRun(4000) 个 rune 的
// 文档走 **visual** 布局：只记可视行起点、points 是空的，于是翻页落到稀疏分支
// （lineForOffset / lineStart 定位）。语义必须和全量 points 那条路一致。
func TestPageDownOnVisualLayoutStepsByViewportHeight(t *testing.T) {
	n := paintArea(t, "tk-page-visual", numberedLines(800)) // 800×6 = 4800 rune > 4000
	st := stateFor(n)

	if !st.layout.visual || len(st.layout.points) != 0 {
		t.Fatalf("样本没走 visual 布局：visual=%v points=%d",
			st.layout.visual, len(st.layout.points))
	}
	step := pageStep(t, n)

	MoveCaret(n, 0, 3)
	assertCaret(t, n, 0, 3)
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, step, 3)
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, 2*step, 3)
	press(n, runtime.KeyPageUp)
	assertCaret(t, n, step, 3)

	// 首页再上翻要夹到第 0 行，末页再下翻要落到文档末尾。
	MoveCaret(n, 0, 3)
	press(n, runtime.KeyPageUp)
	assertCaret(t, n, 0, 3)
	MoveCaret(n, 799, 0)
	press(n, runtime.KeyPageDown)
	if st.caret != len(st.value) {
		t.Fatalf("末行 PageDown 后光标偏移 = %d，期望文档末尾 %d", st.caret, len(st.value))
	}
}

// TestPageDownOnLargeDocumentStepsByViewportHeight 稀疏布局的另一个入口
// （nowrap + ≥ largeLayoutBytes）走的是 large 分支。样本故意让最后一行只有一个
// 字符：翻到末尾那一页时「起点 + 列」会越过行尾，必须夹到行尾而不是算出越界偏移。
func TestPageDownOnLargeDocumentStepsByViewportHeight(t *testing.T) {
	const bodyLines = 29999
	text := strings.Repeat("abcdefgh\n", bodyLines) + "x"
	if len(text) < largeLayoutBytes {
		t.Fatalf("样本 %d 字节，不足以触发稀疏布局（阈值 %d）", len(text), largeLayoutBytes)
	}

	n := focusArea(t, "tk-page-large", text, &runtime.Props{Nowrap: true})
	// 不跑整棵绘制树（27 万字节没必要真画），只调 measure 拿高度：它内部就会
	// 建立稀疏布局，正是 movePage 的 large 分支要用到的。
	_, h := measureTextarea(&mockContext{w: 500, h: 400}, n, 400, 0,
		renderer.TextStyle{FontSize: 13}, theme.Dark)
	n.H = h

	st := stateFor(n)
	if !st.layout.large {
		t.Fatalf("nowrap + 大文档应使用 large 布局：large=%v visual=%v",
			st.layout.large, st.layout.visual)
	}
	if got := st.layout.maxLine; got != bodyLines {
		t.Fatalf("总行号 = %d，期望 %d", got, bodyLines)
	}
	step := pageStep(t, n)

	// 往下翻：步长 = 每页行数，列保持。
	MoveCaret(n, 0, 3)
	assertCaret(t, n, 0, 3)
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, step, 3)
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, 2*step, 3)
	press(n, runtime.KeyPageUp)
	assertCaret(t, n, step, 3)

	// 首页再上翻：目标行号为负，应夹到第 0 行。
	MoveCaret(n, 0, 3)
	press(n, runtime.KeyPageUp)
	assertCaret(t, n, 0, 3)

	// 可视高度不足一行时翻页行数要钳到 1 行，否则大文件里翻页会原地不动。
	MoveCaret(n, 0, 3)
	st.movePage(1, false, st.layout.lineH/2)
	assertCaret(t, n, 1, 3)

	// 末页再下翻：目标行号越过末行，应夹到最后一行。
	MoveCaret(n, bodyLines-1, 0)
	press(n, runtime.KeyPageDown)
	if got, want := st.caret, bodyLines*9; got != want {
		t.Fatalf("末行 PageDown 后偏移 = %d，期望末行起点 %d", got, want)
	}
	assertCaret(t, n, bodyLines, 0)

	// 末行只有 1 个字符，而光标列是 7：「起点 + 列」越过行尾，应夹到文档末尾。
	MoveCaret(n, bodyLines-6, 7)
	press(n, runtime.KeyPageDown)
	if got, want := st.caret, len(text); got != want {
		t.Fatalf("列超过末行长度时偏移 = %d，期望文档末尾 %d", got, want)
	}
	assertCaret(t, n, bodyLines, 1)
}

// TestPageDownOnOneLineTallTextareaStepsOneLine 高度被夹到 24px 下限时，可视文本
// 高度只剩 12px，还不够一行（16.25px），翻页行数会算成 0 —— 必须钳到 1 行，
// 否则这种「只有一行高」的自动增长框里翻页完全没反应。
func TestPageDownOnOneLineTallTextareaStepsOneLine(t *testing.T) {
	const id = "tk-page-short"
	ResetInputState(id)
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState(id)
	})

	n := Textarea(runtime.Props{
		ID:    id,
		Value: numberedLines(10),
		Style: runtime.Style{Width: 400, MinHeight: 1, MaxHeight: 24},
	})
	runtime.SetFocus(id)
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)

	st := stateFor(n)
	if st.layout == nil {
		t.Fatal("绘制后应建立布局")
	}
	if vh := viewportH(n); vh >= st.layout.lineH {
		t.Fatalf("样本没到「可视高度不足一行」：viewportH=%.1f lineH=%.2f", vh, st.layout.lineH)
	}

	MoveCaret(n, 0, 1)
	press(n, runtime.KeyPageDown)
	assertCaret(t, n, 1, 1) // 前进一行，而不是原地不动
}

// TestVerticalKeysBeforeLayoutAreNoOps 未绘制过的编辑框收到纵向 / 翻页键只能静默
// 返回，不能 panic：布局是绘制（measure）时才建的，caretOffset() / moveToLineCol()
// 都不会顺手建布局，而下游 pointAt / lineForOffset 会直接解引用它
// （len(lay.points)、lay.text），layout 为 nil 必炸。
// moveLine 与 movePage 开头各有一道 st.layout == nil 守卫，挡的正是这件事。
//
// 这两道守卫很容易被误判成冗余 —— 紧邻的 st.layout.sparse() 是 nil-safe 的，
// 看上去「反正下游也容忍 nil」。所以要有用例守着：删掉任一道，这里就会 panic。
func TestVerticalKeysBeforeLayoutAreNoOps(t *testing.T) {
	const id = "tk-nolayout"
	n := focusArea(t, id, numberedLines(30), nil)
	st := stateFor(n)
	if st.layout != nil {
		t.Fatal("聚焦本身不该建立布局（布局由绘制建立），否则这条用例失去意义")
	}

	// 直接摆好光标：未布局时 moveToLineCol 的语义不是这条用例要测的。
	st.caret, st.anchor = 5, 5
	for _, key := range []int{runtime.KeyUp, runtime.KeyDown, runtime.KeyPageUp, runtime.KeyPageDown} {
		press(n, key)
	}
	if st.caret != 5 || st.anchor != 5 {
		t.Fatalf("未布局时纵向键动了光标：caret=%d anchor=%d，期望 (5,5)", st.caret, st.anchor)
	}
	if st.layout != nil {
		t.Fatal("纵向键不该建立布局")
	}
}

// TestSingleLineInputIgnoresPageKeys 单行输入框不认翻页键（要留给上层的焦点切换），
// 既不吞事件也不动光标。
func TestSingleLineInputIgnoresPageKeys(t *testing.T) {
	n, got := newInput("tk-oneline", "hello", nil)
	runtime.SetFocus("tk-oneline")
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState("tk-oneline")
	})

	MoveCaret(n, 0, 2)
	for _, key := range []int{runtime.KeyPageUp, runtime.KeyPageDown} {
		if runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: key, Down: true}) {
			t.Fatalf("单行输入框不该吞掉翻页键（key=%#x）", key)
		}
	}
	if l, c := CaretLineCol(n); l != 0 || c != 2 {
		t.Fatalf("单行框翻页后光标动了：(%d,%d)，期望 (0,2)", l, c)
	}
	if *got != "hello" {
		t.Fatalf("单行框翻页后文本变了：%q", *got)
	}
}

// TestTextInputInsertsAtCaretAndNotifies 输入法上屏：插在光标处（不是追加到末尾），
// 并且回调 onInput 收到新值。
func TestTextInputInsertsAtCaretAndNotifies(t *testing.T) {
	n, got := newArea(t, "tk-ime", "ab", nil)
	runtime.SetFocus("tk-ime")
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState("tk-ime")
	})

	assertCaret(t, n, 0, 2) // 初始光标在文末
	if !runtime.DispatchTextInput([]*runtime.VNode{n}, "你好") {
		t.Fatal("上屏应被处理")
	}
	if v := stateFor(n).value; v != "ab你好" {
		t.Fatalf("value = %q，期望 %q", v, "ab你好")
	}
	if *got != "ab你好" {
		t.Fatalf("onInput 收到 %q，期望 %q", *got, "ab你好")
	}

	// 把光标挪到中间再上屏一次：必须插在光标处。
	MoveCaret(n, 0, 2)
	if !runtime.DispatchTextInput([]*runtime.VNode{n}, "X") {
		t.Fatal("第二次上屏应被处理")
	}
	if v := stateFor(n).value; v != "abX你好" {
		t.Fatalf("插入位置不对：%q，期望 %q", v, "abX你好")
	}
	assertCaret(t, n, 0, 3)
}

// TestTextInputRejectsEmptyNilAndUnfocused 上屏的三道前置拒绝：空串、空节点、
// 没有焦点。逐条直接调 handleTextInput，避免被 runtime.DispatchTextInput 的
// 前置判断短路掉、导致这些守卫其实没人验。
func TestTextInputRejectsEmptyNilAndUnfocused(t *testing.T) {
	n, got := newArea(t, "tk-ime2", "ab", nil)

	runtime.SetFocus("")
	if handleTextInput(n, "X") {
		t.Fatal("没有焦点时上屏不该被处理")
	}
	if runtime.DispatchTextInput([]*runtime.VNode{n}, "X") {
		t.Fatal("没有焦点时派发上屏也不该被处理")
	}

	runtime.SetFocus("tk-ime2")
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState("tk-ime2")
	})
	if handleTextInput(n, "") {
		t.Fatal("空串上屏应返回 false")
	}
	if handleTextInput(nil, "X") {
		t.Fatal("handleTextInput(nil) 应返回 false")
	}
	if v := stateFor(n).value; v != "ab" {
		t.Fatalf("被拒绝的上屏改动了文本：%q", v)
	}
	if *got != "ab" {
		t.Fatalf("被拒绝的上屏触发了 onInput：%q", *got)
	}
}

// TestViewportHExcludesVerticalPadding viewportH 是「节点高度减去上下内边距」，
// 并且不足内边距时钳到 0（负数会让翻页行数算出负值）。
func TestViewportHExcludesVerticalPadding(t *testing.T) {
	n := paintArea(t, "tk-vh", "abc")
	if got, want := viewportH(n), n.H-2*inputTopPad; got != want {
		t.Fatalf("viewportH = %.1f，期望 %.1f（n.H=%.1f）", got, want, n.H)
	}
	if viewportH(nil) != 0 {
		t.Fatalf("viewportH(nil) = %.1f，期望 0", viewportH(nil))
	}

	short := &runtime.VNode{}
	short.H = inputTopPad // 恰好等于上下内边距之和 → 0
	if got := viewportH(short); got != 0 {
		t.Fatalf("viewportH(高度=内边距) = %.1f，期望 0", got)
	}
	short.H = 0 // 0 - 2*inputTopPad < 0 → 钳到 0
	if got := viewportH(short); got != 0 {
		t.Fatalf("viewportH(高度不足) = %.1f，期望 0", got)
	}
}

// TestSelectRangeAndSelectAllUseByteOffsets 选区接口按字节给区间；落在 UTF-8
// 中间字节的边界要被拉回字符起点（不能切出半个汉字），from==to 视为没有选区。
func TestSelectRangeAndSelectAllUseByteOffsets(t *testing.T) {
	const value = "你好世界"
	n := focusArea(t, "tk-selapi", value, nil)

	SelectRange(n, 3, 6) // 第二个字「好」
	from, to, has := SelectedRange(n)
	if !has || from != 3 || to != 6 {
		t.Fatalf("SelectedRange = [%d,%d) has=%v，期望 [3,6)", from, to, has)
	}
	if got := stateFor(n).selectedText(); got != "好" {
		t.Fatalf("选中文本 = %q，期望 %q", got, "好")
	}

	// 1 和 4 都落在字符中间：应被拉回 0 和 3。
	SelectRange(n, 1, 4)
	if from, to, has = SelectedRange(n); !has || from != 0 || to != 3 {
		t.Fatalf("中间字节边界 = [%d,%d) has=%v，期望 [0,3)", from, to, has)
	}

	SelectRange(n, 3, 3)
	if _, _, has = SelectedRange(n); has {
		t.Fatal("from == to 不应算选区")
	}

	SelectAll(n)
	if from, to, has = SelectedRange(n); !has || from != 0 || to != len(value) {
		t.Fatalf("SelectAll = [%d,%d) has=%v，期望 [0,%d)", from, to, has, len(value))
	}

	// 空节点不应 panic：状态栏 / 右键菜单在没有焦点框时会拿到 nil。
	SelectRange(nil, 0, 1)
	SelectAll(nil)
	MoveCaret(nil, 0, 1)
	if _, _, has := SelectedRange(nil); has {
		t.Fatal("SelectedRange(nil) 不该报告选区")
	}
	if l, c := CaretLineCol(nil); l != 0 || c != 0 {
		t.Fatalf("CaretLineCol(nil) = (%d,%d)，期望 (0,0)", l, c)
	}
}

// shiftKey 派发一次带 Shift 的按键（扩展选区）。
func shiftKey(n *runtime.VNode, key int) {
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: key, Shift: true, Down: true})
}

// TestUpDownOnVisualLayoutMovesLineKeepingColumn visual 布局（> maxLayoutRun 个 rune）
// 下按上下键走 moveLine 的**稀疏分支**。
//
// 补用例之前这整段（约 27 条语句）一行没跑过 —— 也就是说「大文件 / 超长行模式下
// 按上下键」这个行为全仓没有任何用例守着。movePage 早就有人守（同文件里那两条
// PageDown 用例），但 movePage **不分支 sparse**，所以「movePage 满了」推不出
// 「moveLine 也满了」。
//
// 样本用 numberedLines：行宽**逐行不同**（"line9\n" 6 字节、"line10\n" 7 字节），
// 正好顺带覆盖「目标行比当前列短 → 夹到目标行尾」。
func TestUpDownOnVisualLayoutMovesLineKeepingColumn(t *testing.T) {
	n := paintArea(t, "tk-line-visual", numberedLines(800)) // 6290 rune > 4000
	st := stateFor(n)

	if !st.layout.visual || len(st.layout.points) != 0 {
		t.Fatalf("样本没走 visual 布局：visual=%v points=%d",
			st.layout.visual, len(st.layout.points))
	}
	if st.layout.maxLine != 800 {
		t.Fatalf("总行号 = %d，期望 800（末尾 '\\n' 之后还有一个空行）", st.layout.maxLine)
	}

	// 列保持：第 0 行第 3 列 → 第 1 行第 3 列，来回各一次。
	MoveCaret(n, 0, 3)
	assertCaret(t, n, 0, 3)
	press(n, runtime.KeyDown)
	assertCaret(t, n, 1, 3)
	press(n, runtime.KeyDown)
	assertCaret(t, n, 2, 3)
	press(n, runtime.KeyUp)
	assertCaret(t, n, 1, 3)

	// 目标行比当前列短 → 夹到目标行尾，而不是算出越界偏移。
	// 第 10 行 "line10" 有 6 个字符，列 6 已在行尾；上移到第 9 行（5 个字符）
	// 只能落到它的行尾（列 5）。
	MoveCaret(n, 10, 6)
	assertCaret(t, n, 10, 6)
	press(n, runtime.KeyUp)
	assertCaret(t, n, 9, 5)

	// 移动过光标就必须请求「把光标滚进可视区」：scrollSync 是绘制时唯一的依据
	// （textedit_paint.go 的 scrollToCaret 分支）。全仓此前没有一条用例碰过它。
	MoveCaret(n, 0, 3)
	st.scrollSync = false
	press(n, runtime.KeyDown)
	if !st.scrollSync {
		t.Fatal("稀疏分支移动光标后没有置 scrollSync，视口不会跟着走")
	}
	assertCaret(t, n, 1, 3)

	// Shift+Down 只动 caret、不动 anchor（选区扩展）。
	MoveCaret(n, 0, 1)
	anchorBefore := st.anchor
	shiftKey(n, runtime.KeyDown)
	if st.anchor != anchorBefore {
		t.Fatalf("Shift+Down 把 anchor 从 %d 挪到了 %d", anchorBefore, st.anchor)
	}
	if st.caret == anchorBefore {
		t.Fatal("Shift+Down 没有移动 caret")
	}
	if from, to, has := SelectedRange(n); !has || from != anchorBefore || to != st.caret {
		t.Fatalf("Shift+Down 后选区 = [%d,%d) has=%v，期望 [%d,%d)",
			from, to, has, anchorBefore, st.caret)
	}
}

// TestUpDownAtSparseDocumentEdgesAreNoOps 稀疏分支的两个越界口子：目标行被**夹回
// 当前行**，于是光标原地不动 —— 首行再上移仍是原列，末行再下移仍是原位置。
//
// 注意这和全量 points 那条路**故意不一致**（见下一条用例：points 路径首行上移会
// 走到 (0,0)、末行下移会跳到文档末尾）。两条路各自的行为都钉住，改动才会显形。
func TestUpDownAtSparseDocumentEdgesAreNoOps(t *testing.T) {
	n := paintArea(t, "tk-line-visual-edge", numberedLines(800))
	st := stateFor(n)

	// 首行第 3 列再上移：target 夹到 0 == 当前行 → 光标留在原列。
	MoveCaret(n, 0, 3)
	press(n, runtime.KeyUp)
	assertCaret(t, n, 0, 3)

	// 末行（末尾 '\n' 之后的空行）再下移：target 夹到 maxLine == 当前行 → 不动。
	last := st.layout.maxLine
	MoveCaret(n, last, 0)
	assertCaret(t, n, last, 0)
	press(n, runtime.KeyDown)
	assertCaret(t, n, last, 0)
}

// TestUpDownAtDocumentEdgesOnPointsLayout 全量 points 布局（普通文档）下，上下键
// 越界时走的是两条**早退分支**（不是夹回当前行）：
//
//	首行再上移  → caret = 0（回到行首）
//	末行再下移  → caret = len(value)（跳到文档末尾）
//
// 样本 30 行 ≈ 200 字节，远小于 maxLayoutRun，确保走 points 路径。
func TestUpDownAtDocumentEdgesOnPointsLayout(t *testing.T) {
	n := paintArea(t, "tk-line-points", numberedLines(30))
	st := stateFor(n)

	if st.layout.sparse() || len(st.layout.points) == 0 {
		t.Fatalf("样本没走全量 points 布局：sparse=%v points=%d",
			st.layout.sparse(), len(st.layout.points))
	}

	// 正常上下移动也要置 scrollSync（下面两条早退分支不置 —— 见用例结尾说明）。
	MoveCaret(n, 1, 3)
	st.scrollSync = false
	press(n, runtime.KeyUp)
	if !st.scrollSync {
		t.Fatal("points 分支移动光标后没有置 scrollSync")
	}
	assertCaret(t, n, 0, 3)

	// 首行第 3 列再上移：target = -1 → 光标回 (0,0)。
	MoveCaret(n, 0, 3)
	assertCaret(t, n, 0, 3)
	press(n, runtime.KeyUp)
	assertCaret(t, n, 0, 0)

	// 末行（末尾 '\n' 之后的空行）再下移：target 越过 maxLine → 跳到文档末尾。
	last := st.layout.maxLine
	MoveCaret(n, last, 0)
	assertCaret(t, n, last, 0)
	press(n, runtime.KeyDown)
	if st.caret != len(st.value) {
		t.Fatalf("末行 KeyDown 后偏移 = %d，期望文档末尾 %d", st.caret, len(st.value))
	}
	assertCaret(t, n, last, 0)

	// 说明（不是断言）：上面两条**早退分支**都不置 st.scrollSync，而稀疏分支的
	// 夹取路径会置。当前无害 —— 这两条只在光标已经在首行 / 末行时触发，目标位置
	// 本来就在视口里 —— 但记一笔，别当成「两条路完全一致」。
}

// TestUpDownOnLargeDocumentKeepsColumn large 布局（nowrap + ≥ largeLayoutBytes）是
// 稀疏分支的另一个入口，语义必须和 visual 那条一致：列保持、越界夹到行尾。
func TestUpDownOnLargeDocumentKeepsColumn(t *testing.T) {
	const bodyLines = 29999
	text := strings.Repeat("abcdefgh\n", bodyLines) + "x"

	n := focusArea(t, "tk-line-large", text, &runtime.Props{Nowrap: true})
	_, h := measureTextarea(&mockContext{w: 500, h: 400}, n, 400, 0,
		renderer.TextStyle{FontSize: 13}, theme.Dark)
	n.H = h

	st := stateFor(n)
	if !st.layout.large {
		t.Fatalf("nowrap + 大文档应使用 large 布局：large=%v visual=%v",
			st.layout.large, st.layout.visual)
	}

	// 列保持：第 0 行第 4 列 → 第 1 行第 4 列，来回各一次。
	MoveCaret(n, 0, 4)
	assertCaret(t, n, 0, 4)
	press(n, runtime.KeyDown)
	assertCaret(t, n, 1, 4)
	press(n, runtime.KeyUp)
	assertCaret(t, n, 0, 4)

	// 末行只有 1 个字符（"x"）：从第 29998 行第 4 列下移，目标行装不下第 4 列，
	// 必须夹到末行行尾（列 1），不能算出越界偏移。
	MoveCaret(n, bodyLines-1, 4)
	assertCaret(t, n, bodyLines-1, 4)
	press(n, runtime.KeyDown)
	assertCaret(t, n, bodyLines, 1)
}
