package windows

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procScreenToClient   = user32.NewProc("ScreenToClient")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procPostMessageW     = user32.NewProc("PostMessageW")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procIsWindow         = user32.NewProc("IsWindow")
	procIsZoomed         = user32.NewProc("IsZoomed")
	procFillRect         = user32.NewProc("FillRect")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procSetFocus         = user32.NewProc("SetFocus")
	procGetKeyState      = user32.NewProc("GetKeyState")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procSetCursor        = user32.NewProc("SetCursor")
	procGetClassLongPtrW = user32.NewProc("GetClassLongPtrW")
	procCreateIcon       = user32.NewProc("CreateIcon")
	procMonitorFromWin   = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW  = user32.NewProc("GetMonitorInfoW")
	procDwmSetWindowAttr = dwmapi.NewProc("DwmSetWindowAttribute")

	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procTextOutW         = gdi32.NewProc("TextOutW")
	procGetTextMetricsW  = gdi32.NewProc("GetTextMetricsW")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkColor       = gdi32.NewProc("SetBkColor")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procCreateRectRgn    = gdi32.NewProc("CreateRectRgn")
	procSelectClipRgn    = gdi32.NewProc("SelectClipRgn")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procShellExecuteW    = shell32.NewProc("ShellExecuteW")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	SW_SHOW             = 5
	SW_MINIMIZE         = 6
	SW_MAXIMIZE         = 3
	SW_RESTORE          = 9
	SW_SHOWNORMAL       = 1
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
	WM_DESTROY          = 0x0002
	WM_PAINT            = 0x000F
	WM_SIZE             = 0x0005
	WM_APP              = 0x8000
	WM_GSH_REFRESH      = WM_APP + 1
	WM_NCLBUTTONDOWN    = 0x00A1
	WM_CLOSE            = 0x0010
	CS_HREDRAW          = 0x0002
	CS_VREDRAW          = 0x0001
	IDC_ARROW           = 32512
	WM_SETCURSOR        = 0x0020
	GCLP_HICON          = ^uintptr(13) // -14: window class large icon
	GCLP_HICONSM        = ^uintptr(33) // -34: window class small icon
	GCLP_HCURSOR        = ^uintptr(11) // -12: 类光标句柄（64 位用 GetClassLongPtrW）
	DWMWA_BORDER_COLOR  = 34
	DWMWA_CORNER_PREF   = 33
	DWMWA_NC_POLICY     = 2
	DWMWA_COLOR_NONE    = 0xFFFFFFFE
	DWMNCRP_DISABLED    = 1
	DWMCP_DONOTROUND    = 1
	CW_USEDEFAULT       = 0x80000000
	TRANSPARENT         = 1
	SWP_NOZORDER        = 0x0004
	SWP_NOACTIVATE      = 0x0010
	SWP_FRAMECHANGED    = 0x0020
	HOLLOW_BRUSH        = 5
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X int32
	Y int32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

func GetModuleHandle() (syscall.Handle, error) {
	ret, _, err := procGetModuleHandleW.Call(0)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func RegisterClassEx(wc *WNDCLASSEXW) error {
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if ret == 0 {
		return err
	}
	return nil
}

func CreateWindowEx(className, title *uint16, style uint32, x, y, width, height int32, parent, menu, instance syscall.Handle) (syscall.Handle, error) {
	ret, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		0,
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func DefWindowProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return ret
}

func GetMessage(msg *MSG, hwnd syscall.Handle) bool {
	ret, _, _ := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		0,
		0,
	)
	return ret != 0
}

func TranslateMessage(msg *MSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func DispatchMessage(msg *MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func PostQuitMessage(exitCode int) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func SetFocus(hwnd syscall.Handle) {
	procSetFocus.Call(uintptr(hwnd))
}

func IsKeyDown(key int32) bool {
	ret, _, _ := procGetKeyState.Call(uintptr(key))
	return int16(ret) < 0
}

func ShowWindow(hwnd syscall.Handle, cmd int) {
	procShowWindow.Call(uintptr(hwnd), uintptr(cmd))
}

func DestroyWindow(hwnd syscall.Handle) bool {
	ret, _, _ := procDestroyWindow.Call(uintptr(hwnd))
	return ret != 0
}

func IsWindow(hwnd syscall.Handle) bool {
	ret, _, _ := procIsWindow.Call(uintptr(hwnd))
	return ret != 0
}

func IsZoomed(hwnd syscall.Handle) bool {
	ret, _, _ := procIsZoomed.Call(uintptr(hwnd))
	return ret != 0
}

func GetDC(hwnd syscall.Handle) syscall.Handle {
	ret, _, _ := procGetDC.Call(uintptr(hwnd))
	return syscall.Handle(ret)
}

func ReleaseDC(hwnd, hdc syscall.Handle) {
	procReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
}

func BeginPaint(hwnd syscall.Handle, ps *PAINTSTRUCT) syscall.Handle {
	ret, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps)))
	return syscall.Handle(ret)
}

func EndPaint(hwnd syscall.Handle, ps *PAINTSTRUCT) {
	procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ps)))
}

func GetClientRect(hwnd syscall.Handle, rect *RECT) {
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
}

func GetWindowRect(hwnd syscall.Handle, rect *RECT) bool {
	ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	return ret != 0
}

func ScreenToClient(hwnd syscall.Handle, point *POINT) bool {
	ret, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(point)))
	return ret != 0
}

func SetWindowPos(hwnd syscall.Handle, x, y, width, height int32) bool {
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd), 0,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		uintptr(SWP_NOZORDER|SWP_NOACTIVATE|SWP_FRAMECHANGED),
	)
	return ret != 0
}

