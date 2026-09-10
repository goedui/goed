package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var procSHFileOperationW = shell32.NewProc("SHFileOperationW")

const (
	foDelete          = 3
	fofAllowUndo      = 0x40  // 移入回收站而非永久删除
	fofNoConfirmation = 0x10  // 不弹系统确认框（确认由应用内对话框负责）
	fofSilent         = 0x4   // 不显示进度对话框
	fofNoErrorUI      = 0x400 // 错误交给调用方展示
)

// shFileOpStructW 是 SHFILEOPSTRUCTW 的最小绑定；字段顺序与平台对齐
// （amd64 下 wFunc 与 pFrom 之间由编译器自动填充对齐字节）。
type shFileOpStructW struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

// DeleteToRecycleBin 把文件或目录移入系统回收站。目录会整体递归删除。
func DeleteToRecycleBin(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// pFrom 要求双 NUL 结尾的单字符串列表。
	from := utf16.Encode([]rune(filepath.Clean(absPath)))
	from = append(from, 0, 0)
	op := shFileOpStructW{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	ret, _, callErr := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	if ret != 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("SHFileOperationW: %w", callErr)
		}
		return fmt.Errorf("delete failed with code %d", ret)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("operation aborted")
	}
	return nil
}

// RevealInExplorer opens the containing folder of a document in Windows
// Explorer. For a directory path it opens that directory directly.
//
// 通过 ShellExecuteW 调用 explorer.exe 并以 UTF-16 指针传参，
// 避免命令行拼接/转义导致的路径问题（等效于在目录下执行 `explorer .`）。
func RevealInExplorer(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	folder := filepath.Clean(absPath)
	if info, statErr := os.Stat(folder); statErr != nil || !info.IsDir() {
		// 文件：打开其所在目录。
		folder = filepath.Dir(folder)
	}

	operation, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	program, err := syscall.UTF16PtrFromString("explorer.exe")
	if err != nil {
		return err
	}
	arguments, err := syscall.UTF16PtrFromString(folder)
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(operation)),
		uintptr(unsafe.Pointer(program)),
		uintptr(unsafe.Pointer(arguments)),
		0,
		SW_SHOWNORMAL,
	)
	if ret <= 32 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("ShellExecuteW: %w", callErr)
		}
		return fmt.Errorf("explorer.exe failed with code %d", ret)
	}
	return nil
}
