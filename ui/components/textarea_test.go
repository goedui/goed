package components

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestLargeTextareaUsesSparseLayout(t *testing.T) {
	const id = "ta-large-layout"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	text := strings.Repeat("line\n", largeLayoutBytes/5) + "last line"
	n := Textarea(runtime.Props{ID: id, Value: text, Nowrap: true})
	st := stateFor(n)
	lay := layoutText(&mockContext{w: 400, h: 400}, st, text, 0, renderer.TextStyle{FontSize: 13})

	if !lay.large {
		t.Fatal("large text should use sparse layout")
	}
	if lay.maxLine < largeLayoutBytes/5 {
		t.Fatalf("maxLine = %d, want at least %d", lay.maxLine, largeLayoutBytes/5)
	}
	if got := lay.lineForOffset(len(text) - 1); got != lay.maxLine {
		t.Fatalf("last line = %d, want %d", got, lay.maxLine)
	}
	if got := lay.lineStart(lay.maxLine); got <= 0 {
		t.Fatalf("last line start = %d, want positive offset", got)
	}
}

func TestTextareaLayoutCoversFullWrappedDocument(t *testing.T) {
	const id = "ta-full-doc"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	text := strings.Repeat("abcdefghij\n", 500)
	if utf8.RuneCountInString(text) <= maxLayoutRun {
		t.Fatalf("测试文本应超过 maxLayoutRun=%d", maxLayoutRun)
	}
	n := Textarea(runtime.Props{
		ID:    id,
		Value: text,
		Style: runtime.Style{Width: 240, Height: 80, MinHeight: 80, MaxHeight: 80},
	})
	st := stateFor(n)
	ctx := &mockContext{w: 400, h: 200}
	lay := layoutText(ctx, st, text, 220, renderer.TextStyle{FontSize: 13})
	if !lay.visual {
		t.Fatal("超长文本应走可视行稀疏布局，而不是截断 points")
	}
	if lay.maxLine < 499 {
		t.Fatalf("maxLine = %d，应覆盖全部硬换行", lay.maxLine)
	}
	if got := lay.lineForOffset(len(text)); got != lay.maxLine {
		t.Fatalf("文末行 = %d, want %d", got, lay.maxLine)
	}

	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	maxOff := TextareaScrollMax(id)
	if maxOff <= 0 {
		t.Fatal("超长文档应能向下滚")
	}
	SetTextareaScrollY(id, maxOff)
	if got := TextareaScrollY(id); got < maxOff-1 {
		t.Fatalf("应能滚到文末：got %.1f max %.1f", got, maxOff)
	}
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, -100)
	if got := TextareaScrollY(id); got < maxOff-1 {
		t.Fatalf("滚到底后应停在文末：got %.1f max %.1f", got, maxOff)
	}
}

// newArea 造一个受控多行文本域。
func newArea(t *testing.T, id, value string, extra *runtime.Props) (*runtime.VNode, *string) {
	t.Helper()
	ResetInputState(id)
	got := new(string)
	*got = value
	p := runtime.Props{
		Style:   runtime.Style{Width: 400, MinHeight: 56, MaxHeight: 120},
		ID:      id,
		Value:   value,
		OnInput: func(v string) { *got = v },
	}
	if extra != nil {
		p.Readonly = extra.Readonly
		p.Nowrap = extra.Nowrap
		p.OnSubmit = extra.OnSubmit
		p.Placeholder = extra.Placeholder
	}
	return Textarea(p), got
}

// TestTextareaAutoGrow 内容越多，测得的高度越大（到 MaxHeight 为止）。
func TestTextareaAutoGrow(t *testing.T) {
	ctx := &mockContext{w: 500, h: 400}
	n, _ := newArea(t, "ta-grow", "", nil)
	runtime.SetFocus("ta-grow")
	defer runtime.SetFocus("")

	_, h1 := measureTextarea(ctx, n, 400, 0, renderer.TextStyle{FontSize: 13}, theme.Dark)

	// 高度下限是 MinHeight(56)，先堆到超过它的行数再看增长。
	for i := 0; i < 3; i++ {
		typeRunes(n, "一行内容")
		press(n, runtime.KeyEnter)
	}
	_, h2 := measureTextarea(ctx, n, 400, 0, renderer.TextStyle{FontSize: 13}, theme.Dark)
	if h2 <= h1 {
		t.Fatalf("内容变多后高度应增长：%.1f -> %.1f", h1, h2)
	}
	// 再来好几行：高度被夹在 MaxHeight 里。
	for i := 0; i < 12; i++ {
		press(n, runtime.KeyEnter)
		typeRunes(n, "更多内容")
	}
	_, h3 := measureTextarea(ctx, n, 400, 0, renderer.TextStyle{FontSize: 13}, theme.Dark)
	if h3 > 120+1 {
		t.Fatalf("高度应被 MaxHeight 夹住，实际 %.1f", h3)
	}
}

