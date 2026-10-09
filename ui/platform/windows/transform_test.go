//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

// ===== 矩阵数学（无 COM，任何环境都能跑） =====

// TestRotationMatrix 旋转矩阵的旋转块、平移、中心不动三点都要对。
// 注意 Sincos(90°) 的 cos 是 6e-17 而不是精确 0，旋转块按误差比较 ——
// 这点偏差对渲染毫无影响，不值得为它做角度特判。
func TestRotationMatrix(t *testing.T) {
	// 90° 绕 (100, 50)：右侧的点转到下面（顺时针）。
	m := rotationMatrix(90, 100, 50)
	if !nearEq(m.m11, 0) || !nearEq(m.m12, 1) || !nearEq(m.m21, -1) || !nearEq(m.m22, 0) {
		t.Errorf("90° 的旋转块不对：%+v", m)
	}
	if m.m31 != 150 || m.m32 != -50 {
		t.Errorf("90° 的平移不对：(%v, %v)，期望 (150, -50)", m.m31, m.m32)
	}
	// 中心必须不动。
	x := m.m11*100 + m.m21*50 + m.m31
	y := m.m12*100 + m.m22*50 + m.m32
	if !nearEq(x, 100) || !nearEq(y, 50) {
		t.Errorf("旋转中心应不动：(%v, %v)", x, y)
	}

	// 315°（Word 水印角）：右侧的点往上走，读向是左下 → 右上。
	m = rotationMatrix(315, 0, 0)
	if m.m12 >= 0 {
		t.Errorf("315° 应把右侧的点转到上方（m12 = y 的系数应为负）：%v", m.m12)
	}

	// 0° 就是恒等（Sincos(0) 是精确值）。
	if m := rotationMatrix(0, 7, 9); m != d2d1IdentityMatrix {
		t.Errorf("0° 应为恒等矩阵：%+v", m)
	}
}

func nearEq(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}

// ===== vtable 往返（真 COM：SetTransform 写、GetTransform 读回） =====
//
// 槽位串了不会报错、只会跳到另一个函数上（本包 syscall.go 的老规矩）。
// 这条测试的价值在于：SetTransform(30) 写进去的矩阵能用 GetTransform(31)
// 原样读回来 —— 两个槽位只要有一个不对，读回来的就是垃圾。

func TestSetTransformRoundTrip(t *testing.T) {
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
		t.Skip("建不出渲染目标（隐藏窗口 / D2D 环境）")
	}
	defer release(rt)

	ctx := &d2dContext{rt: rt, dpi: 96}

	ctx.RotateAt(90, 100, 50)
	var m d2d1Matrix3x3F
	syscall.SyscallN(rt.slot(idxGetTransform), uintptr(rt), uintptr(unsafe.Pointer(&m)))
	if want := rotationMatrix(90, 100, 50); m != want {
		t.Fatalf("SetTransform(90°) 读回 %+v，期望 %+v —— 槽位可能串了", m, want)
	}

	ctx.ResetTransform()
	syscall.SyscallN(rt.slot(idxGetTransform), uintptr(rt), uintptr(unsafe.Pointer(&m)))
	if m != d2d1IdentityMatrix {
		t.Fatalf("ResetTransform 后应回到恒等矩阵：%+v", m)
	}
}

// testRenderTarget 建一个隐藏窗口与它上面的渲染目标，只服务槽位往返测试。
// 任何一步失败都返回 0（调用方 skip）：这条测试抓的是「槽位串位」，没有
// D2D 环境时跳过并不损失覆盖。
func testRenderTarget(t *testing.T, factory com) com {
	t.Helper()

	name, _ := syscall.UTF16FromString("goed-rt-test")
	inst := moduleHandle()
	wc := wndClassExW{
		lpfnWndProc:   procDefWindowProcW.Addr(),
		hInstance:     inst,
		lpszClassName: &name[0],
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	atom, _, regErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(wc)
	if atom == 0 {
		if errno, _ := regErr.(syscall.Errno); errno != errorClassAlreadyExists {
			t.Logf("RegisterClassExW 失败：%v", regErr)
			return 0
		}
	}

	// 不 ShowWindow：CreateHwndRenderTarget 只需要句柄，不需要可见。
	// 样式给 0（WS_OVERLAPPED 就是 0）—— 纯粹当 D2D 的渲染表面用。
	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(&name[0])),
		0, // 无标题
		0,
		0, 0, 320, 240,
		0, 0, inst, 0,
	)
	if hwnd == 0 {
		t.Logf("CreateWindowExW 失败：%v", callErr)
		return 0
	}
	defer procDestroyWindow.Call(hwnd)

	rt, err := createHwndRenderTarget(factory, hwnd, 320, 240, 96)
	if err != nil {
		t.Logf("CreateHwndRenderTarget 失败：%v", err)
		return 0
	}
	return rt
}
