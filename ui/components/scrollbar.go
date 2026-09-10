package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

type ScrollBar struct {
	X, Y        int32
	Width       int32
	Height      int32
	Position    int32
	Total       int32
	Visible     int32
	Orientation Orientation

	dragging   bool
	dragStart  int32
	posStart   int32
	thumbHover bool
}

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

func NewVScrollBar(x, y, height int32) *ScrollBar {
	return &ScrollBar{
		X:           x,
		Y:           y,
		Width:       16,
		Height:      height,
		Orientation: Vertical,
		Position:    0,
		Total:       100,
		Visible:     20,
	}
}

// NewHScrollBar 创建横向滚动条。
func NewHScrollBar(x, y, width int32) *ScrollBar {
	return &ScrollBar{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      14,
		Orientation: Horizontal,
		Position:    0,
		Total:       100,
		Visible:     20,
	}
}

// length 返回滚动主轴的像素长度（横向为宽，纵向为高）。
func (s *ScrollBar) length() int32 {
	if s.Orientation == Horizontal {
		return s.Width
	}
	return s.Height
}

// origin 返回主轴原点坐标（横向为 X，纵向为 Y）。
func (s *ScrollBar) origin() int32 {
	if s.Orientation == Horizontal {
		return s.X
	}
	return s.Y
}

// axisPos 取事件坐标在主轴上的分量。
func (s *ScrollBar) axisPos(x, y int32) int32 {
	if s.Orientation == Horizontal {
		return x
	}
	return y
}

// thumbMetrics 计算 thumb 尺寸与主轴位置。内容未超出可视范围时返回 maxPos=0。
func (s *ScrollBar) thumbMetrics() (thumbSize, maxPos, thumbPos int32) {
	thumbSize = (s.Visible * s.length()) / s.Total
	if thumbSize < 20 {
		thumbSize = 20
	}
	maxPos = s.Total - s.Visible
	if maxPos <= 0 || s.length()-thumbSize <= 0 {
		return thumbSize, 0, 0
	}
	thumbPos = (s.Position * (s.length() - thumbSize)) / maxPos
	return
}

func (s *ScrollBar) Render(ctx core.DrawingContext) {
	th := theme.GetTheme()
	// Draw track
	ctx.DrawRect(s.X, s.Y, s.Width, s.Height, th.ScrollTrackBg, true)

	// Calculate thumb
	if s.Total <= s.Visible {
		return
	}

	thumbSize, _, thumbPos := s.thumbMetrics()

	thumbColor := th.ScrollThumbBg
	if s.thumbHover || s.dragging {
		thumbColor = th.ScrollThumbHover
	}

	if s.Orientation == Horizontal {
		ctx.DrawRoundedRect(
			s.X+thumbPos, s.Y+2,
			thumbSize, s.Height-4,
			4, thumbColor, true,
		)
	} else {
		ctx.DrawRoundedRect(
			s.X+2, s.Y+thumbPos,
			s.Width-4, thumbSize,
			4, thumbColor, true,
		)
	}
}

func (s *ScrollBar) HandleMouseMove(x, y int32) {
	if s.Total <= s.Visible || s.length() <= 0 {
		s.thumbHover = false
		return
	}
	pos := s.axisPos(x, y)
	thumbSize, maxPos, thumbPos := s.thumbMetrics()

	if s.dragging {
		delta := pos - s.dragStart
		newPos := s.posStart + (delta*maxPos)/(s.length()-thumbSize)
		if newPos < 0 {
			newPos = 0
		}
		if newPos > maxPos {
			newPos = maxPos
		}
		s.Position = newPos
	} else {
		// Check hover: 主轴落在 thumb 上，且非主轴分量仍在条带内。
		inCross := true
		if s.Orientation == Horizontal {
			inCross = y >= s.Y && y < s.Y+s.Height
		} else {
			inCross = x >= s.X && x < s.X+s.Width
		}
		s.thumbHover = inCross &&
			pos >= s.origin()+thumbPos && pos < s.origin()+thumbPos+thumbSize
	}
}

func (s *ScrollBar) HandleMouseDown(x, y int32) bool {
	if !s.Contains(x, y) {
		return false
	}
	if s.Total <= s.Visible || s.length() <= 0 {
		return true
	}

	thumbSize, maxPos, thumbPos := s.thumbMetrics()
	pos := s.axisPos(x, y)

	// Check if clicking on thumb
	if pos >= s.origin()+thumbPos && pos < s.origin()+thumbPos+thumbSize {
		s.dragging = true
		s.dragStart = pos
		s.posStart = s.Position
		return true
	}

	// Click on track - jump to position (thumb 中心对齐点击点)
	clickPos := pos - s.origin() - thumbSize/2
	newPos := int32(0)
	if s.length()-thumbSize > 0 {
		newPos = (clickPos * maxPos) / (s.length() - thumbSize)
	}
	if newPos < 0 {
		newPos = 0
	}
	if newPos > maxPos {
		newPos = maxPos
	}
	s.Position = newPos
	// 轨道点击后按住可继续拖动。
	s.dragging = true
	s.dragStart = pos
	s.posStart = s.Position

	return true
}

func (s *ScrollBar) HandleMouseUp(x, y int32) bool {
	if s.dragging {
		s.dragging = false
		return true
	}
	return false
}

func (s *ScrollBar) Contains(x, y int32) bool {
	return x >= s.X && x < s.X+s.Width &&
		y >= s.Y && y < s.Y+s.Height
}

func (s *ScrollBar) Scroll(delta int32) {
	newPos := s.Position + delta
	maxPos := s.Total - s.Visible
	if newPos < 0 {
		newPos = 0
	}
	if newPos > maxPos {
		newPos = maxPos
	}
	s.Position = newPos
}
