package windows

import (
	"fmt"
	"syscall"
	"unsafe"
)

// IFileDialog（CLSID_FileOpenDialog）文件夹选择：与"打开文件"同款的
// Vista 现代对话框（带地址栏）。vtable 以具名字段逐项声明，槽位与
// shobjidl_core.h 一一对应（IUnknown 0-2 → IModalWindow::Show 3 →
// IFileDialog 4-19 → IFileOpenDialog::GetResult 20）。

var (
	ole32                    = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procCoCreateInstance     = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	procSHCreateItemFromPars = shell32.NewProc("SHCreateItemFromParsingName")
)

const (
	clsidFileOpenDialog = "{DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}"
	iidIFileDialog      = "{42F85136-DB7E-439C-85F1-E4075D135FC8}"
	iidIShellItem       = "{43826D1E-E718-42EE-BC55-A1E261C37BFE}"
	clsctxInprocServer  = 1
	coinitApartment     = 0x2 // COINIT_APARTMENTTHREADED（与 UI 线程 STA 一致）
	fosPickFolders      = 0x20
	fosForceFileSystem  = 0x40
	fosPathMustExist    = 0x800
	sigdnFilesystemPath = 0x80058000

	hrSFalse         = 0x00000001 // COM 已初始化：成功
	hrRPCChangedMode = 0x80010106 // 已在其他套间初始化：COM 可用
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// guidFromString 解析 "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}" 为 GUID
// 二进制布局：Data1/Data2/Data3 按十六进制大端打包，Data4 逐字节取自
// 字符串的 hex 对。
func guidFromString(s string) guid {
	hexVal := func(r byte) byte {
		switch {
		case r >= '0' && r <= '9':
			return r - '0'
		case r >= 'a' && r <= 'f':
			return r - 'a' + 10
		default:
			return r - 'A' + 10
		}
	}
	digits := make([]byte, 0, 32)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '{' || c == '}' {
			continue
		}
		digits = append(digits, hexVal(c))
	}
	if len(digits) != 32 {
		panic(fmt.Sprintf("invalid GUID %q", s))
	}
	quad32 := func(o int) uint32 {
		v := uint32(0)
		for k := 0; k < 8; k++ {
			v = v<<4 | uint32(digits[o+k])
		}
		return v
	}
	quad16 := func(o int) uint16 {
		v := uint16(0)
		for k := 0; k < 4; k++ {
			v = v<<4 | uint16(digits[o+k])
		}
		return v
	}
	var g guid
	g.Data1 = quad32(0)
	g.Data2 = quad16(8)
	g.Data3 = quad16(12)
	for k := 0; k < 8; k++ {
		g.Data4[k] = digits[16+2*k]<<4 | digits[17+2*k]
	}
	return g
}

// iFileDialogVtbl 按声明顺序列出 IUnknown + IModalWindow::Show +
// IFileDialog 全部 16 个方法 + IFileOpenDialog::GetResult，共 21 个
// vtable 条目（shobjidl_core.h）。
type iFileDialogVtbl struct {
	queryInterface      uintptr // 0
	addRef              uintptr // 1
	release             uintptr // 2
	show                uintptr // 3
	setFileTypes        uintptr // 4
	setFileTypeIndex    uintptr // 5
	getFileTypeIndex    uintptr // 6
	getFileName         uintptr // 7
	setFileName         uintptr // 8
	setOptions          uintptr // 9
	getOptions          uintptr // 10
	setDefaultFolder    uintptr // 11
	setFolder           uintptr // 12
	getFolder           uintptr // 13
	setDefaultExtension uintptr // 14
	setDefaultFileName  uintptr // 15
	setOkButtonLabel    uintptr // 16
	setFileNameLabel    uintptr // 17
	advise              uintptr // 18
	unadvise            uintptr // 19
	getResult           uintptr // 20
}

// shellItemVtbl IShellItem 前 8 个条目（GetDisplayName = slot 5）。
type shellItemVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	bindToHandler  uintptr
	getParent      uintptr
	getDisplayName uintptr
	getAttributes  uintptr
	compare        uintptr
}

// vtblOf 读取 COM 对象的 vtable 指针。
func vtblOf(obj unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(obj)
}

// vtblSlot 把 vtable 字段偏移换算为槽位序号（供测试断言）。
func vtblSlot(fieldOffset uintptr) int {
	return int(fieldOffset / unsafe.Sizeof(uintptr(0)))
}

// comMethod 调用 COM 方法（this 已含在 args 中），返回 HRESULT。
func comMethod(fn uintptr, args ...uintptr) uintptr {
	hr, _, _ := syscall.SyscallN(fn, args...)
	return hr
}

