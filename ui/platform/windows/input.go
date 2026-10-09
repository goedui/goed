//go:build windows

package windows

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func keyDown(vk int) bool {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

// currentOSThreadID 返回当前 OS 线程 id。宿主登记 UI 线程，平台层据此判断
// 某次调用是否已经在 UI 线程上。
func currentOSThreadID() uint32 {
	id, _, _ := procGetCurrentThreadId.Call()
	return uint32(id)
}

var (
	modComdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")

	procGetSaveFileNameW       = modComdlg32.NewProc("GetSaveFileNameW")
	procGetOpenFileNameW       = modComdlg32.NewProc("GetOpenFileNameW")
	procSHGetKnownFolderPath   = modShell32.NewProc("SHGetKnownFolderPath")
	procSHGetKnownFolderIDList = modShell32.NewProc("SHGetKnownFolderIDList")
	procSHBrowseForFolderW     = modShell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW   = modShell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree          = modOle32.NewProc("CoTaskMemFree")

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
	cfUnicodeText      = 13
	gmemMoveable       = 0x0002
	ofnExplorer        = 0x00080000
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnOverwritePrompt = 0x00000002
	ofnHidereadonly    = 0x00000004
	maxPath            = 260

	// 老式「选择文件夹」对话框（SHBrowseForFolderW）的 flag。
	//
	// bifEditBox 是**用户要的「所有路径」**：老式对话框没有地址栏，不给它一个
	// 能输入路径的框，选不到就是选不到。
	bifReturnOnlyFSDirs = 0x00000001
	bifEditBox          = 0x00000010
	bifNewDialogStyle   = 0x00000040
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

// utf16PtrToString 读一个**以 NUL 结尾、长度未知**的原生 UTF-16 串。
//
// 和 utf16SliceToString 的分工：那个的调用方自己知道缓冲区有多大（定长数组），
// 这个只拿到一个指针——长度得靠走到 NUL 才知道。所以必须有 maxNativeString 兜底，
// 否则指针一旦不是字符串就会一直读下去，直到踩进不可读的页把进程带走。
//
// ok=false 表示「指针为空」或「到上限还没见到 NUL」。**这里绝不返回截断的串**：
// 半条路径比没有路径更糟——调用方会拿它去 Stat，然后报一个看不懂的「文件不存在」。
func utf16PtrToString(p *uint16) (string, bool) {
	if p == nil {
		return "", false
	}
	for n := 0; n < maxNativeString; n++ {
		if *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n)*2)) == 0 {
			return syscall.UTF16ToString(unsafe.Slice(p, n)), true
		}
	}
	return "", false
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
	path, ok := utf16PtrToString(p)
	if !ok {
		return ""
	}
	return path
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

// folderDialogEnv 是挑「选择文件夹」后端的开关，取值 "modern" 时用 IFileOpenDialog
// （就是资源管理器那种带导航窗格、地址栏、面包屑的界面）。
//
// **默认是老式的**，因为它是本机唯一确定能弹出来的那个。Common Item Dialog 在这台机器上
// 从第 0 秒起就不抽消息：四个 COM 调用（CoInitializeEx / CoCreateInstance / SetOptions /
// SetTitle）全部 S_OK，窗口也建出来了（#32770 下 DUIViewWndClassName、NamespaceTreeControl、
// Breadcrumb Parent、Address Band Root、SysTreeView32 一应俱全），但
// SendMessageTimeout(WM_NULL) 一直超时、IsHungAppWindow 从第 2 秒起为 true、Show 永不返回、
// WM_CLOSE 与 WM_COMMAND(IDCANCEL) 都进不去。所以它一旦被弹出来就**收不回去**，只能连进程
// 一起结束 —— 这就是默认关掉的全部理由。
//
// 这**不是本文件实现的问题**，五组对照全部复现（现场在 tmp/ 下）：
//
//	Go 宿主          tmp/dlg-new.log、dlg-newmain.log、dlg-app5.log
//	Python(ctypes) 宿主 tmp/nongo-py-folder.log（非 Go，排除 Go 运行时）
//	声明 / 不声明 DPI 感知 tmp/nongo-py-nodpi.log（排除 DPI 虚拟化）
//	选文件夹 / 选文件  tmp/dlg-min.log（不设任何 options 的普通「打开文件」）
//	自建 COM / comdlg32 tmp/dlg-comdlg.log（GetOpenFileNameW，微软自己的入口）
//
// 而同一台机器上老式 SHBrowseForFolderW 完全正常（tmp/dlg-shot.log）。环境侧也查过：
// 窗口站 WinSta0、桌面 Default、Job 对象 UILIMIT=0x0（排除沙箱/会话限制）；无 NoDrives
// 策略；WSearch 在跑。系统是 Windows 10 22H2 / 19045，最后一个热修复停在 2023-12。
//
// 结论：卡在 DirectUI 宿主的现代 shell 对话框自身，**环境相关**，换台机器可能就好。
// 设成 "modern" 就切过去；切之前先自己确认那个会话弹得出 IFileOpenDialog，弹不出会卡住整个界面。
const folderDialogEnv = "GOED_FOLDER_DIALOG"

