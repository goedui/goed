//go:build linux

package linux

// 命中区，几何对齐 Windows 自定义标题栏：
// 未最大化时边缘用于缩放；标题栏右侧三块是最小化 / 最大化 / 关闭。
const (
	netMoveResizeSizeTopLeft     = 0
	netMoveResizeSizeTop         = 1
	netMoveResizeSizeTopRight    = 2
	netMoveResizeSizeRight       = 3
	netMoveResizeSizeBottomRight = 4
	netMoveResizeSizeBottom      = 5
	netMoveResizeSizeBottomLeft  = 6
	netMoveResizeSizeLeft        = 7
	netMoveResizeMove            = 8
)

const (
	hitClient = iota + 1
	hitCaption
	hitMin
	hitMax
	hitClose
	hitLeft
	hitRight
	hitTop
	hitBottom
	hitTopLeft
	hitTopRight
	hitBottomLeft
	hitBottomRight
)

func hitTest(px, py, cw, ch, border, titleH, btnW int, maximized bool) int {
	if cw <= 0 || ch <= 0 {
		return hitClient
	}
	if !maximized && border > 0 {
		left := px < border
		right := px >= cw-border
		top := py < border
		bottom := py >= ch-border
		switch {
		case top && left:
			return hitTopLeft
		case top && right:
			return hitTopRight
		case bottom && left:
			return hitBottomLeft
		case bottom && right:
			return hitBottomRight
		case left:
			return hitLeft
		case right:
			return hitRight
		case top:
			return hitTop
		case bottom:
			return hitBottom
		}
	}
	if py >= 0 && py < titleH && px >= 0 && px < cw {
		if btnW > 0 {
			if px >= cw-btnW {
				return hitClose
			}
			if px >= cw-2*btnW {
				return hitMax
			}
			if px >= cw-3*btnW {
				return hitMin
			}
		}
		return hitCaption
	}
	return hitClient
}

func hitToCaption(hit int) int {
	switch hit {
	case hitMin:
		return 1
	case hitMax:
		return 2
	case hitClose:
		return 3
	default:
		return 0
	}
}

func hitToMoveResize(hit int) int {
	switch hit {
	case hitTopLeft:
		return netMoveResizeSizeTopLeft
	case hitTop:
		return netMoveResizeSizeTop
	case hitTopRight:
		return netMoveResizeSizeTopRight
	case hitRight:
		return netMoveResizeSizeRight
	case hitBottomRight:
		return netMoveResizeSizeBottomRight
	case hitBottom:
		return netMoveResizeSizeBottom
	case hitBottomLeft:
		return netMoveResizeSizeBottomLeft
	case hitLeft:
		return netMoveResizeSizeLeft
	case hitCaption:
		return netMoveResizeMove
	default:
		return -1
	}
}
