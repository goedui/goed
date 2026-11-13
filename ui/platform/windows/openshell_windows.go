//go:build windows

package windows

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// ofnAllowMultiSelect 让「打开文件」对话框一次选多个文件（comdlg32 的 OFN_ALLOWMULTISELECT）。
const ofnAllowMultiSelect = 0x00000200

// openFilesBuf 是多选对话框的返回缓冲区（utf16 单元数）。
// 多选时 comdlg32 往里塞「目录\0文件1\0文件2\0…\0\0」，路径一长很容易越过
// 默认的 MAX_PATH，所以这里按 32K 字符给，约为 64KB 栈空间。
const openFilesBuf = 32768

// swShowNormal 是 ShellExecuteW 的显示方式（SW_SHOWNORMAL）。
const swShowNormal = 5

// OpenFilesDialog 打开「选择多个文件」对话框。
//
// 返回 (路径列表, 选中了吗, 出错了吗)。**用户取消不算错**：ok=false、err=nil。
// 缓冲区语义和单选不同：多选时第一段是目录、后面每段是文件名，要自己拼回来。
func OpenFilesDialog(hwnd uintptr, title, filter string) ([]string, bool, error) {
	if title == "" {
		title = "选择文件"
	}
	var file [openFilesBuf]uint16
	ofn := openFileNameW{
		hwndOwner:       hwnd,
		lpstrFilter:     parseFilter(filter),
		nFilterIndex:    1,
		lpstrFile:       &file[0],
		nMaxFile:        uint32(len(file)),
		lpstrTitle:      utf16Ptr(title),
		lpstrInitialDir: utf16Ptr(downloadsDir()),
		flags:           ofnExplorer | ofnAllowMultiSelect | ofnPathMustExist | ofnFileMustExist,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	ok, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ok == 0 {
		return nil, false, nil
	}
	return parseMultiSelect(file[:]), true, nil
}

// parseMultiSelect 拆 comdlg32 的多选返回：NUL 分隔、双 NUL 结束。
//
//	单文件：整段就是一个完整路径。
//	多文件：第一段是目录，其余是文件名，逐个拼成完整路径。
func parseMultiSelect(buf []uint16) []string {
	var parts []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		if i > start {
			parts = append(parts, syscall.UTF16ToString(buf[start:i]))
		}
		// 连着两个 NUL 就是结束。
		if i+1 < len(buf) && buf[i+1] == 0 {
			break
		}
		start = i + 1
	}
	if len(parts) == 0 {
		return nil
	}
	if len(parts) == 1 {
		return parts
	}
	dir := parts[0]
	out := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

// OpenPath 用系统关联程序打开一个文件 / 文件夹（Windows 上就是 ShellExecute "open"）。
// 找不到关联程序时返回错误，由调用方决定怎么提示。
func OpenPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return syscall.EINVAL
	}
	op := "open"
	p, _ := syscall.UTF16PtrFromString(path)
	verb, _ := syscall.UTF16PtrFromString(op)
	r, _, err := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(p)),
		0, 0,
		swShowNormal,
	)
	// ShellExecute 的返回值不是 HINSTANCE 句柄：<= 32 全是失败码。
	if r <= 32 {
		if err != nil && err != syscall.Errno(0) {
			return err
		}
		return syscall.EINVAL
	}
	return nil
}

var procShellExecuteW = modShell32.NewProc("ShellExecuteW")
