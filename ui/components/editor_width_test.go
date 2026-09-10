package components

import "testing"

func TestIsWideRune(t *testing.T) {
	wide := []rune{'中', '文', '，', '。', '（', '）', 'ア', '한', '；'}
	for _, r := range wide {
		if !isWideRune(r) {
			t.Fatalf("rune %q (U+%04X) should be wide", r, r)
		}
	}
	narrow := []rune{'a', 'Z', '0', ' ', '_', '.', '{'}
	for _, r := range narrow {
		if isWideRune(r) {
			t.Fatalf("rune %q (U+%04X) should be narrow", r, r)
		}
	}
}

func TestCalcDisplayWidthMixedLine(t *testing.T) {
	e := NewEditor()
	// "中文 abc"：2 个全角（2×2）+ 1 空格 + 3 半角 = 8 单位宽。
	line := "中文 abc"
	if got := e.CalcDisplayWidth(line, 8); got != 8*e.CharWidth {
		t.Fatalf("width = %d, want %d", got, 8*e.CharWidth)
	}
	// 光标在第 1 个中文后 = 2 单位宽。
	if got := e.CalcDisplayWidth(line, 1); got != 2*e.CharWidth {
		t.Fatalf("width after 1 wide char = %d, want %d", got, 2*e.CharWidth)
	}
	// 校准值优先于估算 CharWidth。
	e.calibratedCharW = 8
	if got := e.CalcDisplayWidth(line, 8); got != 8*8 {
		t.Fatalf("calibrated width = %d, want %d", got, 64)
	}
}

func TestPixelToLineColWideChars(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"中文 abc"}
	e.VisibleLines = 3
	e.visibleMapDirty = true
	e.LineHeight = 24
	e.calibratedCharW = 8 // 半角 8px、全角 16px
	// PixelToLineCol 扣除 gutter(20) + 边距(8)；x=28 对应行首。
	origin := int32(e.FoldGutter.Width + 8)

	cases := []struct {
		pixelX  int32 // 相对行首的像素偏移
		wantCol int
	}{
		{0, 0},   // "中" 左半边
		{7, 0},   // "中" 右半边内 → 仍定位 0（<w/2 判定）
		{8, 1},   // 进入 "文"
		{28, 2},  // 空格左半边 → 空格
		{36, 3},  // 'a' 左半边 → 'a'
		{64, 6},  // 行尾（'c' 之后，共 6 rune）
		{200, 6}, // 超出行宽 → 行尾
	}
	for _, c := range cases {
		gotLine, gotCol := e.PixelToLineCol(origin+c.pixelX, 12)
		if gotLine != 0 || gotCol != c.wantCol {
			t.Fatalf("PixelToLineCol(%d) = %d:%d, want 0:%d", c.pixelX, gotLine, gotCol, c.wantCol)
		}
	}
}

func TestPixelToLineColRoundTrip(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"混合 line 中英文"}
	e.VisibleLines = 3
	e.visibleMapDirty = true
	e.LineHeight = 24
	e.calibratedCharW = 8
	origin := int32(e.FoldGutter.Width + 8)

	line := e.Lines[0]
	// 每一列的正向宽度 ↔ 反向命中必须一致（列 → X → 列）。
	for col := 0; col <= len([]rune(line)); col++ {
		x := e.CalcDisplayWidth(line, col)
		gotLine, gotCol := e.PixelToLineCol(origin+x, 12)
		if gotLine != 0 || gotCol != col {
			t.Fatalf("round trip col %d: PixelToLineCol = %d:%d", col, gotLine, gotCol)
		}
	}
}
