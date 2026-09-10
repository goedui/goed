package components

import (
	"strings"
	"testing"
)

// benchLines 构造 n 行、每行约 100 字符（含中文）的文档。
func benchLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = strings.Repeat("abcdefg hijklmn opqrst ", 3) + "中文测试注释"
	}
	return lines
}

func BenchmarkGetText10K(b *testing.B) {
	e := NewEditor()
	e.Lines = benchLines(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.GetText()
	}
}

// BenchmarkMaxLineWidthFirstEdit 验证 #8：编辑一行后重算 maxLineWidth
// 应只重测变更行，而非全文档 rune 解码。
func BenchmarkMaxLineWidthAfterEdit(b *testing.B) {
	e := NewEditor()
	e.Lines = benchLines(10000)
	e.calibratedCharW = 9
	e.maxLineWidth() // 建立快照

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Lines[i%len(e.Lines)] += "x" // 改一行
		e.maxLineWidthDirty = true
		e.maxLineWidth()
	}
}

// BenchmarkMaxLineWidthFull 模拟旧实现的全量重算作为对比基线。
func BenchmarkMaxLineWidthFull(b *testing.B) {
	e := NewEditor()
	e.Lines = benchLines(10000)
	e.calibratedCharW = 9

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.widthSnapshotLines = nil // 强制全量
		e.maxLineWidthDirty = true
		e.maxLineWidth()
	}
}

func BenchmarkInsertCharLongLine(b *testing.B) {
	e := NewEditor()
	e.Lines = []string{strings.Repeat("x", 10000)}
	e.CursorLine = 0
	e.CursorCol = 5000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.InsertChar('a')
	}
}
