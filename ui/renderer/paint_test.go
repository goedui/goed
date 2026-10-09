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
	aligns []Align
	rects  int
	images int
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
	m.aligns = append(m.aligns, style.Align)
}
func (m *mockContext) MeasureText(text string, _ float32, style TextStyle) (float32, float32) {
	return float32(len(text)) * style.FontSize * 0.5, style.FontSize * 1.25
}
func (m *mockContext) DrawImage([]byte, float32, float32, float32, float32)            { m.images++ }
func (m *mockContext) DrawPixels([]byte, int, int, float32, float32, float32, float32) {}
func (m *mockContext) PushClip(float32, float32, float32, float32)                     {}
func (m *mockContext) PopClip()                                                        {}

type roundStrokeCtx struct {
	mockContext
	rounds int
	r      float32
}

func (c *roundStrokeCtx) DrawRoundedRect(_, _, _, _, r float32, _ Color, _ float32) {
	c.rounds++
	c.r = r
}

func TestStrokeRoundedUsesRoundedRect(t *testing.T) {
	ctx := &roundStrokeCtx{}
	StrokeRounded(ctx, 10, 20, 200, 38, 12, 1, RGB(47, 107, 242))
	if ctx.rounds != 1 || ctx.r != 12 {
		t.Fatalf("圆角描边应走 DrawRoundedRect：rounds=%d r=%v", ctx.rounds, ctx.r)
	}
	if ctx.rects != 0 {
		t.Fatalf("有圆角矩形描边时不应再拼点：rects=%d", ctx.rects)
	}
}

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

// 文本节点自己写了 Align 才算数：盒子比文字宽时（显式宽度 / 被撑满 /
// layoutFlow 给的整行宽），文字按它对齐。
//
// 容器的 Align 不参与 —— 那是「子节点往哪边摆」，不是「文字往哪边画」。
// 这两件事混在一起的话，`VStack{Align: Center}` 或 Tabs（给自己写死 Center）
// 底下每个 Text 子节点都会被居中，而它们的盒子常常是撑满的，正文就莫名居中。
func TestTextAlignOnlyFromOwnStyle(t *testing.T) {
	// 1) 节点自己写 Align → 生效。
	ctx := &mockContext{w: 480, h: 360}
	n := runtime.Text(runtime.Props{Style: runtime.Style{Width: 200, Align: runtime.Center}}, "居中")
	PaintTree(ctx, []*runtime.VNode{n}, theme.Light)
	if len(ctx.aligns) != 1 || ctx.aligns[0] != AlignCenter {
		t.Fatalf("节点自己的 Align 没生效：%v", ctx.aligns)
	}

	// 2) 容器的 Align 不能漏到子节点上。
	ctx2 := &mockContext{w: 480, h: 360}
	parent := runtime.H("vstack", runtime.Props{Style: runtime.Style{Align: runtime.Center}},
		runtime.Text("子节点"))
	PaintTree(ctx2, []*runtime.VNode{parent}, theme.Light)
	if len(ctx2.aligns) == 0 {
		t.Fatal("没画到子节点的文本")
	}
	for i, a := range ctx2.aligns {
		if a != AlignStart {
			t.Errorf("第 %d 段文本被容器的 Align 影响了：%v", i, a)
		}
	}

	// 3) "stretch" 是「拉满」不是「靠边」，没有文本对齐语义，不该改动画法。
	ctx3 := &mockContext{w: 480, h: 360}
	n3 := runtime.Text(runtime.Props{Style: runtime.Style{Width: 200, Align: runtime.Stretch}}, "拉满")
	PaintTree(ctx3, []*runtime.VNode{n3}, theme.Light)
	if len(ctx3.aligns) != 1 || ctx3.aligns[0] != AlignStart {
		t.Errorf("stretch 不该映射成文本对齐：%v", ctx3.aligns)
	}
}

func TestPaintSkipsNodesOutsideClip(t *testing.T) {
	ctx := &mockContext{w: 200, h: 200}
	visible := runtime.Text(runtime.Props{Style: runtime.Style{Height: 20}}, "可见")
	hidden := runtime.Text(runtime.Props{Style: runtime.Style{Height: 20}}, "隐藏")
	visible.X, visible.Y, visible.W, visible.H = 0, 0, 80, 20
	hidden.X, hidden.Y, hidden.W, hidden.H = 0, 400, 80, 20
	PushCullClip(0, 0, 200, 40)
	paintLaidOut(ctx, []*runtime.VNode{visible, hidden}, TextStyle{FontSize: 13}, theme.Light)
	PopCullClip()
	foundHidden := false
	foundVisible := false
	for _, s := range ctx.texts {
		if s == "隐藏" {
			foundHidden = true
		}
		if s == "可见" {
			foundVisible = true
		}
	}
	if !foundVisible {
		t.Fatal("裁剪内的节点应绘制")
	}
	if foundHidden {
		t.Fatal("裁剪外的节点不应绘制")
	}
}
