package components

import (
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// Textarea 多行文本域。
//
// Attrs:
//
//	value        当前文本
//	placeholder  占位文案
//	readonly     只读
//	maxHeight    最大高度（内容超出后内部滚动，并显示滚动条）
//	minHeight    最小高度
//	onInput(v)   文本变化
//	onSubmit(v)  Ctrl+Enter 提交；返回 true 表示已处理
//
// 高度随内容自动增长（measure 按可视行数计算），光标 / 选区 / 组合串 /
// 撤销重做与 Input 共用同一套编辑核心（textedit_state.go 起的一族文件，
// 见该文件顶部的分文件索引）。
func init() {
	renderer.Register("textarea", renderer.Widget{
		Measure: measureTextarea,
		Paint:   paintTextarea,
	})
}

func Textarea(parts ...any) *runtime.VNode {
	n := runtime.H("textarea", parts...)
	if n.OnClick == nil {
		n.OnClick = func() {
			finishPointerSelect(n)
		}
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		if handleEditKey(n, e, true) {
			return true
		}
		// handleEditKey 已经把 Enter 的分支处理完了（接了 onSubmit 就是提交，
		// 没接就是换行）。这里只兜一个「没有焦点」之类的边角情况：
		// Ctrl+Enter 作为提交的历史别名，仍要能触发 onSubmit。
		if e.Key == runtime.KeyEnter && e.Ctrl && e.Down {
			if submit := onString(n, "onSubmit"); submit != nil {
				submit(stateFor(n).value)
				return true
			}
		}
		return false
	})
	n.Attrs["onComposition"] = runtime.CompositionHandler(func(text string, cursor int) bool {
		return handleComposition(n, text, cursor)
	})
	n.Attrs["onTextInput"] = runtime.TextInputHandler(func(text string) bool {
		return handleTextInput(n, text)
	})
	n.Attrs["onPointerDown"] = func() { beginPointerSelect(n) }
	n.Attrs["onPointerDrag"] = func() { dragPointerSelect(n) }
	// 滚轮在内容超高时内部滚动。delta 正值表示向上滚，偏移随之减小。
	n.Attrs["onWheel"] = func(x, y, delta float32) {
		st := stateFor(n)
		st.scrollY -= delta * 48
		// 钳制和绘制用同一个函数，避免滚轮设的值下一帧被收窄。
		st.scrollY = clampScrollY(st, n.H)
		// 用户主动浏览时不再让光标跟随把它拉回去（记事本行为）。
		st.scrollSync = false
		platform.RequestFrame()
	}
	return n
}

// textStyleOf 取节点样式里的字体设置（未设置时回退主题）。
func textStyleOf(n *runtime.VNode, style renderer.TextStyle, th theme.Theme) renderer.TextStyle {
	st := renderer.TextStyle{
		FontFamily: style.FontFamily,
		FontSize:   style.FontSize,
		Color:      renderer.ColorFrom(th.Foreground),
	}
	if st.FontFamily == "" {
		st.FontFamily = th.FontFamily
	}
	if st.FontSize <= 0 {
		st.FontSize = th.FontSize
		if st.FontSize <= 0 {
			st.FontSize = 13
		}
	}
	return st
}

func measureTextarea(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	st := stateFor(n)
	w := n.Style.Width
	if w <= 0 {
		w = maxW
	}
	if w <= 0 {
		w = 240
	}
	textStyle := textStyleOf(n, style, th)
	// nowrap 时不按宽度折行，高度只由硬换行决定。
	wrapW := w - 2*inputPad
	if renderer.AttrBool(n, "nowrap") {
		wrapW = 0
	}
	lay := layoutOf(ctx, st, wrapW, textStyle)
	h := float32(lay.maxLine+1)*lay.lineH + 2*inputTopPad
	if min := n.Style.MinHeight; min > 0 && h < min {
		h = min
	}
	if max := n.Style.MaxHeight; max > 0 && h > max {
		h = max
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}
	if h < 24 {
		h = 24
	}
	return w, h
}

func paintTextarea(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	paintEditable(ctx, n, style, th, stateFor(n), editPaint{
		multiline:     true,
		scrollToCaret: true,
		showScroll:    true,
	})
}
