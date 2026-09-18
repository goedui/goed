//go:build windows

package windows

import (
	"syscall"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modOle32    = syscall.NewLazyDLL("ole32.dll")
	modD2D1     = syscall.NewLazyDLL("d2d1.dll")
	modDWrite   = syscall.NewLazyDLL("dwrite.dll")
	modShcore   = syscall.NewLazyDLL("shcore.dll")
	modDwmapi   = syscall.NewLazyDLL("dwmapi.dll")
)

var (
	procRegisterClassExW              = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW               = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow                 = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW                = modUser32.NewProc("DefWindowProcW")
	procShowWindow                    = modUser32.NewProc("ShowWindow")
	procPostMessageW                  = modUser32.NewProc("PostMessageW")
	procUpdateWindow                  = modUser32.NewProc("UpdateWindow")
	procGetMessageW                   = modUser32.NewProc("GetMessageW")
	procTranslateMessage              = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW              = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage               = modUser32.NewProc("PostQuitMessage")
	procBeginPaint                    = modUser32.NewProc("BeginPaint")
	procEndPaint                      = modUser32.NewProc("EndPaint")
	procInvalidateRect                = modUser32.NewProc("InvalidateRect")
	procGetClientRect                 = modUser32.NewProc("GetClientRect")
	procLoadCursorW                   = modUser32.NewProc("LoadCursorW")
	procLoadIconW                     = modUser32.NewProc("LoadIconW")
	procCreateIconFromResourceEx      = modUser32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon                   = modUser32.NewProc("DestroyIcon")
	procSendMessageW                  = modUser32.NewProc("SendMessageW")
	procGetSystemMetrics              = modUser32.NewProc("GetSystemMetrics")
	procGetSystemMetricsForDpi        = modUser32.NewProc("GetSystemMetricsForDpi")
	procSetWindowPos                  = modUser32.NewProc("SetWindowPos")
	procAdjustWindowRectEx            = modUser32.NewProc("AdjustWindowRectEx")
	procAdjustWindowRectExForDpi      = modUser32.NewProc("AdjustWindowRectExForDpi")
	procGetDpiForWindow               = modUser32.NewProc("GetDpiForWindow")
	procGetDpiForSystem               = modUser32.NewProc("GetDpiForSystem")
	procSetProcessDpiAwarenessContext = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procScreenToClient                = modUser32.NewProc("ScreenToClient")
	procClientToScreen                = modUser32.NewProc("ClientToScreen")
	procIsZoomed                      = modUser32.NewProc("IsZoomed")
	procTrackMouseEvent               = modUser32.NewProc("TrackMouseEvent")
	procGetWindowRect                 = modUser32.NewProc("GetWindowRect")
	procSetCapture                    = modUser32.NewProc("SetCapture")
	procReleaseCapture = modUser32.NewProc("ReleaseCapture")
	procGetKeyState    = modUser32.NewProc("GetKeyState")
	procSetWindowTextW = modUser32.NewProc("SetWindowTextW")

	procGetModuleHandleW         = modKernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId       = modKernel32.NewProc("GetCurrentThreadId")
	procGetUserDefaultLocaleName = modKernel32.NewProc("GetUserDefaultLocaleName")

	procCoInitializeEx = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize = modOle32.NewProc("CoUninitialize")

	procD2D1CreateFactory   = modD2D1.NewProc("D2D1CreateFactory")
	procDWriteCreateFactory = modDWrite.NewProc("DWriteCreateFactory")

	procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")

	procDwmExtendFrameIntoClientArea = modDwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procDwmSetWindowAttribute        = modDwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsClipChildren     = 0x02000000
	wsClipSiblings     = 0x04000000

	swShow     = 5
	swMaximize = 3
	swMinimize = 6
	swRestore  = 9

	wmDestroy         = 0x0002
	wmSize            = 0x0005
	wmActivate        = 0x0006
	wmSetFocus        = 0x0007
	wmPaint           = 0x000F
	wmClose           = 0x0010
	wmQuit            = 0x0012
	wmEraseBkgnd      = 0x0014
	wmNcCreate        = 0x0081
	wmNcCalcSize      = 0x0083
	wmNcHitTest       = 0x0084
	wmNcPaint         = 0x0085
	wmNcActivate      = 0x0086
	wmNcMouseMove     = 0x00A0
	wmNcLButtonDown   = 0x00A1
	wmNcLButtonUp     = 0x00A2
	wmNcLButtonDblClk = 0x00A3
	wmMouseMove       = 0x0200
	wmLButtonDown     = 0x0201
	wmLButtonUp       = 0x0202
	wmKeyDown         = 0x0100
	wmKeyUp           = 0x0101
	wmChar            = 0x0102
	wmSysKeyDown      = 0x0104
	wmSysKeyUp        = 0x0105
	wmMouseWheel      = 0x020A
	wmNcMouseLeave    = 0x02A2
	wmAppPost         = 0x8001
	vkShift           = 0x10
	vkControl         = 0x11
	vkMenu            = 0x12
	wmMouseLeave      = 0x02A3
	wmDpiChanged      = 0x02E0
	wmSetIcon         = 0x0080

	sizeRestored  = 0
	sizeMinimized = 1
	sizeMaximized = 2

	csHRedraw = 0x0002
	csVRedraw = 0x0001

	idcArrow       = 32512
	idiApplication = 32512

	smCxScreen       = 0
	smCyScreen       = 1
	smCXFrame        = 32
	smCYFrame        = 33
	smCXPaddedBorder = 92

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020

	htClient      = 1
	htCaption     = 2
	htMinButton   = 8
	htMaxButton   = 9
	htLeft        = 10
	htRight       = 11
	htTop         = 12
	htTopLeft     = 13
	htTopRight    = 14
	htBottom      = 15
	htBottomLeft  = 16
	htBottomRight = 17
	htClose       = 20

	tmeLeave     = 0x00000002
	tmeNonClient = 0x00000010

	dwmwaUseImmersiveDarkMode   = 20
	dwmwaUseImmersiveDarkMode19 = 19
	dwmwaWindowCornerPreference = 33
	dwmwcpRound                 = 2

	dwriteTextAlignLeading   = 0
	dwriteTextAlignCenter    = 2
	dwriteTextAlignTrailing  = 1
	dwriteParagraphNear      = 0
	dwriteParagraphCenter    = 2
	dwriteParagraphFar       = 1
	dwriteWordWrappingWrap      = 0
	dwriteWordWrappingNoWrap    = 1
	dwriteWordWrappingCharacter = 3

	coinitApartmentThreaded = 0x2

	d2d1FactoryTypeSingleThreaded = 0
	dwriteFactoryTypeShared       = 0

	dxgiFormatB8G8R8A8Unorm     = 87
	d2d1AlphaModePremultiplied  = 1
	d2d1RenderTargetTypeDefault = 0
	d2d1PresentOptionsNone      = 0
	d2d1TextAntialiasClearType  = 1

	dwriteFontWeightNormal  = 400
	dwriteFontStyleNormal   = 0
	dwriteFontStretchNormal = 5


	d2dDrawTextClip            = 0x00000002
	d2dDrawTextEnableColorFont = 0x00000004

	processPerMonitorDPIAware = 2

	localeNameMax = 85
)

