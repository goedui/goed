//go:build windows

package windows

import "testing"

func TestSignedWordPreservesNegativeScreenCoordinates(t *testing.T) {
	const packed = uintptr(uint32(0xfff0) | uint32(0x0018)<<16)
	x, y := clientPointFromLParam(packed)
	if x != -16 || y != 24 {
		t.Fatalf("clientPointFromLParam() = (%d, %d), want (-16, 24)", x, y)
	}
}

func TestClientPointFromLParam(t *testing.T) {
	const packed = uintptr(uint32(320) | uint32(40)<<16)
	x, y := clientPointFromLParam(packed)
	if x != 320 || y != 40 {
		t.Fatalf("clientPointFromLParam() = (%d, %d), want (320, 40)", x, y)
	}
}

// TestWindowClassRegistersArrowCursor 是鼠标转圈 bug 的回归测试：
// 窗口类若不注册光标（HCursor == NULL），Windows 不会把进程启动时的
// busy/app-starting 光标恢复成箭头，启动后鼠标会一直转圈。
func TestWindowClassRegistersArrowCursor(t *testing.T) {
	w, err := NewFrameless("cursor-test", 200, 120)
	if err != nil {
		t.Skipf("无法创建测试窗口: %v", err)
	}
	defer DestroyWindow(w.hwnd)

	if w.hCursor == 0 {
		t.Fatal("窗口应持有箭头光标句柄 LoadCursor(IDC_ARROW)")
	}
	if classCursor := GetClassCursor(w.hwnd); classCursor == 0 {
		t.Fatal("窗口类 HCursor 未注册（为 NULL），启动后鼠标会一直保持转圈状态")
	}
}

func TestWindowClassRegistersGoedIcons(t *testing.T) {
	w, err := NewFrameless("icon-test", 200, 120)
	if err != nil {
		t.Skipf("无法创建测试窗口: %v", err)
	}
	defer DestroyWindow(w.hwnd)

	if icon := GetClassIcon(w.hwnd, GCLP_HICON); icon == 0 {
		t.Fatal("窗口类未注册 Goed 大图标")
	}
	if icon := GetClassIcon(w.hwnd, GCLP_HICONSM); icon == 0 {
		t.Fatal("窗口类未注册 Goed 小图标")
	}
}

func TestWMCloseInvokesOnCloseWithoutDestroying(t *testing.T) {
	w, err := NewFrameless("close-test", 200, 120)
	if err != nil {
		t.Skipf("无法创建测试窗口: %v", err)
	}
	defer DestroyWindow(w.hwnd)

	called := false
	w.SetOnClose(func() { called = true })
	if ret := w.wndProc(w.hwnd, WM_CLOSE, 0, 0); ret != 0 {
		t.Fatalf("WM_CLOSE with onClose should return 0, got %d", ret)
	}
	if !called {
		t.Fatal("SetOnClose callback must run on WM_CLOSE (Alt+F4 / 系统关闭)")
	}
	if !IsWindow(w.hwnd) {
		t.Fatal("WM_CLOSE interceptor must not destroy the window until Close() is called")
	}
}
