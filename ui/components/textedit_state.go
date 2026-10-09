// 共享的文本编辑核心：Input（单行）与 Textarea（多行）都基于它。
//
// 编辑状态（文本 / 光标 / 选区 / 组合串 / 撤销）不放在 props 里，而是按 id 存在
// editState：节点每帧重建，拿 props 的 value 当编辑基准的话，同一帧里到达的多个
// 字符会互相覆盖 —— 快速连打、输入法整串上屏都会丢字。props 只负责「外部改值」。
//
// 按职责拆在同包的几个文件里：state（状态本体、取状态、基础编辑、闪烁）、layout
// （可视行与「坐标 ↔ 字节偏移」）、keys（按键 / 输入法 / 指针）、paint（绘制）、
// api（应用层入口）。
package components

import (
	"sync"
	"unicode/utf8"

	"github.com/goedui/goed/ui/caret"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
)

// 各文件共用的大小常量。
// LayoutDebug 记录每帧请求过的换行宽度（测试用）。
var LayoutDebug []float32

const (
	inputPad     = float32(10)
	inputTopPad  = float32(6)
	caretWidth   = float32(1.6)
	minCaretH    = float32(12)
	maxLayoutRun = 4000 // 超过此 rune 数改走可视行稀疏布局，不再截断文本
	undoLimit    = 50
)

// editState 是一个编辑框的权威状态。
type editState struct {
	value  string
	caret  int // 光标（选区活动端）
	anchor int // 选区固定端；等于 caret 表示没有选区
	comp   string
	compAt int
	compCu int
	// lastProp 是上一次绘制时 props 的 value：只有它变了才算「应用改了值」，
	// 同一帧里的旧节点（props 还是老值）不能拿来覆盖编辑缓冲。
	lastProp string
	// 撤销 / 重做
	undo []editSnap
	redo []editSnap
	// 最近一次绘制缓存的光标矩形（客户区 DIP），供输入法定位候选窗。
	caretRect [4]float32
	caretOK   bool
	// 多行时的滚动偏移（px，向下为正）
	scrollY float32
	viewH   float32
	// notifiedScrollY 是 onScroll 上一次报过的偏移，没变就不打扰应用。
	notifiedScrollY float32
	// scrollSync 表示刚发生过编辑或光标移动，下一次绘制要把光标滚进可视
	// 范围。用户用滚轮浏览时不能被强行拉回光标处（记事本行为）。
	scrollSync bool
	// scrollX 是不换行模式下的横向滚动偏移（px，向右为正）。
	scrollX float32
	// contentW 是最长行宽，横向滚动条按它算。文本没变就复用，不每帧重量。
	contentW    float32
	contentWFor string
	// 滚动条拖动：0 没拖，1 竖，2 横。grab 是按下时指针相对滑块起点的距离。
	barDrag int
	barGrab float32
	layout     *inputLayout
	layouts    [2]*inputLayout
	glyphW     map[rune]float32
	glyphLineH float32
}

type editSnap struct {
	value  string
	caret  int
	anchor int
}

var (
	editMu         sync.Mutex
	editStates     = map[string]*editState{}
	editInputHooks = map[string]func(string){}
)

func rememberInput(id string, fn func(string)) {
	if id == "" {
		return
	}
	editMu.Lock()
	if fn == nil {
		delete(editInputHooks, id)
	} else {
		editInputHooks[id] = fn
	}
	editMu.Unlock()
}

func (st *editState) dropLayout() {
	if st == nil {
		return
	}
	st.layout = nil
	st.layouts[0] = nil
	st.layouts[1] = nil
}

// ResetInputState 丢弃某个编辑框的内部状态（切换会话等场景可用）。
func ResetInputState(id string) {
	editMu.Lock()
	delete(editStates, id)
	delete(editInputHooks, id)
	editMu.Unlock()
}

