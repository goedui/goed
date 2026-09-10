package components

import (
	"strings"
	"testing"
	"time"

	"github.com/goedui/goed/ui/syntax"
)

// TestLargeFileScrollInteractions 复现「打开 1M+ 的 js 文件无法滚动」：
// 用 3 万行（约 1MB）js 内容初始化编辑器后依次触发滚轮、滚动条几何、
// minmap 命中与 Render 的关键路径，任何一步耗时超过阈值即失败。
func TestLargeFileScrollInteractions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-file perf test in short mode")
	}
	e := NewEditor()
	const rows = 30000
	line := strings.Repeat("fn a(", 40)
	e.Lines = make([]string, rows)
	for i := range e.Lines {
		e.Lines[i] = line
	}
	e.FilePath = "big.js"
	e.VisibleLines = 40
	e.LineHeight = 24
	e.visibleMapDirty = true

	// 1. 滚轮：单次事件应在几十毫秒内生效（此前会全量 tokenize + maxLineWidth）。
	start := time.Now()
	e.HandleMouseWheel(-120, false)
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("first wheel took %v; scrolling a large file should be fast", d)
	}
	if e.ScrollLine == 0 {
		t.Fatal("wheel did not scroll the large document")
	}

	// 2. 连续滚轮（模拟滚动一屏）。
	start = time.Now()
	for i := 0; i < 40; i++ {
		e.HandleMouseWheel(-120, false)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("40 wheel events took %v; scroll is effectively frozen", d)
	}

	// 3. 语法高亮：全量 tokenize 一次性成本应有界。
	start = time.Now()
	tokens := syntax.Highlight(e.Lines, e.FilePath)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Highlight of %d lines took %v", rows, d)
	}
	if len(tokens) != rows {
		t.Fatalf("Highlight returned %d token rows, want %d", len(tokens), rows)
	}

	// 4. 滚动条几何同步（Render 前置路径）。
	if !e.updateScrollBarGeometry(0, 0, 1200, 960, 20, 40) {
		t.Fatal("scrollbar should be active for a large file")
	}

	// 5. visibleLineMap 缓存后重复访问必须 O(1)。
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = len(e.visibleLineMap())
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("visibleLineMap not cached: 1000 calls took %v", d)
	}
}