// OpenFolderDialog 打开系统「选择文件夹」对话框。
//
// 返回值是 (路径, 选中了吗, 出错了吗)。**用户取消不算错**：ok=false、err=nil。
// 后端由 folderDialogEnv 决定，默认老式（见该常量的说明）。
func OpenFolderDialog(hwnd uintptr, title string) (string, bool, error) {
	if title == "" {
		title = "选择文件夹"
	}
	if useModernFolderDialog() {
		return openFolderDialogModern(hwnd, title)
	}
	return openFolderDialogLegacy(hwnd, title)
}

// useModernFolderDialog 报告该不该走 IFileOpenDialog 后端。
//
// 单独拎出来是为了能被用例直接钉住：把判断写在 OpenFolderDialog 里面，就只能靠
// 「真弹一次对话框」来验证，而现代后端在这台机器上根本弹不出来（见 folderDialogEnv）。
func useModernFolderDialog() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(folderDialogEnv)), "modern")
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

// openFolderDialogLegacy 是老式对话框（shell32 的 SHBrowseForFolderW）。
//
// 相对改造前有两处改动，都是为了「所有盘和路径」：
//
//   - **pidlRoot 从「桌面」换成「此电脑」**。原来传 0（桌面根），树的根就是桌面，
//     一级子项是「用户目录 / 此电脑 / 库 / 网络 + 一屏桌面文件夹」——盘符藏在
//     「此电脑」底下，界面上第一眼只看得见「桌面 → 用户目录」，这就是 bug 现场
//     （实测树内容见 tmp/dlg-old2.log）。换成此电脑之后，四个盘符直接是树的一级子项。
//   - **加上 bifEditBox**：多一个能直接输入路径的输入框。老式对话框没有地址栏，
//     不给这个框，「选到别的路径」在界面上就没有入口。
func openFolderDialogLegacy(hwnd uintptr, title string) (string, bool, error) {
	var display [maxPath + 4]uint16
	bi := browseInfoW{
		hwndOwner:      hwnd,
		pidlRoot:       computerFolderIDList(),
		pszDisplayName: &display[0],
		lpszTitle:      utf16Ptr(title),
		ulFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle | bifEditBox,
	}
	// pidlRoot 是 shell 分配的任务内存，SHBrowseForFolderW 不会接管它。
	if bi.pidlRoot != 0 {
		defer procCoTaskMemFree.Call(bi.pidlRoot)
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

// computerFolderIDList 取「此电脑」的 PIDL。
//
// 拿不到就返回 0 —— 0 是桌面根，也就是改造前的行为。宁可退回旧样子，也不要给一个
// 打不开的对话框。
func computerFolderIDList() uintptr {
	var pidl uintptr
	hr, _, _ := procSHGetKnownFolderIDList.Call(
		uintptr(unsafe.Pointer(&folderIDComputerFolder)), 0, 0, uintptr(unsafe.Pointer(&pidl)),
	)
	if hresult(hr).failed() {
		return 0
	}
	return pidl
}

// comInit 确保当前线程的 COM 已初始化（STA），返回一个「还回去」的收尾函数。
//
// 对话框是 COM 对象，要求调用线程先 CoInitializeEx。正常路径下 UI 线程在 host.go
// 里已经初始化过（同样是 STA），这里补的那次返回 S_FALSE——按 COM 的规矩
// **S_FALSE 也占一次引用计数，必须配对一次 CoUninitialize**，所以收尾函数不能省，
// 也不能拿返回值当「要不要还」的依据。
//
// 线程已经以别的套间模型初始化过（RPC_E_CHANGED_MODE）时返回空的收尾函数：
// COM 是可用的，但这次调用没增加引用计数，去 CoUninitialize 等于把别人的还掉。
func comInit() (func(), error) {
	r, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	switch hr := hresult(r); {
	case hr == 0 || hr == sFalse: // S_OK / S_FALSE
		return func() { procCoUninitialize.Call() }, nil
	case hr == hrChangedMode:
		return func() {}, nil
	default:
		return func() {}, fmt.Errorf("github.com/goedui/goed/ui: CoInitializeEx 失败: %v", hr)
	}
}

// openFolderDialogModern 走 Vista+ 的 IFileOpenDialog + FOS_PICKFOLDERS。
//
// 界面比老式的好：左侧是标准导航窗格（快速访问 / 此电脑 / 全部盘符），顶部有地址栏
// 能直接粘任意路径，长路径也不受 MAX_PATH 限制。**但它默认不开**——原因见
// folderDialogEnv 那段说明：这台机器上它 Show 之后不抽消息，会连着界面一起卡住。
//
// 用户取消不算错：ok=false、err=nil。Show 用 HRESULT_FROM_WIN32(ERROR_CANCELLED)
// 报取消，和真失败共用失败位，所以必须先比 hrCancelled 再判 failed——顺序反了每次
// 取消都会变成一条报错提示。
func openFolderDialogModern(hwnd uintptr, title string) (string, bool, error) {
	// 套间对象是**绑线程**的：CoInitializeEx、CoCreateInstance、Show 三件事必须
	// 落在同一个 OS 线程上。而 goroutine 会在 syscall 返回处被调度器搬到别的
	// 线程（exitsyscall 之后不保证回原线程），一搬走 CoCreateInstance 就报
	// CO_E_NOTINITIALIZED（0x800401F0）——这条不是推测，是本文件那条用例先挂出来
	// 的现场。所以整段钉住。
	//
	// 生产路径上调用方就是 UI 线程，它已经被 host.go 的消息循环永久
	// LockOSThread 着；这里多锁一层只是把计数 +1，defer 里还回去，净效果为零。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	done, err := comInit()
	if err != nil {
		return "", false, err
	}
	defer done()

	var dlg com
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)),
		uintptr(unsafe.Pointer(&dlg)),
	)
	if hresult(hr).failed() || dlg == 0 {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 创建文件夹对话框失败: %v", hresult(hr))
	}
	defer release(dlg)

	// FOS_PICKFOLDERS 是「选目录」的唯一开关；FOS_FORCEFILESYSTEM 把「库」
	// 「网络位置」这类没有文件系统路径的虚拟节点挡掉——留着它们，用户点完确定
	// 只会撞上下面 GetDisplayName 的失败。
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogSetOptions), uintptr(dlg), fosPickFolders|fosForceFileSystem)
	if hresult(hr).failed() {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 文件夹对话框 SetOptions 失败: %v", hresult(hr))
	}
	// 标题是**新分配的** Go 内存，只有一个局部变量指着它。转成 uintptr 之后
	// 那个引用就断了（uintptr 不是指针，GC 不认），所以必须 KeepAlive 到调用之后
	// ——否则 SetTitle 有可能读到一块已经被回收的内存。&dlg/&item/&raw 不用管：
	// 它们后面还要用，本来就是活的。
	titlePtr := utf16Ptr(title)
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogSetTitle), uintptr(dlg), uintptr(unsafe.Pointer(titlePtr)))
	runtime.KeepAlive(titlePtr)
	if hresult(hr).failed() {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 文件夹对话框 SetTitle 失败: %v", hresult(hr))
	}

	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogShow), uintptr(dlg), hwnd)
	if hresult(hr) == hrCancelled {
		return "", false, nil
	}
	if hresult(hr).failed() {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 文件夹对话框 Show 失败: %v", hresult(hr))
	}

	var item com
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogGetResult), uintptr(dlg), uintptr(unsafe.Pointer(&item)))
	if hresult(hr).failed() || item == 0 {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 文件夹对话框 GetResult 失败: %v", hresult(hr))
	}
	defer release(item)

	// SIGDN_FILESYSPATH 拿到的串由 COM 任务分配器分配，必须用 CoTaskMemFree 还，
	// 不能用 Go 的 GC 或 free——它是 CoTaskMemAlloc 那一套。
	var raw *uint16
	hr, _, _ = syscall.SyscallN(
		item.slot(idxItemGetDisplayName),
		uintptr(item),
		uintptr(sigdnFileSysPath),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hresult(hr).failed() || raw == nil {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 取文件夹路径失败: %v", hresult(hr))
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(raw)))

	path, ok := utf16PtrToString(raw)
	if !ok {
		return "", false, fmt.Errorf("github.com/goedui/goed/ui: 文件夹路径读取失败（超长或未终止）")
	}
	return path, true, nil
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
