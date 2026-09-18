//go:build windows

package windows

import (
	"syscall"
	"unsafe"
)

func keyDown(vk int) bool {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

// currentOSThreadID 返回当前 OS 线程 id。宿主用它登记 UI 线程，
// 平台层据此判断某次调用是否已经在 UI 线程上。
func currentOSThreadID() uint32 {
	id, _, _ := procGetCurrentThreadId.Call()
	return uint32(id)
}

var (
	modComdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")

	procGetSaveFileNameW     = modComdlg32.NewProc("GetSaveFileNameW")
	procGetOpenFileNameW     = modComdlg32.NewProc("GetOpenFileNameW")
	procSHGetKnownFolderPath = modShell32.NewProc("SHGetKnownFolderPath")
	procSHBrowseForFolderW   = modShell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = modShell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree        = modOle32.NewProc("CoTaskMemFree")

	procOpenClipboard              = modUser32.NewProc("OpenClipboard")
	procCloseClipboard             = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard             = modUser32.NewProc("EmptyClipboard")
	procGetClipboardData           = modUser32.NewProc("GetClipboardData")
	procSetClipboardData           = modUser32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = modUser32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc                = modKernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock               = modKernel32.NewProc("GlobalUnlock")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
	ofnExplorer   = 0x00080000
	ofnPathMustExist = 0x00000800
	ofnFileMustExist = 0x00001000
	ofnOverwritePrompt = 0x00000002
	ofnHidereadonly = 0x00000004
	maxPath = 260
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

func utf16SliceToString(u []uint16) string {
	n := 0
	for n < len(u) && u[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(u[:n])
}

func downloadsDir() string {
	folder := guid{
		Data1: 0x374DE290,
		Data2: 0x123F,
		Data3: 0x4565,
		Data4: [8]byte{0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B},
	}
	var p *uint16
	hr, _, _ := procSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(&folder)), 0, 0, uintptr(unsafe.Pointer(&p)))
	if hresult(hr).failed() || p == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
	n := 0
	for {
		c := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n)*2))
		if c == 0 {
			break
		}
		n++
		if n > 4096 {
			return ""
		}
	}
	buf := unsafe.Slice(p, n)
	return syscall.UTF16ToString(buf)
}

func encodeFilter(pairs [][2]string) *uint16 {
	filter := []uint16{}
	appendZ := func(t string) {
		p, _ := syscall.UTF16FromString(t)
		if len(p) > 0 {
			filter = append(filter, p[:len(p)-1]...)
		}
		filter = append(filter, 0)
	}
	for _, pair := range pairs {
		appendZ(pair[0])
		appendZ(pair[1])
	}
	filter = append(filter, 0)
	return &filter[0]
}

func imageFilter() *uint16 {
	return encodeFilter([][2]string{
		{"图片", "*.png;*.jpg;*.jpeg"},
		{"PNG", "*.png"},
		{"JPEG", "*.jpg;*.jpeg"},
	})
}

func parseFilter(filter string) *uint16 {
	if filter == "" {
		return encodeFilter([][2]string{
			{"所有文件", "*.*"},
			{"文本", "*.txt;*.md;*.go;*.json;*.xml;*.yml;*.yaml"},
		})
	}
	parts := splitFilter(filter)
	if len(parts) == 0 {
		return encodeFilter([][2]string{{"所有文件", "*.*"}})
	}
	pairs := make([][2]string, 0, (len(parts)+1)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		pairs = append(pairs, [2]string{parts[i], parts[i+1]})
	}
	if len(parts)%2 == 1 {
		pairs = append(pairs, [2]string{parts[len(parts)-1], "*.*"})
	}
	return encodeFilter(pairs)
}

