package windows

import (
	"syscall"
	"unsafe"
)

const (
	wmMouseLeave  = 0x02A3
	wmNCMouseMove = 0x00A0
	tmeLeave      = 0x00000002
	tmeCancel     = 0x80000000
)

type trackMouseEvent struct {
	Size      uint32
	Flags     uint32
	Window    syscall.Handle
	HoverTime uint32
}

var procTrackMouseEvent = user32.NewProc("TrackMouseEvent")

// SetOnMouseLeave registers notification for leaving the client area.
func (w *FramelessWindow) SetOnMouseLeave(fn func()) { w.onMouseLeave = fn }

func (w *FramelessWindow) trackMouseLeave(hwnd syscall.Handle) {
	if w.trackingMouse {
		return
	}
	event := trackMouseEvent{Size: uint32(unsafe.Sizeof(trackMouseEvent{})), Flags: tmeLeave, Window: hwnd}
	result, _, _ := procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&event)))
	w.trackingMouse = result != 0
}

func (w *FramelessWindow) clearMouseTracking(hwnd syscall.Handle) {
	if !w.trackingMouse {
		return
	}
	w.trackingMouse = false
	event := trackMouseEvent{Size: uint32(unsafe.Sizeof(trackMouseEvent{})), Flags: tmeLeave | tmeCancel, Window: hwnd}
	procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&event)))
	if w.onMouseLeave != nil {
		w.onMouseLeave()
	}
}
