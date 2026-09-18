package components

import (
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// newInput 造一个受控输入框：onInput 写进 got，节点本身不重建 —— 这正是
// 「同一帧里连打多个字符」的场景（旧实现会因为 value 还是旧值而丢字）。
func newInput(id string, initial string, extra *runtime.Props) (*runtime.VNode, *string) {
	got := new(string)
	*got = initial
	p := runtime.Props{
		Style:   runtime.Style{Width: 300, Height: 40},
		ID:      id,
		Value:   initial,
		OnInput: func(v string) { *got = v },
	}
	if extra != nil {
		p.Multiline = extra.Multiline
		p.Readonly = extra.Readonly
		p.Password = extra.Password
		p.Placeholder = extra.Placeholder
		p.OnSubmit = extra.OnSubmit
	}
	ResetInputState(id)
	return Input(p), got
}

func typeRunes(n *runtime.VNode, text string) {
	for _, r := range text {
		runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Char: r, Down: true})
	}
}

func press(n *runtime.VNode, key int) {
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: key, Down: true})
}

// TestInputKeepsAllCharsInOneFrame 是这次修复的核心回归点：
// 编辑基准不能取「上一次渲染的 props」，否则同一帧内的多个字符会互相覆盖。
func TestInputKeepsAllCharsInOneFrame(t *testing.T) {
	n, got := newInput("t-rapid", "", nil)
	runtime.SetFocus("t-rapid")
	defer runtime.SetFocus("")

	typeRunes(n, "你好世界")
	if *got != "你好世界" {
		t.Fatalf("同一帧连打多个字符被丢了：%q", *got)
	}
	typeRunes(n, "ABC")
	if *got != "你好世界ABC" {
		t.Fatalf("连续输入结果不对：%q", *got)
	}
}

func TestInputCaretMoveAndBackspace(t *testing.T) {
	n, got := newInput("t-caret", "abcd", nil)
	runtime.SetFocus("t-caret")
	defer runtime.SetFocus("")

	press(n, runtime.KeyBackspace)
	if *got != "abc" {
		t.Fatalf("退格应删掉末尾字符：%q", *got)
	}
	// 光标左移两格后在中间插入
	press(n, runtime.KeyLeft)
	press(n, runtime.KeyLeft)
	typeRunes(n, "X")
	if *got != "aXbc" {
		t.Fatalf("应在光标处插入：%q", *got)
	}
	// Delete 删光标后的字符
	press(n, runtime.KeyDelete)
	if *got != "aXc" {
		t.Fatalf("Delete 应删光标后一个字符：%q", *got)
	}
	// Home 回到行首再插入
	press(n, runtime.KeyHome)
	typeRunes(n, "1")
	if *got != "1aXc" {
		t.Fatalf("Home 后插入应在行首：%q", *got)
	}
	press(n, runtime.KeyEnd)
	typeRunes(n, "2")
	if *got != "1aXc2" {
		t.Fatalf("End 后插入应在行尾：%q", *got)
	}
}

func TestInputCompositionThenCommit(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, got := newInput("t-ime", "", nil)
	runtime.SetFocus("t-ime")
	defer runtime.SetFocus("")

	// 输入法开始拼字：组合串由我们绘制
	handled := runtime.DispatchComposition([]*runtime.VNode{n}, "nihao", 5)
	if !handled {
		t.Fatal("焦点输入框应消费组合串（自己绘制预编辑）")
	}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	found := false
	for _, s := range ctx.texts {
		if s == "nihao" {
			found = true
		}
	}
	if !found {
		t.Fatalf("组合串应被绘制出来：%v", ctx.texts)
	}
	if _, _, _, _, ok := FocusedCaretRect(); !ok {
		t.Fatal("绘制后应记录光标矩形（供输入法定位候选窗）")
	}

	// 组合串清空（系统上屏的那条消息会先清预编辑）
	runtime.DispatchComposition([]*runtime.VNode{n}, "", 0)

	// 上屏文本仍走 WM_CHAR 通路
	typeRunes(n, "你好")
	if *got != "你好" {
		t.Fatalf("上屏文本应插入：%q", *got)
	}
	if _, _, _, _, ok := FocusedCaretRect(); !ok {
		t.Fatal("上屏后光标矩形仍应可用")
	}
}

