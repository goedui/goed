package windows

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Additional Win32 functions needed
var (
	msimg32                    = syscall.NewLazyDLL("msimg32.dll")
	procAlphaBlend             = msimg32.NewProc("AlphaBlend")
	procCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procCreatePen              = gdi32.NewProc("CreatePen")
	procMoveToEx               = gdi32.NewProc("MoveToEx")
	procLineTo                 = gdi32.NewProc("LineTo")
	procFrameRect              = user32.NewProc("FrameRect")
	procSetBkMode              = gdi32.NewProc("SetBkMode")
	procGetTextExtentPoint32W  = gdi32.NewProc("GetTextExtentPoint32W")
	procEllipse                = gdi32.NewProc("Ellipse")
	procPolygon                = gdi32.NewProc("Polygon")
	procStretchDIBits          = gdi32.NewProc("StretchDIBits")
	procSetStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	procSetBrushOrgEx          = gdi32.NewProc("SetBrushOrgEx")
	procSetDIBitsToDevice      = gdi32.NewProc("SetDIBitsToDevice")
	procStretchBlt             = gdi32.NewProc("StretchBlt")
)

const (
	halftone     = 4 // HALFTONE：StretchDIBits 高质量缩放模式
	dibRGBColors = 0 // DIB_RGB_COLORS
)

const (
	SRCCOPY  = 0x00CC0020
	PS_SOLID = 0
	OPAQUE   = 2
)

type SIZE struct {
	Cx int32
	Cy int32
}

// TEXTMETRICW 是完整的 Win32 TEXTMETRICW 布局（11 个 LONG + 8 个 BYTE，
// 共 52 字节）。字段大小必须与 API 期望严格一致：GetTextMetricsW 会写入
// 整个结构体，缺字段会越界写、破坏相邻堆内存（表现为延迟随机崩溃）。
type TEXTMETRICW struct {
	tmHeight           int32
	tmAscent           int32
	tmDescent          int32
	tmInternalLeading  int32
	tmExternalLeading  int32
	tmAveCharWidth     int32
	tmMaxCharWidth     int32
	tmWeight           int32
	tmOverhang         int32
	tmDigitizedAspectX int32
	tmDigitizedAspectY int32
	tmFirstChar        uint16
	tmLastChar         uint16
	tmDefaultChar      uint16
	tmBreakChar        uint16
	tmItalic           uint8
	tmUnderlined       uint8
	tmStruckOut        uint8
	tmPitchAndFamily   uint8
	tmCharSet          uint8
}

// GetTextMetrics 查询指定 DC 当前字体的度量。
func GetTextMetrics(hdc syscall.Handle, tm *TEXTMETRICW) {
	// 布局守卫：结构体大小必须精确等于 Win32 TEXTMETRICW（60 字节 =
	// 11 个 LONG + 4 个 WORD + 8 个 BYTE（含对齐填充）。GetTextMetricsW
	// 会写入整个结构体，缺字段会越界写、破坏相邻堆内存。
	const textMetricWSize = 60
	if unsafe.Sizeof(*tm) != textMetricWSize {
		panic(fmt.Sprintf("TEXTMETRICW size = %d, want %d", unsafe.Sizeof(*tm), textMetricWSize))
	}
	procGetTextMetricsW.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(tm)),
	)
}

func CreateCompatibleDC(hdc syscall.Handle) syscall.Handle {
	ret, _, _ := procCreateCompatibleDC.Call(uintptr(hdc))
	return syscall.Handle(ret)
}

func CreateCompatibleBitmap(hdc syscall.Handle, width, height int32) syscall.Handle {
	ret, _, _ := procCreateCompatibleBitmap.Call(uintptr(hdc), uintptr(width), uintptr(height))
	return syscall.Handle(ret)
}

func BitBlt(hdcDest syscall.Handle, x, y, width, height int32, hdcSrc syscall.Handle, xSrc, ySrc int32, rop uint32) {
	procBitBlt.Call(
		uintptr(hdcDest),
		uintptr(x), uintptr(y),
		uintptr(width), uintptr(height),
		uintptr(hdcSrc),
		uintptr(xSrc), uintptr(ySrc),
		uintptr(rop),
	)
}

