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
	procReleaseCapture                = modUser32.NewProc("ReleaseCapture")
	procGetKeyState                   = modUser32.NewProc("GetKeyState")
	procSetWindowTextW                = modUser32.NewProc("SetWindowTextW")

	procGetModuleHandleW         = modKernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId       = modKernel32.NewProc("GetCurrentThreadId")
	procGetUserDefaultLocaleName = modKernel32.NewProc("GetUserDefaultLocaleName")

	procCoInitializeEx   = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize   = modOle32.NewProc("CoUninitialize")
	procCoCreateInstance = modOle32.NewProc("CoCreateInstance")

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
	wmRButtonDown     = 0x0204
	wmRButtonUp       = 0x0205
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

	dwriteTextAlignLeading      = 0
	dwriteTextAlignCenter       = 2
	dwriteTextAlignTrailing     = 1
	// DWRITE_PARAGRAPH_ALIGNMENT：0 Near、1 Far、**2 Center**。
	//
	// 和上面的 TEXT_ALIGNMENT 不是同一套顺序：TEXT 是 0 Leading、1 Trailing、
	// **2 Center**；PARAGRAPH 是 0 Near、1 Far、**2 Center**。两边的 Center 都是 2，
	// 只有中间那个值含义不同。把 Center 写成 1 时，按钮和标题栏的 VAlign=Center
	// 实际落到 Far（贴底），垂直居中就没了。
	dwriteParagraphNear         = 0
	dwriteParagraphCenter       = 2
	dwriteParagraphFar          = 1
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
	dwriteFontStyleItalic   = 2
	dwriteFontStretchNormal = 5

	d2dDrawTextClip            = 0x00000002
	d2dDrawTextEnableColorFont = 0x00000004

	processPerMonitorDPIAware = 2

	localeNameMax = 85

	// 现代「选择文件夹」对话框（IFileOpenDialog）用的开关。
	//
	// fosPickFolders 是「选目录」的**唯一**开关：不加它拿到的是选文件的对话框，
	// 点进目录只会「进去」而不是「选中」。
	//
	// fosForceFileSystem 把「库」「网络位置」这类**没有文件系统路径**的虚拟节点
	// 过滤掉。留着它们的后果不是多几个选项，而是用户选中之后
	// GetDisplayName(SIGDN_FILESYSPATH) 直接失败——界面上点了确定，什么都没发生。
	fosPickFolders     = 0x00000020
	fosForceFileSystem = 0x00000040

	// SIGDN_FILESYSPATH：向 IShellItem 要「文件系统路径」形态的显示名。
	// 它是 SIGDN 枚举里带 0x8000 位的那一档（不是字符串名、不是 URL）。
	sigdnFileSysPath = 0x80058000

	clsctxInprocServer = 0x1

	// maxNativeString 是读「长度未知的原生 UTF-16 串」时的硬上限。
	//
	// Windows 路径的上限是 32767 个 UTF-16 码元（\\?\ 长路径），这里取同一个
	// 量级：真串一定在里面停住；万一指针不是字符串（或者没有终止符），也不会
	// 一路读下去把进程读崩——撞上限时调用方报错，而不是交出一条截断的路径。
	maxNativeString = 32768
)

