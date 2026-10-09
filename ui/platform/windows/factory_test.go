//go:build windows

package windows

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

func TestDirectWriteTextFormat(t *testing.T) {
	// COM 的套间是**绑线程**的：CoInitializeEx 与 CoUninitialize 必须落在同一条 OS
	// 线程上。不锁线程时 goroutine 会在两次调用之间被调度器搬走，于是「还」的那次
	// 落在另一条线程上——本线程的引用计数没减、那条线程被无端 CoUninitialize 一次，
	// 而且它还会污染线程池，让后面依赖「线程是干净的」的用例莫名其妙地跳过。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
	format, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, dwriteFontStyleNormal, loc)
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

	// 斜体是 CreateTextFormat 的 fontStyle 参数，不是画布变换。它不是「把正体
	// 斜过来」那么单纯：字体有 italic 面就换面（步进会跟着变，实测 Segoe UI
	// 16px 的 HelloWorld 是 79.41 → 77.88），没有才合成倾斜。所以量宽与画字
	// 必须共用同一个 format —— 一旦两边拿的 format 不同，行内折行就会和实际
	// 绘制对不上。这里钉住的是「斜体能量、能建 layout，且步进没有量级差别」。
	italic, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, dwriteFontStyleItalic, loc)
	if err != nil {
		t.Fatal(err)
	}
	defer release(italic)

	italicLayout, err := createTextLayout(dwrite, &buf[0], n, italic, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer release(italicLayout)

	im, err := textLayoutMetrics(italicLayout)
	if err != nil {
		t.Fatal(err)
	}
	if im.width <= 0 || im.height <= 0 {
		t.Fatalf("斜体度量异常: %+v", im)
	}
	if delta := im.width - m.width; delta > m.width*0.1 || delta < -m.width*0.1 {
		t.Errorf("斜体与正体的步进差了 %.1f%%（%.3f vs %.3f），超出字形差异的合理范围",
			delta/m.width*100, im.width, m.width)
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
