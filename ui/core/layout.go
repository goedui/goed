package core

// LayoutNode 布局节点
type LayoutNode struct {
	Component Component
	Flex      int32 // 弹性因子（0=固定，>0=弹性）
	Basis     int32 // 基础大小
	MinWidth  int32
	MinHeight int32
	MaxWidth  int32
	MaxHeight int32
}

// Layout 布局接口
type Layout interface {
	Calculate(constraints Constraints) Size
	Arrange(bounds Rect)
}

// Constraints 布局约束
type Constraints struct {
	MinWidth  int32
	MinHeight int32
	MaxWidth  int32
	MaxHeight int32
}

// Size 尺寸
type Size struct {
	Width  int32
	Height int32
}

// FlexDirection 弹性方向
type FlexDirection int

const (
	Row FlexDirection = iota
	Column
)

// JustifyContent 主轴对齐
type JustifyContent int

const (
	JustifyStart JustifyContent = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
	JustifySpaceAround
)

// AlignItems 交叉轴对齐
type AlignItems int

const (
	AlignStart AlignItems = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// FlexLayout Flexbox 布局
type FlexLayout struct {
	Direction      FlexDirection
	JustifyContent JustifyContent
	AlignItems     AlignItems
	Gap            int32
	Children       []LayoutNode
}

func NewFlexLayout(direction FlexDirection) *FlexLayout {
	return &FlexLayout{
		Direction:      direction,
		JustifyContent: JustifyStart,
		AlignItems:     AlignStretch,
		Gap:            0,
	}
}

// Calculate 计算布局大小
func (fl *FlexLayout) Calculate(constraints Constraints) Size {
	if len(fl.Children) == 0 {
		return Size{constraints.MinWidth, constraints.MinHeight}
	}

	var totalWidth, totalHeight int32

	if fl.Direction == Row {
		// 横向布局
		for i, child := range fl.Children {
			bounds := child.Component.GetBounds()
			totalWidth += bounds.W
			if bounds.H > totalHeight {
				totalHeight = bounds.H
			}
			if i < len(fl.Children)-1 {
				totalWidth += fl.Gap
			}
		}
	} else {
		// 纵向布局
		for i, child := range fl.Children {
			bounds := child.Component.GetBounds()
			totalHeight += bounds.H
			if bounds.W > totalWidth {
				totalWidth = bounds.W
			}
			if i < len(fl.Children)-1 {
				totalHeight += fl.Gap
			}
		}
	}

	return Size{totalWidth, totalHeight}
}

// Arrange 排列子组件
func (fl *FlexLayout) Arrange(bounds Rect) {
	if len(fl.Children) == 0 {
		return
	}

	if fl.Direction == Row {
		fl.arrangeRow(bounds)
	} else {
		fl.arrangeColumn(bounds)
	}
}

func (fl *FlexLayout) arrangeRow(bounds Rect) {
	// 计算固定和弹性部分
	var fixedWidth, totalFlex int32
	for _, child := range fl.Children {
		if child.Flex == 0 {
			fixedWidth += child.Component.GetBounds().W
		} else {
			totalFlex += child.Flex
		}
	}

	// 计算间隙
	gapTotal := fl.Gap * int32(len(fl.Children)-1)
	fixedWidth += gapTotal

	// 剩余空间
	remainingWidth := bounds.W - fixedWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	// 排列
	leading, extraGap := int32(0), int32(0)
	if totalFlex == 0 {
		leading, extraGap = justifySpacing(fl.JustifyContent, remainingWidth, len(fl.Children))
	}
	x := bounds.X + leading
	for _, child := range fl.Children {
		childBounds := child.Component.GetBounds()

		// 计算宽度
		var width int32
		if child.Flex == 0 || totalFlex == 0 {
			width = childBounds.W
			if child.Basis > 0 {
				width = child.Basis
			}
		} else {
			width = (remainingWidth * child.Flex) / totalFlex
		}
		width = clampSize(width, child.MinWidth, child.MaxWidth)

		// 计算高度（根据对齐方式）
		height := childBounds.H
		var y int32
		switch fl.AlignItems {
		case AlignStart:
			y = bounds.Y
		case AlignCenter:
			y = bounds.Y + (bounds.H-height)/2
		case AlignEnd:
			y = bounds.Y + bounds.H - height
		case AlignStretch:
			y = bounds.Y
			height = bounds.H
		}
		height = clampSize(height, child.MinHeight, child.MaxHeight)

		// 设置位置
		child.Component.SetBounds(Rect{x, y, width, height})
		x += width + fl.Gap + extraGap
	}
}

func (fl *FlexLayout) arrangeColumn(bounds Rect) {
	// 计算固定和弹性部分
	var fixedHeight, totalFlex int32
	for _, child := range fl.Children {
		if child.Flex == 0 {
			fixedHeight += child.Component.GetBounds().H
		} else {
			totalFlex += child.Flex
		}
	}

	// 计算间隙
	gapTotal := fl.Gap * int32(len(fl.Children)-1)
	fixedHeight += gapTotal

	// 剩余空间
	remainingHeight := bounds.H - fixedHeight
	if remainingHeight < 0 {
		remainingHeight = 0
	}

	// 排列
	leading, extraGap := int32(0), int32(0)
	if totalFlex == 0 {
		leading, extraGap = justifySpacing(fl.JustifyContent, remainingHeight, len(fl.Children))
	}
	y := bounds.Y + leading
	for _, child := range fl.Children {
		childBounds := child.Component.GetBounds()

		// 计算高度
		var height int32
		if child.Flex == 0 || totalFlex == 0 {
			height = childBounds.H
			if child.Basis > 0 {
				height = child.Basis
			}
		} else {
			height = (remainingHeight * child.Flex) / totalFlex
		}
		height = clampSize(height, child.MinHeight, child.MaxHeight)

		// 计算宽度（根据对齐方式）
		width := childBounds.W
		var x int32
		switch fl.AlignItems {
		case AlignStart:
			x = bounds.X
		case AlignCenter:
			x = bounds.X + (bounds.W-width)/2
		case AlignEnd:
			x = bounds.X + bounds.W - width
		case AlignStretch:
			x = bounds.X
			width = bounds.W
		}
		width = clampSize(width, child.MinWidth, child.MaxWidth)

		// 设置位置
		child.Component.SetBounds(Rect{x, y, width, height})
		y += height + fl.Gap + extraGap
	}
}

func justifySpacing(j JustifyContent, free int32, count int) (leading, gap int32) {
	if free <= 0 || count == 0 {
		return
	}
	switch j {
	case JustifyCenter:
		leading = free / 2
	case JustifyEnd:
		leading = free
	case JustifySpaceBetween:
		if count > 1 {
			gap = free / int32(count-1)
		}
	case JustifySpaceAround:
		gap = free / int32(count)
		leading = gap / 2
	}
	return
}

func clampSize(value, min, max int32) int32 {
	if min > 0 && value < min {
		value = min
	}
	if max > 0 && value > max {
		value = max
	}
	if value < 0 {
		return 0
	}
	return value
}