func stateFor(n *runtime.VNode) *editState {
	id := n.ID()
	prop := renderer.AttrString(n, "value")
	if id == "" {
		return &editState{value: prop, caret: len(prop), anchor: len(prop), lastProp: prop}
	}
	editMu.Lock()
	defer editMu.Unlock()
	st := editStates[id]
	if st == nil {
		st = &editState{value: prop, caret: len(prop), anchor: len(prop), lastProp: prop}
		editStates[id] = st
		return st
	}
	if prop != st.lastProp {
		st.lastProp = prop
		// props 真的变了：应用侧改了值。
		if prop != st.value {
			st.value = prop
			st.comp = ""
			st.compAt = 0
			st.compCu = 0
			st.caret = len(prop)
			st.anchor = st.caret
			st.undo = nil
			st.redo = nil
			st.dropLayout()
			st.scrollY = 0
		}
	}
	st.caret = clampCaret(st.value, st.caret)
	st.anchor = clampCaret(st.value, st.anchor)
	if st.compAt > len(st.value) {
		st.comp = ""
		st.compAt = 0
		st.compCu = 0
	}
	return st
}

// ===== 基础操作 ========================================================

func clampCaret(s string, off int) int {
	if off < 0 {
		return 0
	}
	if off > len(s) {
		return len(s)
	}
	for off > 0 && off < len(s) && !utf8.RuneStart(s[off]) {
		off--
	}
	return off
}

func prevRuneStart(s string, off int) int {
	if off <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(s[:off])
	return off - size
}

// text 是「正文 + 组合串」的合成结果，绘制与布局都用它。
func (st *editState) text() string {
	if st.comp == "" {
		return st.value
	}
	at := clampCaret(st.value, st.compAt)
	return st.value[:at] + st.comp + st.value[at:]
}

// caretOffset 是光标在合成文本里的字节偏移（组合串打开时落在其后）。
func (st *editState) caretOffset() int {
	if st.comp == "" {
		return clampCaret(st.value, st.caret)
	}
	return clampCaret(st.value, st.compAt) + len(st.comp)
}

// selRange 返回排序后的选区；has 为 false 表示没有选区。
func (st *editState) selRange() (from, to int, has bool) {
	if st.anchor == st.caret {
		return 0, 0, false
	}
	from, to = st.caret, st.anchor
	if from > to {
		from, to = to, from
	}
	from = clampCaret(st.value, from)
	to = clampCaret(st.value, to)
	if from >= to {
		return 0, 0, false
	}
	return from, to, true
}

func (st *editState) collapse(at int) {
	st.caret = clampCaret(st.value, at)
	st.anchor = st.caret
	st.scrollSync = true
}

// snapshot 在改动文本前压栈（撤销用）。
func (st *editState) snapshot() {
	st.undo = append(st.undo, editSnap{value: st.value, caret: st.caret, anchor: st.anchor})
	if len(st.undo) > undoLimit {
		st.undo = st.undo[len(st.undo)-undoLimit:]
	}
	st.redo = nil
}

// undo / redo 恢复快照；返回是否生效。
func (st *editState) undoStep() bool {
	if len(st.undo) == 0 {
		return false
	}
	snap := st.undo[len(st.undo)-1]
	st.undo = st.undo[:len(st.undo)-1]
	st.redo = append(st.redo, editSnap{value: st.value, caret: st.caret, anchor: st.anchor})
	st.apply(snap)
	st.dropLayout()
	return true
}

func (st *editState) redoStep() bool {
	if len(st.redo) == 0 {
		return false
	}
	snap := st.redo[len(st.redo)-1]
	st.redo = st.redo[:len(st.redo)-1]
	st.undo = append(st.undo, editSnap{value: st.value, caret: st.caret, anchor: st.anchor})
	st.apply(snap)
	st.dropLayout()
	return true
}

func (st *editState) apply(snap editSnap) {
	st.value = snap.value
	st.caret = clampCaret(st.value, snap.caret)
	st.anchor = clampCaret(st.value, snap.anchor)
	st.comp = ""
	st.compAt = 0
	st.compCu = 0
	st.scrollSync = true
}

