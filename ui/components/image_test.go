package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestFitContainKeepsAspectRatio(t *testing.T) {
	x, y, w, h := fitContain(800, 400, 1024, 1024)
	if w != h {
		t.Fatalf("正方形应保持 1:1: %v x %v", w, h)
	}
	if w != 400 || x != 200 || y != 0 {
		t.Fatalf("应垂直铺满并水平居中: x=%v y=%v w=%v h=%v", x, y, w, h)
	}
	x, y, w, h = fitContain(400, 800, 1536, 1024)
	wantH := float32(400) * 1024 / 1536
	if w != 400 || h != wantH || x != 0 {
		t.Fatalf("宽图应水平铺满并保持比例: x=%v y=%v w=%v h=%v wantH=%v", x, y, w, h, wantH)
	}
}

func TestPaintImageUsesRoundedBackground(t *testing.T) {
	ctx := &mockContext{w: 200, h: 200}
	n := Image(runtime.Props{
		Style: runtime.Style{Width: 80, Height: 80, Radius: 12},
		ID:    "thumb",
	})
	n.X, n.Y, n.W, n.H = 0, 0, 80, 80
	paintImage(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if ctx.rects == 0 {
		t.Fatal("圆角缩略图应填充圆角底")
	}
}