func PostMessage(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) bool {
	ret, _, _ := procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return ret != 0
}

func SetWindowText(hwnd syscall.Handle, title string) bool {
	ptr := syscall.StringToUTF16Ptr(title)
	ret, _, _ := procSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(ptr)))
	return ret != 0
}

func GetSystemMetrics(index int32) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(ret)
}

func FillRect(hdc syscall.Handle, rect *RECT, brush syscall.Handle) {
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rect)), uintptr(brush))
}

const (
	fontFixedPitch   = 1
	fontModernFamily = 0x30
)

func fontPitch(name string) uintptr {
	if strings.Contains(strings.ToLower(name), "cascadia") || strings.EqualFold(name, "consolas") {
		return uintptr(fontFixedPitch | fontModernFamily)
	}
	// FF_SWISS (0x20) with DEFAULT_PITCH is the correct hint for Segoe UI and
	// other proportional interface fonts.
	return uintptr(0x20)
}

func SelectObject(hdc, obj syscall.Handle) syscall.Handle {
	ret, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(obj))
	return syscall.Handle(ret)
}

func TextOut(hdc syscall.Handle, x, y int32, text string) {
	utf16Text := syscall.StringToUTF16(sanitizeWindowsText(text))
	procTextOutW.Call(
		uintptr(hdc),
		uintptr(x),
		uintptr(y),
		uintptr(unsafe.Pointer(&utf16Text[0])),
		uintptr(len(utf16Text)-1),
	)
}

// OpenExternalTerminal asks the Windows shell to create a real console
// window. ShellExecute avoids inheriting Goed's console/window state, which
// can leave a directly-started cmd.exe invisible or attached to the wrong
// host.
func OpenExternalTerminal(workdir string) error {
	if workdir == "" {
		workdir = "."
	}
	operation := syscall.StringToUTF16Ptr("open")
	file := syscall.StringToUTF16Ptr("cmd.exe")
	params := syscall.StringToUTF16Ptr("/K")
	directory := syscall.StringToUTF16Ptr(workdir)
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(operation)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(directory)),
		SW_SHOWNORMAL,
	)
	if ret <= 32 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("ShellExecuteW: %w", callErr)
		}
		return fmt.Errorf("ShellExecuteW failed with code %d", ret)
	}
	return nil
}

// Windows' StringToUTF16 rejects embedded NUL characters. Files opened in the
// editor can contain arbitrary bytes, so replace NUL with a visible marker at
// the rendering boundary instead of allowing a malformed token to crash UI.
func sanitizeWindowsText(text string) string {
	return strings.ReplaceAll(text, "\x00", "\uFFFD")
}

func SetTextColor(hdc syscall.Handle, color uint32) {
	procSetTextColor.Call(uintptr(hdc), uintptr(color))
}

func SetBkColor(hdc syscall.Handle, color uint32) {
	procSetBkColor.Call(uintptr(hdc), uintptr(color))
}

func CreateSolidBrush(color uint32) syscall.Handle {
	ret, _, _ := procCreateSolidBrush.Call(uintptr(color))
	return syscall.Handle(ret)
}

func RGB(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func InvalidateRect(hwnd syscall.Handle) {
	procInvalidateRect.Call(uintptr(hwnd), 0, 1) // NULL rect, erase background
}

// LoadCursor 加载系统预定义光标（instance=0 时 name 为 IDC_* 资源 ID）
func LoadCursor(name uintptr) syscall.Handle {
	ret, _, _ := procLoadCursorW.Call(0, name)
	return syscall.Handle(ret)
}

// SetCursor 设置当前光标，返回上一个光标
func SetCursor(cursor syscall.Handle) syscall.Handle {
	ret, _, _ := procSetCursor.Call(uintptr(cursor))
	return syscall.Handle(ret)
}

// GetClassCursor 读取窗口类的光标句柄（用于测试验证类光标已注册）
func GetClassCursor(hwnd syscall.Handle) syscall.Handle {
	ret, _, _ := procGetClassLongPtrW.Call(uintptr(hwnd), GCLP_HCURSOR)
	return syscall.Handle(ret)
}

func GetClassIcon(hwnd syscall.Handle, index uintptr) syscall.Handle {
	ret, _, _ := procGetClassLongPtrW.Call(uintptr(hwnd), index)
	return syscall.Handle(ret)
}

func createIcon(width, height int, andBits, xorBits []byte) syscall.Handle {
	if len(andBits) == 0 || len(xorBits) == 0 {
		return 0
	}
	ret, _, _ := procCreateIcon.Call(
		0,
		uintptr(width),
		uintptr(height),
		1,
		32,
		uintptr(unsafe.Pointer(&andBits[0])),
		uintptr(unsafe.Pointer(&xorBits[0])),
	)
	return syscall.Handle(ret)
}

// setFramelessDWMAttributes prevents the system inactive-window frame from
// adding a bright border around the custom-painted window.
func setFramelessDWMAttributes(hwnd syscall.Handle) {
	ncPolicy := uint32(DWMNCRP_DISABLED)
	procDwmSetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(DWMWA_NC_POLICY),
		uintptr(unsafe.Pointer(&ncPolicy)),
		unsafe.Sizeof(ncPolicy),
	)
	borderColor := uint32(DWMWA_COLOR_NONE)
	procDwmSetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(DWMWA_BORDER_COLOR),
		uintptr(unsafe.Pointer(&borderColor)),
		unsafe.Sizeof(borderColor),
	)
	cornerPreference := uint32(DWMCP_DONOTROUND)
	procDwmSetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(DWMWA_CORNER_PREF),
		uintptr(unsafe.Pointer(&cornerPreference)),
		unsafe.Sizeof(cornerPreference),
	)
}
