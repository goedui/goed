package components

import (
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestRichTextNaturalWidthAndCJKWrapping(t *testing.T) {
	ctx := &mockContext{w: 500, h: 600}
	n := RichText(runtime.Props{}, TextRun{Text: "Chinese 中文 identifierLongerThanTheLine"})
	w, h := measureRichText(ctx, n, 0, 0, renderer.TextStyle{FontSize: 13}, theme.Dark)
	if w <= 100 || h > 25 {
		t.Fatalf("natural paragraph measured %gx%g", w, h)
	}
	_, wrapped := measureRichText(ctx, n, 80, 0, renderer.TextStyle{FontSize: 13}, theme.Dark)
	if wrapped <= h {
		t.Fatal("narrow paragraph did not wrap")
	}
	layoutRichText(ctx, n, 20, 30, 80, wrapped, renderer.TextStyle{FontSize: 13}, theme.Dark)
	fragments := n.Attrs["fragments"].([]richFragment)
	var text strings.Builder
	for _, f := range fragments {
		text.WriteString(f.text)
		if f.x+f.w > 80.1 {
			t.Fatalf("fragment %q runs past paragraph width", f.text)
		}
		if f.y+f.h > wrapped+.1 {
			t.Fatalf("fragment %q runs past measured height", f.text)
		}
	}
	if text.String() != n.Text {
		t.Fatalf("wrapping lost text: %q", text.String())
	}
}

func TestRichTextLinksHaveSeparateHitAreasAfterWrap(t *testing.T) {
	clicked := 0
	n := RichText(runtime.Props{Style: runtime.Style{Width: 110}},
		TextRun{Text: "plain "}, TextRun{Text: "link-long-enough-to-wrap", Bold: true, Underline: true, OnClick: func() { clicked++ }}, TextRun{Text: " tail"})
	ctx := &mockContext{w: 200, h: 500}
	root := VStack(runtime.Props{Style: runtime.Style{Flex: 1}}, n)
	renderer.PaintTree(ctx, []*runtime.VNode{root}, theme.Dark)
	hits := runtime.CollectHits([]*runtime.VNode{n})
	if len(hits) < 2 {
		t.Fatalf("wrapped link has only %d hit area(s)", len(hits))
	}
	for _, hit := range hits {
		if hit.X < n.X || hit.X+hit.W > n.X+n.W+.1 || hit.Y+hit.H > n.Y+n.H+.1 {
			t.Fatalf("link hit extends beyond its paragraph: %+v", hit)
		}
		hit.OnClick()
	}
	if clicked != len(hits) {
		t.Fatal("link callback missing")
	}
}

func TestRichTextCodePreservesIndentationAndStyles(t *testing.T) {
	n := RichText(runtime.Props{Style: runtime.Style{Width: 300}},
		TextRun{Text: "    return ", Color: runtime.RGB(10, 20, 30)}, TextRun{Text: "true", Bold: true})
	ctx := &mockContext{w: 400, h: 100}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	var rendered string
	bold := false
	for _, d := range ctx.draws {
		rendered += d.text
		bold = bold || d.style.Weight == 700
	}
	if rendered != "    return true" || !bold {
		t.Fatalf("code rendering = %q, bold=%v", rendered, bold)
	}
}