// insert 在光标处插入文本（有选区时先替换选区；组合中先丢组合串）。
func (st *editState) insert(text string) {
	if _, _, has := st.selRange(); has {
		st.deleteSelection()
	}
	at := st.caret
	if st.comp != "" {
		at = clampCaret(st.value, st.compAt)
		st.comp = ""
		st.compAt = 0
		st.compCu = 0
	}
	at = clampCaret(st.value, at)
	st.value = st.value[:at] + text + st.value[at:]
	st.caret = at + len(text)
	st.anchor = st.caret
	st.dropLayout()
	st.scrollSync = true
}

// replaceSelected 用 text 替换当前选区（无选区等于 insert）。
func (st *editState) replaceSelected(text string) {
	st.insert(text)
}

func (st *editState) deleteSelection() bool {
	from, to, has := st.selRange()
	if !has {
		return false
	}
	st.value = st.value[:from] + st.value[to:]
	st.collapse(from)
	st.dropLayout()
	return true
}

func (st *editState) selectedText() string {
	from, to, has := st.selRange()
	if !has {
		return ""
	}
	return st.value[from:to]
}

// backspace 删除：有选区删选区，否则删光标前一个字符（组合中取消组合）。
func (st *editState) backspace() bool {
	if st.comp != "" {
		st.comp = ""
		st.compAt = 0
		st.compCu = 0
		st.dropLayout()
		return true
	}
	if _, _, has := st.selRange(); has {
		return st.deleteSelection()
	}
	if st.caret <= 0 {
		return false
	}
	st.snapshot()
	start := prevRuneStart(st.value, st.caret)
	st.value = st.value[:start] + st.value[st.caret:]
	st.caret = start
	st.anchor = st.caret
	st.dropLayout()
	st.scrollSync = true
	return true
}

// deleteForward 删除：有选区删选区，否则删光标后一个字符。
func (st *editState) deleteForward() bool {
	if _, _, has := st.selRange(); has {
		return st.deleteSelection()
	}
	if st.caret >= len(st.value) {
		return false
	}
	st.snapshot()
	_, size := utf8.DecodeRuneInString(st.value[st.caret:])
	st.value = st.value[:st.caret] + st.value[st.caret+size:]
	st.dropLayout()
	st.scrollSync = true
	return true
}

// move 移动光标；extend 为 true 时扩展选区。
func (st *editState) move(delta int, extend bool) {
	if !extend {
		from, to, has := st.selRange()
		if has {
			// 先塌缩到对应端再走一步
			if delta < 0 {
				st.caret, st.anchor = from, from
			} else {
				st.caret, st.anchor = to, to
			}
		}
	}
	st.moveBy(delta)
	if !extend {
		st.anchor = st.caret
	}
	st.scrollSync = true
}

func (st *editState) moveBy(delta int) {
	if delta < 0 {
		for i := 0; i < -delta; i++ {
			st.caret = prevRuneStart(st.value, st.caret)
		}
		return
	}
	for i := 0; i < delta; i++ {
		if st.caret >= len(st.value) {
			break
		}
		_, size := utf8.DecodeRuneInString(st.value[st.caret:])
		st.caret += size
	}
}

func (st *editState) selectAll() {
	st.anchor = 0
	st.caret = len(st.value)
	st.scrollSync = true
}

// selectAllStay 全选，但不把视口拉到文末。scrollSync 会在下一帧把光标滚进
// 可视区，长文档里那就是跳到最后一行。
func (st *editState) selectAllStay() {
	st.anchor = 0
	st.caret = len(st.value)
	st.scrollSync = false
}

// compCursorBytes 把组合串内的字符偏移换算成字节偏移。
func compCursorBytes(st *editState) int {
	if st.comp == "" || st.compCu <= 0 {
		return 0
	}
	r := []rune(st.comp)
	if st.compCu > len(r) {
		return len(st.comp)
	}
	return len(string(r[:st.compCu]))
}

// ===== 光标闪烁 ========================================================
//
// 相位本体在 ui/caret：屏幕上同时只有一个可见光标，输入框和代码编辑区（editor 的
// mcode）共用同一份相位，才不会一快一慢地各闪各的。这里只留两个薄封装，
// textedit_keys.go / textedit_paint.go 里那一堆调用点不用动。

func blinkVisible() bool { return caret.Visible() }

func resetBlink() { caret.Reset() }