const (
	idxRelease = 2

	idxFactoryCreateHwndRenderTarget = 14

	idxCreateSolidColorBrush = 8
	idxDrawLine              = 15
	idxFillRectangle         = 17
	idxDrawText              = 27
	idxSetTextAntialiasMode  = 34
	idxClear                 = 47
	idxBeginDraw             = 48
	idxEndDraw               = 49
	idxResize                = 58

	idxBrushSetColor = 8

	// IDWriteFactory 虚表（含 IUnknown 0-2）：
	// 3 GetSystemFontCollection
	// 4 CreateCustomFontCollection
	// 5 RegisterFontCollectionLoader
	// 6 UnregisterFontCollectionLoader
	// 7 CreateFontFileReference
	// 8 CreateCustomFontFileReference
	// 9 CreateFontFace
	// 10 CreateRenderingParams
	// 11 CreateMonitorRenderingParams
	// 12 CreateCustomRenderingParams
	// 13 RegisterFontFileLoader
	// 14 UnregisterFontFileLoader
	// 15 CreateTextFormat
	// 16 CreateTypography
	// 17 GetGdiInterop
	// 18 CreateTextLayout
	idxCreateTextFormat = 15
	idxCreateTextLayout = 18

	idxSetTextAlignment      = 3
	idxSetParagraphAlignment = 4
	idxSetWordWrapping       = 5
	idxGetLineMetrics        = 59
	idxGetMetrics            = 60
	idxHitTestTextPosition   = 65

	errorClassAlreadyExists syscall.Errno = 1410
)

type hresult int32

func (h hresult) failed() bool { return h < 0 }

func (h hresult) Error() string {
	return "HRESULT 0x" + hex32(uint32(h))
}

func hex32(v uint32) string {
	const digits = "0123456789ABCDEF"
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = digits[v&0xF]
		v >>= 4
	}
	return string(b[:])
}

type rect struct {
	left, top, right, bottom int32
}

type margins struct {
	cxLeftWidth, cxRightWidth, cyTopHeight, cyBottomHeight int32
}

type nccalcsizeParams struct {
	rgrc  [3]rect
	lppos uintptr
}

type trackMouseEvent struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   uintptr
	dwHoverTime uint32
}

