// Package composables - 组合式函数
// 撤销/重做功能
package composables

// History 撤销/重做管理
type History struct {
	undoStack []Operation
	redoStack []Operation
	maxSize   int
}

// Operation 操作记录
type Operation struct {
	Type     OperationType
	Position int
	OldText  string
	NewText  string
}

// OperationType 操作类型
type OperationType int

const (
	OpInsert OperationType = iota
	OpDelete
	OpReplace
)

// NewHistory 创建历史记录管理器
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &History{
		undoStack: make([]Operation, 0, maxSize),
		redoStack: make([]Operation, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Record 记录操作
func (h *History) Record(op Operation) {
	// 清空 redo 栈
	h.redoStack = h.redoStack[:0]

	// 添加到 undo 栈
	h.undoStack = append(h.undoStack, op)

	// 限制栈大小
	if len(h.undoStack) > h.maxSize {
		h.undoStack = h.undoStack[1:]
	}
}

// Undo 撤销操作
func (h *History) Undo() *Operation {
	if len(h.undoStack) == 0 {
		return nil
	}

	// 弹出操作
	op := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]

	// 压入 redo 栈
	h.redoStack = append(h.redoStack, op)

	return &op
}

// Redo 重做操作
func (h *History) Redo() *Operation {
	if len(h.redoStack) == 0 {
		return nil
	}

	// 弹出操作
	op := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]

	// 压入 undo 栈
	h.undoStack = append(h.undoStack, op)

	return &op
}

// CanUndo 是否可以撤销
func (h *History) CanUndo() bool {
	return len(h.undoStack) > 0
}

// CanRedo 是否可以重做
func (h *History) CanRedo() bool {
	return len(h.redoStack) > 0
}

// Clear 清空历史
func (h *History) Clear() {
	h.undoStack = h.undoStack[:0]
	h.redoStack = h.redoStack[:0]
}
