//go:build windows

package windows

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// 这一组用例测「选择文件夹」对话框。
//
// 被修的病：用户报「选择工作空间只能选到 此电脑的用户目录，期望所有盘和路径」。
// 根因在 openFolderDialogLegacy 的 pidlRoot —— 原来传 0（桌面根），树的根就成了
// 「桌面」，一级子项是「用户目录 / 此电脑 / 库 / 网络 + 一屏桌面文件夹」，盘符藏在
// 「此电脑」底下，而且老式对话框没有地址栏（现场：tmp/dlg-old2.log）。修法是把
// pidlRoot 换成 FOLDERID_ComputerFolder（树根直接是「此电脑」，盘符就在它下面）并
// 加上 BIF_EDITBOX（不给输入框，「选到任意路径」在界面上就没有入口）。
//
// 为什么非要开真窗口：这两件事**都不是纯计算的**。pidlRoot 传错不会编译失败，只会
// 让树根变成另一个文件夹；flag 少一位不会报错，只会少一个控件。只有把对话框真的弹
// 出来、再去读它的窗口树和树控件内容，才谈得上「验过」。
//
// 读控件文本一律走 SendMessageTimeoutW 而不是 GetWindowTextW：对**本进程内**的窗口
// GetWindowText 是发 WM_GETTEXT、同步等对方线程处理，对方一旦不抽消息就会把整个测试
// 进程挂死，连失败信息都印不出来（这台机器上的 Common Item Dialog 就是这样，探针第一
// 次跑就被它挂掉过）。加了超时之后，卡住的表现是「读回空串」，用例报错、进程照常退出。

var (
	procEnumWindows              = modUser32.NewProc("EnumWindows")
	procEnumChildWindows         = modUser32.NewProc("EnumChildWindows")
	procGetClassNameW            = modUser32.NewProc("GetClassNameW")
	procGetDlgCtrlID             = modUser32.NewProc("GetDlgCtrlID")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procIsWindow                 = modUser32.NewProc("IsWindow")
	procIsWindowVisible          = modUser32.NewProc("IsWindowVisible")
	procSendMessageTimeoutW      = modUser32.NewProc("SendMessageTimeoutW")
	procSHGetNameFromIDList      = modShell32.NewProc("SHGetNameFromIDList")
)

const (
	// legacyDialogClass 是老式对话框（含 SHBrowseForFolderW）的顶层窗口类名。
	legacyDialogClass = "#32770"
	// modernDialogClass 是现代 Common Item Dialog 的 DirectUI 宿主窗口类名。
	// 本机弹不出来，只用来做「不该出现」的反面断言。
	modernDialogClass = "DUIViewWndClassName"
	// shellNameSpaceClass 是老式「选择文件夹」对话框内层的 shell 命名空间控件。
	// 光认 #32770 会抓错（别的模态框也是这个类），加上这一条才是它独有的签名。
	shellNameSpaceClass = "SHBrowseForFolder ShellNameSpace Control"
	// treeViewClass 是目录树的窗口类名。
	treeViewClass = "SysTreeView32"

	wmGetText = 0x000D
	wmCommand = 0x0111
	idCancel  = 2 // 取消按钮的控件 id

	smtoBlock       = 0x0001
	smtoAbortIfHung = 0x0002

	// SysTreeView32 的消息号与 TVITEMW 的字段位，都是 comctl32 的固定值。
	tvmFirst    = 0x1100
	tvmGetNext  = tvmFirst + 10 // TVM_GETNEXTITEM
	tvmGetItemW = tvmFirst + 62 // TVM_GETITEMW
	tvgnRoot    = 0x0000
	tvifText    = 0x0001

	// sigdnNormalDisplay 是 SIGDN 的「普通显示名」档（0 就是它）。
	sigdnNormalDisplay = 0

	// textTimeoutMS 是读控件文本的等待上限。
	//
	// 正常的老式对话框毫秒级就回（探针用 50ms 都够）；给到 1 秒是留余量，同时保证
	// 「目标线程不抽消息」这种真故障在用例里表现为失败而不是永久挂起。
	textTimeoutMS = 1000

	// dialogTitle 是塞进 lpszTitle 的标题。
	//
	// 注意它**不会**出现在顶层窗口的标题栏上：shell 把顶层 #32770 的标题写成
	// 「浏览文件夹」，我们传的串落在里面那个 Static 上（实测见 tmp/dlg-shot.log）。
	// 所以定位对话框不能靠标题，靠类名 + 进程 + 内层 shell 控件。
	dialogTitle = "github.com/goedui/goed-test 选择文件夹"

	// folderDialogUIChildEnv 把这条用例切成「父进程」和「子进程」两半。
	//
	// 值非空时当前进程就是子进程，值本身是**结论文件的路径**。
	folderDialogUIChildEnv = "GOED_FOLDER_DIALOG_UI_CHILD"
)

