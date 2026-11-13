package components

import (
	"fmt"
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/runtime"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"strings"
)

var markdownParser = goldmark.New(goldmark.WithExtensions(extension.GFM))

// Markdown renders native GFM blocks. OnSelect receives clicked link URLs.
func Markdown(p runtime.Props, source string) *runtime.VNode {
	data := []byte(source)
	doc := markdownParser.Parser().Parse(text.NewReader(data))
	var onLink func(string)
	onLink, _ = p.OnSelect.(func(string))
	if p.Style.Gap <= 0 {
		p.Style.Gap = 12
	}
	p.Style.Align = runtime.Stretch
	return VStack(p, markdownBlocks(doc, data, onLink))
}

func markdownBlocks(parent ast.Node, data []byte, onLink func(string)) []*runtime.VNode {
	var nodes []*runtime.VNode
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if node := markdownBlock(child, data, onLink); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func markdownBlock(node ast.Node, data []byte, onLink func(string)) *runtime.VNode {
	switch n := node.(type) {
	case *ast.Heading:
		sizes := []float32{24, 20, 17, 15, 14, 13}
		return RichText(runtime.Props{Style: runtime.Style{FontSize: sizes[n.Level-1], Weight: 600}}, markdownInline(n, data, TextRun{}, onLink)...)
	case *ast.Paragraph, *ast.TextBlock:
		return RichText(runtime.Props{}, markdownInline(n, data, TextRun{}, onLink)...)
	case *ast.FencedCodeBlock:
		return markdownCode(string(n.Lines().Value(data)), string(n.Language(data)))
	case *ast.CodeBlock:
		return markdownCode(string(n.Lines().Value(data)), "")
	case *ast.Blockquote:
		return HStack(runtime.Props{Style: runtime.Style{Gap: 12, Align: runtime.Stretch}},
			Box(runtime.Props{Style: runtime.Style{Width: 3, Background: runtime.RGB(76, 76, 76)}}),
			VStack(runtime.Props{Style: runtime.Style{Flex: 1, Align: runtime.Stretch, Gap: 8, Color: runtime.RGB(157, 157, 157)}}, markdownBlocks(n, data, onLink)),
		)
	case *ast.List:
		var rows []*runtime.VNode
		index := n.Start
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			mark := "•"
			if n.IsOrdered() {
				mark = fmt.Sprintf("%d.", index)
				index++
			}
			rows = append(rows, HStack(runtime.Props{Style: runtime.Style{Gap: 8, Align: runtime.Start}},
				runtime.Text(runtime.Props{Style: runtime.Style{Width: 20, FontSize: 13}}, mark),
				VStack(runtime.Props{Style: runtime.Style{Flex: 1, Gap: 6, Align: runtime.Stretch}}, markdownBlocks(item, data, onLink)),
			))
		}
		return VStack(runtime.Props{Style: runtime.Style{Gap: 6, Align: runtime.Stretch}}, rows)
	case *ast.ThematicBreak:
		return Divider(runtime.Props{Style: runtime.Style{Background: runtime.RGB(60, 60, 60)}})
	case *extast.Table:
		return markdownTable(n, data, onLink)
	case *ast.HTMLBlock:
		return runtime.Text(runtime.Props{Style: runtime.Style{FontFamily: "Consolas", FontSize: 12}}, string(n.Lines().Value(data)))
	default:
		if n.HasChildren() {
			return VStack(runtime.Props{Style: runtime.Style{Gap: 8}}, markdownBlocks(n, data, onLink))
		}
	}
	return nil
}

