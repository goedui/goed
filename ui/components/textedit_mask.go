package components

import (
	"strings"
	"unicode/utf8"
)

// passwordMask 是密码框的显示字符。它是 3 字节，显示偏移与真实值的字节
// 偏移不再相等，所有跨边界换算都必须走 displayOffset / valueOffsetFromDisplay。
const passwordMask = "•"

// ===== 密码显示偏移换算 ==================================================
//
// 密码框把每个 rune 换成一个圆点再交给布局，布局里的 off 是显示文本偏移，
// 而 st.caret / st.anchor / selRange 是真实值偏移。两组偏移不能直接混用：
// 一个圆点 3 字节，混用时光标和选区会偏到 1/3 处。下面这组方法只在
// 布局边界上转换，编辑核心内部始终使用真实值偏移。

func maskText(s string) string {
	if s == "" {
		return ""
	}
	return strings.Repeat(passwordMask, utf8.RuneCountInString(s))
}

func runeByteOffset(s string, n int) int {
	if n <= 0 {
		return 0
	}
	i := 0
	for n > 0 && i < len(s) {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		n--
	}
	return i
}

// setMasked 切换密码显示；布局缓存按文本缓存，切换后必须作废。
func (st *editState) setMasked(masked bool) {
	if st.masked == masked {
		return
	}
	st.masked = masked
	st.dropLayout()
}

// displayText 是布局与绘制实际使用的文本：密码框把每个 rune 换成圆点，
// 组合串保持原样（输入法预编辑仍可见）。
func (st *editState) displayText() string {
	if !st.masked {
		return st.text()
	}
	if st.comp == "" {
		return maskText(st.value)
	}
	at := clampCaret(st.value, st.compAt)
	return maskText(st.value[:at]) + st.comp + maskText(st.value[at:])
}

// displaySpan 返回 value[from:to] 在显示文本里的字节长度。
func (st *editState) displaySpan(from, to int) int {
	from = clampCaret(st.value, from)
	to = clampCaret(st.value, to)
	if to <= from {
		return 0
	}
	if !st.masked {
		return to - from
	}
	return utf8.RuneCountInString(st.value[from:to]) * len(passwordMask)
}

// displayOffset 把「value + 组合串」里的字节偏移换算成显示文本偏移。
func (st *editState) displayOffset(compOff int) int {
	compOff = clampCaret(st.text(), compOff)
	if st.comp == "" {
		return st.displaySpan(0, compOff)
	}
	at := clampCaret(st.value, st.compAt)
	pre := st.displaySpan(0, at)
	switch {
	case compOff <= at:
		return st.displaySpan(0, compOff)
	case compOff <= at+len(st.comp):
		return pre + compOff - at
	default:
		return pre + len(st.comp) + st.displaySpan(at, compOff-len(st.comp))
	}
}

// displayValueOffset 把真实值里的偏移换算到显示文本（组合串占的字节要算进去）。
func (st *editState) displayValueOffset(valueOff int) int {
	valueOff = clampCaret(st.value, valueOff)
	if st.comp == "" {
		return st.displaySpan(0, valueOff)
	}
	at := clampCaret(st.value, st.compAt)
	pre := st.displaySpan(0, at)
	if valueOff <= at {
		return st.displaySpan(0, valueOff)
	}
	return pre + len(st.comp) + st.displaySpan(at, valueOff)
}

// displayCaretOffset 是光标在显示文本里的偏移（组合串内再按候选光标走）。
func (st *editState) displayCaretOffset() int {
	return st.displayOffset(st.caretOffset() + compCursorBytes(st))
}

// valueOffsetFromDisplay 把显示文本偏移换算回真实值的字节偏移。
// 落在组合串里时收在组合串起点——应用层的选区 / 光标只认真实值。
func (st *editState) valueOffsetFromDisplay(displayOff int) int {
	if st.comp == "" {
		return st.unmaskValue(0, displayOff)
	}
	at := clampCaret(st.value, st.compAt)
	pre := st.displaySpan(0, at)
	switch {
	case displayOff <= pre:
		return st.unmaskValue(0, displayOff)
	case displayOff <= pre+len(st.comp):
		return at
	default:
		return st.unmaskValue(at, displayOff-pre-len(st.comp))
	}
}

func (st *editState) unmaskValue(from, displayLen int) int {
	from = clampCaret(st.value, from)
	if displayLen <= 0 {
		return from
	}
	if !st.masked {
		return clampCaret(st.value, from+displayLen)
	}
	return from + runeByteOffset(st.value[from:], displayLen/len(passwordMask))
}