// dialogOutcome 是后台那次 OpenFolderDialog 的返回值。
//
// 单独起个包级类型（而不是在用例里现声明一个匿名结构体）：等待窗口的那个 helper
// 要收同一个 channel，匿名类型跨函数传不过去。
type dialogOutcome struct {
	path string
	ok   bool
	err  error
}

// tvItemW 是 TVITEMW（comctl32 v6 的十字段版）。
//
// 显式写出填充字段：布局错了不会编译失败，只会让 TVM_GETITEMW 读到错位的指针然后
// 崩或者返回垃圾。用例里配了一条 Sizeof 断言把 56 钉住。
type tvItemW struct {
	mask           uint32
	_              uint32 // 对齐填充：hItem 是 8 字节指针
	hItem          uintptr
	state          uint32
	stateMask      uint32
	pszText        *uint16
	cchTextMax     int32
	iImage         int32
	iSelectedImage int32
	cChildren      int32
	lParam         uintptr
}

func windowClass(hwnd uintptr) string {
	var buf [256]uint16
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return utf16SliceToString(buf[:n])
}

func windowPID(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func windowAlive(hwnd uintptr) bool {
	ok, _, _ := procIsWindow.Call(hwnd)
	return ok != 0
}

func windowVisible(hwnd uintptr) bool {
	v, _, _ := procIsWindowVisible.Call(hwnd)
	return v != 0
}

// sendMessageTimeout 发一条消息并等结果，ok=false 表示超时。
//
// 超时和「消息回了 0」必须分开：在用例里前者是故障（目标线程不抽消息），后者可能
// 只是「控件是空的」。混成一个零值，断言失败的信息就会指向错误的方向。
func sendMessageTimeout(hwnd uintptr, msg, wparam, lparam uintptr) (uintptr, bool) {
	var res uintptr
	r, _, _ := procSendMessageTimeoutW.Call(
		hwnd, msg, wparam, lparam,
		smtoBlock|smtoAbortIfHung, textTimeoutMS,
		uintptr(unsafe.Pointer(&res)),
	)
	if r == 0 {
		return 0, false
	}
	return res, true
}

// windowText 读窗口文本，读不到（含超时）返回空串。
func windowText(hwnd uintptr) string {
	var buf [512]uint16
	res, ok := sendMessageTimeout(hwnd, wmGetText, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(&buf[0])
	if !ok || res == 0 {
		return ""
	}
	return utf16SliceToString(buf[:])
}

// topLevelWindows 收齐本进程里类名等于 class 的顶层窗口。
//
// 回调里**只收句柄**，属性等枚举完再查：EnumWindows 回调期间系统持着窗口列表的锁，
// 在回调里发同步消息等于锁着全局窗口锁去等另一个线程（探针踩过这个坑）。所以下面
// descendantWindows 也是同样的两段式。
func topLevelWindows(class string) []uintptr {
	pid := uint32(os.Getpid())
	var all []uintptr
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		all = append(all, hwnd)
		return 1
	})
	procEnumWindows.Call(cb, 0)

	var out []uintptr
	for _, hwnd := range all {
		if windowPID(hwnd) == pid && windowClass(hwnd) == class {
			out = append(out, hwnd)
		}
	}
	return out
}

type childWindow struct {
	hwnd    uintptr
	class   string
	text    string
	id      int32
	visible bool
}

// descendantWindows 列出 parent 下**所有**后代窗口（EnumChildWindows 是递归的）。
func descendantWindows(parent uintptr) []childWindow {
	var handles []uintptr
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		handles = append(handles, hwnd)
		return 1
	})
	procEnumChildWindows.Call(parent, cb, 0)

	out := make([]childWindow, 0, len(handles))
	for _, hwnd := range handles {
		id, _, _ := procGetDlgCtrlID.Call(hwnd)
		out = append(out, childWindow{
			hwnd:    hwnd,
			class:   windowClass(hwnd),
			text:    windowText(hwnd),
			id:      int32(id),
			visible: windowVisible(hwnd),
		})
	}
	return out
}

