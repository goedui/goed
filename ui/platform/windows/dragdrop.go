//go:build windows

package windows

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// 拖入走 DragAcceptFiles / WM_DROPFILES，这条没问题，留着。
//
// 拖出用 Shell 的 IDataObject（资源管理器从它取 CF_HDROP），再补一个
// IDropSource。OLE 拖放要求线程先 OleInitialize；只调 CoInitializeEx 不够。
// 数据对象由 SHCreateDataObject 构造，pidlFolder 与相对子 PIDL 必须配对。

const (
	wmDropFiles = 0x0233

	dropeffectCopy = 1
	dropeffectMove = 2

	ddSDrop              = 0x00040100
	ddSCancel            = 0x00040101
	ddSUseDefaultCursors = 0x00040102

	sOK          = 0
	eNoInterface = 0x80004002

	mkLButton = 0x0001
	vkLButton = 0x01
)

// errDragCanceled 表示用户取消，或松手时指针下没有接收方。
var errDragCanceled = errors.New("拖放已取消")

var (
	procDragAcceptFiles    = modShell32.NewProc("DragAcceptFiles")
	procDragQueryFileW     = modShell32.NewProc("DragQueryFileW")
	procDragFinish         = modShell32.NewProc("DragFinish")
	procILCreateFromPathW  = modShell32.NewProc("ILCreateFromPathW")
	procILFindLastID       = modShell32.NewProc("ILFindLastID")
	procSHCreateDataObject = modShell32.NewProc("SHCreateDataObject")
	procILFree             = modShell32.NewProc("ILFree")

	procOleInitialize   = modOle32.NewProc("OleInitialize")
	procOleUninitialize = modOle32.NewProc("OleUninitialize")
	procDoDragDrop      = modOle32.NewProc("DoDragDrop")

	// GetKeyState 只看本线程消息队列，ReleaseCapture 之后经常是 0。
	// 物理按键用 GetAsyncKeyState，否则 DoDragDrop 一进来就放下。
	procGetAsyncKeyState = modUser32.NewProc("GetAsyncKeyState")
)

// iidIDataObject {0000010e-0000-0000-C000-000000000046}
var iidIDataObject = guid{
	Data1: 0x0000010e,
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// iidIDropSource {00000121-0000-0000-C000-000000000046}
var iidIDropSource = guid{
	Data1: 0x00000121,
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// iidIUnknown {00000000-0000-0000-C000-000000000046}
var iidIUnknown = guid{
	Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
}

// ===== 拖入 ==============================================================

// EnableFileDrop 让窗口接受资源管理器拖来的文件（后续走 WM_DROPFILES）。
func EnableFileDrop(hwnd uintptr) {
	if hwnd != 0 {
		procDragAcceptFiles.Call(hwnd, 1)
	}
}

// queryDroppedFiles 取 WM_DROPFILES 里的路径列表并释放句柄。
func queryDroppedFiles(hDrop uintptr) []string {
	if hDrop == 0 {
		return nil
	}
	defer procDragFinish.Call(hDrop)
	n, _, _ := procDragQueryFileW.Call(hDrop, 0xFFFFFFFF, 0, 0)
	if n == 0 {
		return nil
	}
	out := make([]string, 0, n)
	for i := uintptr(0); i < n; i++ {
		size, _, _ := procDragQueryFileW.Call(hDrop, i, 0, 0)
		if size == 0 {
			continue
		}
		buf := make([]uint16, size+1)
		procDragQueryFileW.Call(hDrop, i, uintptr(unsafe.Pointer(&buf[0])), size+1)
		out = append(out, syscall.UTF16ToString(buf[:size]))
	}
	return out
}

// ===== 拖出 ==============================================================

// oleInit 初始化 OLE 拖放。CoInitializeEx 已经调过时这里补 OLE 状态。
// S_OK / S_FALSE 都占一次引用，必须配对 OleUninitialize。
func oleInit() (func(), error) {
	r, _, _ := procOleInitialize.Call(0)
	hr := hresult(r)
	switch {
	case hr == 0 || hr == sFalse:
		return func() { procOleUninitialize.Call() }, nil
	case hr == hrChangedMode:
		return func() {}, nil
	default:
		return func() {}, hr
	}
}

// shellPidlSet 持有一个父目录 PIDL 和它下面的相对子 PIDL。
// SHCreateDataObject 的 apidl 必须是相对 pidlFolder 的子项；传完整 PIDL
// 会得到一个没有任何可用格式的对象。
type shellPidlSet struct {
	folder   uintptr
	absolute []uintptr
	children []uintptr
}

func (s shellPidlSet) free() {
	for _, pidl := range s.absolute {
		procILFree.Call(pidl)
	}
	if s.folder != 0 {
		procILFree.Call(s.folder)
	}
}

// makeShellPidlSet 把同一目录下的路径转换成 Shell 要求的 PIDL 形式。
func makeShellPidlSet(paths []string) (shellPidlSet, error) {
	set := shellPidlSet{}
	fail := func(err error) (shellPidlSet, error) {
		set.free()
		return shellPidlSet{}, err
	}
	parentPath := ""
	for _, p := range paths {
		if p == "" {
			continue
		}
		absolutePath, err := filepath.Abs(p)
		if err != nil {
			return fail(err)
		}
		absolutePath = filepath.Clean(absolutePath)
		parent := filepath.Dir(absolutePath)
		if parentPath == "" {
			parentPath = parent
		} else if !strings.EqualFold(parentPath, parent) {
			return fail(errors.New("拖出的文件必须位于同一目录"))
		}
		u, err := syscall.UTF16PtrFromString(absolutePath)
		if err != nil {
			return fail(err)
		}
		pidl, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(u)))
		if pidl == 0 {
			return fail(errors.New("无法取得拖出项目"))
		}
		set.absolute = append(set.absolute, pidl)
		child, _, _ := procILFindLastID.Call(pidl)
		if child == 0 {
			return fail(errors.New("无法取得拖出项目"))
		}
		set.children = append(set.children, child)
	}
	if len(set.children) == 0 {
		return fail(errors.New("没有可拖出的路径"))
	}
	u, err := syscall.UTF16PtrFromString(parentPath)
	if err != nil {
		return fail(err)
	}
	set.folder, _, _ = procILCreateFromPathW.Call(uintptr(unsafe.Pointer(u)))
	if set.folder == 0 {
		return fail(errors.New("无法取得拖出目录"))
	}
	return set, nil
}

