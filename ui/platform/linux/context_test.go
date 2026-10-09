//go:build linux

package linux

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

// solidPNG 造一张纯色 PNG，给圆角图片的用例当素材。
func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("造 PNG 失败：%v", err)
	}
	return buf.Bytes()
}

// 圆角图片：背景纯红、图纯绿，四个角应当还是红的，四条边中点应当是绿的。
// 和 Windows 后端那条像素用例判据一致——两个后端对同一件事要给出同样的结果。
func TestDrawImageRoundedClipsCorners(t *testing.T) {
	// dpi=96，所以 DIP 与设备像素一一对应，坐标不用换算。
	ctx := newContext(120, 100, 96)
	ctx.Clear(renderer.Color{R: 1, A: 1})
	ctx.DrawImageRounded(solidPNG(t, 60, 50, color.NRGBA{G: 255, A: 255}), 10, 10, 60, 50, 12)

	// 四条边的中点必须被图片铺满（否则「一张都没画」也会让角上的判据通过）。
	for _, c := range []struct {
		name string
		x, y int
	}{
		{"上边中点", 40, 10},
		{"下边中点", 40, 59},
		{"左边中点", 10, 35},
		{"右边中点", 69, 35},
	} {
		r, g, _, _ := ctx.pixel(c.x, c.y)
		if g < 200 || r > 60 {
			t.Errorf("%s (%d,%d) 不是绿色（r=%d g=%d），图片没铺满矩形", c.name, c.x, c.y, r, g)
		}
	}
	// 圆角要真的切掉：包围盒四角应当还是背景红。
	for _, c := range []struct {
		name string
		x, y int
	}{
		{"左上角", 10, 10},
		{"右上角", 69, 10},
		{"左下角", 10, 59},
		{"右下角", 69, 59},
	} {
		r, g, _, _ := ctx.pixel(c.x, c.y)
		if r < 200 || g > 60 {
			t.Errorf("%s (%d,%d) 不是背景色（r=%d g=%d），圆角没切掉", c.name, c.x, c.y, r, g)
		}
	}
	// 角心再往里一点应该已经是图了：证明切掉的是角、不是整块。
	if r, g, _, _ := ctx.pixel(25, 13); g < 200 || r > 60 {
		t.Errorf("圆角内 (25,13) 不是绿色（r=%d g=%d），切多了", r, g)
	}
}

// r <= 0 时不该切角，等价于 DrawImage。
func TestDrawImageRoundedZeroRadiusIsSquare(t *testing.T) {
	ctx := newContext(120, 100, 96)
	ctx.Clear(renderer.Color{R: 1, A: 1})
	ctx.DrawImageRounded(solidPNG(t, 60, 50, color.NRGBA{G: 255, A: 255}), 10, 10, 60, 50, 0)
	r, g, _, _ := ctx.pixel(10, 10)
	if g < 200 || r > 60 {
		t.Errorf("r=0 时角上应是图片本身（r=%d g=%d）", r, g)
	}
}

// 覆盖率函数本身的边界：角心处为 1、角外为 0、弧上介于两者之间。
func TestCornerCoverage(t *testing.T) {
	const x0, y0, x1, y1 = 0, 0, 100, 100
	if got := cornerCoverage(50, 50, x0, y0, x1, y1, 20); got != 1 {
		t.Errorf("矩形正中覆盖率应为 1，得到 %v", got)
	}
	if got := cornerCoverage(0, 0, x0, y0, x1, y1, 20); got != 0 {
		t.Errorf("角上（离角心 28.3 > 20）覆盖率应为 0，得到 %v", got)
	}
	// 弧上那一圈像素的覆盖率应严格介于 0 和 1 之间——抗锯齿就靠它。
	// (5,6) 的像素中心 (5.5,6.5) 离角心 (20,20) 约 19.81，落在半径 20 的弧上。
	if got := cornerCoverage(5, 6, x0, y0, x1, y1, 20); got <= 0 || got >= 1 {
		t.Errorf("弧上的覆盖率应介于 0 与 1 之间，得到 %v", got)
	}
	// 半径给 0（或负数）时不做圆角。
	if got := cornerCoverage(0, 0, x0, y0, x1, y1, 0); got != 1 {
		t.Errorf("半径为 0 时覆盖率应为 1，得到 %v", got)
	}
	// 非角方块区域直接放行，和半径无关。
	if got := cornerCoverage(50, 5, x0, y0, x1, y1, 20); got != 1 {
		t.Errorf("上边中部不在角方块里，覆盖率应为 1，得到 %v", got)
	}
}

func (c *swContext) pixel(x, y int) (r, g, b, a byte) {
	if x < 0 || y < 0 || x >= c.pixelW || y >= c.pixelH {
		return 0, 0, 0, 0
	}
	off := y*c.stride + x*4
	return c.pix[off+2], c.pix[off+1], c.pix[off], c.pix[off+3]
}