type point struct {
	x, y int32
}

type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type d2d1PixelFormat struct {
	format    uint32
	alphaMode uint32
}

type d2d1RenderTargetProperties struct {
	targetType  uint32
	pixelFormat d2d1PixelFormat
	dpiX        float32
	dpiY        float32
	usage       uint32
	minLevel    uint32
}

type d2d1SizeU struct {
	width  uint32
	height uint32
}

type d2d1HwndRenderTargetProperties struct {
	hwnd           uintptr
	pixelSize      d2d1SizeU
	presentOptions uint32
	_              uint32
}

type d2d1Point2F struct {
	x, y float32
}

type d2d1ColorF struct {
	r, g, b, a float32
}

type d2d1RectF struct {
	left, top, right, bottom float32
}

type dwriteTextMetrics struct {
	left                             float32
	top                              float32
	width                            float32
	widthIncludingTrailingWhitespace float32
	height                           float32
	layoutWidth                      float32
	layoutHeight                     float32
	maxBidiReorderingDepth           uint32
	lineCount                        uint32
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidID2D1Factory = guid{
		Data1: 0x06152247,
		Data2: 0x6f50,
		Data3: 0x465a,
		Data4: [8]byte{0x92, 0x45, 0x11, 0x8b, 0xfd, 0x3b, 0x60, 0x07},
	}
	iidIDWriteFactory = guid{
		Data1: 0xb859ee5a,
		Data2: 0xd838,
		Data3: 0x4b5b,
		Data4: [8]byte{0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48},
	}
)

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func utf16Buf(s string) ([]uint16, uint32) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return []uint16{0}, 0
	}
	n := len(u)
	if n > 0 && u[n-1] == 0 {
		n--
	}
	return u, uint32(n)
}

func moduleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

func loadCursor(id uintptr) uintptr {
	h, _, _ := procLoadCursorW.Call(0, id)
	return h
}

func loadIcon(id uintptr) uintptr {
	h, _, _ := procLoadIconW.Call(0, id)
	return h
}

func enableDPI() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (HANDLE)-4
	if err := procSetProcessDpiAwarenessContext.Find(); err == nil {
		ctx := ^uintptr(3)
		r, _, _ := procSetProcessDpiAwarenessContext.Call(ctx)
		if r != 0 {
			return
		}
	}
	if err := procSetProcessDpiAwareness.Find(); err == nil {
		procSetProcessDpiAwareness.Call(processPerMonitorDPIAware)
	}
}

func systemDPI() uint32 {
	if err := procGetDpiForSystem.Find(); err == nil {
		r, _, _ := procGetDpiForSystem.Call()
		if r != 0 {
			return uint32(r)
		}
	}
	return 96
}

func windowDPI(hwnd uintptr) uint32 {
	if hwnd != 0 && procGetDpiForWindow.Find() == nil {
		r, _, _ := procGetDpiForWindow.Call(hwnd)
		if r != 0 {
			return uint32(r)
		}
	}
	return systemDPI()
}

func userLocale() []uint16 {
	var buf [localeNameMax]uint16
	n, _, _ := procGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), localeNameMax)
	if n == 0 {
		u, _ := syscall.UTF16FromString("en-US")
		return u
	}
	out := make([]uint16, n)
	copy(out, buf[:n])
	return out
}

func signedLOWORD(v uintptr) int32 { return int32(int16(uint16(v))) }
func signedHIWORD(v uintptr) int32 { return int32(int16(uint16(v >> 16))) }

func dipToPx(dip, dpi float32) int32 {
	v := dip * dpi / 96
	if v < 1 {
		return 1
	}
	return int32(v + 0.5)
}

func resizeBorderPx(dpi uint32) int32 {
	if procGetSystemMetricsForDpi.Find() == nil {
		frame, _, _ := procGetSystemMetricsForDpi.Call(smCXFrame, uintptr(dpi))
		pad, _, _ := procGetSystemMetricsForDpi.Call(smCXPaddedBorder, uintptr(dpi))
		n := int32(frame + pad)
		if n < 1 {
			return 1
		}
		return n
	}
	frame, _, _ := procGetSystemMetrics.Call(smCXFrame)
	pad, _, _ := procGetSystemMetrics.Call(smCXPaddedBorder)
	n := int32(frame + pad)
	if n < 1 {
		return 1
	}
	return n
}

func adjustWindowRect(rc *rect, style uint32, dpi uint32) {
	if procAdjustWindowRectExForDpi.Find() == nil {
		procAdjustWindowRectExForDpi.Call(
			uintptr(unsafe.Pointer(rc)),
			uintptr(style),
			0,
			0,
			uintptr(dpi),
		)
		return
	}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(rc)), uintptr(style), 0, 0)
}