const (
	idxRelease = 2

	idxFactoryCreateHwndRenderTarget = 14
	// ID2D1Factory 虚表（含 IUnknown 0-2、ID2D1Resource::GetFactory 3）：
	// 4 ReloadSystemMetrics、5 GetDesktopDpi、6 CreateRectangleGeometry…（绝对序号，
	// 与 idxFactoryCreateHwndRenderTarget=14 同一基准，不是相对偏移）。
	idxFactoryCreateRoundedRectGeometry = 6
	// 7 CreateEllipseGeometry、8 CreateGeometryGroup、9 CreateTransformedGeometry、
	// 10 CreatePathGeometry、11 CreateStrokeStyle。序号按 d2d1.h 声明顺序，
	// 与上面 6 / 14 同一基准。stroke_test.go 会建一条路径再 Open 验证。
	idxFactoryCreatePathGeometry = 10
	idxFactoryCreateStrokeStyle  = 11

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

	// IDWriteFactory 虚表（含 IUnknown 0-2）：3 GetSystemFontCollection、4 CreateCustom-
	// FontCollection、5/6 Register/UnregisterFontCollectionLoader、7 CreateFontFileReference、
	// 8 CreateCustomFontFileReference、9 CreateFontFace、10-12 *RenderingParams、
	// 13/14 Register/UnregisterFontFileLoader、15 CreateTextFormat、16 CreateTypography、
	// 17 GetGdiInterop、18 CreateTextLayout。序号按 dwrite.h 声明顺序数出来的。
	// IDWriteFontFamily 虚表（IUnknown 0-2；继承自 IDWriteFontList 的
	// GetFontCollection / GetFontCount 是 3/4）：
	// 5 GetFont、6 GetFamilyNames、7 GetFirstMatchingFont、8 GetMatchingFonts。
	idxGetFamilyNames = 6
	// 7 GetFirstMatchingFont、8 GetMatchingFonts。
	idxGetFirstMatchingFont = 7

	// IDWriteLocalizedStrings 虚表（IUnknown 0-2）：
	// 3 GetCount、4 FindLocaleName、5 GetLocaleNameLength、6 GetLocaleName、
	// 7 GetStringLength、8 GetString。
	idxStringsFindLocale   = 4
	idxStringsStringLength = 7
	idxStringsGetString    = 8

	// IDWriteFont 虚表（IUnknown 0-2）：3 GetFontFamily、4 GetWeight、5 GetStretch、
	// 6 GetStyle、7 IsSymbolFont、8 GetFaceNames、9 GetInformationalStrings、
	// 10 GetSimulations、11 GetMetrics、12 HasCharacter、13 CreateFontFace。
	idxFontHasCharacter = 12

	// IDWriteFontFace 虚表（IUnknown 0-2）：3 GetType、4 GetFiles、5 GetIndex、
	// 6 GetSimulations、7 IsSymbolFont、8 GetMetrics、9 GetGlyphCount、
	// 10 GetDesignGlyphMetrics、11 GetGlyphIndices、12 TryGetFontTable、
	// 13 ReleaseFontTable、14 GetGlyphRunOutline、15 GetRecommendedRenderingMode、
	// 16 GetGdiCompatibleMetrics、17 GetGdiCompatibleGlyphMetrics、
	// 18 GetDesignGlyphAdvances、19 GetGdiCompatibleGlyphAdvances。
	idxFaceGlyphCount      = 9
	idxFaceDesignAdvances  = 18

	idxGetSystemFontCollection = 3
	idxCreateTextFormat        = 15
	idxCreateTextLayout        = 18

	// IDWriteFontCollection 虚表（IUnknown 0-2）：
	// 3 GetFontFamilyCount、4 GetFontFamily、5 FindFamilyName、6 GetFontFromFontFace。
	idxFontFamilyCount = 3
	idxGetFontFamily   = 4
	idxFindFamilyName  = 5

	idxSetTextAlignment      = 3
	idxSetParagraphAlignment = 4
	idxSetWordWrapping       = 5
	dwriteLineSpacingUniform = 1
	idxSetLineSpacing        = 10
	idxGetLineMetrics        = 59
	idxGetMetrics            = 60
	idxHitTestTextPosition   = 65

	// IFileOpenDialog 虚表（IUnknown 0-2、IModalWindow::Show 3）：
	// 4  SetFileTypes
	// 5  SetFileTypeIndex
	// 6  GetFileTypeIndex
	// 7  Advise
	// 8  Unadvise
	// 9  SetOptions
	// 10 GetOptions
	// 11 SetDefaultFolder
	// 12 SetFolder
	// 13 GetFolder
	// 14 GetCurrentSelection
	// 15 SetFileName
	// 16 GetFileName
	// 17 SetTitle
	// 18 SetOkButtonLabel
	// 19 SetFileNameLabel
	// 20 GetResult
	//
	// 和上面 ID2D1Factory 那一组一样是**绝对序号**（下标 0/1/2 是
	// QueryInterface/AddRef/Release），对象指针直接当 this 用。槽位写错不会
	// 报错，只会跳到另一个函数上去——所以 folderdialog_test.go 里对
	// SetOptions/GetOptions 做了一次**往返**断言：槽位若串了，取回来的位就不同。
	idxDialogShow       = 3
	idxDialogSetOptions = 9
	idxDialogGetOptions = 10
	idxDialogSetTitle   = 17
	idxDialogGetResult  = 20

	// IShellItem 虚表（IUnknown 0-2）：
	// 3 BindToHandler
	// 4 GetParent
	// 5 GetDisplayName
	idxItemGetDisplayName = 5

	errorClassAlreadyExists syscall.Errno = 1410
)