func findDescendant(parent uintptr, class string) uintptr {
	for _, c := range descendantWindows(parent) {
		if c.class == class {
			return c.hwnd
		}
	}
	return 0
}

// findFolderDialog 找本进程里那个「选择文件夹」对话框。
//
// 判据是「顶层 + 本进程 + #32770 + 有 SHBrowseForFolder ShellNameSpace Control 后代」
// 四条一起。只看类名会抓错别的模态框；只看标题也不行——shell 会把顶层窗口标题改成
// 「浏览文件夹」，我们传的 lpszTitle 落在里面那个 Static 上（tmp/dlg-shot.log）。
func findFolderDialog() uintptr {
	for _, hwnd := range topLevelWindows(legacyDialogClass) {
		if findDescendant(hwnd, shellNameSpaceClass) != 0 {
			return hwnd
		}
	}
	return 0
}

// dumpTopLevel 把本进程的顶层窗口全打出来。用在「等不到对话框」这类失败上——
// 有这张表，才知道是没弹出来、还是弹成了别的样子。
func dumpTopLevel(t *testing.T) {
	t.Helper()
	pid := uint32(os.Getpid())
	var all []uintptr
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		all = append(all, hwnd)
		return 1
	})
	procEnumWindows.Call(cb, 0)
	for _, hwnd := range all {
		if windowPID(hwnd) == pid {
			t.Logf("顶层 %#010x vis=%-5v class=%-30q text=%q",
				hwnd, windowVisible(hwnd), windowClass(hwnd), windowText(hwnd))
		}
	}
}

// treeRootText 读树控件根节点的文本。
//
// TVM_GETNEXTITEM(TVGN_ROOT) 拿根句柄，TVM_GETITEMW 拿文本。**只读，绝不展开**：
// 跨线程去 TVM_EXPAND 会把进程挂死（shell 填子项要回对话框的消息循环，而循环正卡在
// 这次调用上），探针为这个坑丢过两次现场。盘符在不在根下面由「根是不是此电脑」回答
// ——「此电脑」在 Windows 里就是装盘符的那个虚拟文件夹。
func treeRootText(tree uintptr) (string, bool) {
	root, ok := sendMessageTimeout(tree, tvmGetNext, tvgnRoot, 0)
	if !ok {
		return "", false
	}
	if root == 0 {
		return "", true // 消息通了，但这棵树没有根节点
	}
	var buf [512]uint16
	item := tvItemW{
		mask:       tvifText,
		hItem:      root,
		pszText:    &buf[0],
		cchTextMax: int32(len(buf)),
	}
	_, ok = sendMessageTimeout(tree, tvmGetItemW, 0, uintptr(unsafe.Pointer(&item)))
	runtime.KeepAlive(&buf[0])
	if !ok {
		return "", false
	}
	return utf16SliceToString(buf[:]), true
}

// computerFolderDisplayName 独立地问一次「此电脑」的显示名。
//
// 用途是给「树根是此电脑」那条断言一个**不依赖界面**的对照：两边都从同一个
// KNOWNFOLDERID 出发，本地化怎么变都一致。写死「此电脑」会在英文系统上假红；写死
// 「桌面」又抓不到「pidlRoot 还是桌面根」这个 bug。
//
// 这里必须过 comInit：SHGetNameFromIDList 和 SHGetKnownFolderIDList 不一样，它要求
// 调用线程已经初始化过 COM——不初始化会拿到 CO_E_NOTINITIALIZED(0x800401F0)，而不是
// 「名字读不出来」这种更像样的错（这条是实测出来的，第一版用例就栽在这儿）。
func computerFolderDisplayName(t *testing.T) string {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	done, err := comInit()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	pidl := computerFolderIDList()
	if pidl == 0 {
		t.Fatal("SHGetKnownFolderIDList(FOLDERID_ComputerFolder) 返回空 PIDL")
	}
	defer procCoTaskMemFree.Call(pidl)

	var p *uint16
	hr, _, _ := procSHGetNameFromIDList.Call(pidl, sigdnNormalDisplay, uintptr(unsafe.Pointer(&p)))
	if hresult(hr).failed() || p == nil {
		t.Fatalf("SHGetNameFromIDList: %v", hresult(hr))
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))

	name, ok := utf16PtrToString(p)
	if !ok || name == "" {
		t.Fatalf("「此电脑」的显示名读不出来（ok=%v name=%q）", ok, name)
	}
	return name
}

