package renderer

import (
	"testing"

	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// measureProbe 数 MeasureText 被调了几次，用来验证跨帧测量缓存。
type measureProbe struct {
	mockContext
	dpi float32
	n   int
}

func (m *measureProbe) DPI() float32 { return m.dpi }

func (m *measureProbe) MeasureText(text string, limit float32, style TextStyle) (float32, float32) {
	m.n++
	return m.mockContext.MeasureText(text, limit, style)
}

func textTree(s string) []*runtime.VNode {
	return []*runtime.VNode{runtime.Text(s)}
}

// TestMeasureCacheReusesTextAcrossTrees：文本没变的重建帧不该再问后端。
//
// 这里特意造两棵**不同的**节点树（重建帧就是新树），节点级的测量记忆帮不上忙，
// 只有按 (文本, 限宽, 字体) 的跨帧缓存能命中。
func TestMeasureCacheReusesTextAcrossTrees(t *testing.T) {
	ResetMeasureCache()
	ctx := &measureProbe{mockContext: mockContext{w: 800, h: 600}, dpi: 96}

	PaintTree(ctx, textTree("同样的文本"), theme.Dark)
	if ctx.n == 0 {
		t.Fatal("首帧应当向后端测量")
	}

	ctx.n = 0
	PaintTree(ctx, textTree("同样的文本"), theme.Dark)
	if ctx.n != 0 {
		t.Fatalf("文本未变的重建帧应命中缓存，仍测量 %d 次", ctx.n)
	}

	// 文本变了就要重新量。
	PaintTree(ctx, textTree("换了一段文本"), theme.Dark)
	if ctx.n == 0 {
		t.Fatal("文本变化后应当重新测量")
	}
}

// TestMeasureCacheIsolatedPerBackend：换了后端（类型或 DPI 变了）必须清缓存，
// 否则会把另一个后端的尺寸用过来。
func TestMeasureCacheIsolatedPerBackend(t *testing.T) {
	ResetMeasureCache()
	ctx96 := &measureProbe{mockContext: mockContext{w: 800, h: 600}, dpi: 96}
	PaintTree(ctx96, textTree("同样的文本"), theme.Dark)

	ctx144 := &measureProbe{mockContext: mockContext{w: 800, h: 600}, dpi: 144}
	PaintTree(ctx144, textTree("同样的文本"), theme.Dark)
	if ctx144.n == 0 {
		t.Fatal("DPI 变了应当重新测量")
	}
}
