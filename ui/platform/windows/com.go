//go:build windows

package windows

import (
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"
)

// com 是原生 COM 接口指针：必须用 uintptr 而不是带指针字段的 Go 结构体，
// 否则 GC 会把 Direct2D 对象当 Go 堆来扫描。
type com uintptr

func (obj com) slot(idx int) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(obj))
	return *(*uintptr)(unsafe.Pointer(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
}

func release(obj com) {
	if obj == 0 {
		return
	}
	syscall.SyscallN(obj.slot(idxRelease), uintptr(obj))
}

func d2d1CreateFactory() (com, error) {
	var factory com
	iid := iidID2D1Factory
	hr, _, _ := procD2D1CreateFactory.Call(
		uintptr(d2d1FactoryTypeSingleThreaded),
		uintptr(unsafe.Pointer(&iid)),
		0,
		uintptr(unsafe.Pointer(&factory)),
	)
	runtime.KeepAlive(iid)
	if hresult(hr).failed() || factory == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: D2D1CreateFactory 失败: %v", hresult(hr))
	}
	return factory, nil
}

func dwriteCreateFactory() (com, error) {
	var factory com
	iid := iidIDWriteFactory
	hr, _, _ := procDWriteCreateFactory.Call(
		uintptr(dwriteFactoryTypeShared),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&factory)),
	)
	runtime.KeepAlive(iid)
	if hresult(hr).failed() || factory == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: DWriteCreateFactory 失败: %v", hresult(hr))
	}
	return factory, nil
}

func createHwndRenderTarget(factory com, hwnd uintptr, pxW, pxH uint32, dpi float32) (com, error) {
	rtProps := d2d1RenderTargetProperties{
		targetType: d2d1RenderTargetTypeDefault,
		pixelFormat: d2d1PixelFormat{
			format:    dxgiFormatB8G8R8A8Unorm,
			alphaMode: d2d1AlphaModePremultiplied,
		},
		dpiX: dpi,
		dpiY: dpi,
	}
	hwndProps := d2d1HwndRenderTargetProperties{
		hwnd: hwnd,
		pixelSize: d2d1SizeU{
			width:  pxW,
			height: pxH,
		},
		presentOptions: d2d1PresentOptionsNone,
	}
	var rt com
	hr, _, _ := syscall.SyscallN(
		factory.slot(idxFactoryCreateHwndRenderTarget),
		uintptr(factory),
		uintptr(unsafe.Pointer(&rtProps)),
		uintptr(unsafe.Pointer(&hwndProps)),
		uintptr(unsafe.Pointer(&rt)),
	)
	runtime.KeepAlive(rtProps)
	runtime.KeepAlive(hwndProps)
	if hresult(hr).failed() || rt == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: CreateHwndRenderTarget 失败: %v", hresult(hr))
	}
	return rt, nil
}

func createSolidColorBrush(rt com) (com, error) {
	color := d2d1ColorF{0, 0, 0, 1}
	var brush com
	hr, _, _ := syscall.SyscallN(
		rt.slot(idxCreateSolidColorBrush),
		uintptr(rt),
		uintptr(unsafe.Pointer(&color)),
		0,
		uintptr(unsafe.Pointer(&brush)),
	)
	runtime.KeepAlive(color)
	if hresult(hr).failed() || brush == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: CreateSolidColorBrush 失败: %v", hresult(hr))
	}
	return brush, nil
}

// createTextFormat 建一个 IDWriteTextFormat。fontStyle 取 dwriteFontStyleNormal /
// dwriteFontStyleItalic —— 斜体在**建 format 时**决定，建完改不了。
func createTextFormat(dwrite com, family *uint16, size float32, weight, fontStyle int, locale *uint16) (com, error) {
	if weight <= 0 {
		weight = dwriteFontWeightNormal
	}
	var format com
	// fontSize 是第 7 个参数（下标 6），在 amd64 上走栈，IEEE 位模式可以直接传。
	hr, _, _ := syscall.SyscallN(
		dwrite.slot(idxCreateTextFormat),
		uintptr(dwrite),
		uintptr(unsafe.Pointer(family)),
		0,
		uintptr(uint32(weight)),
		uintptr(uint32(fontStyle)),
		uintptr(dwriteFontStretchNormal),
		uintptr(math.Float32bits(size)),
		uintptr(unsafe.Pointer(locale)),
		uintptr(unsafe.Pointer(&format)),
	)
	runtime.KeepAlive(family)
	runtime.KeepAlive(locale)
	if hresult(hr).failed() || format == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: CreateTextFormat 失败: %v", hresult(hr))
	}
	syscall.SyscallN(format.slot(idxSetWordWrapping), uintptr(format), uintptr(dwriteWordWrappingCharacter))
	return format, nil
}

func createTextLayout(dwrite com, text *uint16, length uint32, format com, maxW, maxH float32) (com, error) {
	var layout com
	hr, _, _ := syscall.SyscallN(
		dwrite.slot(idxCreateTextLayout),
		uintptr(dwrite),
		uintptr(unsafe.Pointer(text)),
		uintptr(length),
		uintptr(format),
		uintptr(math.Float32bits(maxW)),
		uintptr(math.Float32bits(maxH)),
		uintptr(unsafe.Pointer(&layout)),
	)
	runtime.KeepAlive(text)
	if hresult(hr).failed() || layout == 0 {
		return 0, fmt.Errorf("github.com/goedui/goed/ui: CreateTextLayout 失败: %v", hresult(hr))
	}
	return layout, nil
}

func textLayoutMetrics(layout com) (dwriteTextMetrics, error) {
	var m dwriteTextMetrics
	hr, _, _ := syscall.SyscallN(
		layout.slot(idxGetMetrics),
		uintptr(layout),
		uintptr(unsafe.Pointer(&m)),
	)
	if hresult(hr).failed() {
		return m, hresult(hr)
	}
	return m, nil
}
