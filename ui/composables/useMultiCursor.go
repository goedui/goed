// Package composables - 组合式函数
// 多光标功能
package composables

// Cursor 单个光标
type Cursor struct {
	Position  Position
	Selection *Selection // 可选的选择区域
}

// MultiCursor 多光标管理
type MultiCursor struct {
	Cursors    []Cursor
	PrimaryIdx int  // 主光标索引
	Enabled    bool // 是否启用多光标模式
}

// NewMultiCursor 创建多光标管理器
func NewMultiCursor(initialPos Position) *MultiCursor {
	return &MultiCursor{
		Cursors: []Cursor{
			{Position: initialPos, Selection: nil},
		},
		PrimaryIdx: 0,
		Enabled:    false,
	}
}

// AddCursor 添加新光标
func (mc *MultiCursor) AddCursor(pos Position) {
	// 检查是否已存在相同位置的光标
	for _, cursor := range mc.Cursors {
		if cursor.Position == pos {
			return
		}
	}

	mc.Cursors = append(mc.Cursors, Cursor{
		Position:  pos,
		Selection: nil,
	})
	mc.Enabled = true
}

// RemoveCursor 移除指定索引的光标
func (mc *MultiCursor) RemoveCursor(idx int) {
	if idx < 0 || idx >= len(mc.Cursors) {
		return
	}

	// 不允许删除最后一个光标
	if len(mc.Cursors) <= 1 {
		return
	}

	mc.Cursors = append(mc.Cursors[:idx], mc.Cursors[idx+1:]...)

	// 调整主光标索引
	if mc.PrimaryIdx >= len(mc.Cursors) {
		mc.PrimaryIdx = len(mc.Cursors) - 1
	}

	// 如果只剩一个光标，退出多光标模式
	if len(mc.Cursors) == 1 {
		mc.Enabled = false
	}
}

// GetPrimary 获取主光标
func (mc *MultiCursor) GetPrimary() *Cursor {
	if mc.PrimaryIdx >= 0 && mc.PrimaryIdx < len(mc.Cursors) {
		return &mc.Cursors[mc.PrimaryIdx]
	}
	return nil
}

// GetAll 获取所有光标
func (mc *MultiCursor) GetAll() []Cursor {
	return mc.Cursors
}

// Clear 清除所有次要光标，保留主光标
func (mc *MultiCursor) Clear() {
	if mc.PrimaryIdx >= 0 && mc.PrimaryIdx < len(mc.Cursors) {
		mc.Cursors = []Cursor{mc.Cursors[mc.PrimaryIdx]}
	} else {
		mc.Cursors = []Cursor{{Position: Position{Line: 0, Column: 0}}}
	}
	mc.PrimaryIdx = 0
	mc.Enabled = false
}

// Sort 按位置排序光标
func (mc *MultiCursor) Sort() {
	// 简单冒泡排序
	for i := 0; i < len(mc.Cursors)-1; i++ {
		for j := 0; j < len(mc.Cursors)-i-1; j++ {
			if mc.compareCursors(&mc.Cursors[j], &mc.Cursors[j+1]) > 0 {
				mc.Cursors[j], mc.Cursors[j+1] = mc.Cursors[j+1], mc.Cursors[j]

				// 更新主光标索引
				if mc.PrimaryIdx == j {
					mc.PrimaryIdx = j + 1
				} else if mc.PrimaryIdx == j+1 {
					mc.PrimaryIdx = j
				}
			}
		}
	}
}

// compareCursors 比较两个光标的位置
func (mc *MultiCursor) compareCursors(c1, c2 *Cursor) int {
	if c1.Position.Line < c2.Position.Line {
		return -1
	}
	if c1.Position.Line > c2.Position.Line {
		return 1
	}
	if c1.Position.Column < c2.Position.Column {
		return -1
	}
	if c1.Position.Column > c2.Position.Column {
		return 1
	}
	return 0
}

// MergeDuplicates 合并重叠的光标
func (mc *MultiCursor) MergeDuplicates() {
	if len(mc.Cursors) <= 1 {
		return
	}

	mc.Sort()

	unique := []Cursor{mc.Cursors[0]}
	for i := 1; i < len(mc.Cursors); i++ {
		// 如果当前光标与上一个不同，则保留
		if mc.Cursors[i].Position != unique[len(unique)-1].Position {
			unique = append(unique, mc.Cursors[i])
		}
	}

	mc.Cursors = unique

	// 重新定位主光标
	if mc.PrimaryIdx >= len(mc.Cursors) {
		mc.PrimaryIdx = len(mc.Cursors) - 1
	}

	if len(mc.Cursors) == 1 {
		mc.Enabled = false
	}
}

// UpdateCursor 更新指定光标的位置
func (mc *MultiCursor) UpdateCursor(idx int, pos Position) {
	if idx >= 0 && idx < len(mc.Cursors) {
		mc.Cursors[idx].Position = pos
	}
}

// UpdateSelection 更新指定光标的选择
func (mc *MultiCursor) UpdateSelection(idx int, sel *Selection) {
	if idx >= 0 && idx < len(mc.Cursors) {
		mc.Cursors[idx].Selection = sel
	}
}
