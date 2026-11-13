package components

import (
	"strings"
	"testing"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestMarkdownGFMCodeCopyAndLinks(t *testing.T) {
	const code = "func main() {\n    println(42)\n}"
	source := "# Heading\n\nA **bold** and ~~old~~ [link](https://example.com).\n\n- [x] checked\n- [ ] open\n\n> quote\n\n```go\n" + code + "\n```\n\n| Name | Value |\n| --- | --- |\n| answer | 42 |"
	var url, copied string
	previous := platform.ClipboardSet
	platform.ClipboardSet = func(s string) { copied = s }
	t.Cleanup(func() { platform.ClipboardSet = previous })
	n := Markdown(runtime.Props{Style: runtime.Style{Width: 480, Color: runtime.RGB(204, 204, 204)}, OnSelect: func(s string) { url = s }}, source)
	ctx := &mockContext{w: 500, h: 1200}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	var copyButton *runtime.VNode
	var link *runtime.VNode
	runtime.Walk([]*runtime.VNode{n}, func(node *runtime.VNode) bool {
		if node.Props.Title == "复制代码" {
			copyButton = node
		}
		if node.Kind == runtime.KindText && node.Text == "link" && node.OnClick != nil {
			link = node
		}
		return false
	})
	if copyButton == nil || link == nil {
		t.Fatal("code copy or link action missing")
	}
	copyButton.OnClick()
	link.OnClick()
	if copied != code || url != "https://example.com" {
		t.Fatalf("actions copied=%q url=%q", copied, url)
	}
	for _, want := range []string{"Heading", "bold", "old", "checked", "open", "quote", "Name", "answer", "☑", "☐"} {
		if !strings.Contains(strings.Join(ctx.texts, ""), want) {
			t.Errorf("rendered markdown missing %q", want)
		}
	}
	bold, strike := false, false
	for _, draw := range ctx.draws {
		bold = bold || (draw.text == "bold" && draw.style.Weight == 700)
	}
	runtime.Walk([]*runtime.VNode{n}, func(node *runtime.VNode) bool {
		runs, _ := node.Attrs["runs"].([]TextRun)
		for _, run := range runs {
			strike = strike || run.Strike
		}
		return false
	})
	if !bold || !strike {
		t.Fatal("inline formatting was discarded")
	}
}
