package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

func TestAppIconRaster(t *testing.T) {
	rgba := App.RGBA(16, renderer.ColorFrom(theme.Dark.Foreground), renderer.ColorFrom(theme.Dark.Titlebar))
	if len(rgba) != 16*16*4 {
		t.Fatalf("RGBA 尺寸错误: %d", len(rgba))
	}
	opaque := 0
	for i := 3; i < len(rgba); i += 4 {
		if rgba[i] > 0 {
			opaque++
		}
	}
	if opaque < 16 {
		t.Fatalf("图标几乎是空的: 不透明像素 %d", opaque)
	}
}

func TestAppShellIcon(t *testing.T) {
	for _, size := range []int{16, 32, 64} {
		img := App.WindowsIcon(size)
		if len(img) < 40+size*size*4 {
			t.Fatalf("%d 图标编码过短: %d", size, len(img))
		}
		if got := int(img[0]); got != 40 {
			t.Fatalf("BITMAPINFOHEADER 大小应为 40，得到 %d", got)
		}
	}
}

func TestAppIconName(t *testing.T) {
	if App.Name != "app" || App.Paint == nil {
		t.Fatal("默认应用图标未注册")
	}
}
