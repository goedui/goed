package components

import (
	"reflect"
	"testing"
)

func TestSplitLinesPreservesUTF8(t *testing.T) {
	got := splitLines("标题\r\n中文\n末尾")
	want := []string{"标题", "中文", "末尾"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitLines() = %#v, want %#v", got, want)
	}
}

func TestEditorSetTextRoundTripsUTF8(t *testing.T) {
	const input = "高性能的纯 Go 编辑器\n第二行"
	editor := NewEditor()
	editor.SetText(input)
	if got := editor.GetText(); got != input {
		t.Fatalf("GetText() = %q, want %q", got, input)
	}
}

func TestPixelToLineColUsesRenderedTextOrigin(t *testing.T) {
	editor := NewEditor()
	editor.Lines = []string{"abcd"}
	editor.VisibleLines = 1

	// Rendered text starts after the 20px fold gutter and 8px inset.
	if line, col := editor.PixelToLineCol(28, 0); line != 0 || col != 0 {
		t.Fatalf("PixelToLineCol at text origin = (%d, %d), want (0, 0)", line, col)
	}
	if line, col := editor.PixelToLineCol(37, 0); line != 0 || col != 1 {
		t.Fatalf("PixelToLineCol at one glyph advance = (%d, %d), want (0, 1)", line, col)
	}
}
