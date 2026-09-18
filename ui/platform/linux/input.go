//go:build linux

package linux

// X11 KeySym → 引擎 VK_*（与 runtime/event.go / Windows 编号一致）。
const (
	xkBackSpace = 0xff08
	xkTab       = 0xff09
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
	xkDelete    = 0xffff
	xkHome      = 0xff50
	xkLeft      = 0xff51
	xkUp        = 0xff52
	xkRight     = 0xff53
	xkDown      = 0xff54
	xkPageUp    = 0xff55
	xkPageDown  = 0xff56
	xkEnd       = 0xff57
	xkF11       = 0xffc8
	xkShiftL    = 0xffe1
	xkShiftR    = 0xffe2
	xkControlL  = 0xffe3
	xkControlR  = 0xffe4
	xkAltL      = 0xffe9
	xkAltR      = 0xffea
	xkSpace     = 0x0020
)

const (
	vkBackspace = 0x08
	vkTab       = 0x09
	vkEnter     = 0x0D
	vkShift     = 0x10
	vkControl   = 0x11
	vkEscape    = 0x1B
	vkSpace     = 0x20
	vkEnd       = 0x23
	vkHome      = 0x24
	vkLeft      = 0x25
	vkUp        = 0x26
	vkRight     = 0x27
	vkDown      = 0x28
	vkPageUp    = 0x21
	vkPageDown  = 0x22
	vkDelete    = 0x2E
	vkF11       = 0x7A
)

const (
	shiftMask   = 1 << 0
	controlMask = 1 << 2
	mod1Mask    = 1 << 3
)

func mapKeySym(sym uint64) (key int, char rune) {
	switch sym {
	case xkBackSpace:
		return vkBackspace, 0
	case xkTab:
		return vkTab, 0
	case xkReturn:
		return vkEnter, 0
	case xkEscape:
		return vkEscape, 0
	case xkDelete:
		return vkDelete, 0
	case xkHome:
		return vkHome, 0
	case xkLeft:
		return vkLeft, 0
	case xkUp:
		return vkUp, 0
	case xkRight:
		return vkRight, 0
	case xkDown:
		return vkDown, 0
	case xkPageUp:
		return vkPageUp, 0
	case xkPageDown:
		return vkPageDown, 0
	case xkEnd:
		return vkEnd, 0
	case xkF11:
		return vkF11, 0
	case xkShiftL, xkShiftR:
		return vkShift, 0
	case xkControlL, xkControlR:
		return vkControl, 0
	case xkSpace:
		return vkSpace, ' '
	}
	if sym >= 'a' && sym <= 'z' {
		return int(sym - 'a' + 'A'), rune(sym)
	}
	if sym >= 'A' && sym <= 'Z' {
		return int(sym), rune(sym)
	}
	if sym >= 0x20 && sym <= 0xff {
		return 0, rune(sym)
	}
	return 0, 0
}