// TestTextareaCtrlEnterSubmit Ctrl+Enter 触发提交，普通 Enter 换行。
// TestTextareaEnterSubmitsShiftEnterNewlines 固化「回车发送 / Shift+Enter 换行」。
//
// 约定（与主流聊天输入框一致）：
//   - 接了 onSubmit 的多行框：单独 Enter 提交，Shift+Enter 换行；
//   - 没接 onSubmit 的纯编辑器：Enter 仍换行（否则「按回车没反应」）；
//   - Ctrl+Enter 是历史别名，继续当提交处理。
func TestTextareaEnterSubmitsShiftEnterNewlines(t *testing.T) {
	mk := func(id string, withSubmit bool, submitted *string) *runtime.VNode {
		ResetInputState(id)
		p := runtime.Props{
			Style:   runtime.Style{Width: 400, MinHeight: 56, MaxHeight: 120},
			ID:      id,
			Value:   "",
			OnInput: func(string) {},
		}
		if withSubmit {
			p.OnSubmit = func(v string) { *submitted = v }
		}
		return Textarea(p)
	}
	enter := func(n *runtime.VNode, shift, ctrl bool) {
		runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{
			Key: runtime.KeyEnter, Shift: shift, Ctrl: ctrl, Down: true,
		})
	}

	// 1) 接了 onSubmit：Enter 提交，且不往内容里塞换行。
	var submitted string
	n := mk("ta-enter", true, &submitted)
	runtime.SetFocus("ta-enter")
	defer runtime.SetFocus("")
	typeRunes(n, "在吗")
	enter(n, false, false)
	if submitted != "在吗" {
		t.Fatalf("Enter 应提交 %q，实际 submitted=%q", "在吗", submitted)
	}
	if got := stateFor(n).value; got != "在吗" {
		t.Fatalf("Enter 提交后不该换行，实际内容 %q", got)
	}

	// 2) 接了 onSubmit：Shift+Enter 换行，不提交。
	submitted = ""
	n2 := mk("ta-shift", true, &submitted)
	runtime.SetFocus("ta-shift")
	typeRunes(n2, "一")
	enter(n2, true, false)
	typeRunes(n2, "二")
	if submitted != "" {
		t.Fatalf("Shift+Enter 不该提交，实际 submitted=%q", submitted)
	}
	if got := stateFor(n2).value; got != "一\n二" {
		t.Fatalf("Shift+Enter 应换行，实际内容 %q", got)
	}

	// 3) 没接 onSubmit 的纯编辑器：Enter 仍然换行。
	n3 := mk("ta-plain", false, new(string))
	runtime.SetFocus("ta-plain")
	typeRunes(n3, "a")
	enter(n3, false, false)
	typeRunes(n3, "b")
	if got := stateFor(n3).value; got != "a\nb" {
		t.Fatalf("未接 onSubmit 时 Enter 应换行，实际 %q", got)
	}

	// 4) Ctrl+Enter 继续当提交（老习惯别弄坏）。
	submitted = ""
	n4 := mk("ta-ctrl", true, &submitted)
	runtime.SetFocus("ta-ctrl")
	typeRunes(n4, "再试")
	enter(n4, false, true)
	if submitted != "再试" {
		t.Fatalf("Ctrl+Enter 应继续提交，实际 submitted=%q", submitted)
	}
}

