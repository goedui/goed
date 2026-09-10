package windows

import (
	"fmt"
	"syscall"
	"unsafe"
)

// OpenImageDialog selects a PNG or JPEG file for image editing.
func OpenImageDialog(owner syscall.Handle) (string, bool, error) {
	buffer := make([]uint16, 32768)
	filter := utf16Buffer("图片 (*.png;*.jpg;*.jpeg)\x00*.png;*.jpg;*.jpeg")
	of := openFileName{
		StructSize: uint32(unsafe.Sizeof(openFileName{})), Owner: owner,
		Filter: &filter[0], FilterIndex: 1, File: &buffer[0], MaxFile: uint32(len(buffer)),
		Title: syscall.StringToUTF16Ptr("选择参考图片"),
		Flags: ofNoChangeDir | ofPathMustExist | ofFileMustExist,
	}
	if result, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&of))); result == 0 {
		code, _, _ := comdlg32.NewProc("CommDlgExtendedError").Call()
		if code != 0 {
			return "", false, fmt.Errorf("Windows 对话框错误 0x%X", code)
		}
		return "", false, nil
	}
	return syscall.UTF16ToString(buffer), true, nil
}
