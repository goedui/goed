package windows

import (
	"syscall"
	"unsafe"
)

const (
	WS_POPUP       = 0x80000000
	WS_THICKFRAME  = 0x00040000
	WS_CAPTION     = 0x00C00000
	WS_SYSMENU     = 0x00080000
	WS_MINIMIZEBOX = 0x00020000
	WS_MAXIMIZEBOX = 0x00010000
	WM_NCHITTEST   = 0x0084
	WM_NCCALCSIZE  = 0x0083
	WM_NCPAINT     = 0x0085
	WM_NCACTIVATE  = 0x0086
	WM_ERASEBKGND  = 0x0014
	WM_LBUTTONDOWN = 0x0201
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_CHAR        = 0x0102
	WM_MOUSEWHEEL  = 0x020A
	HTCAPTION      = 2
	HTCLIENT       = 1
	HTTOP          = 12
	HTBOTTOM       = 15
	HTLEFT         = 10
	HTRIGHT        = 11
	HTTOPLEFT      = 13
	HTTOPRIGHT     = 14
	HTBOTTOMLEFT   = 16
	HTBOTTOMRIGHT  = 17
)

// Virtual Key Codes (exported)
const (
	VK_BACK    = 0x08
	VK_TAB     = 0x09
	VK_RETURN  = 0x0D
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_ESCAPE  = 0x1B
	VK_SPACE   = 0x20
	VK_PRIOR   = 0x21 // Page Up
	VK_NEXT    = 0x22 // Page Down
	VK_END     = 0x23
	VK_HOME    = 0x24
	VK_LEFT    = 0x25
	VK_UP      = 0x26
	VK_RIGHT   = 0x27
	VK_DOWN    = 0x28
	VK_DELETE  = 0x2E
	VK_F11     = 0x7A
)

type FramelessWindow struct {
	hwnd           syscall.Handle
	instance       syscall.Handle
	hCursor        syscall.Handle // 类光标（箭头），缺少它 Windows 会一直显示启动转圈光标
	onPaint        func(syscall.Handle)
	onMouseMove    func(x, y int32)
	onMouseLeave   func()
	trackingMouse  bool
	onMouseDown    func(x, y int32, button int)
	onMouseUp      func(x, y int32, button int)
	onMouseWheel   func(delta, x, y int32)
	onKeyDown      func(key int32)
	onKeyUp        func(key int32)
	onChar         func(char rune)
	onResize       func(width, height int32)
	onClose        func()
	onDestroy      func()
	isDragRegion   func(x, y int32) bool
	borderWidth    int32
	titleBarHeight int32
	fullscreen     bool
	restoreRect    RECT
	width          int32
	height         int32
}

func NewFrameless(title string, width, height int32) (*FramelessWindow, error) {
	instance, err := GetModuleHandle()
	if err != nil {
		return nil, err
	}

	w := &FramelessWindow{
		instance:       instance,
		borderWidth:    8,
		titleBarHeight: 40,
		width:          width,
		height:         height,
	}

	// 修复启动时鼠标转圈：注册类光标。若类光标为 NULL，Windows 不会
	// 自动把启动时的 busy 光标恢复成箭头，导致窗口内一直显示转圈。
	w.hCursor = LoadCursor(IDC_ARROW)

	className := syscall.StringToUTF16Ptr("GoedFramelessWindow")
	icons := loadGoedIcons()

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(w.wndProc),
		HInstance:     instance,
		HIcon:         icons.large,
		HCursor:       w.hCursor,
		HbrBackground: CreateSolidBrush(RGB(31, 31, 31)),
		LpszClassName: className,
		HIconSm:       icons.small,
	}

	if err := RegisterClassEx(&wc); err != nil {
		return nil, err
	}

	hwnd, err := CreateWindowEx(
		className,
		syscall.StringToUTF16Ptr(title),
		// Keep the window hidden until DWM attributes are applied below. Showing
		// it during CreateWindowEx lets Windows paint a default white frame first.
		WS_POPUP|WS_THICKFRAME,
		100, 100,
		width, height,
		0, 0, instance,
	)
	if err != nil {
		return nil, err
	}

	setFramelessDWMAttributes(hwnd)
	w.hwnd = hwnd
	return w, nil
}