// TestTextareaSelectionAndUndo 选区 + 撤销重做。
func TestTextareaSelectionAndUndo(t *testing.T) {
	n, got := newArea(t, "ta-undo", "abcdef", nil)
	runtime.SetFocus("ta-undo")
	defer runtime.SetFocus("")

	// Ctrl+A 全选后输入 → 替换全部
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyA, Ctrl: true, Down: true})
	typeRunes(n, "X")
	if *got != "X" {
		t.Fatalf("全选后输入应替换全部：%q", *got)
	}
	// Ctrl+Z 撤销
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyZ, Ctrl: true, Down: true})
	if *got != "abcdef" {
		t.Fatalf("撤销应恢复原文本：%q", *got)
	}
	// Shift+Right 选中一个字符后输入替换
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyHome, Down: true})
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyRight, Shift: true, Down: true})
	typeRunes(n, "A")
	if *got != "Abcdef" {
		t.Fatalf("Shift 选区替换失败：%q", *got)
	}
	// Ctrl+Y 重做
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyZ, Ctrl: true, Down: true})
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyY, Ctrl: true, Down: true})
	if *got != "Abcdef" {
		t.Fatalf("重做结果不对：%q", *got)
	}
}

// TestTextareaMultilineHomeEnd 多行的 Home / End 只到当前行，Ctrl+Home/End 才到文档两端。
func TestTextareaMultilineHomeEnd(t *testing.T) {
	n, _ := newArea(t, "ta-line", "abc\ndefgh\nij", nil)
	runtime.SetFocus("ta-line")
	defer runtime.SetFocus("")

	// 初始光标在文档末尾（stateFor 把新状态的光标放在结尾）。
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyHome, Ctrl: true, Down: true})
	if line, col := CaretLineCol(n); line != 0 || col != 0 {
		t.Fatalf("Ctrl+Home 应到文档开头：(%d,%d)", line, col)
	}
	// End 只到当前行行尾，不跳到文档末尾。
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyEnd, Down: true})
	if line, col := CaretLineCol(n); line != 0 || col != 3 {
		t.Fatalf("End 应停在当前行行尾 (0,3)：(%d,%d)", line, col)
	}
	// Home 回到行首；随后 ↓ 需要布局，先手工建一次（首帧之前没有布局）。
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyHome, Down: true})
	ctx := &mockContext{w: 400, h: 400}
	st := stateFor(n)
	layoutOf(ctx, st, 380, renderer.TextStyle{FontSize: 13})
	press(n, runtime.KeyDown)
	if line, col := CaretLineCol(n); line != 1 || col != 0 {
		t.Fatalf("↓ 应到第二行行首 (1,0)：(%d,%d)", line, col)
	}
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyEnd, Down: true})
	if line, col := CaretLineCol(n); line != 1 || col != 5 {
		t.Fatalf("End 应停在第二行行尾 (1,5)：(%d,%d)", line, col)
	}
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyEnd, Ctrl: true, Down: true})
	if line, col := CaretLineCol(n); line != 2 || col != 2 {
		t.Fatalf("Ctrl+End 应到文档结尾 (2,2)：(%d,%d)", line, col)
	}
}

// TestTextareaTabInsertsIndent 多行里 Tab 插入制表符而不是切焦点。
func TestTextareaTabInsertsIndent(t *testing.T) {
	n, got := newArea(t, "ta-tab", "a", nil)
	runtime.SetFocus("ta-tab")
	defer runtime.SetFocus("")

	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyTab, Down: true})
	if *got != "a\t" {
		t.Fatalf("Tab 应插入制表符：%q", *got)
	}
}

// TestTextareaReadonlyBlocksEdits 只读框拒绝改动，但仍允许移动光标与全选。
func TestTextareaReadonlyBlocksEdits(t *testing.T) {
	n, got := newArea(t, "ta-ro", "只读内容", &runtime.Props{Readonly: true})
	runtime.SetFocus("ta-ro")
	defer runtime.SetFocus("")

	typeRunes(n, "X")
	press(n, runtime.KeyBackspace)
	press(n, runtime.KeyEnter)
	if *got != "只读内容" {
		t.Fatalf("只读框不应被改动：%q", *got)
	}
	if EditOp(n, "cut") || EditOp(n, "paste") || EditOp(n, "undo") {
		t.Fatal("只读框的改动类命令应被拒绝")
	}
	// 光标移动与全选不受限制（记事本行为）。
	press(n, runtime.KeyHome)
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyA, Ctrl: true, Down: true})
	if _, _, has := SelectedRange(n); !has {
		t.Fatal("只读框仍应支持 Ctrl+A 全选")
	}
}