func TestInputCompositionReplacedByInsert(t *testing.T) {
	// 组合串没被显式清掉就来了字符（某些输入法会这样）时，应替换而不是叠加。
	n, got := newInput("t-ime2", "前", nil)
	runtime.SetFocus("t-ime2")
	defer runtime.SetFocus("")

	runtime.DispatchComposition([]*runtime.VNode{n}, "shi", 3)
	typeRunes(n, "是")
	if *got != "前是" {
		t.Fatalf("组合串应被提交的字符替换：%q", *got)
	}
}

func TestInputExternalValueWins(t *testing.T) {
	// 应用侧改值（发送后清空 / 点案例卡回填）时，内部的编辑缓冲必须让位。
	_, got := newInput("t-ext", "abc", nil)
	runtime.SetFocus("t-ext")
	defer runtime.SetFocus("")

	// 模拟下一次渲染：props 变成 "案例标题"
	n2 := Input(runtime.Props{
		Style:   runtime.Style{Width: 300, Height: 40},
		ID:      "t-ext",
		Value:   "案例标题",
		OnInput: func(v string) { *got = v },
	})
	typeRunes(n2, "！")
	if *got != "案例标题！" {
		t.Fatalf("外部改值后应以 props 为基准：%q", *got)
	}
}

func TestInputClickPlacesCaret(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, got := newInput("t-click", "abcdef", nil)
	runtime.SetFocus("t-click")
	defer runtime.SetFocus("")

	// 先画一帧，得到布局缓存（mock 的字符宽度 = 字号 * 0.5）。
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	// 点在第 3 个字符附近：x = 输入框左内边距 + 2 字符宽（mock 下 13*0.5=6.5/字符）
	renderer.SetPointer(n.X+inputPad+13, n.Y+10, false)
	n.OnClick()
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	typeRunes(n, "X")
	if *got != "abXcdef" {
		t.Fatalf("点击应把光标放到第 3 个字符前：%q", *got)
	}
}

func TestInputDragSelectsText(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, _ := newInput("t-drag", "abcdef", nil)
	defer runtime.SetFocus("")

	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	charW := float32(13) * 0.5
	renderer.SetPointer(n.X+inputPad+charW, n.Y+10, true)
	beginPointerSelect(n)
	from, to, has := SelectedRange(n)
	if has {
		t.Fatalf("按下时应塌缩光标，实际选区 [%d,%d)", from, to)
	}
	renderer.SetPointer(n.X+inputPad+charW*4, n.Y+10, true)
	dragPointerSelect(n)
	from, to, has = SelectedRange(n)
	if !has {
		t.Fatal("拖动后应形成选区")
	}
	if from != 1 || to != 4 {
		t.Fatalf("选区应对齐拖过的字符：[%d,%d) want [1,4)", from, to)
	}
	if sel := stateFor(n).selectedText(); sel != "bcd" {
		t.Fatalf("选中文本应为 bcd，实际 %q", sel)
	}
	n.OnClick()
	from, to, has = SelectedRange(n)
	if !has || from != 1 || to != 4 {
		t.Fatalf("抬起后应保留选区：[%d,%d) has=%v", from, to, has)
	}
}

func TestInputDragSelectsOverflowText(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	text := "abcdefghijklmnopqrstuvwxyz"
	n, _ := newInput("t-overflow", text, nil)
	n.Style.Width = 80
	n.X, n.Y, n.W, n.H = 0, 0, 80, 32
	defer runtime.SetFocus("")

	paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if lay := stateFor(n).layout; lay == nil || lay.maxLine != 0 {
		t.Fatalf("单行溢出不应折行，maxLine=%v", lay)
	}
	renderer.SetPointer(n.X+inputPad, n.Y+10, true)
	beginPointerSelect(n)
	renderer.SetPointer(n.X+n.W+40, n.Y+10, true)
	dragPointerSelect(n)
	from, to, has := SelectedRange(n)
	if !has || from != 0 {
		t.Fatalf("应从开头选起：[%d,%d) has=%v", from, to, has)
	}
	if to <= 10 {
		t.Fatalf("拖出右边缘应选到溢出部分，实际 to=%d", to)
	}
	if stateFor(n).scrollX <= 0 {
		t.Fatal("拖出右边缘后应横向滚动")
	}
}