func (w *FramelessWindow) wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_NCCALCSIZE:
		// Remove the system non-client frame. WS_THICKFRAME is retained for
		// resize hit-testing, which is implemented below in WM_NCHITTEST.
		return 0

	case WM_NCPAINT:
		// The entire window is painted by the client backbuffer.
		return 0

	case WM_NCACTIVATE:
		if wparam == 0 {
			w.clearMouseTracking(hwnd)
		}
		// Do not let activation changes make Windows repaint a system frame.
		return 1

	case WM_ERASEBKGND:
		// The backbuffer paints the complete client area.
		return 1

	case WM_NCHITTEST:
		return w.handleHitTest(lparam)

	case WM_SETCURSOR:
		// 客户区内强制箭头光标（双保险）；边框/标题栏交给系统处理，
		// 以保留调整大小和拖动的方向光标。
		if w.hCursor != 0 && syscall.Handle(wparam) == hwnd && int32(lparam&0xFFFF) == HTCLIENT {
			SetCursor(w.hCursor)
			return 1
		}
		return DefWindowProc(hwnd, msg, wparam, lparam)

	case WM_LBUTTONDOWN:
		SetFocus(hwnd)
		x, y := clientPointFromLParam(lparam)
		if w.onMouseDown != nil {
			w.onMouseDown(x, y, 0) // 0 = left button
		}

		return 0

	case WM_LBUTTONUP:
		x, y := clientPointFromLParam(lparam)
		if w.onMouseUp != nil {
			w.onMouseUp(x, y, 0) // 0 = left button
		}
		return 0

	case WM_RBUTTONDOWN:
		SetFocus(hwnd)
		x, y := clientPointFromLParam(lparam)
		if w.onMouseDown != nil {
			w.onMouseDown(x, y, 1) // 1 = right button
		}
		return 0

	case WM_RBUTTONUP:
		x, y := clientPointFromLParam(lparam)
		if w.onMouseUp != nil {
			w.onMouseUp(x, y, 1) // 1 = right button
		}
		return 0

	case WM_MOUSEMOVE:
		w.trackMouseLeave(hwnd)
		x, y := clientPointFromLParam(lparam)
		if w.onMouseMove != nil {
			w.onMouseMove(x, y)
		}
		return 0

	case wmMouseLeave:
		w.clearMouseTracking(hwnd)
		return 0

	case wmNCMouseMove:
		// Caption drag pixels and resize borders do not send WM_MOUSEMOVE.
		w.clearMouseTracking(hwnd)
		return DefWindowProc(hwnd, msg, wparam, lparam)

	case WM_MOUSEWHEEL:
		// wparam 高位字是滚轮增量；lParam 是屏幕坐标（不同于鼠标消息的
		// 客户区坐标），需转换为客户区坐标供组件做区域路由。
		delta := int32(int16(wparam >> 16))
		if w.onMouseWheel != nil {
			pt := POINT{X: signedWord(lparam), Y: signedWord(lparam >> 16)}
			ScreenToClient(w.hwnd, &pt)
			w.onMouseWheel(delta, pt.X, pt.Y)
		}
		return 0

	case WM_SIZE:
		width := int32(lparam & 0xFFFF)
		height := int32((lparam >> 16) & 0xFFFF)
		w.width, w.height = width, height
		if w.onResize != nil {
			w.onResize(width, height)
		}

	case WM_GSH_REFRESH:
		// Delayed fullscreen refreshes are posted back to the window thread so
		// application layout state is never mutated from a timer goroutine.
		if w.onResize != nil {
			var rect RECT
			GetClientRect(w.hwnd, &rect)
			w.onResize(rect.Right-rect.Left, rect.Bottom-rect.Top)
		}
		InvalidateRect(w.hwnd)
		return 0

	case WM_KEYDOWN:
		if w.onKeyDown != nil {
			w.onKeyDown(int32(wparam))
		}
		return 0

	case WM_KEYUP:
		if w.onKeyUp != nil {
			w.onKeyUp(int32(wparam))
		}
		return 0

	case WM_CHAR:
		if w.onChar != nil {
			w.onChar(rune(wparam))
		}
		return 0

	case WM_CLOSE:
		// 应用层可拦截关闭（未保存确认）。回调内再 DestroyWindow 才真正退出；
		// 未设置回调时走 DefWindowProc，保持系统默认销毁行为。
		if w.onClose != nil {
			w.onClose()
			return 0
		}

	case WM_DESTROY:
		if w.onDestroy != nil {
			w.onDestroy()
		}
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

