// Package composables - 组合式函数
// 文本选择功能
package composables

// Selection 文本选择
type Selection struct {
	Start Position
	End   Position
}

// Position 位置
type Position struct {
	Line   int
	Column int
}

// NewSelection 创建选择
func NewSelection(start, end Position) *Selection {
	return &Selection{
		Start: start,
		End:   end,
	}
}

// IsEmpty 是否为空选择
func (s *Selection) IsEmpty() bool {
	return s.Start == s.End
}

// Normalize 规范化（确保 Start <= End）
func (s *Selection) Normalize() *Selection {
	if s.Start.Line > s.End.Line ||
		(s.Start.Line == s.End.Line && s.Start.Column > s.End.Column) {
		return &Selection{
			Start: s.End,
			End:   s.Start,
		}
	}
	return s
}

// Contains 是否包含位置
func (s *Selection) Contains(pos Position) bool {
	norm := s.Normalize()

	// 在起始行之前
	if pos.Line < norm.Start.Line {
		return false
	}

	// 在结束行之后
	if pos.Line > norm.End.Line {
		return false
	}

	// 在起始行上
	if pos.Line == norm.Start.Line && pos.Column < norm.Start.Column {
		return false
	}

	// 在结束行上
	if pos.Line == norm.End.Line && pos.Column > norm.End.Column {
		return false
	}

	return true
}
