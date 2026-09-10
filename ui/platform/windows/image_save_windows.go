package windows

import (
	"fmt"
	"syscall"
	"unsafe"
)

// SaveImageDialog opens an owned save dialog in the user's Downloads folder.
// The bool is false for cancellation; native dialog errors are returned separately.
func SaveImageDialog(owner syscall.Handle, name, format string) (string, bool, error) {
	buffer := make([]uint16, 32768)
	copy(buffer, syscall.StringToUTF16(name))
	filter := utf16Buffer(fmt.Sprintf("%s 图片 (*.%s)\x00*.%s", format, format, format))
	dir := syscall.StringToUTF16Ptr(DownloadsDirectory())
	title := syscall.StringToUTF16Ptr("保存图片")
	ext := syscall.StringToUTF16Ptr(format)
	of := openFileName{
		StructSize: uint32(unsafe.Sizeof(openFileName{})), Owner: owner,
		Filter: &filter[0], FilterIndex: 1, File: &buffer[0], MaxFile: uint32(len(buffer)),
		InitialDir: dir, Title: title, DefExtension: ext, Flags: ofNoChangeDir | ofPathMustExist | ofOverwritePrompt,
	}
	if result, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&of))); result == 0 {
		code, _, _ := comdlg32.NewProc("CommDlgExtendedError").Call()
		if code != 0 {
			return "", false, fmt.Errorf("Windows 对话框错误 0x%X", code)
		}
		return "", false, nil
	}
	return syscall.UTF16ToString(buffer), true, nil
}
