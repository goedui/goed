package windows

import (
	"syscall"
	"unsafe"
)

type Window struct {
	hwnd     syscall.Handle
	instance syscall.Handle
	onPaint  func(syscall.Handle)
}

func New(title string, width, height int32) (*Window, error) {
	instance, err := GetModuleHandle()
	if err != nil {
		return nil, err
	}

	w := &Window{instance: instance}

	className := syscall.StringToUTF16Ptr("GoedWindowClass")
	icons := loadGoedIcons()

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(w.wndProc),
		HInstance:     instance,
		HIcon:         icons.large,
		LpszClassName: className,
		HIconSm:       icons.small,
	}

	if err := RegisterClassEx(&wc); err != nil {
		return nil, err
	}

	hwnd, err := CreateWindowEx(
		className,
		syscall.StringToUTF16Ptr(title),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		100, 100,
		width, height,
		0, 0, instance,
	)
	if err != nil {
		return nil, err
	}

	w.hwnd = hwnd
	return w, nil
}

func (w *Window) wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_DESTROY:
		PostQuitMessage(0)
		return 0
	case WM_PAINT:
		if w.onPaint != nil {
			var ps PAINTSTRUCT
			hdc := BeginPaint(hwnd, &ps)
			w.onPaint(hdc)
			EndPaint(hwnd, &ps)
			return 0
		}
	}
	return DefWindowProc(hwnd, msg, wparam, lparam)
}

func (w *Window) SetOnPaint(fn func(syscall.Handle)) {
	w.onPaint = fn
}

func (w *Window) Show() {
	ShowWindow(w.hwnd, SW_SHOW)
}

func (w *Window) Run() error {
	var msg MSG
	for GetMessage(&msg, 0) {
		DispatchMessage(&msg)
	}
	return nil
}
