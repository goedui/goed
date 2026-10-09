//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

// TestPathGeometryOpen 钉住 CreatePathGeometry / Open 的虚表槽位。
// 槽位串了不会报编译错，只会在运行时跳到别的函数上，所以必须真 COM 往返。
func TestPathGeometryOpen(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		t.Fatalf("CoInitializeEx: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()

	factory, err := d2d1CreateFactory()
	if err != nil {
		t.Skipf("Direct2D 不可用：%v", err)
	}
	defer release(factory)

	var path com
	hr, _, _ = syscall.SyscallN(
		factory.slot(idxFactoryCreatePathGeometry),
		uintptr(factory),
		uintptr(unsafe.Pointer(&path)),
	)
	if hresult(hr).failed() || path == 0 {
		t.Fatalf("CreatePathGeometry 槽位 %d 失败: %v", idxFactoryCreatePathGeometry, hresult(hr))
	}
	defer release(path)

	var sink com
	hr, _, _ = syscall.SyscallN(path.slot(idxPathOpen), uintptr(path), uintptr(unsafe.Pointer(&sink)))
	if hresult(hr).failed() || sink == 0 {
		t.Fatalf("PathGeometry::Open 槽位 %d 失败: %v", idxPathOpen, hresult(hr))
	}
	release(sink)
}

func TestCreateStrokeStyleSlot(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		t.Fatalf("CoInitializeEx: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()
	factory, err := d2d1CreateFactory()
	if err != nil {
		t.Skip(err)
	}
	defer release(factory)
	props := d2d1StrokeStyleProperties{
		startCap: d2d1CapStyleRound, endCap: d2d1CapStyleRound, dashCap: d2d1CapStyleRound,
		lineJoin: d2d1LineJoinRound, miterLimit: 10,
	}
	if unsafe.Sizeof(props) != 28 {
		t.Fatalf("stroke props 应为 28 字节，实际 %d", unsafe.Sizeof(props))
	}
	var style com
	hr, _, _ = syscall.SyscallN(
		factory.slot(idxFactoryCreateStrokeStyle),
		uintptr(factory),
		uintptr(unsafe.Pointer(&props)),
		0, 0,
		uintptr(unsafe.Pointer(&style)),
	)
	runtime.KeepAlive(props)
	if hresult(hr).failed() || style == 0 {
		t.Fatalf("CreateStrokeStyle hr=%v style=%v", hresult(hr), style)
	}
	release(style)
}

func TestRoundStrokeDraws(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		t.Fatalf("CoInitializeEx: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()

	factory, err := d2d1CreateFactory()
	if err != nil {
		t.Skipf("Direct2D 不可用：%v", err)
	}
	defer release(factory)
	rt := testRenderTarget(t, factory)
	if rt == 0 {
		t.Skip("建不出渲染目标")
	}
	defer release(rt)
	brush, err := createSolidColorBrush(rt)
	if err != nil {
		t.Fatal(err)
	}
	defer release(brush)

	ctx := &d2dContext{rt: rt, brush: brush, d2d: factory, dpi: 96}
	defer ctx.releaseStroke()
	defer ctx.releaseShadows()
	syscall.SyscallN(rt.slot(idxBeginDraw), uintptr(rt))
	ok := ctx.DrawRoundStroke([]renderer.StrokePoint{
		{X: 4, Y: 20},
		{X: 16, Y: 4, Curve: true, C1X: 8, C1Y: 20, C2X: 12, C2Y: 4},
		{X: 28, Y: 20},
	}, 2, renderer.RGB(40, 90, 220), false)
	hr, _, _ = syscall.SyscallN(rt.slot(idxEndDraw), uintptr(rt), 0, 0)
	if !ok || hresult(hr).failed() {
		t.Fatalf("圆头路径把设备画坏了: ok=%v hr=%v", ok, hresult(hr))
	}
	syscall.SyscallN(rt.slot(idxBeginDraw), uintptr(rt))
	ctx.FillShadowRoundedRect(20, 20, 80, 36, 12, 8, 4, renderer.RGB(255, 255, 255), renderer.RGBA(15, 23, 42, 80))
	renderer.StrokeRounded(ctx, 8, 8, 180, 36, 12, 1, renderer.RGB(47, 107, 242))
	hr, _, _ = syscall.SyscallN(rt.slot(idxEndDraw), uintptr(rt), 0, 0)
	if hresult(hr).failed() {
		t.Fatalf("阴影或圆角描边把设备画坏了: hr=%v", hresult(hr))
	}
}