func TestInputNonFocusedIgnoresKeys(t *testing.T) {
	n, got := newInput("t-blur", "abc", nil)
	runtime.SetFocus("") // 没有焦点
	typeRunes(n, "X")
	if *got != "abc" {
		t.Fatalf("未聚焦的输入框不应接收输入：%q", *got)
	}
}

func TestInputMultilineWrapsLongToken(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	key := "sk-" + strings.Repeat("Nc9Ne9DATEZIPvFl3x3vqeVMHCLxacZaOU4Qp9Q94VZ7jn0l", 2)
	n, _ := newInput("t-wrap-key", key, &runtime.Props{Multiline: true})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 140
	runtime.SetFocus("t-wrap-key")
	defer runtime.SetFocus("")

	paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	lay := stateFor(n).layout
	if lay == nil || lay.maxLine < 1 {
		t.Fatalf("超长密钥应在提示词框内折行，maxLine=%v", lay)
	}
	if p := lay.pointAt(len("sk-")); p.line != 0 {
		t.Fatalf("连字符处不应提前断行，实际 line=%d", p.line)
	}
	end := lay.pointAt(len(key))
	if end.line != lay.maxLine {
		t.Fatalf("光标应落在最后一行：line=%d max=%d", end.line, lay.maxLine)
	}
}

func TestInputMultilineEnterInsertsNewline(t *testing.T) {
	n, got := newInput("t-ml", "第一行", &runtime.Props{Multiline: true})
	runtime.SetFocus("t-ml")
	defer runtime.SetFocus("")
	press(n, runtime.KeyEnter)
	typeRunes(n, "第二行")
	if *got != "第一行\n第二行" {
		t.Fatalf("多行输入回车应换行：%q", *got)
	}
}

// TestInputMultilinePaintsFromTop 是提示词框的回归：固定高度的多行 Input
// 必须顶对齐绘制，不能沿用单行的垂直居中，否则光标在第一行、文字在中间。
func TestInputMultilinePaintsFromTop(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, _ := newInput("t-ml-paint", "描述你想创建的图片", &runtime.Props{Multiline: true})
	n.X, n.Y, n.W, n.H = 10, 20, 300, 140
	runtime.SetFocus("t-ml-paint")
	defer runtime.SetFocus("")

	paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if len(ctx.draws) == 0 {
		t.Fatal("应绘制文本")
	}
	d := ctx.draws[0]
	if d.style.VAlign != renderer.AlignStart {
		t.Fatalf("多行输入文字应对齐顶部，实际 VAlign=%v", d.style.VAlign)
	}
	wantY := n.Y + inputTopPad
	if d.y != wantY {
		t.Fatalf("多行输入文字应从顶部内边距开始：got y=%.1f want %.1f", d.y, wantY)
	}
}

func TestInputSingleLinePaintsCentered(t *testing.T) {
	ctx := &mockContext{w: 400, h: 200}
	n, _ := newInput("t-sl-paint", "hello", nil)
	n.X, n.Y, n.W, n.H = 10, 20, 300, 38

	paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if len(ctx.draws) == 0 {
		t.Fatal("应绘制文本")
	}
	d := ctx.draws[0]
	if d.style.VAlign != renderer.AlignCenter {
		t.Fatalf("单行输入文字应垂直居中，实际 VAlign=%v", d.style.VAlign)
	}
	if d.y != n.Y {
		t.Fatalf("单行输入文字绘制起点应为控件顶部：got y=%.1f want %.1f", d.y, n.Y)
	}
}
