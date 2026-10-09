// 编辑框对外的操作入口。
//
// 这些符号经 github.com/goedui/goed/ui 转发给应用层，用于状态栏行列、查找后选中定位、
// 弹出菜单的剪切 / 复制 / 粘贴 / 撤销等场景；组件内部不调用它们。
//
// 编辑状态的权威定义见 textedit_state.go，此处只做「节点 + 参数」到
// 内部方法的转接。
package components

import (
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// FocusedCaretRect 返回当前焦点编辑框的光标矩形（客户区 DIP），
// 宿主用它把输入法候选窗贴到光标旁边。ok 为 false 表示暂无可用位置。
func FocusedCaretRect() (x, y, w, h float32, ok bool) {
	id := runtime.FocusedID()
	if id == "" {
		return 0, 0, 0, 0, false
	}
	editMu.Lock()
	defer editMu.Unlock()
	st := editStates[id]
	if st == nil || !st.caretOK {
		return 0, 0, 0, 0, false
	}
	r := st.caretRect
	return r[0], r[1], r[2], r[3], true
}

// CaretLineCol 读取编辑框光标的 (行, 列)，均为 0 基，列按 rune 计。
// 未挂载或无状态的节点返回 (0, 0)。
func CaretLineCol(n *runtime.VNode) (line, col int) {
	if n == nil {
		return 0, 0
	}
	return stateFor(n).caretLineCol()
}

// MoveCaret 把编辑框光标移到 0 基的 (行, 列)，并让光标滚入可视范围。
// 「打开文件后定位到第 1 行第 1 列」这类场景由应用层在渲染后调用。
func MoveCaret(n *runtime.VNode, line, col int) {
	if n == nil {
		return
	}
	stateFor(n).moveToLineCol(line, col)
}

// SelectedRange 返回当前选区的字节区间；has 为 false 表示没有选区。
func SelectedRange(n *runtime.VNode) (from, to int, has bool) {
	if n == nil {
		return 0, 0, false
	}
	return stateFor(n).selRange()
}

// SelectRange 选中 [from, to) 字节区间，并把光标滚入可视范围。
func SelectRange(n *runtime.VNode, from, to int) {
	if n == nil {
		return
	}
	st := stateFor(n)
	st.caret = clampCaret(st.value, from)
	st.anchor = clampCaret(st.value, to)
	st.scrollSync = true
}

// SelectAll 全选。
func SelectAll(n *runtime.VNode) {
	if n == nil {
		return
	}
	stateFor(n).selectAll()
}

// SelectAllStay 全选，但保持当前滚动位置。光标仍在文末，只是不滚过去。
func SelectAllStay(n *runtime.VNode) {
	if n == nil {
		return
	}
	stateFor(n).selectAllStay()
}

// EditOp 对编辑框执行一次标准编辑命令，供弹出菜单 / 工具栏调用。
// 支持 "undo"、"redo"、"cut"、"copy"、"paste"、"delete"、"selectall"。
// 返回是否执行（只读框收到改动类命令时返回 false）。
func EditOp(n *runtime.VNode, op string) bool {
	if n == nil {
		return false
	}
	return editOpState(stateFor(n), renderer.AttrBool(n, "readonly") || !renderer.Enabled(n), onString(n, "onInput"), op)
}

// EditOpID 按编辑框 id 执行同一套命令。菜单回调发生在布局之外，那时拿不到节点。
// onInput 用创建编辑框时记下的回调，剪切 / 粘贴才能把新文本交回应用。
func EditOpID(id, op string) bool {
	st := textareaState(id)
	if st == nil || id == "" {
		return false
	}
	editMu.Lock()
	onInput := editInputHooks[id]
	editMu.Unlock()
	return editOpState(st, false, onInput, op)
}

func editOpState(st *editState, readonly bool, onInput func(string), op string) bool {
	if st == nil {
		return false
	}
	notify := func() {
		resetBlink()
		if onInput != nil {
			onInput(st.value)
		}
	}
	switch op {
	case "undo":
		if readonly || !st.undoStep() {
			return false
		}
		notify()
		return true
	case "redo":
		if readonly || !st.redoStep() {
			return false
		}
		notify()
		return true
	case "copy":
		if platform.ClipboardSet == nil {
			return false
		}
		if sel := st.selectedText(); sel != "" {
			platform.ClipboardSet(sel)
			return true
		}
		if st.value == "" {
			return false
		}
		platform.ClipboardSet(st.value)
		return true
	case "cut":
		if readonly || platform.ClipboardSet == nil {
			return false
		}
		sel := st.selectedText()
		if sel == "" {
			return false
		}
		platform.ClipboardSet(sel)
		st.snapshot()
		st.deleteSelection()
		notify()
		return true
	case "paste":
		if readonly || platform.ClipboardGet == nil {
			return false
		}
		text := platform.ClipboardGet()
		if text == "" {
			return false
		}
		st.snapshot()
		st.insert(text)
		notify()
		return true
	case "delete":
		if readonly {
			return false
		}
		if _, _, has := st.selRange(); !has && st.caret >= len(st.value) {
			return false
		}
		st.snapshot()
		if !st.deleteSelection() {
			st.deleteForward()
		}
		notify()
		return true
	case "selectall":
		st.selectAll()
		return true
	}
	return false
}

func textareaState(id string) *editState {
	if id == "" {
		return nil
	}
	editMu.Lock()
	defer editMu.Unlock()
	return editStates[id]
}

// TextareaScrollY 返回多行编辑框当前纵向滚动偏移。
func TextareaScrollY(id string) float32 {
	st := textareaState(id)
	if st == nil {
		return 0
	}
	return st.scrollY
}

// TextareaScrollMax 返回多行编辑框可滚上限。尚未布局时为 0。
func TextareaScrollMax(id string) float32 {
	st := textareaState(id)
	if st == nil || st.layout == nil {
		return 0
	}
	maxOff := float32(st.layout.maxLine+1)*st.layout.lineH + 2*inputTopPad - st.viewH
	if maxOff < 0 {
		return 0
	}
	return maxOff
}

// PageTextarea 按视口能放下的行数翻一页（dir>0 向下）。
// 与 PageUp/PageDown 同一条 movePage：光标和滚动一起走，不靠标题比例换算。
func PageTextarea(id string, dir float32) {
	st := textareaState(id)
	if st == nil || st.layout == nil || dir == 0 {
		return
	}
	sign := 1
	if dir < 0 {
		sign = -1
	}
	st.movePage(sign, false, st.viewH-2*inputTopPad)
}

// SetTextareaScrollY 设置多行编辑框纵向滚动偏移（会钳进合法范围）。
func SetTextareaScrollY(id string, y float32) {
	st := textareaState(id)
	if st == nil {
		return
	}
	st.scrollY = y
	st.scrollSync = false
	if st.viewH > 0 {
		st.scrollY = clampScrollY(st, st.viewH)
	} else if st.scrollY < 0 {
		st.scrollY = 0
	}
}
