// 编辑框的交互：按键、输入法组合串、上屏文本、指针落点。
//
// 按键处理是 Input（单行）与 Textarea（多行）共用的入口，组件外壳只负责把
// onKeyDown / onComposition / onTextInput / onClick 转进来，多行差异由
// multiline 参数与节点上的 multiline attr 表达。
package components

import (
	"strings"
	"unicode/utf8"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// viewportH 返回多行编辑框的可视文本高度（px），供翻页计算行数。
func viewportH(n *runtime.VNode) float32 {
	if n == nil {
		return 0
	}
	h := n.H - 2*inputTopPad
	if h < 0 {
		return 0
	}
	return h
}

// isMutatingEditKey 报告这次按键是否会改动文本。只读编辑框用它过滤输入，
// 同时放行光标移动与 Ctrl+A / Ctrl+C，对齐记事本只读模式的行为。
func isMutatingEditKey(e runtime.KeyEvent) bool {
	if e.Char >= 32 && !e.Ctrl {
		return true
	}
	if !e.Down {
		return false
	}
	switch e.Key {
	case runtime.KeyBackspace, runtime.KeyDelete, runtime.KeyEnter, runtime.KeyTab:
		return true
	}
	if !e.Ctrl {
		return false
	}
	switch e.Key {
	case runtime.KeyX, runtime.KeyV, runtime.KeyZ, runtime.KeyY:
		return true
	}
	return false
}

func handleEditKey(n *runtime.VNode, e runtime.KeyEvent, multiline bool) bool {
	if n == nil || !renderer.Enabled(n) {
		return false
	}
	// 以节点上的 multiline attr 为准：组件类型只决定默认值。
	if renderer.AttrBool(n, "multiline") {
		multiline = true
	}
	if !runtime.IsFocused(n) {
		return false
	}
	if renderer.AttrBool(n, "readonly") && isMutatingEditKey(e) {
		return true
	}
	st := stateFor(n)
	onInput := func() {
		resetBlink()
		if fn := onString(n, "onInput"); fn != nil {
			fn(st.value)
		}
	}
	if e.Char != 0 && e.Char >= 32 {
		if e.Ctrl {
			return true
		}
		st.snapshot()
		st.insert(string(e.Char))
		onInput()
		return true
	}
	if !e.Down && e.Char == 0 {
		return false
	}
	shift := e.Shift
	switch e.Key {
	case runtime.KeyBackspace:
		if st.backspace() {
			onInput()
		}
		return true
	case runtime.KeyDelete:
		if st.deleteForward() {
			onInput()
		}
		return true
	case runtime.KeyLeft:
		if e.Ctrl {
			st.moveWord(-1, shift)
		} else {
			st.move(-1, shift)
		}
		resetBlink()
		return true
	case runtime.KeyRight:
		if e.Ctrl {
			st.moveWord(1, shift)
		} else {
			st.move(1, shift)
		}
		resetBlink()
		return true
	case runtime.KeyUp:
		if multiline {
			st.moveLine(-1, shift)
			resetBlink()
			return true
		}
		return false
	case runtime.KeyDown:
		if multiline {
			st.moveLine(1, shift)
			resetBlink()
			return true
		}
		return false
	case runtime.KeyPageUp:
		if multiline {
			st.movePage(-1, shift, viewportH(n))
			resetBlink()
			return true
		}
		return false
	case runtime.KeyPageDown:
		if multiline {
			st.movePage(1, shift, viewportH(n))
			resetBlink()
			return true
		}
		return false
	case runtime.KeyHome:
		// 多行：行首（软换行也算一行）；Ctrl+Home：文档开头。
		if multiline && !e.Ctrl {
			start, _, _ := st.lineBounds()
			st.caret = start
		} else {
			st.caret = 0
		}
		if !shift {
			st.anchor = st.caret
		}
		st.scrollSync = true
		resetBlink()
		return true
	case runtime.KeyEnd:
		// 多行：行尾；Ctrl+End：文档结尾。
		if multiline && !e.Ctrl {
			_, end, _ := st.lineBounds()
			st.caret = end
		} else {
			st.caret = len(st.value)
		}
		if !shift {
			st.anchor = st.caret
		}
		st.scrollSync = true
		resetBlink()
		return true
	case runtime.KeyTab:
		// 多行里 Tab 插入制表符；单行留给上层的焦点切换。
		if multiline {
			st.snapshot()
			st.insert("\t")
			onInput()
			return true
		}
		return false
	case runtime.KeyEnter:
		if multiline {
			// 约定：Shift+Enter 换行，单独 Enter 提交（与主流聊天输入框一致）。
			// 没接 onSubmit 的多行框（纯编辑器）保持原行为——Enter 就是换行，
			// 否则用户会「按回车没反应」。
			if e.Shift {
				st.snapshot()
				st.insert("\n")
				onInput()
				return true
			}
			submit := onString(n, "onSubmit")
			// Ctrl+Enter 是历史用法，继续当提交处理，别把老习惯弄坏。
			if submit != nil && (e.Ctrl || !e.Shift) {
				submit(st.value)
				return true
			}
			if !e.Ctrl {
				st.snapshot()
				st.insert("\n")
				onInput()
				return true
			}
			return false
		}
		// 单行：回车即提交。
		if submit := onString(n, "onSubmit"); submit != nil && !e.Ctrl && !e.Shift {
			submit(st.value)
			return true
		}
		return false
	case runtime.KeyA:
		if e.Ctrl {
			st.selectAll()
			resetBlink()
			return true
		}
	case runtime.KeyZ:
		if e.Ctrl && !e.Shift {
			if st.undoStep() {
				onInput()
				resetBlink()
			}
			return true
		}
		if e.Ctrl && e.Shift {
			if st.redoStep() {
				onInput()
				resetBlink()
			}
			return true
		}
	case runtime.KeyY:
		if e.Ctrl {
			if st.redoStep() {
				onInput()
				resetBlink()
			}
			return true
		}
	case runtime.KeyC:
		if e.Ctrl && platform.ClipboardSet != nil {
			if sel := st.selectedText(); sel != "" {
				platform.ClipboardSet(sel)
			} else {
				platform.ClipboardSet(st.value)
			}
			return true
		}
	case runtime.KeyX:
		if e.Ctrl && platform.ClipboardSet != nil {
			if sel := st.selectedText(); sel != "" {
				platform.ClipboardSet(sel)
				st.snapshot()
				st.deleteSelection()
				onInput()
			} else {
				platform.ClipboardSet(st.value)
				st.snapshot()
				st.value = ""
				st.caret = 0
				st.anchor = 0
				st.layout = nil
				onInput()
			}
			return true
		}
	case runtime.KeyV:
		if e.Ctrl && platform.ClipboardGet != nil {
			if text := platform.ClipboardGet(); text != "" {
				st.snapshot()
				st.insert(text)
				onInput()
			}
			return true
		}
	}
	return e.Char != 0
}

// moveWord 按词移动（Ctrl+Left/Right）：连续的字词字符算一段。
func (st *editState) moveWord(delta int, extend bool) {
	if !extend {
		from, to, has := st.selRange()
		if has {
			if delta < 0 {
				st.caret, st.anchor = from, from
			} else {
				st.caret, st.anchor = to, to
			}
		}
	}
	isWord := func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return true
		case r >= 0x2E80: // CJK 一字一词
			return false
		}
		return false
	}
	if delta < 0 {
		for st.caret > 0 {
			r, size := utf8.DecodeLastRuneInString(st.value[:st.caret])
			st.caret -= size
			if !isWord(r) {
				break
			}
		}
	} else {
		for st.caret < len(st.value) {
			r, size := utf8.DecodeRuneInString(st.value[st.caret:])
			st.caret += size
			if !isWord(r) {
				break
			}
		}
	}
	if !extend {
		st.anchor = st.caret
	}
	st.scrollSync = true
}