func (w *FramelessWindow) handleHitTest(lparam uintptr) uintptr {
	var rect RECT
	GetClientRect(w.hwnd, &rect)

	point := POINT{X: signedWord(lparam), Y: signedWord(lparam >> 16)}
	if !ScreenToClient(w.hwnd, &point) {
		return HTCLIENT
	}
	x, y := point.X, point.Y

	border := w.borderWidth
	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top

	// Corners
	if x < border && y < border {
		return HTTOPLEFT
	}
	if x >= width-border && y < border {
		return HTTOPRIGHT
	}
	if x < border && y >= height-border {
		return HTBOTTOMLEFT
	}
	if x >= width-border && y >= height-border {
		return HTBOTTOMRIGHT
	}

	// Edges
	if x < border {
		return HTLEFT
	}
	if x >= width-border {
		return HTRIGHT
	}
	if y < border {
		return HTTOP
	}
	if y >= height-border {
		return HTBOTTOM
	}

	// Keep custom controls in the client area. Other title-bar pixels use
	// the native caption hit-test, which provides system drag behavior.
	if y >= 0 && y < w.titleBarHeight && (w.isDragRegion == nil || w.isDragRegion(x, y)) {
		return HTCAPTION
	}

	return HTCLIENT
}

func (w *FramelessWindow) SetOnPaint(fn func(syscall.Handle)) {
	w.onPaint = fn
}

func (w *FramelessWindow) SetOnMouseMove(fn func(x, y int32)) {
	w.onMouseMove = fn
}

func (w *FramelessWindow) SetOnMouseDown(fn func(x, y int32, button int)) {
	w.onMouseDown = fn
}

func (w *FramelessWindow) SetTitle(title string) {
	if w != nil && w.hwnd != 0 {
		SetWindowText(w.hwnd, title)
	}
}

func (w *FramelessWindow) SetOnMouseUp(fn func(x, y int32, button int)) {
	w.onMouseUp = fn
}

func (w *FramelessWindow) SetOnMouseWheel(fn func(delta, x, y int32)) {
	w.onMouseWheel = fn
}

func (w *FramelessWindow) SetOnKeyDown(fn func(key int32)) {
	w.onKeyDown = fn
}

func (w *FramelessWindow) SetOnKeyUp(fn func(key int32)) {
	w.onKeyUp = fn
}

func (w *FramelessWindow) SetOnChar(fn func(char rune)) {
	w.onChar = fn
}

func (w *FramelessWindow) SetOnResize(fn func(width, height int32)) {
	w.onResize = fn
}

func (w *FramelessWindow) SetOnClose(fn func()) {
	w.onClose = fn
}

func (w *FramelessWindow) SetOnDestroy(fn func()) {
	w.onDestroy = fn
}

func (w *FramelessWindow) SetDragRegion(fn func(x, y int32) bool) {
	w.isDragRegion = fn
}

func (w *FramelessWindow) SetTitleBarHeight(height int32) {
	if height >= 0 {
		w.titleBarHeight = height
	}
}

func (w *FramelessWindow) Show() {
	ShowWindow(w.hwnd, SW_SHOW)
}

// ClientSize returns the current drawable client area. The first WM_SIZE can
// arrive while the window is being created, before an application registers
// its resize callback, so callers should use this during painting as well.
func (w *FramelessWindow) ClientSize() (int32, int32) {
	if w == nil || w.hwnd == 0 {
		return 0, 0
	}
	var rect RECT
	GetClientRect(w.hwnd, &rect)
	return rect.Right - rect.Left, rect.Bottom - rect.Top
}

func (w *FramelessWindow) Minimize() {
	ShowWindow(w.hwnd, SW_MINIMIZE)
}

func (w *FramelessWindow) ToggleMaximize() {
	if IsZoomed(w.hwnd) {
		ShowWindow(w.hwnd, SW_RESTORE)
	} else {
		ShowWindow(w.hwnd, SW_MAXIMIZE)
	}
}

// Close 强制销毁窗口。未保存拦截应走 SetOnClose / WM_CLOSE，确认后再调用 Close。
func (w *FramelessWindow) Close() {
	DestroyWindow(w.hwnd)
}

func (w *FramelessWindow) Invalidate() {
	InvalidateRect(w.hwnd)
}

func (w *FramelessWindow) PostResizeRefresh() {
	if w != nil && w.hwnd != 0 {
		PostMessage(w.hwnd, WM_GSH_REFRESH, 0, 0)
	}
}

func (w *FramelessWindow) Run() error {
	var msg MSG
	for GetMessage(&msg, 0) {
		TranslateMessage(&msg)
		DispatchMessage(&msg)
	}
	return nil
}

func signedWord(value uintptr) int32 {
	return int32(int16(value & 0xffff))
}

func clientPointFromLParam(lparam uintptr) (int32, int32) {
	return signedWord(lparam), signedWord(lparam >> 16)
}
