//go:build windows

package windows

import (
	"testing"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

func TestDirectWriteTextFormat(t *testing.T) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		t.Fatalf("CoInitializeEx: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()

	dwrite, err := dwriteCreateFactory()
	if err != nil {
		t.Fatal(err)
	}
	defer release(dwrite)

	locale := userLocale()
	var loc *uint16
	if len(locale) > 0 {
		loc = &locale[0]
	}
	format, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, loc)
	if err != nil {
		t.Fatal(err)
	}
	defer release(format)

	buf, n := utf16Buf("HelloWorld")
	layout, err := createTextLayout(dwrite, &buf[0], n, format, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer release(layout)

	m, err := textLayoutMetrics(layout)
	if err != nil {
		t.Fatal(err)
	}
	if m.width <= 0 || m.height <= 0 {
		t.Fatalf("DirectWrite 度量异常: %+v", m)
	}
}

func TestD2DFactory(t *testing.T) {
	factory, err := d2d1CreateFactory()
	if err != nil {
		t.Fatal(err)
	}
	release(factory)
}

func TestNativeStructLayout(t *testing.T) {
	if unsafe.Sizeof(wndClassExW{}) != 80 {
		t.Fatalf("WNDCLASSEXW 大小应为 80，实际 %d", unsafe.Sizeof(wndClassExW{}))
	}
	if unsafe.Sizeof(d2d1RenderTargetProperties{}) != 28 {
		t.Fatalf("D2D1_RENDER_TARGET_PROPERTIES 大小应为 28，实际 %d", unsafe.Sizeof(d2d1RenderTargetProperties{}))
	}
	if unsafe.Sizeof(d2d1HwndRenderTargetProperties{}) != 24 {
		t.Fatalf("D2D1_HWND_RENDER_TARGET_PROPERTIES 大小应为 24，实际 %d", unsafe.Sizeof(d2d1HwndRenderTargetProperties{}))
	}
}

func TestHostPaintsHelloWorld(t *testing.T) {
	painted := false
	err := Run(Options{
		Title:  "github.com/goedui/goed-test",
		Width:  320,
		Height: 240,
		Paint: func(ctx renderer.Context) {
			if ctx.Width() <= 0 || ctx.Height() <= 0 {
				t.Errorf("绘制表面尺寸无效: %v x %v", ctx.Width(), ctx.Height())
			}
			ctx.Clear(renderer.ColorFrom(theme.Light.Background))
			ctx.DrawText("HelloWorld", 24, 24, 280, 40, renderer.TextStyle{
				FontFamily: theme.Light.FontFamily,
				FontSize:   theme.Light.FontSize,
				Color:      renderer.ColorFrom(theme.Light.Foreground),
				Weight:     400,
			})
			painted = true
			procPostQuitMessage.Call(0)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !painted {
		t.Fatal("窗口未进入 Direct2D 绘制")
	}
}
