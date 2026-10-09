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

// mockStroke 记录一次描边（DrawRect）。焦点环这类「只看颜色」的行为要靠它：
// 只数次数看不出画的是什么色，禁用输入框有没有留焦点环就查不出来。
type mockStroke struct {
	x, y, w, h float32
	c          renderer.Color
	width      float32
}

type mockContext struct {
	w, h   float32
	clears []renderer.Color
	texts  []string
	rects  int
	lines  int
	draws  []mockTextDraw
	rounds []mockRound
	// fills 记下每个 FillRect 的矩形。只数次数看不出「画在哪儿」，
	// 标题栏按钮这类靠位置区分的行为需要几何。
	fills []mockRound
	// strokes 记下每个 DrawRect 的颜色与线宽。
	strokes []mockStroke
}

func (m *mockContext) Width() float32  { return m.w }
func (m *mockContext) Height() float32 { return m.h }
func (m *mockContext) DPI() float32    { return 96 }
func (m *mockContext) Clear(c renderer.Color) {
	m.clears = append(m.clears, c)
}
func (m *mockContext) FillRect(x, y, w, h float32, c renderer.Color) {
	m.rects++
	m.fills = append(m.fills, mockRound{x: x, y: y, w: w, h: h, c: c})
}
func (m *mockContext) FillRoundedRect(x, y, w, h, r float32, c renderer.Color) {
	m.rects++
	m.rounds = append(m.rounds, mockRound{x: x, y: y, w: w, h: h, r: r, c: c})
}
func (m *mockContext) DrawRect(x, y, w, h float32, c renderer.Color, width float32) {
	m.strokes = append(m.strokes, mockStroke{x: x, y: y, w: w, h: h, c: c, width: width})
}
func (m *mockContext) DrawLine(float32, float32, float32, float32, renderer.Color, float32) {
	m.lines++
}
func (m *mockContext) DrawText(text string, x, y, w, h float32, style renderer.TextStyle) {
	m.texts = append(m.texts, text)
	m.draws = append(m.draws, mockTextDraw{text: text, x: x, y: y, w: w, h: h, style: style})
}
func (m *mockContext) MeasureText(text string, _ float32, style renderer.TextStyle) (float32, float32) {
	return float32(len(text)) * style.FontSize * 0.5, style.FontSize * 1.25
}
func (m *mockContext) DrawImage([]byte, float32, float32, float32, float32)            {}
func (m *mockContext) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (m *mockContext) PushClip(float32, float32, float32, float32)                     {}
func (m *mockContext) PopClip()                                                        {}

type countingMeasure struct {
	w, h float32
	n    int
}

func (m *countingMeasure) Width() float32                                              { return m.w }
func (m *countingMeasure) Height() float32                                             { return m.h }
func (m *countingMeasure) DPI() float32                                                { return 96 }
func (m *countingMeasure) Clear(renderer.Color)                                        {}
func (m *countingMeasure) FillRect(float32, float32, float32, float32, renderer.Color) {}
func (m *countingMeasure) FillRoundedRect(float32, float32, float32, float32, float32, renderer.Color) {
}
func (m *countingMeasure) DrawRect(float32, float32, float32, float32, renderer.Color, float32)    {}
func (m *countingMeasure) DrawLine(float32, float32, float32, float32, renderer.Color, float32)    {}
func (m *countingMeasure) DrawText(string, float32, float32, float32, float32, renderer.TextStyle) {}
func (m *countingMeasure) MeasureText(text string, _ float32, style renderer.TextStyle) (float32, float32) {
	m.n++
	size := style.FontSize
	if size <= 0 {
		size = 13
	}
	return float32(len([]rune(text))) * size * 0.5, size * 1.25
}
func (m *countingMeasure) DrawImage([]byte, float32, float32, float32, float32)            {}
func (m *countingMeasure) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (m *countingMeasure) PushClip(float32, float32, float32, float32)                     {}
func (m *countingMeasure) PopClip()                                                        {}

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