// OpenFolderDialog shows the modern file dialog in folder-picking mode
// (FOS_PICKFOLDERS) — 与"打开文件"对话框同款、带地址栏。
//
// 返回选中目录；用户取消返回 ok=false；失败返回具体错误供状态栏展示。
func OpenFolderDialog(owner syscall.Handle, initialDir string) (string, bool, error) {
	// 线程未初始化 COM 时初始化；S_FALSE 表示本线程已初始化（成功）。
	// RPC_E_CHANGED_MODE 表示已在其他套间初始化，COM 仍可用，直接继续。
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartment)
	switch uhr := uint32(hr); {
	case uhr == 0 || uhr == hrSFalse:
		defer procCoUninitialize.Call()
	case uhr == hrRPCChangedMode:
		// 已在其他套间：继续使用现有 COM 环境。
	default:
		return "", false, fmt.Errorf("CoInitializeEx failed: 0x%08x", hr)
	}
	dialog, err := createFileOpenDialog()
	if err != nil {
		return "", false, err
	}
	vtbl := (*iFileDialogVtbl)(vtblOf(dialog))
	defer syscall.SyscallN(vtbl.release, uintptr(unsafe.Pointer(dialog)))

	// GetOptions → 追加 FOS_PICKFOLDERS → SetOptions（DWORD 按值传参）。
	var options uint32
	if hr := comMethod(vtbl.getOptions, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(&options))); hr != 0 {
		return "", false, fmt.Errorf("GetOptions failed: 0x%08x", hr)
	}
	options |= fosPickFolders | fosForceFileSystem | fosPathMustExist
	if hr := comMethod(vtbl.setOptions, uintptr(unsafe.Pointer(dialog)), uintptr(options)); hr != 0 {
		return "", false, fmt.Errorf("SetOptions failed: 0x%08x", hr)
	}
	if initialDir != "" {
		if item, err := parseShellItem(initialDir); err == nil {
			itemVtbl := (*shellItemVtbl)(vtblOf(item))
			comMethod(vtbl.setFolder, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(item)))
			comMethod(itemVtbl.release, uintptr(unsafe.Pointer(item)))
		}
	}

	// Show：模态弹出；用户取消返回非 0 HRESULT。
	if hr := comMethod(vtbl.show, uintptr(unsafe.Pointer(dialog)), uintptr(owner)); hr != 0 {
		return "", false, nil // 取消或失败均静默关闭
	}

	var item unsafe.Pointer
	if hr := comMethod(vtbl.getResult, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(&item))); hr != 0 || item == nil {
		return "", false, fmt.Errorf("GetResult failed: 0x%08x", hr)
	}
	path, err := shellItemPath(item)
	itemVtbl := (*shellItemVtbl)(vtblOf(item))
	comMethod(itemVtbl.release, uintptr(unsafe.Pointer(item)))
	if err != nil {
		return "", false, err
	}
	return path, path != "", nil
}

// createFileOpenDialog 创建 IFileDialog 实例。
func createFileOpenDialog() (unsafe.Pointer, error) {
	clsid := guidFromString(clsidFileOpenDialog)
	iid := guidFromString(iidIFileDialog)
	var dialog unsafe.Pointer
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if hr != 0 || dialog == nil {
		return nil, fmt.Errorf("CoCreateInstance failed: 0x%08x", hr)
	}
	return dialog, nil
}

// parseShellItem 由路径创建 IShellItem（供 SetFolder 使用）。
func parseShellItem(path string) (unsafe.Pointer, error) {
	iid := guidFromString(iidIShellItem)
	units := utf16Encode(path)
	var item unsafe.Pointer
	hr, _, _ := procSHCreateItemFromPars.Call(
		uintptr(unsafe.Pointer(&units[0])),
		0,
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(&item)),
	)
	if hr != 0 || item == nil {
		return nil, fmt.Errorf("SHCreateItemFromParsingName failed: 0x%08x", hr)
	}
	return item, nil
}

// shellItemPath 经 IShellItem::GetDisplayName 取文件系统路径。
func shellItemPath(item unsafe.Pointer) (string, error) {
	vtbl := (*shellItemVtbl)(vtblOf(item))
	var displayName unsafe.Pointer
	if hr := comMethod(vtbl.getDisplayName, uintptr(unsafe.Pointer(item)), sigdnFilesystemPath, uintptr(unsafe.Pointer(&displayName))); hr != 0 || displayName == nil {
		return "", fmt.Errorf("GetDisplayName failed: 0x%08x", hr)
	}
	defer procCoTaskMemFree.Call(uintptr(displayName))
	units := unsafe.Slice((*uint16)(displayName), 1<<15)
	for i, u := range units {
		if u == 0 {
			return string(utf16Decode(units[:i])), nil
		}
	}
	return "", nil
}

func utf16Encode(s string) []uint16 {
	runes := []rune(s)
	out := make([]uint16, 0, len(runes)+1)
	for _, r := range runes {
		out = append(out, uint16(r))
	}
	return append(out, 0)
}

func utf16Decode(units []uint16) []rune {
	runes := make([]rune, 0, len(units))
	for _, u := range units {
		runes = append(runes, rune(u))
	}
	return runes
}
