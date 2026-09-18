package windows

import (
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// 输入法（IMM32）支持。
//
// 自定义绘制的文本框必须自己处理 WM_IME_* 消息，否则会出现三个典型毛病：
//   - 拼音/日文等预编辑串看不到：系统把组合窗画在窗口左上角，位置完全不对；
//   - 候选窗不跟着光标走；
//   - 上屏字符丢失或者重复。
//
// 这里的做法：
//   - 预编辑串从 IME 上下文读出来交给应用自己画（应用消费了就不再让系统画组合窗）；
//   - 每次组合变化都用 ImmSetCompositionWindow(CFS_POINT) 把候选窗钉在光标右下角；
//   - 上屏（GCS_RESULTSTR）不自己插入，交给 DefWindowProc 继续翻成 WM_IME_CHAR → WM_CHAR，
//     这样走的是引擎已有的 WM_CHAR 通路，不会出现「自己插一次、系统再送一次」的重复。
const (
	wmImeSetContext       = 0x0281
	wmImeStartComposition = 0x010D
	wmImeEndComposition   = 0x010E
	wmImeComposition      = 0x010F

	gcsCompStr   = 0x0008
	gcsCursorPos = 0x0080
	gcsResultStr = 0x0800

	cfsPoint = 0x0002

	iscShowUICompositionWindow = 0x0008
)

// compositionForm 对应 Win32 COMPOSITIONFORM。
type compositionForm struct {
	dwStyle      uint32
	ptCurrentPos point
	rcArea       rect
}

var (
	modImm32                     = syscall.NewLazyDLL("imm32.dll")
	procImmGetContext            = modImm32.NewProc("ImmGetContext")
	procImmReleaseContext        = modImm32.NewProc("ImmReleaseContext")
	procImmGetCompositionStringW = modImm32.NewProc("ImmGetCompositionStringW")
	procImmSetCompositionWindow  = modImm32.NewProc("ImmSetCompositionWindow")
)

// immContext 取窗口的输入法上下文，用完要 immRelease。
func immContext(hwnd uintptr) uintptr {
	if hwnd == 0 {
		return 0
	}
	himc, _, _ := procImmGetContext.Call(hwnd)
	return himc
}

func immRelease(hwnd, himc uintptr) {
	if himc != 0 {
		procImmReleaseContext.Call(hwnd, himc)
	}
}

// immCompositionString 读取组合串（GCS_COMPSTR）。
func immCompositionString(himc uintptr) string {
	if himc == 0 {
		return ""
	}
	size, _, _ := procImmGetCompositionStringW.Call(himc, uintptr(gcsCompStr), 0, 0)
	n := int32(size)
	if n <= 0 {
		return ""
	}
	buf := make([]uint16, n/2+1)
	procImmGetCompositionStringW.Call(
		himc,
		uintptr(gcsCompStr),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(n),
	)
	runtime.KeepAlive(buf)
	used := n / 2
	for used > 0 && buf[used-1] == 0 {
		used--
	}
	return string(utf16.Decode(buf[:used]))
}

// immCompositionCursor 读取组合串内的光标位置（GCS_CURSORPOS，单位：字符）。
func immCompositionCursor(himc uintptr) int {
	if himc == 0 {
		return 0
	}
	pos, _, _ := procImmGetCompositionStringW.Call(himc, uintptr(gcsCursorPos), 0, 0)
	if int32(pos) < 0 {
		return 0
	}
	return int(pos)
}

// immPlaceComposition 把候选窗贴到客户区物理坐标 (x, y)。
// 用 CFS_POINT 时系统只显示候选窗、不画内联组合串，内联部分由应用绘制。
func immPlaceComposition(himc uintptr, x, y int32) {
	if himc == 0 {
		return
	}
	cf := compositionForm{
		dwStyle:      cfsPoint,
		ptCurrentPos: point{x: x, y: y},
	}
	procImmSetCompositionWindow.Call(himc, uintptr(unsafe.Pointer(&cf)))
	runtime.KeepAlive(cf)
}

// syncComposition 处理 WM_IME_STARTCOMPOSITION / WM_IME_COMPOSITION。
// 返回 true 表示这条消息由我们消化掉（不交给 DefWindowProc）。
func (h *host) syncComposition(lParam uintptr) bool {
	himc := immContext(h.hwnd)
	if himc == 0 {
		return false
	}
	defer immRelease(h.hwnd, himc)
	h.placeCompositionAtCaret(himc)

	// 上屏：清掉本地预编辑显示，剩下交给 DefWindowProc 翻成 WM_CHAR。
	if lParam&gcsResultStr != 0 {
		h.notifyComposition("", 0)
		h.requestFrame()
		return false
	}
	text := immCompositionString(himc)
	cursor := immCompositionCursor(himc)
	handled := h.notifyComposition(text, cursor)
	h.requestFrame()
	return handled
}

// placeCompositionAtCaret 把输入法候选窗移到当前光标右下角。
func (h *host) placeCompositionAtCaret(himc uintptr) {
	if h.opts.Caret == nil {
		return
	}
	cx, cy, _, ch, ok := h.opts.Caret()
	if !ok {
		return
	}
	scale := h.dpi / 96
	if scale <= 0 {
		scale = 1
	}
	immPlaceComposition(himc, int32(cx*scale), int32((cy+ch)*scale))
}

// notifyComposition 把组合串交给应用，返回应用是否消费。
func (h *host) notifyComposition(text string, cursor int) bool {
	if h.opts.Composition == nil {
		return false
	}
	return h.opts.Composition(text, cursor)
}

// endComposition 处理 WM_IME_ENDCOMPOSITION：清掉预编辑显示。
func (h *host) endComposition() {
	h.notifyComposition("", 0)
	h.requestFrame()
}
