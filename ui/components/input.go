package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// Input 单行文本输入。
//
// Attrs:
//
//	value        当前文本（外部改值时以 props 为准）
//	placeholder  占位文案
//	password     密码显示
//	readonly     只读（不显示光标、不接受输入）
//	onInput(v)   文本变化
//	onSubmit(v)  回车提交；返回 true 表示已处理
//	onBlur()     焦点离开
//
// 编辑状态（光标 / 选区 / 组合串 / 撤销）见 textedit_state.go，
// 交互在 textedit_keys.go，绘制在 textedit_paint.go。
func init() {
	renderer.Register("input", renderer.Widget{
		Measure: measureInput,
		Paint:   paintInput,
	})
}

func Input(parts ...any) *runtime.VNode {
	n := runtime.H("input", parts...)
	if n.OnClick == nil {
		n.OnClick = func() {
			finishPointerSelect(n)
		}
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		return handleEditKey(n, e, false)
	})
	n.Attrs["onComposition"] = runtime.CompositionHandler(func(text string, cursor int) bool {
		return handleComposition(n, text, cursor)
	})
	n.Attrs["onTextInput"] = runtime.TextInputHandler(func(text string) bool {
		return handleTextInput(n, text)
	})
	n.Attrs["onPointerDown"] = func() { beginPointerSelect(n) }
	n.Attrs["onPointerDrag"] = func() { dragPointerSelect(n) }
	runtime.SetBlurHook(n.ID(), onVoid(n, "onBlur"))
	return n
}

func measureInput(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = maxH
	_ = style
	_ = th
	w := n.Style.Width
	if w <= 0 {
		w = maxW
	}
	if w <= 0 {
		w = 160
	}
	h := n.Style.Height
	if h <= 0 {
		h = 32
	}
	return w, h
}

func paintInput(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	// 与按键处理一致：节点上的 multiline attr 把 Input 当成多行框。
	// 否则固定高度的提示词框会把文字垂直居中，光标却停在第一行。
	multiline := renderer.AttrBool(n, "multiline")
	paintEditable(ctx, n, style, th, stateFor(n), editPaint{
		multiline:     multiline,
		scrollToCaret: multiline,
		showScroll:    multiline,
	})
}
