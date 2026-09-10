package windows

import "unsafe"

// monitorInfo 是 Win32 MONITORINFO 的最小绑定。
type monitorInfo struct {
	CbSize    uint32
	RcMonitor RECT // 显示器整体边界（虚拟桌面坐标）
	RcWork    RECT // 工作区（扣除任务栏）
	DwFlags   uint32
}

const monitorDefaultToNearest = 2

// monitorBounds 返回 hwnd 所在显示器的整体边界。多显示器下
// SM_CXSCREEN 只描述主屏且原点固定 (0,0)，扩展屏窗口必须用
// MonitorFromWindow 定位所在屏，否则全屏会跳到主显示器。
func monitorBounds(hwnd uintptr) (RECT, bool) {
	hmon, _, _ := procMonitorFromWin.Call(hwnd, monitorDefaultToNearest)
	if hmon == 0 {
		return RECT{}, false
	}
	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if ret, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi))); ret == 0 {
		return RECT{}, false
	}
	return mi.RcMonitor, true
}

// ToggleFullscreen switches between the saved window rectangle and the full
// monitor bounds. Frameless windows already own their title bar, so changing
// the bounds is sufficient and keeps the existing resize/layout callbacks.
func (w *FramelessWindow) ToggleFullscreen() {
	if w == nil || w.hwnd == 0 {
		return
	}
	if w.fullscreen {
		r := w.restoreRect
		SetWindowPos(w.hwnd, r.Left, r.Top, r.Right-r.Left, r.Bottom-r.Top)
		ShowWindow(w.hwnd, SW_SHOW)
		w.fullscreen = false
		return
	}
	var r RECT
	if !GetWindowRect(w.hwnd, &r) {
		return
	}
	w.restoreRect = r
	monitor, ok := monitorBounds(uintptr(w.hwnd))
	if !ok {
		// 极端回退：主屏（原路径）。
		width := GetSystemMetrics(SM_CXSCREEN)
		height := GetSystemMetrics(SM_CYSCREEN)
		if width <= 0 || height <= 0 {
			return
		}
		SetWindowPos(w.hwnd, 0, 0, width, height)
		ShowWindow(w.hwnd, SW_SHOW)
		w.fullscreen = true
		return
	}
	SetWindowPos(w.hwnd, monitor.Left, monitor.Top, monitor.Right-monitor.Left, monitor.Bottom-monitor.Top)
	ShowWindow(w.hwnd, SW_SHOW)
	w.fullscreen = true
}

func (w *FramelessWindow) IsFullscreen() bool {
	return w != nil && w.fullscreen
}
