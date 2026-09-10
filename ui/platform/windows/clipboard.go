package windows

import (
	"syscall"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	procOpenClipboard          = user32.NewProc("OpenClipboard")
	procCloseClipboard         = user32.NewProc("CloseClipboard")
	procEmptyClipboard         = user32.NewProc("EmptyClipboard")
	procGetClipboardData       = user32.NewProc("GetClipboardData")
	procSetClipboardData       = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvail = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc            = kernel32.NewProc("GlobalAlloc")
	procGlobalFree             = kernel32.NewProc("GlobalFree")
	procGlobalLock             = kernel32.NewProc("GlobalLock")
	procGlobalUnlock           = kernel32.NewProc("GlobalUnlock")
	procGlobalSize             = kernel32.NewProc("GlobalSize")
	procRtlMoveMemory          = kernel32.NewProc("RtlMoveMemory")
)

// ReadClipboardText returns Unicode text from the Windows clipboard.
func ReadClipboardText(owner syscall.Handle) string {
	if available, _, _ := procIsClipboardFormatAvail.Call(cfUnicodeText); available == 0 {
		return ""
	}
	if opened, _, _ := procOpenClipboard.Call(uintptr(owner)); opened == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return ""
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(handle)

	size, _, _ := procGlobalSize.Call(handle)
	if size < 2 {
		return ""
	}
	units := make([]uint16, int(size/2))
	procRtlMoveMemory.Call(
		uintptr(unsafe.Pointer(&units[0])),
		ptr,
		uintptr(len(units)*2),
	)
	return syscall.UTF16ToString(units)
}

// WriteClipboardText replaces the Windows clipboard with Unicode text.
func WriteClipboardText(owner syscall.Handle, text string) bool {
	if opened, _, _ := procOpenClipboard.Call(uintptr(owner)); opened == 0 {
		return false
	}
	defer procCloseClipboard.Call()

	if emptied, _, _ := procEmptyClipboard.Call(); emptied == 0 {
		return false
	}
	units := syscall.StringToUTF16(text)
	size := uintptr(len(units) * 2)
	handle, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return false
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return false
	}
	procRtlMoveMemory.Call(
		ptr,
		uintptr(unsafe.Pointer(&units[0])),
		uintptr(len(units)*2),
	)
	procGlobalUnlock.Call(handle)
	if result, _, _ := procSetClipboardData.Call(cfUnicodeText, handle); result != 0 {
		// Ownership transfers to the clipboard after success.
		return true
	}
	procGlobalFree.Call(handle)
	return false
}