// TestTextareaMoveCaretAndCaretLineCol 应用层可以直接读写光标位置。
func TestTextareaMoveCaretAndCaretLineCol(t *testing.T) {
	n, _ := newArea(t, "ta-move", "alpha\nbeta\ngamma", nil)

	MoveCaret(n, 2, 3)
	if line, col := CaretLineCol(n); line != 2 || col != 3 {
		t.Fatalf("MoveCaret 后应为 (2,3)：(%d,%d)", line, col)
	}
	// 越界的行列会被夹到合法范围。
	MoveCaret(n, 99, 99)
	if line, col := CaretLineCol(n); line != 2 || col != 5 {
		t.Fatalf("越界应被夹到文档末尾：(%d,%d)", line, col)
	}
}

// TestTextareaNowrapKeepsLongLineSingle 关闭自动换行后，长行不产生软换行，
// 高度只由硬换行决定（记事本「自动换行」取消勾选后的行为）。
func TestArrowKeysRevealCaretHorizontally(t *testing.T) {
	const id = "ta-arrow-x"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })
	text := strings.Repeat("a", 80)
	n := Textarea(runtime.Props{
		ID: id, Value: text, Nowrap: true,
		Style: runtime.Style{Width: 80, Height: 40, FontSize: 14},
	})
	n.X, n.Y, n.W, n.H = 0, 0, 80, 40
	ctx := &mockContext{w: 200, h: 80}
	style := renderer.TextStyle{FontSize: 14}
	runtime.SetFocus(id)
	t.Cleanup(func() { runtime.SetFocus("") })
	MoveCaret(n, 0, 0)
	paintTextarea(ctx, n, style, theme.Light)
	for i := 0; i < len(text); i++ {
		press(n, runtime.KeyRight)
		paintTextarea(ctx, n, style, theme.Light)
	}
	st := stateFor(n)
	endX := st.layout.pointAtLineEnd(0).x
	viewW := n.W - 2*inputPad
	want := endX - viewW
	if want < 0 {
		want = 0
	}
	if st.scrollX+1 < want {
		t.Fatalf("right arrow should reach line end, scrollX=%.1f want>=%.1f", st.scrollX, want)
	}
	scrolled := st.scrollX
	for i := 0; i < len(text); i++ {
		press(n, runtime.KeyLeft)
		paintTextarea(ctx, n, style, theme.Light)
	}
	if st.scrollX >= scrolled {
		t.Fatalf("left arrow should scroll back, scrollX=%.1f was %.1f", st.scrollX, scrolled)
	}
	if st.scrollX > 1 {
		t.Fatalf("caret at start should show column 0, scrollX=%.1f", st.scrollX)
	}
}


func TestTextareaNowrapKeepsLongLineSingle(t *testing.T) {
	ctx := &mockContext{w: 500, h: 400}
	style := renderer.TextStyle{FontSize: 13}
	long := "abcdefghijklmnopqrstuvwxyz0123456789"

	ResetInputState("ta-nowrap")
	n := Textarea(runtime.Props{
		Style:  runtime.Style{Width: 200, MaxHeight: 400},
		ID:     "ta-nowrap",
		Value:  long,
		Nowrap: true,
	})
	_, hNowrap := measureTextarea(ctx, n, 200, 0, style, theme.Dark)

	ResetInputState("ta-wrap")
	wrapped := Textarea(runtime.Props{
		Style: runtime.Style{Width: 200, MaxHeight: 400},
		ID:    "ta-wrap",
		Value: long,
	})
	_, hWrapped := measureTextarea(ctx, wrapped, 200, 0, style, theme.Dark)

	if hWrapped <= hNowrap {
		t.Fatalf("自动换行应比不换行高：nowrap=%.1f wrap=%.1f", hNowrap, hWrapped)
	}
	if hNowrap > 2*style.FontSize+2*inputTopPad {
		t.Fatalf("不换行时应保持单行高度：%.1f", hNowrap)
	}
}