type hresult int32

// hresultOf 把无符号 HRESULT 字面量转成有符号的 hresult。
//
// 需要绕这一下是因为 0x8007xxxx / 0x8001xxxx 都超出 int32：Go 的**常量**转换要求
// 目标类型能表示该值，`hresult(0x800704C7)` 是编译期错误（constant overflows
// int32）。走一次带参数的函数，参数就成了非常量，转换退化成运行时截断——合法，
// 而且正是我们要的（失败位保持不变）。
func hresultOf(v uint32) hresult { return hresult(v) }

var (
	// hrCancelled 是 HRESULT_FROM_WIN32(ERROR_CANCELLED)：用户在对话框里点了取消。
	//
	// 它**必须**和「真失败」分开。IFileOpenDialog::Show 用同一个失败位（最高位）
	// 报取消，一旦当成 error 往上抛，每次点取消都会变成界面上一条红色提示——
	// 而取消是这个对话框最常见的结局。
	hrCancelled = hresultOf(0x800704C7)
	// hrChangedMode 是 RPC_E_CHANGED_MODE：线程已经以**别的**套间模型初始化过 COM。
	//
	// 这种「失败」其实是成功的：COM 可用，只是这次调用没增加引用计数，所以不能
	// 配对 CoUninitialize——还了就等于把别人的（或系统自己的一次）引用还掉了。
	hrChangedMode = hresultOf(0x80010106)
)

// sFalse 是 COM 惯用的「成功，但没做事」返回值。
const sFalse = 1

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

// dwriteLineMetrics 对应 DWRITE_LINE_METRICS。
// 字段顺序按 dwrite.h：length、trailingWhitespaceLength、newlineLength、
// height、baseline、isTrimmed。
type dwriteLineMetrics struct {
	length                   uint32
	trailingWhitespaceLength uint32
	newlineLength            uint32
	height                   float32
	baseline                 float32
	isTrimmed                int32
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

	// CLSID_FileOpenDialog {DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}：
	// 现代文件/文件夹对话框的 coclass。同一个对象靠 FOS_PICKFOLDERS 在
	// 「选文件」和「选目录」之间切换——没有单独的 CLSID_FolderOpenDialog。
	clsidFileOpenDialog = guid{
		Data1: 0xdc1c5a9c,
		Data2: 0xe88a,
		Data3: 0x4dde,
		Data4: [8]byte{0xa5, 0xa1, 0x60, 0xf8, 0x2a, 0x20, 0xae, 0xf7},
	}
	// FOLDERID_ComputerFolder {0AC0837C-BBF8-452A-850D-79D08E667CA7}：「此电脑」。
	// 老式对话框拿它当 pidlRoot，树根才是盘符而不是桌面。
	folderIDComputerFolder = guid{
		Data1: 0x0ac0837c,
		Data2: 0xbbf8,
		Data3: 0x452a,
		Data4: [8]byte{0x85, 0x0d, 0x79, 0xd0, 0x8e, 0x66, 0x7c, 0xa7},
	}
	// IID_IFileOpenDialog {D57C7288-D4AD-4768-BE02-9D969532D960}。
	// 注意不是 IID_IFileDialog：要 GetResult 就必须拿这一层的接口，
	// 而它是 IFileDialog 的派生接口，前 27 个槽位完全一致。
	iidIFileOpenDialog = guid{
		Data1: 0xd57c7288,
		Data2: 0xd4ad,
		Data3: 0x4768,
		Data4: [8]byte{0xbe, 0x02, 0x9d, 0x96, 0x95, 0x32, 0xd9, 0x60},
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