func DeleteDC(hdc syscall.Handle) {
	procDeleteDC.Call(uintptr(hdc))
}

func DeleteObject(obj syscall.Handle) {
	procDeleteObject.Call(uintptr(obj))
}

func CreatePen(style int32, width int32, color uint32) syscall.Handle {
	ret, _, _ := procCreatePen.Call(uintptr(style), uintptr(width), uintptr(color))
	return syscall.Handle(ret)
}

func MoveToEx(hdc syscall.Handle, x, y int32) {
	procMoveToEx.Call(uintptr(hdc), uintptr(x), uintptr(y), 0)
}

func LineTo(hdc syscall.Handle, x, y int32) {
	procLineTo.Call(uintptr(hdc), uintptr(x), uintptr(y))
}

func RoundRect(hdc syscall.Handle, left, top, right, bottom, ellipseWidth, ellipseHeight int32) {
	procRoundRect.Call(
		uintptr(hdc), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom),
		uintptr(ellipseWidth), uintptr(ellipseHeight),
	)
}

func Ellipse(hdc syscall.Handle, left, top, right, bottom int32) {
	procEllipse.Call(uintptr(hdc), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
}

func Polygon(hdc syscall.Handle, points *POINT, count int32) {
	procPolygon.Call(uintptr(hdc), uintptr(unsafe.Pointer(points)), uintptr(count))
}

func CreateRectRgn(left, top, right, bottom int32) syscall.Handle {
	ret, _, _ := procCreateRectRgn.Call(uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
	return syscall.Handle(ret)
}

func SelectClipRgn(hdc, rgn syscall.Handle) {
	procSelectClipRgn.Call(uintptr(hdc), uintptr(rgn))
}

func GetStockObject(index int32) syscall.Handle {
	ret, _, _ := procGetStockObject.Call(uintptr(index))
	return syscall.Handle(ret)
}

func FrameRect(hdc syscall.Handle, rect *RECT, brush syscall.Handle) {
	procFrameRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rect)), uintptr(brush))
}

func SetBkMode(hdc syscall.Handle, mode int32) {
	procSetBkMode.Call(uintptr(hdc), uintptr(mode))
}

// bitmapInfoHeader 是 BITMAPINFOHEADER 的最小对齐定义。
type bitmapInfoHeader struct {
	Size         uint32
	Width        int32
	Height       int32
	Planes       uint16
	BitCount     uint16
	Compression  uint32
	SizeImage    uint32
	XPelsPerM    int32
	YPelsPerM    int32
	ClrUsed      uint32
	ClrImportant uint32
}

// blendFunction 对应 Win32 BLENDFUNCTION。
type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

const (
	acSrcOver  = 0 // AC_SRC_OVER
	acSrcAlpha = 1 // 源像素自带 alpha 时使用（本实现用常量 alpha，不设此项）
)

// CreateDIBSection 包装 gdi32!CreateDIBSection。像素指针通过 ppBits 出参
// 返回；实现把 Call 返回的 uintptr 立即写进出参 slot（syscall 包对
// lpvBits 的约定），避免 uintptr 中转换转换点散落。
func CreateDIBSection(hdc syscall.Handle, bi *bitmapInfoHeader, usage uint32, ppBits **byte, section uintptr, offset uint32) syscall.Handle {
	// 由 syscall 直接把像素指针写入 *byte slot：slot 的类型即指针，
	// 不存在 uintptr → unsafe.Pointer 的转换点，vet 合规。
	ret, _, _ := procCreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(bi)),
		uintptr(usage),
		uintptr(unsafe.Pointer(ppBits)),
		section,
		uintptr(offset),
	)
	return syscall.Handle(ret)
}

func GetTextExtentPoint32(hdc syscall.Handle, text string, size *SIZE) {
	utf16Text := syscall.StringToUTF16(sanitizeWindowsText(text))
	procGetTextExtentPoint32W.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&utf16Text[0])),
		uintptr(len(utf16Text)-1),
		uintptr(unsafe.Pointer(size)),
	)
}