// TestFolderDialogBackendSwitch 钉住后端开关。
//
// 它读一个环境变量，是纯函数，所以能直接测——这很要紧：现代后端在这台机器上弹不出来
// （见 folderDialogEnv），「真弹一次」这种验证方式对它根本不成立。默认必须是老式，
// 否则界面会在弹不出 Common Item Dialog 的机器上一起卡住。
func TestFolderDialogBackendSwitch(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want bool
	}{
		{"", false},
		{"legacy", false},
		{"LEGACY", false},
		{"modern2", false},
		{"modern", true},
		{"MODERN", true},
		{" modern ", true},
	} {
		t.Run(fmt.Sprintf("%q", tc.env), func(t *testing.T) {
			t.Setenv(folderDialogEnv, tc.env)
			if got := useModernFolderDialog(); got != tc.want {
				t.Errorf("GOED_FOLDER_DIALOG=%q 时 useModernFolderDialog()=%v，期望 %v",
					tc.env, got, tc.want)
			}
		})
	}
}

// TestComputerFolderIDListIsVirtualRoot 钉住「pidlRoot 拿到的是此电脑，不是某个真实目录」。
//
// 三条判据缺一不可：
//   - PIDL 非空：拿不到就返回 0，而 0 是桌面根 —— 那正是要修掉的老行为；
//   - 显示名非空：能读出来才说明是个可用的 shell 项；
//   - **没有文件系统路径**：这是「此电脑」和「用户目录 / 桌面」的分水岭。此电脑是虚拟
//     文件夹，SHGetPathFromIDListW 对它返回 0；一旦有人把 GUID 换成 FOLDERID_Profile
//     或 FOLDERID_Desktop，这里立刻会拿到 C:\Users\... 而报红。
func TestComputerFolderIDListIsVirtualRoot(t *testing.T) {
	pidl := computerFolderIDList()
	if pidl == 0 {
		t.Fatal("SHGetKnownFolderIDList(FOLDERID_ComputerFolder) 返回空 PIDL —— 会退回桌面根（改造前的行为）")
	}
	defer procCoTaskMemFree.Call(pidl)

	t.Logf("「此电脑」的显示名 = %q", computerFolderDisplayName(t))

	var path [maxPath + 4]uint16
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok != 0 {
		t.Fatalf("「此电脑」竟然有文件系统路径 %q —— 这个 PIDL 指向的是真实目录，不是虚拟的此电脑",
			utf16SliceToString(path[:]))
	}
}

// TestFolderDialogLegacyRootsAtComputerAndShowsEditBox 把老式对话框真的弹出来，确认
//
//  1. 目录树的根是「此电脑」—— pidlRoot 换对了；传 0 时这里是「桌面」，
//     传用户目录时是「alwaysbetter」（三种情形都有现场日志）；
//  2. 有一个**可见**的 Edit 控件 —— BIF_EDITBOX 生效了。注意判据是「可见」不是
//     「存在」：改造前那个 Edit 也在窗口里，只是 vis=false（tmp/dlg-old2.log）；
//  3. 取消 → ok=false、err=nil（取消和失败共用失败位，这条最容易被写反）。
//
// **这条用例分父子两半跑**，理由见 runLegacyDialogUIChild 上面那段注释：弹过老式
// 对话框的进程退不出，所以真正弹对话框的那一半必须关在子进程里。
func TestFolderDialogLegacyRootsAtComputerAndShowsEditBox(t *testing.T) {
	if resPath := os.Getenv(folderDialogUIChildEnv); resPath != "" {
		// 子进程分支：跑真正的断言，把结论落盘。
		//
		// 直接单独跑子进程（手工调 GOED_FOLDER_DIALOG_UI_CHILD）时，t.Error 让结论
		// 在测试输出里也看得见；父进程不看这里的 PASS/FAIL，只看结论文件。
		result := legacyDialogUI(t)
		if err := os.WriteFile(resPath, []byte(result), 0o644); err != nil {
			t.Errorf("写结论文件 %s: %v", resPath, err)
		}
		if strings.HasPrefix(result, "FAIL") {
			t.Error(result)
		}
		return
	}
	runLegacyDialogUIChild(t)
}

