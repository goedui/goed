package runtime

// 对齐值，对应 CSS align-items / justify-content。写在 Style.Align / Style.Justify 上。
const (
	Start   = "start"
	Center  = "center"
	End     = "end"
	Stretch = "stretch"
)

// overflow 值，对应 CSS overflow / overflow-x / overflow-y。
// 空字符串是 visible。轴上的 OverflowX / OverflowY 盖过 Overflow。
const (
	OverflowVisible = "visible"
	OverflowHidden  = "hidden"
	OverflowAuto    = "auto"
	OverflowScroll  = "scroll"
)

// 定位，对应 CSS position。空字符串是 static。
const (
	PositionRelative = "relative"
	PositionAbsolute = "absolute"
)

// Inset 标记哪条边写过。值为 0 的边必须置位，否则当成 auto。
const (
	InsetTop uint8 = 1 << iota
	InsetRight
	InsetBottom
	InsetLeft
	InsetAll = InsetTop | InsetRight | InsetBottom | InsetLeft
)

// OverflowAxis 解析某一轴的最终值。轴字段非空时盖过 Overflow；都空则 visible。
// horizontal 为 true 时读 OverflowX。
func OverflowAxis(s Style, horizontal bool) string {
	v := s.OverflowY
	if horizontal {
		v = s.OverflowX
	}
	if v == "" {
		v = s.Overflow
	}
	switch v {
	case OverflowHidden, OverflowAuto, OverflowScroll, OverflowVisible:
		return v
	default:
		return OverflowVisible
	}
}

// OverflowVisibleAxis 报告节点在该轴上是否允许内容画出盒子。
// n 为 nil 时按 visible（没有盒子可裁）。
func OverflowVisibleAxis(n *VNode, horizontal bool) bool {
	if n == nil {
		return true
	}
	return OverflowAxis(n.Style, horizontal) == OverflowVisible
}

// OverflowClips 报告该轴是否裁剪（hidden / auto / scroll）。visible 不裁。
func OverflowClips(s Style, horizontal bool) bool {
	return OverflowAxis(s, horizontal) != OverflowVisible
}

// OverflowScrolls 报告该轴是否可滚。auto 与 scroll 都可滚；hidden 只裁。
func OverflowScrolls(s Style, horizontal bool) bool {
	switch OverflowAxis(s, horizontal) {
	case OverflowAuto, OverflowScroll:
		return true
	default:
		return false
	}
}

// IsAbsolute 报告节点是否脱离文档流。
func IsAbsolute(n *VNode) bool {
	return n != nil && n.Style.Position == PositionAbsolute
}

// IsPositioned 报告节点是否是绝对定位的包含块（relative / absolute）。
func IsPositioned(n *VNode) bool {
	if n == nil {
		return false
	}
	return n.Style.Position == PositionRelative || n.Style.Position == PositionAbsolute
}

// InsetSpecified 报告这条边写过。非 0 视为写过；要钉在 0 上必须置 Inset 位。
func InsetSpecified(s Style, bit uint8, v float32) bool {
	return s.Inset&bit != 0 || v != 0
}

// SetParent 记下布局父节点，供绝对定位找包含块。
func SetParent(n, p *VNode) {
	if n != nil {
		n.parent = p
	}
}

// ContainingBlock 返回最近 relative/absolute 祖先的 padding 边。没有则 ok=false。
func ContainingBlock(n *VNode) (x, y, w, h float32, ok bool) {
	if n == nil {
		return 0, 0, 0, 0, false
	}
	for p := n.parent; p != nil; p = p.parent {
		if !IsPositioned(p) {
			continue
		}
		pad := p.Style.Padding
		return p.X + pad, p.Y + pad, p.W - 2*pad, p.H - 2*pad, true
	}
	return 0, 0, 0, 0, false
}
