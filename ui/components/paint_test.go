package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

type mockTextDraw struct {
	text       string
	x, y, w, h float32
	style      renderer.TextStyle
}

// mockRound 记录一次圆角矩形绘制，便于断言几何与颜色（switch 等控件用）。
type mockRound struct {
	x, y, w, h, r float32
	c             renderer.Color
}

type mockContext struct {
	w, h   float32
	clears []renderer.Color
	texts  []string
	rects  int
	draws  []mockTextDraw
	rounds []mockRound
}

func (m *mockContext) Width() float32  { return m.w }
func (m *mockContext) Height() float32 { return m.h }
func (m *mockContext) DPI() float32    { return 96 }
func (m *mockContext) Clear(c renderer.Color) {
	m.clears = append(m.clears, c)
}
func (m *mockContext) FillRect(float32, float32, float32, float32, renderer.Color) {
	m.rects++
}
func (m *mockContext) FillRoundedRect(x, y, w, h, r float32, c renderer.Color) {
	m.rects++
	m.rounds = append(m.rounds, mockRound{x: x, y: y, w: w, h: h, r: r, c: c})
}
func (m *mockContext) DrawRect(float32, float32, float32, float32, renderer.Color, float32) {}
func (m *mockContext) DrawLine(float32, float32, float32, float32, renderer.Color, float32) {}
func (m *mockContext) DrawText(text string, x, y, w, h float32, style renderer.TextStyle) {
	m.texts = append(m.texts, text)
	m.draws = append(m.draws, mockTextDraw{text: text, x: x, y: y, w: w, h: h, style: style})
}
func (m *mockContext) MeasureText(text string, _ float32, style renderer.TextStyle) (float32, float32) {
	return float32(len(text)) * style.FontSize * 0.5, style.FontSize * 1.25
}
func (m *mockContext) DrawImage([]byte, float32, float32, float32, float32) {}
func (m *mockContext) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (m *mockContext) PushClip(float32, float32, float32, float32)          {}
func (m *mockContext) PopClip()                                             {}

func TestPaintTreeWithTitleBar(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	tb := runtime.Tag("titlebar", runtime.Attrs{"title": "HelloWorld"})
	renderer.PaintTree(ctx, []*runtime.VNode{tb, runtime.Text("body")}, theme.Dark)
	if len(ctx.clears) != 1 || ctx.clears[0] != renderer.ColorFrom(theme.Dark.Background) {
		t.Fatalf("清屏颜色不符合深色主题: %+v", ctx.clears)
	}
	foundTitle, foundBody := false, false
	for _, s := range ctx.texts {
		if s == "HelloWorld" {
			foundTitle = true
		}
		if s == "body" {
			foundBody = true
		}
	}
	if !foundTitle || !foundBody {
		t.Fatalf("标题或正文未绘制: %v", ctx.texts)
	}
}

func TestPaintTreeVStackButton(t *testing.T) {
	ctx := &mockContext{w: 480, h: 360}
	tree := VStack(runtime.Props{
		Style: runtime.Style{
			Flex: 1, Padding: 32, Gap: 16,
			Align: runtime.Center, Justify: runtime.Center,
		},
	},
		runtime.Text(runtime.Props{Style: runtime.Style{FontSize: 22, Weight: 600}}, "Hello, github.com/goedui/goed UI!"),
		runtime.Text("你点击了 ", 0, " 次"),
		Button(runtime.Props{OnClick: func() {}}, "点我"),
	)
	renderer.PaintTree(ctx, []*runtime.VNode{tree}, theme.Light)
	foundHello, foundCount, foundBtn := false, false, false
	for _, s := range ctx.texts {
		switch s {
		case "Hello, github.com/goedui/goed UI!":
			foundHello = true
		case "你点击了 0 次":
			foundCount = true
		case "点我":
			foundBtn = true
		}
	}
	if !foundHello || !foundCount || !foundBtn {
		t.Fatalf("vstack 内容未画全: %v", ctx.texts)
	}
	if ctx.rects == 0 {
		t.Fatal("按钮应填充背景")
	}
	hits := runtime.CollectHits([]*runtime.VNode{tree})
	if len(hits) != 1 {
		t.Fatalf("期望 1 个可点击区域，得到 %d", len(hits))
	}
	if runtime.HitTest(hits, hits[0].X+1, hits[0].Y+1) == nil {
		t.Fatal("按钮区域内应能命中")
	}
}
