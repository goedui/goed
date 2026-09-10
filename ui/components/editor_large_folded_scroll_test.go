package components

import (
	"strings"
	"testing"
	"time"
)

// TestLargeFileScrollWithFoldRegions 复现大 js 文件无法滚动：
// 编辑器加载时会 UpdateFoldRegions("auto")（editor.App.go:144），
// 大文件产生数以万计的折叠区域；IsFolded 对每行线性扫描全部区域，
// visibleLineMap 退化为 O(行数×区域数)，滚动时每次重建都要数秒。
func TestLargeFileScrollWithFoldRegions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-file perf test in short mode")
	}
	e := NewEditor()
	const rows = 20000
	// 生成嵌套花括号的 js 风格内容，产生大量折叠区域。
	e.Lines = make([]string, rows)
	for i := 0; i < rows; i += 2 {
		e.Lines[i] = "function f" + strings.Repeat("x", i%7) + "(a) {"
		if i+1 < rows {
			e.Lines[i+1] = "  return {a: a, b: 1, c: \"str\"}; }"
		}
	}
	e.FilePath = "big.js"
	e.VisibleLines = 40
	e.LineHeight = 24

	start := time.Now()
	e.UpdateFoldRegions("auto")
	t.Logf("UpdateFoldRegions took %v, regions=%d", time.Since(start), len(e.FoldingManager.Regions))

	start = time.Now()
	e.HandleMouseWheel(-120, false)
	d := time.Since(start)
	t.Logf("first wheel took %v, ScrollLine=%d", d, e.ScrollLine)
	if d > 300*time.Millisecond {
		t.Fatalf("first wheel after fold detection took %v; scroll is frozen for large files", d)
	}
	if e.ScrollLine == 0 {
		t.Fatal("wheel did not scroll")
	}

	start = time.Now()
	for i := 0; i < 30; i++ {
		e.HandleMouseWheel(-120, false)
	}
	if d := time.Since(start); d > 300*time.Millisecond {
		t.Fatalf("30 wheel events took %v; scroll effectively unusable", d)
	}
}