// moveLine 上下移动一行（多行）：按当前 x 找相邻行最近的插入点。
func (st *editState) moveLine(delta int, extend bool) {
	if st.layout == nil {
		return
	}
	cur := st.layout.pointAt(st.caretOffset())
	target := cur.line + delta
	if target < 0 {
		st.caret = 0
		if !extend {
			st.anchor = st.caret
		}
		return
	}
	if target > st.layout.maxLine {
		st.caret = len(st.value)
		if !extend {
			st.anchor = st.caret
		}
		return
	}
	p := st.layout.pointAtXY(cur.x, (float32(target)+0.5)*st.layout.lineH)
	st.caret = clampCaret(st.value, p.off)
	if !extend {
		st.anchor = st.caret
	}
	st.scrollSync = true
}

// lineBounds 返回光标所在可视行的起止字节偏移（软换行也算一行）。
// 布局尚未建立时退化为按硬换行切分。
func (st *editState) lineBounds() (start, end int, ok bool) {
	at := st.caretOffset()
	if st.layout != nil && len(st.layout.points) > 0 {
		p := st.layout.pointAt(at)
		start, end = -1, -1
		for _, q := range st.layout.points {
			if q.line != p.line {
				continue
			}
			if start < 0 || q.off < start {
				start = q.off
			}
			if q.off > end {
				end = q.off
			}
		}
		if start >= 0 && end >= start {
			return start, end, true
		}
	}
	text := st.value
	at = clampCaret(text, at)
	s := strings.LastIndexByte(text[:at], '\n') + 1
	e := len(text)
	if i := strings.IndexByte(text[at:], '\n'); i >= 0 {
		e = at + i
	}
	return s, e, true
}