// TestEditOpUndoRedoAndDelete 弹出菜单用的标准编辑命令。
func TestEditOpUndoRedoAndDelete(t *testing.T) {
	n, got := newArea(t, "ta-op", "hello", nil)
	runtime.SetFocus("ta-op")
	defer runtime.SetFocus("")

	MoveCaret(n, 0, 0)
	if !EditOp(n, "delete") {
		t.Fatal("光标在文首时 Delete 应删掉后一个字符")
	}
	if *got != "ello" {
		t.Fatalf("Delete 结果不对：%q", *got)
	}
	if !EditOp(n, "selectall") {
		t.Fatal("selectall 应可执行")
	}
	if _, _, has := SelectedRange(n); !has {
		t.Fatal("selectall 后应有选区")
	}
	if !EditOp(n, "undo") {
		t.Fatal("undo 应可执行")
	}
	if *got != "hello" {
		t.Fatalf("undo 应恢复：%q", *got)
	}
	if !EditOp(n, "redo") {
		t.Fatal("redo 应可执行")
	}
	if *got != "ello" {
		t.Fatalf("redo 应重放：%q", *got)
	}
	// 光标已在文档末尾且无选区时，Delete 无事可做。
	MoveCaret(n, 0, 4)
	if EditOp(n, "delete") {
		t.Fatal("文末无选区时 Delete 应返回 false")
	}
}

// TestTextareaWheelScrollsContent 滚轮滚动后偏移要保留到下一帧。
//
// 回归点：editPaint 曾经有个没人赋值的 scrollY 字段，绘制时把它当成当前
// 偏移，于是 st.scrollY 每帧被覆盖回 0 —— 滚轮一滚就被抹掉。
func TestTextareaWheelScrollsContent(t *testing.T) {
	const id = "ta-wheel"
	text := "第一行\n第二行\n第三行\n第四行"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	// 每帧都会重建节点，这里模拟这一点：同一 id 复用同一份编辑状态。
	newNode := func() *runtime.VNode {
		n := Textarea(runtime.Props{
			Style: runtime.Style{Width: 400, MinHeight: 24, MaxHeight: 30},
			ID:    id,
			Value: text,
		})
		// 可视高度只放得下一行，保证内容确实超出。
		n.H = 30
		return n
	}

	ctx := &mockContext{w: 400, h: 400}
	th := theme.Dark
	n := newNode()
	renderer.PaintTree(ctx, []*runtime.VNode{n}, th)
	if st := stateFor(n); st.scrollY != 0 {
		t.Fatalf("初始不应有滚动偏移：%.1f", st.scrollY)
	}

	// 通过真实分发路径向下滚动（delta 为负）。
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, -1)

	afterWheel := stateFor(n).scrollY
	if afterWheel <= 0 {
		t.Fatalf("向下滚动后偏移应大于 0：%.1f", afterWheel)
	}

	// 再画一帧：偏移必须保留，不能被绘制的默认值覆盖。
	next := newNode()
	renderer.PaintTree(ctx, []*runtime.VNode{next}, th)
	if got := stateFor(next).scrollY; got != afterWheel {
		t.Fatalf("重绘后滚动偏移被重置：%.1f → %.1f", afterWheel, got)
	}

	// 继续猛滚：偏移要停在内容底端，不能一帧一个值地抖动。
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, -100)
	first := stateFor(n).scrollY
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, -100)
	second := stateFor(n).scrollY
	if second != first {
		t.Fatalf("到达底端后偏移应停住：%.1f → %.1f", first, second)
	}
	renderer.PaintTree(ctx, []*runtime.VNode{newNode()}, th)
	if third := stateFor(n).scrollY; third != second {
		t.Fatalf("停在底端后重绘仍要一致：%.1f → %.1f", second, third)
	}

	// 向上滚回顶端后偏移归零。
	HandleWheel([]*runtime.VNode{n}, n.X+10, n.Y+10, 100)
	if top := stateFor(n).scrollY; top != 0 {
		t.Fatalf("向上滚应回到顶端：%.1f", top)
	}
}

func TestTextareaFlexSkipUnconstrainedWrap(t *testing.T) {
	const id = "ta-flex-skip"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	text := strings.Repeat("abcdefghij", 80)
	n := Textarea(runtime.Props{
		ID:    id,
		Value: text,
		Style: runtime.Style{Flex: 1, MinHeight: 80},
	})
	st := stateFor(n)
	ctx := &mockContext{w: 800, h: 400}
	style := renderer.TextStyle{FontSize: 13}

	LayoutDebug = nil
	_, h0 := measureTextarea(ctx, n, 0, 200, style, theme.Dark)
	if h0 != 200 {
		t.Fatalf("flex 列无约束宽度应回报父级高度：got %.1f", h0)
	}
	if len(LayoutDebug) != 0 {
		t.Fatalf("无约束宽度不该折行：%v", LayoutDebug)
	}
	if st.layout != nil {
		t.Fatal("无约束宽度不该写入 layout 缓存")
	}

	LayoutDebug = nil
	_, _ = measureTextarea(ctx, n, 430, 200, style, theme.Dark)
	if len(LayoutDebug) == 0 {
		t.Fatal("真实宽度应折行")
	}
	if st.layout == nil || st.layout.wrapW != 410 {
		t.Fatalf("真实宽度 wrapW=%v", st.layout)
	}
}