// runLegacyDialogUIChild 起一个子进程跑这条用例，读它写下的结论。
//
// 为什么要拆进程：实测（tmp/probe-exit-old.log、tmp/probe-exit-new.log、tmp/a.log）
// 在本进程里弹过一次老式 SHBrowseForFolderW 之后，**进程就退不出了**——用例全跑完、
// 打出 PASS 之后卡在进程收尾上，只能靠超时杀。改造前那份实现（pidlRoot=0、没有
// BIF_EDITBOX）一样退不出，所以这不是本次改动引入的，机制也还没查清（见文件末尾
// 「已知问题」）。本文件不打算顺手解决它，但要保证它不会把整个包卡死。
//
// 顺带解决第二个坑：弹过老式对话框之后，同进程里再 CoCreateInstance(CLSID_FileOpenDialog)
// 会永久阻塞（tmp/bisect.log），于是 TestFolderDialogModernOptionsRoundTrip 也会被
// 拖死。把对话框关进子进程之后，主进程从头到尾没弹过它，两条用例互不干扰。
//
// 父进程不等子进程退出（它根本不会退），而是等那个结论文件；拿到就杀。
func runLegacyDialogUIChild(t *testing.T) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("定位测试可执行文件: %v", err)
	}
	resPath := filepath.Join(t.TempDir(), "legacy-dialog-result.txt")

	cmd := exec.Command(exe, "-test.run=^TestFolderDialogLegacyRootsAtComputerAndShowsEditBox$", "-test.v")
	cmd.Env = append(os.Environ(), folderDialogUIChildEnv+"="+resPath)
	var out syncBuffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动子进程: %v", err)
	}
	defer func() {
		// 子进程不会自己退（见上面那段），所以这里必须主动杀，且必须 Wait——
		// 不 Wait 的话 exec 那两个拷贝管道输出的 goroutine 不会结束。
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(60 * time.Second)
	for {
		if raw, err := os.ReadFile(resPath); err == nil {
			result := strings.TrimSpace(string(raw))
			switch {
			case result == "OK":
				// 把子进程的现场转出来：光看一个 PASS 看不出「验了什么」，而这条
				// 用例的全部价值就在那几行里（树根是什么、Edit 可不可见）。
				t.Logf("子进程结论 OK；子进程输出：\n%s", out.String())
				return
			case strings.HasPrefix(result, "SKIP"):
				t.Skipf("子进程跳过：%s", result)
			case strings.HasPrefix(result, "FAIL"):
				t.Errorf("子进程里的对话框断言失败：%s", result)
				t.Logf("子进程输出：\n%s", out.String())
				return
			default:
				// 只写了一半，下一轮再读。
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("子进程 60 秒内没给出结论（结论文件 %s 不存在或读不出来）\n子进程输出：\n%s",
				resPath, out.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// legacyDialogUI 是子进程里真正跑的那半截：弹对话框、读窗口树和树控件、关掉它。
//
// 返回 "OK" / "SKIP: <说明>" / "FAIL: <说明>" 而不是直接 t.Fatalf：结论要跨进程交给
// 父进程，而 Fatalf 会把子进程当场终止、连结论文件都来不及写。t 只用来打日志。
func legacyDialogUI(t *testing.T) string {
	t.Setenv(folderDialogEnv, "")
	if unsafe.Sizeof(tvItemW{}) != 56 {
		return fmt.Sprintf("FAIL: TVITEMW 大小应为 56，实际 %d —— 树控件消息的入参会错位",
			unsafe.Sizeof(tvItemW{}))
	}

	wantRoot := computerFolderDisplayName(t)
	t.Logf("对照：FOLDERID_ComputerFolder 的显示名 = %q", wantRoot)

	ch := make(chan dialogOutcome, 1)
	go func() {
		// 对话框是模态的：SHBrowseForFolderW 在调用线程上跑自己的消息循环，要等用户
		// 操作完才返回。所以必须另起一条线程，让主线程还能去戳它。LockOSThread 是
		// 必须的——建窗口、跑循环、销窗口都得落在同一条 OS 线程上，而 goroutine 会
		// 在 syscall 返回处被调度器搬走。
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		path, ok, err := OpenFolderDialog(0, dialogTitle)
		ch <- dialogOutcome{path, ok, err}
	}()

	hwnd, why := waitForDialog(ch, 8*time.Second)
	if hwnd == 0 {
		dumpTopLevel(t)
		if strings.HasPrefix(why, "worker:") {
			return "FAIL: 对话框还没出现，OpenFolderDialog 就先返回了：" + why
		}
		return "SKIP: " + why
	}
	// 兜底：万一后面某条断言失败了，别把一个活着的模态框留在子进程里。
	defer func() {
		if windowAlive(hwnd) {
			procPostMessageW.Call(hwnd, wmClose, 0, 0)
		}
	}()

	var problems []string
	if modern := topLevelWindows(modernDialogClass); len(modern) != 0 {
		problems = append(problems, fmt.Sprintf("同时出现了现代对话框窗口 %v（类名 %s）",
			modern, modernDialogClass))
	}
	if got := windowText(hwnd); got != "" && got != "浏览文件夹" {
		t.Logf("顶层窗口标题是 %q（shell 一般写「浏览文件夹」，我们传的 lpszTitle 落在里面的 Static 上）", got)
	}

	kids := descendantWindows(hwnd)
	for _, c := range kids {
		t.Logf("子窗口 %#010x class=%-42q id=%-5d vis=%-5v text=%q", c.hwnd, c.class, c.id, c.visible, c.text)
	}

	// 1) 树根 = 此电脑。
	tree := findDescendant(hwnd, treeViewClass)
	switch {
	case tree == 0:
		problems = append(problems, fmt.Sprintf("窗口里没有 %s —— 老式「选择文件夹」一定带一棵目录树", treeViewClass))
	default:
		root, ok := treeRootText(tree)
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("树控件 %#x 的根节点读不出来（%d 毫秒内没回消息）", tree, textTimeoutMS))
		case root != wantRoot:
			problems = append(problems, fmt.Sprintf("目录树的根是 %q，期望 %q —— pidlRoot 没指向 FOLDERID_ComputerFolder，"+
				"盘符就藏在根下面那一层（改造前根是「桌面」）", root, wantRoot))
		}
	}

	// 2) 有可见的输入框 = BIF_EDITBOX 生效。
	var edits []childWindow
	for _, c := range kids {
		if c.class == "Edit" {
			edits = append(edits, c)
		}
	}
	switch {
	case len(edits) == 0:
		problems = append(problems, fmt.Sprintf("没有 Edit 控件 —— BIF_EDITBOX(0x%X) 没进 ulFlags，用户没有地方输路径",
			bifEditBox))
	default:
		visible := false
		for _, e := range edits {
			if e.visible {
				visible = true
			}
		}
		if !visible {
			problems = append(problems, fmt.Sprintf("Edit 控件（id=%d）不可见 —— BIF_EDITBOX(0x%X) 没生效",
				edits[0].id, bifEditBox))
		}
	}

	// 3) 取消不是错误。
	if why := closeDialog(hwnd); why != "" {
		problems = append(problems, why)
	}
	select {
	case r := <-ch:
		if r.err != nil {
			problems = append(problems, fmt.Sprintf("取消不该报错，却拿到 err=%v", r.err))
		}
		if r.ok || r.path != "" {
			problems = append(problems, fmt.Sprintf("取消之后拿到 ok=%v path=%q，期望 false/空串", r.ok, r.path))
		}
	case <-time.After(8 * time.Second):
		problems = append(problems, "取消之后 OpenFolderDialog 没返回（窗口关了，线程却还卡在对话框里）")
	}
	if windowAlive(hwnd) {
		problems = append(problems, fmt.Sprintf("对话框窗口 %#x 还在", hwnd))
	}

	if len(problems) != 0 {
		return "FAIL: " + strings.Join(problems, "；")
	}
	return "OK"
}

// TestComInitPairsRefcount 钉住 comInit 的引用计数。
//
// 生产路径上 UI 线程**已经**以 STA 初始化过 COM（host.go 里那次），所以 comInit 在
// 那里拿到的一定是 S_FALSE。S_FALSE 是「成功，但这次没做事」——按 COM 的规矩它同样
// 占一次引用计数，必须配对一次 CoUninitialize。把 S_FALSE 当失败抛出去，或者干脆不
// 还，两种写法都能编译、都能跑，坏处要到别处才显形。
//
// 观测手法：先探一次这条线程干不干净（干净才继续，脏了就跳过并说明），然后在「干净」
// 的前提下做 2 次 comInit + 2 次归还，最后再裸调一次 CoInitializeEx —— 拿到 S_OK 才
// 说明引用计数真的还干净了；comInit 漏了归还的话，这里会拿到 S_FALSE。
//
// **这条用例必须排在 TestFolderDialogModernOptionsRoundTrip 前面**（Go 按源码顺序跑）。
// 那条用例也调 comInit，一旦 comInit 漏还引用计数，它就会把线程弄脏；而这里「脏了就
// 跳过」的守卫会让这条断言静默跳过——泄漏从此没人看得见。这不是假设：反向验证第一次
// 跑出来就是一条 [缺口]（tmp/verify-folderdialog.log）。顺序反了不会报错，只会少守
// 一个行为，所以这条约束写在这里，而不是留给下一个人去猜。
func TestComInitPairsRefcount(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	probe, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	switch hresult(probe) {
	case 0:
		// 探底成功，还掉这一次，线程恢复成「未初始化」。
		procCoUninitialize.Call()
	case sFalse:
		procCoUninitialize.Call()
		// 跳过在这里是**危险**的：它既可能是线程池被别的用例污染，也可能是 comInit
		// 真的漏还了引用计数——而后者正是本用例要抓的东西。所以把话说全，别让人
		// 把「跳过」当成「没事」。
		t.Skip("这条线程上 COM 已经被初始化过（拿不到干净起点），引用计数那半条断言没法做；" +
			"注意：如果这条跳过是 comInit 漏还引用计数引起的，那么现在没有任何用例守着它")
	default:
		t.Skipf("CoInitializeEx 探底返回 %v，这条线程的套间模型不是本用例能假设的", hresult(probe))
	}

	done1, err := comInit()
	if err != nil {
		t.Fatalf("comInit 第 1 次（S_OK 分支）: %v", err)
	}
	done2, err := comInit()
	if err != nil {
		t.Fatalf("comInit 第 2 次（S_FALSE 分支，就是生产路径上那次）: %v", err)
	}
	done2()
	done1()

	after, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if got := hresult(after); got != 0 {
		t.Fatalf("两次 comInit 都归还之后，这条线程上的 COM 还没回到「未初始化」："+
			"CoInitializeEx=%v（期望 S_OK）—— done() 少还了一次", got)
	}
	procCoUninitialize.Call()
}

// TestFolderDialogModernOptionsRoundTrip 钉住 CLSID/IID 与虚表槽位 9/10/17。
//
// 手法是**往返**而不是「设了就信」：先读一次默认值，再设进去，再读回来。槽位一旦
// 串位（比如 SetOptions 写成了 10），读回来的位就不对；CLSID 或 IID 写错则
// CoCreateInstance 直接失败。这两件事都只有真起一个 COM 对象才看得出来。
//
// 这里**不 Show**：本机弹不出 Common Item Dialog（见 folderDialogEnv），Show 会把
// 测试进程一起带走。所以这条用例能覆盖的只有「对象拿得到、槽位对得上」——
// 界面那半截没有自动化验证，只有 tmp/dlg-new.log 那份现场。
//
// 还有一条隐含前提：**本进程没弹过老式对话框**。弹过之后再 CoCreateInstance 这个
// CLSID 会永久阻塞（tmp/bisect.log），所以老式那条用例必须关在子进程里
// （见 runLegacyDialogUIChild）。这两件事是绑在一起的，别把子进程隔离去掉。
func TestFolderDialogModernOptionsRoundTrip(t *testing.T) {
	// 套间对象绑线程：CoInitializeEx 与 CoCreateInstance 必须落在同一条 OS 线程上。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	done, err := comInit()
	if err != nil {
		t.Fatal(err)
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
		t.Fatalf("CoCreateInstance(CLSID_FileOpenDialog, IID_IFileOpenDialog): %v", hresult(hr))
	}
	defer release(dlg)

	var before uint32
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogGetOptions), uintptr(dlg), uintptr(unsafe.Pointer(&before)))
	if hresult(hr).failed() {
		t.Fatalf("GetOptions(槽位 %d): %v", idxDialogGetOptions, hresult(hr))
	}
	if before&fosPickFolders != 0 {
		t.Fatalf("新建的对话框默认就带 FOS_PICKFOLDERS（GetOptions=0x%X），这条往返测不出东西", before)
	}

	want := uint32(fosPickFolders | fosForceFileSystem)
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogSetOptions), uintptr(dlg), uintptr(want))
	if hresult(hr).failed() {
		t.Fatalf("SetOptions(槽位 %d): %v", idxDialogSetOptions, hresult(hr))
	}
	var after uint32
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogGetOptions), uintptr(dlg), uintptr(unsafe.Pointer(&after)))
	if hresult(hr).failed() {
		t.Fatalf("GetOptions(槽位 %d): %v", idxDialogGetOptions, hresult(hr))
	}
	if after&want != want {
		t.Fatalf("SetOptions(0x%X) 之后 GetOptions 是 0x%X（默认 0x%X）：槽位 %d/%d 至少有一个指错了地方",
			want, after, before, idxDialogSetOptions, idxDialogGetOptions)
	}

	title := utf16Ptr("github.com/goedui/goed-test")
	hr, _, _ = syscall.SyscallN(dlg.slot(idxDialogSetTitle), uintptr(dlg), uintptr(unsafe.Pointer(title)))
	runtime.KeepAlive(title)
	if hresult(hr).failed() {
		t.Fatalf("SetTitle(槽位 %d): %v", idxDialogSetTitle, hresult(hr))
	}
}

