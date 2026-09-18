//go:build windows

package windows

// hitTest 把客户区像素坐标映射成 WM_NCHITTEST 返回值。
// 边框用于缩放；标题栏右侧三块分别是最小化 / 最大化 / 关闭。
func hitTest(px, py, cw, ch, border, titleH, btnW int32, maximized bool) uintptr {
	if cw <= 0 || ch <= 0 {
		return htClient
	}
	if !maximized && border > 0 {
		left := px < border
		right := px >= cw-border
		top := py < border
		bottom := py >= ch-border
		switch {
		case top && left:
			return htTopLeft
		case top && right:
			return htTopRight
		case bottom && left:
			return htBottomLeft
		case bottom && right:
			return htBottomRight
		case left:
			return htLeft
		case right:
			return htRight
		case top:
			return htTop
		case bottom:
			return htBottom
		}
	}
	if py >= 0 && py < titleH && px >= 0 && px < cw {
		if btnW > 0 {
			if px >= cw-btnW {
				return htClose
			}
			if px >= cw-2*btnW {
				return htMaxButton
			}
			if px >= cw-3*btnW {
				return htMinButton
			}
		}
		return htCaption
	}
	return htClient
}

func hitToCaption(hit uintptr) int {
	switch hit {
	case htMinButton:
		return 1
	case htMaxButton:
		return 2
	case htClose:
		return 3
	default:
		return 0
	}
}
