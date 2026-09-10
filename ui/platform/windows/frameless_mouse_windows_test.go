package windows

import (
	"runtime"
	"syscall"
	"testing"
)

func TestFramelessRegistersAndRearmsNativeMouseTracking(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	instance, err := GetModuleHandle()
	if err != nil {
		t.Fatal(err)
	}
	// A hidden built-in window avoids app-class registration and desktop input.
	hwnd, err := CreateWindowEx(syscall.StringToUTF16Ptr("STATIC"), syscall.StringToUTF16Ptr("mouse-tracking-test"), WS_POPUP, 0, 0, 200, 120, 0, 0, instance)
	if err != nil {
		t.Skipf("cannot create hidden test window: %v", err)
	}
	defer DestroyWindow(hwnd)
	w := &FramelessWindow{hwnd: hwnd}
	moves, leaves := 0, 0
	w.SetOnMouseMove(func(x, y int32) { moves++ })
	w.SetOnMouseLeave(func() { leaves++ })
	for n := 1; n <= 2; n++ {
		w.wndProc(hwnd, WM_MOUSEMOVE, 0, uintptr(20|20<<16))
		if !w.trackingMouse || moves != n {
			t.Fatal("native leave tracking was not armed on mouse move")
		}
		w.wndProc(hwnd, wmMouseLeave, 0, 0)
		if w.trackingMouse || leaves != n {
			t.Fatal("native leave was not delivered")
		}
	}
}

func TestFramelessMouseExitNotifications(t *testing.T) {
	for _, msg := range []uint32{wmMouseLeave, wmNCMouseMove, WM_NCACTIVATE} {
		t.Run(map[uint32]string{wmMouseLeave: "outside", wmNCMouseMove: "caption-or-border", WM_NCACTIVATE: "deactivate"}[msg], func(t *testing.T) {
			w := &FramelessWindow{trackingMouse: true}
			calls := 0
			w.SetOnMouseLeave(func() { calls++ })
			w.wndProc(0, msg, 0, 0)
			if w.trackingMouse || calls != 1 {
				t.Fatalf("tracking=%v calls=%d", w.trackingMouse, calls)
			}
			w.wndProc(0, wmMouseLeave, 0, 0)
			if calls != 1 {
				t.Fatal("duplicate leave notification")
			}
			// A subsequent successful TrackMouseEvent registration rearms exit delivery.
			w.trackingMouse = true
			w.wndProc(0, wmMouseLeave, 0, 0)
			if calls != 2 {
				t.Fatal("re-entry did not rearm leave notification")
			}
		})
	}
}