// movePage 上下翻页：按视口能放下的行数移动，保持列位置。
func (st *editState) movePage(dir int, extend bool, viewH float32) {
	if st.layout == nil || len(st.layout.points) == 0 {
		return
	}
	cur := st.layout.pointAt(st.caretOffset())
	lines := 1
	if st.layout.lineH > 0 && viewH > 0 {
		lines = int(viewH / st.layout.lineH)
		if lines < 1 {
			lines = 1
		}
	}
	target := cur.line + dir*lines
	switch {
	case target < 0:
		st.caret = 0
	case target > st.layout.maxLine:
		st.caret = len(st.value)
	default:
		p := st.layout.pointAtXY(cur.x, (float32(target)+0.5)*st.layout.lineH)
		st.caret = clampCaret(st.value, p.off)
	}
	if !extend {
		st.anchor = st.caret
	}
	st.scrollSync = true
}

// handleComposition 更新输入法组合串。返回 true 表示组合串由控件自己绘制，
// 宿主不必让系统画内联组合窗（候选窗仍由系统显示）。
func handleComposition(n *runtime.VNode, text string, cursor int) bool {
	if n == nil || !runtime.IsFocused(n) {
		return false
	}
	st := stateFor(n)
	if text == "" {
		st.comp = ""
		st.compAt = 0
		st.compCu = 0
		st.layout = nil
		resetBlink()
		return true
	}
	if st.comp == "" {
		from, _, has := st.selRange()
		if has {
			st.deleteSelection() // 有选区时组合串替换选区
		}
		st.compAt = st.caret
		_ = from
	}
	st.comp = text
	st.compCu = cursor
	st.layout = nil
	resetBlink()
	return true
}

// handleTextInput 收下一段上屏文本（输入法提交 / 宿主注入）。
func handleTextInput(n *runtime.VNode, text string) bool {
	if n == nil || !runtime.IsFocused(n) || text == "" {
		return false
	}
	st := stateFor(n)
	st.comp = ""
	st.compAt = 0
	st.compCu = 0
	st.snapshot()
	st.insert(text)
	resetBlink()
	if fn := onString(n, "onInput"); fn != nil {
		fn(st.value)
	}
	return true
}

// placeCaretFromPointer 用最近一次指针位置把光标移到点击处。
func placeCaretFromPointer(n *runtime.VNode) {
	extendCaretFromPointer(n, false)
}

// extendCaretFromPointer 把光标放到指针处；extend 为 true 时保留锚点以形成选区。
func extendCaretFromPointer(n *runtime.VNode, extend bool) {
	if n == nil {
		return
	}
	x, y, ok := renderer.Pointer()
	if !ok {
		return
	}
	st := stateFor(n)
	if st.layout == nil {
		return
	}
	localX := x - n.X - inputPad + st.scrollX
	localY := y - n.Y - inputTopPad + st.scrollY
	if extend {
		viewW := n.W - 2*inputPad
		if viewW > 0 {
			switch {
			case x < n.X+inputPad:
				st.scrollX -= (n.X + inputPad - x)
				if st.scrollX < 0 {
					st.scrollX = 0
				}
				localX = st.scrollX
			case x > n.X+inputPad+viewW:
				st.scrollX += x - (n.X + inputPad + viewW)
				end := st.layout.pointAtLineEnd(st.layout.pointAt(st.caretOffset()).line)
				maxOff := end.x - viewW
				if maxOff < 0 {
					maxOff = 0
				}
				if st.scrollX > maxOff {
					st.scrollX = maxOff
				}
				localX = st.scrollX + viewW
			}
		}
	}
	st.caret = clampCaret(st.value, st.layout.pointAtXY(localX, localY).off)
	if !extend {
		st.anchor = st.caret
	}
	st.scrollSync = true
	if extend {
		followCaretX(n, st)
	}
	resetBlink()
}

func followCaretX(n *runtime.VNode, st *editState) {
	if n == nil || st == nil || st.layout == nil {
		return
	}
	viewW := n.W - 2*inputPad
	if viewW <= 0 {
		return
	}
	cx := st.layout.pointAt(st.caretOffset()).x
	switch {
	case cx-st.scrollX < 0:
		st.scrollX = cx
	case cx-st.scrollX > viewW:
		st.scrollX = cx - viewW
	}
	if st.scrollX < 0 {
		st.scrollX = 0
	}
}

func beginPointerSelect(n *runtime.VNode) {
	if n == nil || !renderer.Enabled(n) {
		return
	}
	runtime.SetFocus(n.ID())
	extendCaretFromPointer(n, false)
}

func dragPointerSelect(n *runtime.VNode) {
	if n == nil || !renderer.Enabled(n) || !runtime.IsFocused(n) {
		return
	}
	extendCaretFromPointer(n, true)
}

func finishPointerSelect(n *runtime.VNode) {
	if n == nil || !renderer.Enabled(n) {
		return
	}
	runtime.SetFocus(n.ID())
	if _, _, has := stateFor(n).selRange(); has {
		resetBlink()
		return
	}
	extendCaretFromPointer(n, false)
}