func TestTextareaPaintsVisibleLinesOnly(t *testing.T) {
	const id = "ta-visible"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	var b strings.Builder
	for i := 0; i < 40; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("line-")
		b.WriteString(strings.Repeat("x", 8))
		b.WriteByte(byte('0' + i%10))
	}
	n := Textarea(runtime.Props{
		ID:    id,
		Value: b.String(),
		Style: runtime.Style{Width: 240, Height: 48, MinHeight: 48, MaxHeight: 48},
	})
	ctx := &mockContext{w: 400, h: 200}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if len(ctx.texts) == 0 {
		t.Fatal("应画出文本")
	}
	drawn := ctx.texts[len(ctx.texts)-1]
	if strings.Contains(drawn, "line-x") && strings.Count(drawn, "\n") > 8 {
		t.Fatalf("应只画可见行，得到 %d 行", strings.Count(drawn, "\n")+1)
	}
}

func TestGlyphWidthCachedAcrossRebuild(t *testing.T) {
	const id = "ta-glyph"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })

	text := strings.Repeat("汉字宽", 40)
	n := Textarea(runtime.Props{ID: id, Value: text, Style: runtime.Style{Width: 200}})
	st := stateFor(n)
	ctx := &countingMeasure{w: 400, h: 200}
	style := renderer.TextStyle{FontSize: 13}
	layoutText(ctx, st, text, 180, style)
	first := ctx.n
	if first == 0 {
		t.Fatal("首次折行应量字形")
	}
	st.dropLayout()
	ctx.n = 0
	layoutText(ctx, st, text, 180, style)
	if ctx.n >= first {
		t.Fatalf("字形缓存后测量次数应从 %d 下降，得到 %d", first, ctx.n)
	}
}

func TestSparseSelectAllPaintsHighlightWithoutScroll(t *testing.T) {
	const id = "ta-sparse-sel"
	ResetInputState(id)
	t.Cleanup(func() { ResetInputState(id) })
	text := strings.Repeat("abcdefghij\n", 500)
	n := Textarea(runtime.Props{
		ID: id, Value: text, Nowrap: true,
		Style: runtime.Style{Width: 200, Height: 80, FontSize: 14},
	})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 80
	ctx := &mockContext{w: 200, h: 80}
	paintTextarea(ctx, n, renderer.TextStyle{FontSize: 14}, theme.Light)
	st := stateFor(n)
	st.scrollY = 40
	st.selectAllStay()
	ctx.fills = nil
	paintTextarea(ctx, n, renderer.TextStyle{FontSize: 14}, theme.Light)
	if abs32(st.scrollY-40) > 1 {
		t.Fatalf("select all moved scroll to %.1f", st.scrollY)
	}
	hi := 0
	for _, f := range ctx.fills {
		if f.c.A > 0.2 && f.c.A < 0.5 && f.h > 4 {
			hi++
		}
	}
	if hi == 0 {
		t.Fatal("sparse selection painted no highlight")
	}
}

// TestTextareaSelectionCopy 复制的是选区内容。
func TestTextareaSelectionCopy(t *testing.T) {
	old := platform.ClipboardGet
	defer func() { platform.ClipboardGet = old }()
	var copied string
	platform.ClipboardGet = func() string { return "" }
	platform.ClipboardSet = func(v string) { copied = v }

	n, _ := newArea(t, "ta-copy", "你好世界", nil)
	runtime.SetFocus("ta-copy")
	defer runtime.SetFocus("")

	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyHome, Down: true})
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyRight, Shift: true, Down: true})
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyRight, Shift: true, Down: true})
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyC, Ctrl: true, Down: true})
	if copied != "你好" {
		t.Fatalf("复制应是选区内容：%q", copied)
	}
}
