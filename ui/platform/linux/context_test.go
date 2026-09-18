//go:build linux

package linux

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
)

func TestFillRectAndClip(t *testing.T) {
	ctx := newContext(20, 10, 96)
	ctx.Clear(renderer.Color{A: 1})
	ctx.PushClip(2, 2, 5, 5)
	ctx.FillRect(0, 0, 20, 10, renderer.Color{R: 1, A: 1})
	ctx.PopClip()

	r, _, _, _ := ctx.pixel(4, 4)
	if r != 255 {
		t.Fatalf("裁剪内应为红色，得到 r=%d", r)
	}
	r, _, _, _ = ctx.pixel(9, 4)
	if r != 0 {
		t.Fatalf("裁剪外不应被填充，得到 r=%d", r)
	}
}

func TestMeasureChinese(t *testing.T) {
	ctx := newContext(400, 100, 96)
	style := renderer.TextStyle{FontSize: 13, Color: renderer.Color{R: 1, G: 1, B: 1, A: 1}}
	w, h := ctx.MeasureText("你好世界 Hello", 400, style)
	if w <= 0 || h <= 0 {
		t.Fatalf("中英混排度量应为正值，得到 %v x %v", w, h)
	}
	ctx.Clear(renderer.Color{A: 1})
	ctx.DrawText("你好世界", 0, 0, 400, 100, style)
	found := false
	for y := 0; y < ctx.pixelH && !found; y++ {
		for x := 0; x < ctx.pixelW; x++ {
			r, g, b, _ := ctx.pixel(x, y)
			if r > 32 || g > 32 || b > 32 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("中文 DrawText 没有点亮任何像素")
	}
}

func TestMeasureAndDrawText(t *testing.T) {
	ctx := newContext(400, 100, 96)
	style := renderer.TextStyle{FontSize: 13, Color: renderer.Color{R: 1, G: 1, B: 1, A: 1}}
	w, h := ctx.MeasureText("HelloWorld", 400, style)
	if w <= 0 || h <= 0 {
		t.Fatalf("MeasureText 应为正值，得到 %v x %v", w, h)
	}
	ctx.Clear(renderer.Color{A: 1})
	ctx.DrawText("HelloWorld", 0, 0, 400, 100, style)
	found := false
	for y := 0; y < ctx.pixelH && !found; y++ {
		for x := 0; x < ctx.pixelW; x++ {
			r, g, b, _ := ctx.pixel(x, y)
			if r > 128 && g > 128 && b > 128 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("DrawText 没有点亮任何像素")
	}
}

func TestRoundedRectDegenerate(t *testing.T) {
	ctx := newContext(16, 16, 96)
	ctx.FillRoundedRect(0, 0, 8, 8, 0, renderer.Color{B: 1, A: 1})
	_, _, b, _ := ctx.pixel(2, 2)
	if b != 255 {
		t.Fatalf("r=0 应退化成 FillRect，得到 b=%d", b)
	}
}

func (c *swContext) pixel(x, y int) (r, g, b, a byte) {
	if x < 0 || y < 0 || x >= c.pixelW || y >= c.pixelH {
		return 0, 0, 0, 0
	}
	off := y*c.stride + x*4
	return c.pix[off+2], c.pix[off+1], c.pix[off], c.pix[off+3]
}