func splitFilter(filter string) []string {
	out := []string{}
	cur := ""
	for _, r := range filter {
		if r == '|' || r == 0 {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func runFileDialog(hwnd uintptr, title, name, ext string, save bool) (string, bool, error) {
	return runFileDialogEx(hwnd, title, name, ext, "", save, imageFilter())
}

func runFileDialogEx(hwnd uintptr, title, name, ext, filter string, save bool, encoded *uint16) (string, bool, error) {
	var file [maxPath + 4]uint16
	if name != "" {
		src, _ := syscall.UTF16FromString(name)
		copy(file[:], src)
	}
	if encoded == nil {
		encoded = parseFilter(filter)
	}
	ofn := openFileNameW{
		hwndOwner:       hwnd,
		lpstrFilter:     encoded,
		nFilterIndex:    1,
		lpstrFile:       &file[0],
		nMaxFile:        uint32(len(file)),
		lpstrTitle:      utf16Ptr(title),
		flags:           ofnExplorer | ofnHidereadonly | ofnPathMustExist,
		lpstrInitialDir: utf16Ptr(downloadsDir()),
	}
	if ext != "" {
		ofn.lpstrDefExt = utf16Ptr(ext)
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	var ok uintptr
	if save {
		ofn.flags |= ofnOverwritePrompt
		ok, _, _ = procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	} else {
		ofn.flags |= ofnFileMustExist
		ok, _, _ = procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	}
	if ok == 0 {
		return "", false, nil
	}
	return utf16SliceToString(file[:]), true, nil
}

// SaveImageDialog 打开“另存为”，默认目录是下载文件夹。
func SaveImageDialog(hwnd uintptr, name, format string) (string, bool, error) {
	return runFileDialog(hwnd, "保存图片", name, format, true)
}

// OpenImageDialog 打开“选择图片”。
func OpenImageDialog(hwnd uintptr) (string, bool, error) {
	return runFileDialog(hwnd, "选择参考图", "", "", false)
}

// OpenFileDialog 打开通用“选择文件”对话框。filter 用 | 分隔，例如 "Go|*.go|全部|*.*"。
func OpenFileDialog(hwnd uintptr, title, filter string) (string, bool, error) {
	if title == "" {
		title = "打开"
	}
	return runFileDialogEx(hwnd, title, "", "", filter, false, nil)
}

// SaveFileDialog 打开通用“保存文件”对话框。
func SaveFileDialog(hwnd uintptr, title, name, filter, ext string) (string, bool, error) {
	if title == "" {
		title = "保存"
	}
	return runFileDialogEx(hwnd, title, name, ext, filter, true, nil)
}

type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

const (
	bifReturnOnlyFSDirs = 0x00000001
	bifNewDialogStyle   = 0x00000040
)

// OpenFolderDialog 打开“选择文件夹”对话框。
func OpenFolderDialog(hwnd uintptr, title string) (string, bool, error) {
	if title == "" {
		title = "选择文件夹"
	}
	var display [maxPath + 4]uint16
	bi := browseInfoW{
		hwndOwner:      hwnd,
		pszDisplayName: &display[0],
		lpszTitle:      utf16Ptr(title),
		ulFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", false, nil
	}
	defer procCoTaskMemFree.Call(pidl)
	var path [maxPath + 4]uint16
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", false, nil
	}
	return utf16SliceToString(path[:]), true, nil
}

// ClipboardText 读取 Unicode 剪贴板。
func ClipboardText() string {
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return ""
	}
	defer procCloseClipboard.Call()
	avail, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText)
	if avail == 0 {
		return ""
	}
	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)
	n := 0
	for {
		c := *(*uint16)(unsafe.Pointer(p + uintptr(n)*2))
		if c == 0 {
			break
		}
		n++
		if n > 1<<20 {
			break
		}
	}
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n))
}

// SetWindowText 更新窗口标题（标题栏 / 任务栏 / Alt+Tab 同步）。
func SetWindowText(hwnd uintptr, title string) {
	if hwnd == 0 {
		return
	}
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(title))))
}

// SetClipboardText 写入 Unicode 剪贴板。
func SetClipboardText(s string) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(u)*2))
	if h == 0 {
		return
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u))
	copy(dst, u)
	procGlobalUnlock.Call(h)
	if r, _, _ := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		return
	}
}
