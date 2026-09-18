package theme

import "github.com/goedui/goed/ui/runtime"

// hex 解析 VS Code 色值：#RGB、#RRGGBB、#RRGGBBAA。
func hex(s string) runtime.Color {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	nibble := func(c byte) uint32 {
		switch {
		case c >= '0' && c <= '9':
			return uint32(c - '0')
		case c >= 'a' && c <= 'f':
			return uint32(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			return uint32(c - 'A' + 10)
		default:
			return 0
		}
	}
	var n uint32
	for i := 0; i < len(s); i++ {
		n = n<<4 | nibble(s[i])
	}
	switch len(s) {
	case 3:
		r := uint8(((n >> 8) & 0xF) * 0x11)
		g := uint8(((n >> 4) & 0xF) * 0x11)
		b := uint8((n & 0xF) * 0x11)
		return runtime.RGB(r, g, b)
	case 6:
		return runtime.RGB(uint8(n>>16), uint8(n>>8), uint8(n))
	case 8:
		return runtime.RGBA(uint8(n>>24), uint8(n>>16), uint8(n>>8), uint8(n))
	default:
		return runtime.Color{}
	}
}
