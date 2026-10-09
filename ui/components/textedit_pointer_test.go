// 稀疏布局（> maxLayoutRun 个 rune 的 visual / ≥256KB 的 large）下的指针落点。
//
// 这一族此前没人守，踩过两个同源的坑：
//
//   - pointAtXY 稀疏分支把「行内像素横坐标」当字节列（col := int(x)）：点行
//     中间会把光标推到 start+像素数、再被钳到行尾 —— 大文档里点哪儿光标都跳
//     行尾（md-previewer 打开 2730 行文档时光标总跑到行尾就是它）。
//   - 稀疏分支的 layoutPoint.x 返回「字节差」，而拖选横向滚动的钳制
//     （maxOff := end.x - viewW）与 followCaretX 全按像素用。
//
// 现在两个分支统一像素口径，这里把口径钉死：点击按像素就近落 rune 边界，
// layoutPoint.x 一律是像素宽。
package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
)

// mockCharW 是 mock 后端下 13px 字号的单字节 rune 宽（= 字号 * 0.5，与
// input_test.go 的点击用例同一口径）。CJK 在 mock 下按 3 字节计宽。
const mockCharW = float32(13) * 0.5

// sparseDoc 造一份超过 maxLayoutRun 的文档（走 buildVisualLines 稀疏布局），
// 第 3 行用中英混排，检验宽字符的像素换算。
func sparseDoc() string {
	var b strings.Builder
	for i := 0; i < 800; i++ {
		if i == 3 {
			b.WriteString("\u4f60\u597dabc\n")
			continue
		}
		fmt.Fprintf(&b, "line%d\n", i)
	}
	return b.String()
}

// TestPointerClickOnSparseLayoutLandsAtPixelColumn 点击落点按像素就近换算：
// 点字符右半边光标落在它后面，宽字符（CJK）按实际宽算而不是按字节数。
func TestPointerClickOnSparseLayoutLandsAtPixelColumn(t *testing.T) {
	n := paintArea(t, "tk-sparse-click", sparseDoc())
	defer renderer.SetPointer(0, 0, false)
	lineH := stateFor(n).layout.lineH
	wide := mockCharW * 3 // 「你」「好」在 mock 下各 3 字节宽

	// "line5" 行点第 3 个字符 'n' 的右半 → 光标落在 (5, 3)。
	y := n.Y + inputTopPad + 5.5*lineH
	renderer.SetPointer(n.X+inputPad+2.6*mockCharW, y, true)
	beginPointerSelect(n)
	assertCaret(t, n, 5, 3)

	// 「你好abc」行点「好」的右半 → (3, 2)：两个宽字符占 2 列而不是 6 列。
	y = n.Y + inputTopPad + 3.5*lineH
	renderer.SetPointer(n.X+inputPad+wide+0.75*wide, y, true)
	beginPointerSelect(n)
	assertCaret(t, n, 3, 2)

	// 点到行外右侧钳到行末，不能越到下一行。
	renderer.SetPointer(n.X+inputPad+1000, y, true)
	beginPointerSelect(n)
	assertCaret(t, n, 3, 5) // 「你好abc」行末
}

// TestPointerClickOnWrappedSparseLineLandsOnThatLine 折行出来的可视续行按
// 该行行首换算列（md-previewer 的长接口行正是折行场景）：像素不能加到整个
// 逻辑行的字节起点上。
func TestPointerClickOnWrappedSparseLineLandsOnThatLine(t *testing.T) {
	var doc strings.Builder
	for i := 0; i < 800; i++ {
		if i == 2 {
			doc.WriteString(strings.Repeat("x", 200))
			doc.WriteString("\n")
			continue
		}
		fmt.Fprintf(&doc, "line%d\n", i)
	}
	n := paintArea(t, "tk-sparse-wrap", doc.String())
	defer renderer.SetPointer(0, 0, false)
	perLine := int((n.W - 2*inputPad) / mockCharW)
	if perLine >= 200 || perLine < 4 {
		t.Fatalf("样本应折成多行：每可视行 %d 字符", perLine)
	}

	// 可视行 3 是「x…」逻辑行的第 2 个可视续行（可视行 0/1 是 line0/line1）。
	y := n.Y + inputTopPad + 3.5*stateFor(n).layout.lineH
	renderer.SetPointer(n.X+inputPad+2.6*mockCharW, y, true)
	beginPointerSelect(n)
	assertCaret(t, n, 3, 3)
}

// TestSparseLayoutPointsReportPixelX 稀疏布局的 layoutPoint.x 与全量 points
// 同口径（像素宽）：拖选横向滚动的钳制与 followCaretX 都把它当像素用。
func TestSparseLayoutPointsReportPixelX(t *testing.T) {
	n := paintArea(t, "tk-sparse-px", sparseDoc())
	lay := stateFor(n).layout
	wide := mockCharW * 3

	start := lay.lineStart(3)
	end := lay.lineEnd(3)
	if x := lay.pointAt(start).x; x != 0 {
		t.Fatalf("行首 x = %.1f，期望 0", x)
	}
	if x := lay.pointAt(start + 6).x; x != 2*wide {
		t.Fatalf("「你好」后 x = %.1f，期望 %.1f", x, 2*wide)
	}
	if want := 2*wide + 3*mockCharW; lay.pointAtLineEnd(3).x != want {
		t.Fatalf("行末 x = %.1f，期望像素宽 %.1f", lay.pointAtLineEnd(3).x, want)
	}
	if x := lay.pointAt(end).x; x != 2*wide+3*mockCharW {
		t.Fatalf("pointAt(行末).x = %.1f，期望与行末一致", x)
	}
}
