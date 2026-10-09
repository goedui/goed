//go:build windows

package windows

// hitTest 把客户区像素坐标映射成 WM_NCHITTEST 返回值。
// 边框用于缩放；标题栏右侧三块分别是最小化 / 最大化 / 关闭。
//
// clientHit 与 dpi 是为「应用自绘标题栏」准备的：顶部那条默认整条都是标题栏，
// 应用把标签条画在那里时点击就全被 HTCAPTION 吃掉了。clientHit 非 nil 时，
// 系统按钮之外的每个点都先问它一句「这里是不是应用自己的控件」，是就返回
// htClient 交回客户区。nil 时这条分支不生效，行为与以前完全一致。
func hitTest(px, py, cw, ch, border, titleH, btnW int32, maximized bool,
	clientHit func(x, y float32) bool, dpi float32) uintptr {
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
		if clientHit != nil {
			if dpi <= 0 {
				dpi = 96
			}
			if clientHit(float32(px)*96/dpi, float32(py)*96/dpi) {
				return htClient
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