// closeDialog 把模态对话框关掉，返回空串表示关干净了。
//
// 先发 WM_COMMAND(IDCANCEL)——探针实测这一下就能关（tmp/dlg-shot.log），窗口还在就
// 再补一发 WM_CLOSE。两条都不行就把话说清楚：留着一个活着的模态框，子进程会一直卡着。
func closeDialog(hwnd uintptr) string {
	if !windowAlive(hwnd) {
		return ""
	}
	procPostMessageW.Call(hwnd, wmCommand, idCancel, 0)
	if waitWindowGone(hwnd, 5*time.Second) {
		return ""
	}
	procPostMessageW.Call(hwnd, wmClose, 0, 0)
	if waitWindowGone(hwnd, 5*time.Second) {
		return ""
	}
	return fmt.Sprintf("对话框 %#x 关不掉：取消键和 WM_CLOSE 都送进去了", hwnd)
}

func waitWindowGone(hwnd uintptr, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !windowAlive(hwnd) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitForDialog 等对话框窗口出现，返回 (句柄, "")。
//
// 没等到时返回 (0, 原因)，并且**区分**两种原因，因为它们的性质完全不同：
//   - 前缀 "worker: " —— 后台那次 OpenFolderDialog 先返回了（带着它自己的错误），
//     这是真问题，调用方要报 FAIL；
//   - 其它 —— 窗口压根没出现，多半是这台机器/这个会话弹不出对话框（比如没有交互
//     桌面），调用方要报 SKIP 而不是 FAIL。
func waitForDialog(ch <-chan dialogOutcome, timeout time.Duration) (uintptr, string) {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case r := <-ch:
			return 0, fmt.Sprintf("worker: path=%q ok=%v err=%v", r.path, r.ok, r.err)
		default:
		}
		if hwnd := findFolderDialog(); hwnd != 0 {
			return hwnd, ""
		}
		if time.Now().After(deadline) {
			return 0, fmt.Sprintf("%v 内没等到「%s + %s」的顶层窗口", timeout, legacyDialogClass, shellNameSpaceClass)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// syncBuffer 是给子进程 stdout 用的缓冲：exec 用一条 goroutine 往里写，父进程同时
// 在读，没有这把锁就是数据竞争（失败路径上才会暴露，正好是最需要看输出的时候）。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// ---- 已知问题（本文件不解决，但必须写下来，免得下一个人当它是自己弄坏的） ----
//
// 1. 弹过老式 SHBrowseForFolderW 之后，**进程退不出**。
//    现象：用例全跑完、打出 PASS，然后卡在进程收尾，只能靠超时杀（tmp/a.log 是
//    「PASS 之后被 kill」的现场；探针的 old / new / app 三种模式全都 exit=124，
//    见 tmp/probe-exit-old.log、tmp/probe-exit-new.log、tmp/probe-exit-app.log）。
//    与本次改动无关：old 模式用的就是改造前那份实现（pidlRoot=0、没有 BIF_EDITBOX）。
//    机制未查清。怀疑是进程收尾时某个 DLL 的 DllMain(PROCESS_DETACH) 在等一个已经
//    不属于任何消息循环的套间/线程，但没有证据，不敢下结论。
//    影响面：生产路径上对话框跑在 host.go 那条永久锁定、一直在抽消息的 UI 线程上，
//    与这里「临时锁定一条 goroutine 弹完就解锁」的形状不同，所以不能直接说产品有
//    这个毛病——**但也没验过**。要下结论得单独做一次「真窗口 + 消息循环里弹老式
//    对话框 + 退出」的实验。
//
// 2. 弹过老式对话框之后，同进程里 CoCreateInstance(CLSID_FileOpenDialog) 永久阻塞
//    （现场：tmp/bisect.log，栈停在 ole32 的 CoCreateInstance 里）。
//    这条用例组用「把老式对话框关进子进程」绕开了它，同时也就绕开了问题 1。
