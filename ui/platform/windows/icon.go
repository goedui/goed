//go:build windows

package windows

import (
	"runtime"
	"unsafe"
)

const (
	iconSmall      = 0
	iconBig        = 1
	lrDefaultColor = 0
)

func createIcon(bits []byte, size int) uintptr {
	if len(bits) == 0 || size <= 0 {
		return 0
	}
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&bits[0])),
		uintptr(len(bits)),
		1,
		0x00030000,
		uintptr(size),
		uintptr(size),
		lrDefaultColor,
	)
	runtime.KeepAlive(bits)
	return h
}

func destroyIcon(h uintptr) {
	if h != 0 {
		procDestroyIcon.Call(h)
	}
}

func (h *host) applyIcons() {
	if h.hwnd == 0 {
		return
	}
	if h.iconSm == 0 {
		h.iconSm = createIcon(h.opts.Icon16, 16)
	}
	if h.iconBg == 0 {
		h.iconBg = createIcon(h.opts.Icon32, 32)
	}
	if h.iconSm != 0 {
		procSendMessageW.Call(h.hwnd, wmSetIcon, iconSmall, h.iconSm)
	}
	if h.iconBg != 0 {
		procSendMessageW.Call(h.hwnd, wmSetIcon, iconBig, h.iconBg)
	}
}

func (h *host) releaseIcons() {
	if h.iconSm != 0 {
		destroyIcon(h.iconSm)
		h.iconSm = 0
	}
	if h.iconBg != 0 {
		destroyIcon(h.iconBg)
		h.iconBg = 0
	}
}