// shellDataObject 用父目录 / 子 PIDL 构造 Shell 的 IDataObject。
// 资源管理器认这个，不认手写的 CF_HDROP。
func shellDataObject(paths []string) (uintptr, error) {
	set, err := makeShellPidlSet(paths)
	if err != nil {
		return 0, err
	}
	defer set.free()
	var obj uintptr
	hr, _, _ := procSHCreateDataObject.Call(
		set.folder,
		uintptr(len(set.children)),
		uintptr(unsafe.Pointer(&set.children[0])),
		0,
		uintptr(unsafe.Pointer(&iidIDataObject)),
		uintptr(unsafe.Pointer(&obj)),
	)
	if hresult(hr).failed() || obj == 0 {
		if hr == 0 {
			return 0, errors.New("SHCreateDataObject 没有返回对象")
		}
		return 0, hresult(hr)
	}
	return obj, nil
}
func releasePtr(p uintptr) {
	if p == 0 {
		return
	}
	vtbl := *(*uintptr)(unsafe.Pointer(p))
	rel := *(*uintptr)(unsafe.Pointer(vtbl + 2*unsafe.Sizeof(uintptr(0))))
	syscall.SyscallN(rel, p)
}

// dropSource 只回答「继续拖还是放下」。数据在 Shell 对象里。
type dropSource struct {
	vtbl uintptr
	refs int32
}

var (
	dropOnce sync.Once
	dropVT   [5]uintptr
	dropMu   sync.Mutex
	dropLive = map[uintptr]*dropSource{}
)

func newDropSource() uintptr {
	dropOnce.Do(func() {
		dropVT = [5]uintptr{
			syscall.NewCallback(srcQueryInterface),
			syscall.NewCallback(srcAddRef),
			syscall.NewCallback(srcRelease),
			syscall.NewCallback(srcQueryContinueDrag),
			syscall.NewCallback(srcGiveFeedback),
		}
	})
	o := &dropSource{vtbl: uintptr(unsafe.Pointer(&dropVT[0])), refs: 1}
	p := uintptr(unsafe.Pointer(o))
	dropMu.Lock()
	dropLive[p] = o
	dropMu.Unlock()
	return p
}

func srcQueryInterface(this, riid, ppv uintptr) uintptr {
	if ppv == 0 || riid == 0 {
		return eNoInterface
	}
	id := *(*guid)(unsafe.Pointer(riid))
	if id != iidIUnknown && id != iidIDropSource {
		*(*uintptr)(unsafe.Pointer(ppv)) = 0
		return eNoInterface
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = this
	srcAddRef(this)
	return sOK
}

func srcAddRef(this uintptr) uintptr {
	o := (*dropSource)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&o.refs, 1))
}

func srcRelease(this uintptr) uintptr {
	o := (*dropSource)(unsafe.Pointer(this))
	left := atomic.AddInt32(&o.refs, -1)
	if left <= 0 {
		dropMu.Lock()
		delete(dropLive, this)
		dropMu.Unlock()
	}
	return uintptr(left)
}

func srcQueryContinueDrag(_, escape, keys uintptr) uintptr {
	st, _, _ := procGetAsyncKeyState.Call(vkLButton)
	return dragContinue(escape, keys, st&0x8000 != 0)
}

// dragContinue 决定继续拖、放下还是取消。
//
// grfKeyState 在源窗口 ReleaseCapture 之后经常不带 MK_LBUTTON，
// 只信它会在指针还没离开窗口时立刻放下，文件到不了资源管理器。
// 物理左键仍按下（asyncDown）时必须继续。
func dragContinue(escape, keys uintptr, asyncDown bool) uintptr {
	if escape != 0 {
		return ddSCancel
	}
	if keys&mkLButton != 0 || asyncDown {
		return sOK
	}
	return ddSDrop
}

func srcGiveFeedback(_, _ uintptr) uintptr { return ddSUseDefaultCursors }

// DoDragDropFiles 把磁盘路径拖到桌面或资源管理器。阻塞到松手或取消。
func DoDragDropFiles(paths []string) error {
	if len(paths) == 0 {
		return errors.New("没有可拖出的文件")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	done, err := oleInit()
	if err != nil {
		return err
	}
	defer done()
	data, err := shellDataObject(paths)
	if err != nil {
		return err
	}
	defer releasePtr(data)
	src := newDropSource()
	defer releasePtr(src)
	procReleaseCapture.Call()
	var effect uint32
	// 只给 COPY 时，资源管理器的默认「移动」会拒绝这次放下。
	allowed := uintptr(dropeffectCopy | dropeffectMove)
	hr, _, _ := procDoDragDrop.Call(
		data, src, allowed, uintptr(unsafe.Pointer(&effect)),
	)
	if hresult(hr).failed() {
		return hresult(hr)
	}
	if uint32(hr) == ddSCancel || effect == 0 {
		return errDragCanceled
	}
	return nil
}
