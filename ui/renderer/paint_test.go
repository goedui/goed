package renderer

import (
	"testing"

	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

type mockContext struct {
	w, h   float32
	clears []Color
	texts  []string
	sizes  []float32
	rects  int
}

func (m *mockContext) Width() float32  { return m.w }
func (m *mockContext) Height() float32 { return m.h }
func (m *mockContext) DPI() float32    { return 96 }
func (m *mockContext) Clear(c Color)   { m.clears = append(m.clears, c) }
func (m *mockContext) FillRect(float32, float32, float32, float32, Color) {
	m.rects++
}
func (m *mockContext) FillRoundedRect(float32, float32, float32, float32, float32, Color) {
	m.rects++
}
func (m *mockContext) DrawRect(float32, float32, float32, float32, Color, float32) {}
func (m *mockContext) DrawLine(float32, float32, float32, float32, Color, float32) {}
func (m *mockContext) DrawText(text string, _, _, _, _ float32, style TextStyle) {
	m.texts = append(m.texts, text)
	m.sizes = append(m.sizes, style.FontSize)
}
func (m *mockContext) MeasureText(text string, _ float32, style TextStyle) (float32, float32) {
	return float32(len(text)) * style.FontSize * 0.5, style.FontSize * 1.25
}
func (m *mockContext) DrawImage([]byte, float32, float32, float32, float32) {}
func (m *mockContext) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (m *mockContext) PushClip(float32, float32, float32, float32)          {}
func (m *mockContext) PopClip()                                            {}

func TestPaintTreeHelloWorld(t *testing.T) {
	ctx := &mockContext{w: 800, h: 600}
	PaintTree(ctx, []*runtime.VNode{runtime.Text("HelloWorld")}, theme.Light)
	if len(ctx.clears) != 1 {
		t.Fatalf("期望清屏 1 次，得到 %d", len(ctx.clears))
	}
	if ctx.clears[0] != ColorFrom(theme.Light.Background) {
		t.Fatalf("清屏颜色不符合浅色主题: %+v", ctx.clears[0])
	}
	if len(ctx.texts) != 1 || ctx.texts[0] != "HelloWorld" {
		t.Fatalf("绘制文本不符合预期: %v", ctx.texts)
	}
}

func TestInlineStyleOnText(t *testing.T) {
	ctx := &mockContext{w: 480, h: 360}
	n := runtime.Text(runtime.Props{Style: runtime.Style{FontSize: 22, Weight: 600}}, "Hello, github.com/goedui/goed UI!")
	PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	if len(ctx.texts) != 1 || ctx.texts[0] != "Hello, github.com/goedui/goed UI!" {
		t.Fatalf("文本未绘制: %v", ctx.texts)
	}
	if len(ctx.sizes) != 1 || ctx.sizes[0] != 22 {
		t.Fatalf("行内 FontSize 未生效: %v", ctx.sizes)
	}
}