func markdownInline(parent ast.Node, data []byte, style TextRun, onLink func(string)) []TextRun {
	var runs []TextRun
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		run := style
		switch n := child.(type) {
		case *ast.Text:
			run.Text = string(n.Segment.Value(data))
			if n.HardLineBreak() {
				run.Text += "\n"
			} else if n.SoftLineBreak() {
				run.Text += " "
			}
			runs = append(runs, run)
		case *ast.String:
			run.Text = string(n.Value)
			runs = append(runs, run)
		case *ast.Emphasis:
			if n.Level == 2 {
				run.Bold = true
			} else {
				run.Italic = true
			}
			runs = append(runs, markdownInline(n, data, run, onLink)...)
		case *ast.CodeSpan:
			run.Code = true
			run.Text = string(n.Text(data))
			runs = append(runs, run)
		case *ast.Link:
			url := string(n.Destination)
			run.Color = runtime.RGB(77, 166, 255)
			run.Underline = true
			if onLink != nil {
				run.OnClick = func() { onLink(url) }
			}
			runs = append(runs, markdownInline(n, data, run, onLink)...)
		case *ast.AutoLink:
			url := string(n.URL(data))
			run.Text = string(n.Label(data))
			run.Color = runtime.RGB(77, 166, 255)
			run.Underline = true
			if onLink != nil {
				run.OnClick = func() { onLink(url) }
			}
			runs = append(runs, run)
		case *extast.Strikethrough:
			run.Strike = true
			runs = append(runs, markdownInline(n, data, run, onLink)...)
		case *extast.TaskCheckBox:
			run.Text = "☐ "
			if n.IsChecked {
				run.Text = "☑ "
			}
			runs = append(runs, run)
		case *ast.Image:
			run.Text = "[图片：" + string(n.Text(data)) + "]"
			runs = append(runs, run)
		default:
			runs = append(runs, markdownInline(n, data, run, onLink)...)
		}
	}
	return runs
}

func markdownCode(code, language string) *runtime.VNode {
	bg := runtime.RGB(24, 24, 24)
	header := HStack(runtime.Props{Style: runtime.Style{Padding: 8, Align: runtime.Center, Background: runtime.RGB(43, 43, 43)}},
		runtime.Text(runtime.Props{Style: runtime.Style{Flex: 1, FontSize: 11, Color: runtime.RGB(157, 157, 157)}}, language),
		Button(runtime.Props{Icon: "copy", Title: "复制代码", Style: runtime.Style{Width: 24, Height: 22, Radius: 3, Background: runtime.RGB(43, 43, 43), Border: runtime.RGB(43, 43, 43), Color: runtime.RGB(157, 157, 157)}, OnClick: func() {
			if platform.ClipboardSet != nil {
				platform.ClipboardSet(strings.TrimSuffix(code, "\n"))
			}
		}}),
	)
	lines := strings.Split(strings.TrimSuffix(code, "\n"), "\n")
	var rows []*runtime.VNode
	for _, line := range lines {
		if line == "" {
			line = " "
		}
		rows = append(rows, RichText(runtime.Props{Style: runtime.Style{FontFamily: "Consolas", FontSize: 12, MinHeight: 20, Color: runtime.RGB(204, 204, 204)}}, highlightCode(line, language)...))
	}
	return VStack(runtime.Props{Style: runtime.Style{Background: bg, Radius: 4, Border: runtime.RGB(60, 60, 60), Align: runtime.Stretch}}, header,
		VStack(runtime.Props{Style: runtime.Style{Padding: 10, Gap: 0, Align: runtime.Stretch}}, rows),
	)
}

func markdownTable(table *extast.Table, data []byte, onLink func(string)) *runtime.VNode {
	var rows []*runtime.VNode
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []*runtime.VNode
		_, header := row.(*extast.TableHeader)
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			style := runtime.Style{Flex: 1, Padding: 8, Border: runtime.RGB(60, 60, 60)}
			if header {
				style.Background = runtime.RGB(43, 43, 43)
			}
			cells = append(cells, VStack(runtime.Props{Style: style}, RichText(runtime.Props{}, markdownInline(cell, data, TextRun{Bold: header}, onLink)...)))
		}
		rows = append(rows, HStack(runtime.Props{Style: runtime.Style{Align: runtime.Stretch}}, cells))
	}
	return VStack(runtime.Props{Style: runtime.Style{Align: runtime.Stretch}}, rows)
}
