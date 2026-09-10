//go:build windows

package filewatch

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateFileW  = kernel32.NewProc("CreateFileW")
	procReadDirChgW  = kernel32.NewProc("ReadDirectoryChangesW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	procCreateEventW = kernel32.NewProc("CreateEventW")
	procSetEvent     = kernel32.NewProc("SetEvent")
	procWaitForMulti = kernel32.NewProc("WaitForMultipleObjects")
)

const (
	fileListDirectory      = 0x00000001
	fileShareRead          = 0x00000001
	fileShareWrite         = 0x00000002
	fileShareDelete        = 0x00000004
	openExisting           = 3
	fileFlagBackupSemantics = 0x02000000
	invalidHandleValue     = ^uintptr(0)

	fileNotifyChangeFileName  = 0x00000001
	fileNotifyChangeLastWrite = 0x00000010
	fileNotifyChangeSize      = 0x00000008

	waitObject0 = 0
)

// FILE_NOTIFY_INFORMATION 变体记录（详见 Win32 文档）。
type fileNotifyInformation struct {
	NextEntryOffset uint32
	Action          uint32
	FileNameLength  uint32
	// FileName 紧随其后（UTF-16，非 NUL 结尾）。
}

const (
	fileActionAdded            = 1
	fileActionRemoved          = 2
	fileActionModified         = 3
	fileActionRenamedOldName   = 4
	fileActionRenamedNewName   = 5
)

// windowsWatcher 以同步阻塞的 ReadDirectoryChangesW 循环监视目录子树。
// 每个被监视目录占用一个 goroutine；Close 通过事件唤醒阻塞的等待。
type windowsWatcher struct {
	mu      sync.Mutex
	handles []syscall.Handle
	stopped bool
	stopEvt syscall.Handle
}

// New 返回平台 FileWatcher 实现。
func New() (FileWatcher, error) {
	evt, _, err := procCreateEventW.Call(0, 0, 0, 0)
	if evt == 0 {
		return nil, err
	}
	return &windowsWatcher{stopEvt: syscall.Handle(evt)}, nil
}

func (w *windowsWatcher) WatchDir(dir string) error {
	utf16, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	h, _, err := procCreateFileW.Call(
		uintptr(unsafe.Pointer(utf16)),
		fileListDirectory,
		fileShareRead|fileShareWrite|fileShareDelete,
		0, openExisting,
		fileFlagBackupSemantics,
		0,
	)
	if h == invalidHandleValue || h == 0 {
		return err
	}
	w.mu.Lock()
	w.handles = append(w.handles, syscall.Handle(h))
	stopped := w.stopped
	w.mu.Unlock()
	if stopped {
		procCloseHandle.Call(h)
		return nil
	}
	go w.loop(syscall.Handle(h), dir)
	return nil
}

// WatchFile 通过监视其所在目录实现（子树通知覆盖单文件变更）。
func (w *windowsWatcher) WatchFile(path string) error {
	dir := filepathDir(path)
	if dir == "" {
		return nil
	}
	return w.WatchDir(dir)
}

// notifyBufSize 是通知缓冲大小；必须能容纳一次批量变更，64KB 足够
//（超限时 Windows 会整体丢弃通知并置 bytesReturned=0，需要全量刷新兜底）。
const notifyBufSize = 64 * 1024

func (w *windowsWatcher) loop(handle syscall.Handle, dir string) {
	// 每个监视循环独占一块 64KB 通知缓冲（fsnotify 同款模式）。
	// 缓冲逃逸到堆并在整个循环生命周期内稳定存活；unsafe.Pointer
	// 转换仅出现在 syscall.Call 的参数位置——vet 认可该位置，且
	// Go 运行时对 syscall 参数引用的内存会自动钉住（pinned），
	// 后台写入期间缓冲区不会被回收或移动，避免 0xc0000005。
	buf := make([]byte, notifyBufSize)

	for {
		w.mu.Lock()
		stopped := w.stopped
		w.mu.Unlock()
		if stopped {
			return
		}
		var bytesReturned uint32
		// 签名：ReadDirectoryChangesW(hDir, lpBuffer, nBufferLength,
		// bWatchSubtree, dwNotifyFilter, lpBytesReturned, lpOverlapped,
		// lpCompletionRoutine)——lpBytesReturned 是第 6 个参数。
		ok, _, _ := procReadDirChgW.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&buf[0])),
			notifyBufSize,
			1, // bWatchSubtree
			fileNotifyChangeFileName|fileNotifyChangeLastWrite|fileNotifyChangeSize,
			uintptr(unsafe.Pointer(&bytesReturned)),
			0, 0, // 同步模式：阻塞直至有变更
		)
		if ok == 0 {
			return // 目录句柄失效（目录被删）或已停止
		}
		if bytesReturned == 0 {
			// 缓冲区溢出（批量变更超过 64KB）：通知已整体丢弃，
			// 发出一个全域刷新信号让上层兜底。
			globalDispatch(Event{Path: dir, Kind: EventOverflow})
			continue
		}
		w.emit(dir, buf[:bytesReturned])
	}
}

// emit 解析通知缓冲并把命中文件的事件投递给 dispatcher。
// raw 来自 VirtualAlloc 系统内存包装的切片，bytesReturned 为有效字节数。
func (w *windowsWatcher) emit(dir string, raw []byte) {
	offset := 0
	for {
		if offset+12 > len(raw) {
			return
		}
		info := (*fileNotifyInformation)(unsafe.Pointer(&raw[offset]))
		nameStart := offset + 12
		nameLen := int(info.FileNameLength) / 2
		if nameStart+nameLen*2 > len(raw) {
			return
		}
		nameUTF16 := make([]uint16, nameLen)
		for i := 0; i < nameLen; i++ {
			nameUTF16[i] = *(*uint16)(unsafe.Pointer(&raw[nameStart+i*2]))
		}
		name := syscall.UTF16ToString(nameUTF16)
		kind := EventModified
		switch info.Action {
		case fileActionRemoved:
			kind = EventDeleted
		case fileActionRenamedOldName:
			kind = EventDeleted
		case fileActionAdded, fileActionRenamedNewName, fileActionModified:
			kind = EventModified
		}
		if callback := globalDispatch(Event{Path: joinPath(dir, name), Kind: kind}); !callback {
			return
		}
		if info.NextEntryOffset == 0 {
			return
		}
		offset += int(info.NextEntryOffset)
	}
}

func (w *windowsWatcher) Close() error {
	w.mu.Lock()
	w.stopped = true
	handles := append([]syscall.Handle(nil), w.handles...)
	w.handles = nil
	w.mu.Unlock()
	// SetEvent 唤醒可能阻塞的 WaitForMultipleObjects 使用者（预留）。
	procSetEvent.Call(uintptr(w.stopEvt))
	for _, h := range handles {
		procCloseHandle.Call(uintptr(h)) // 令阻塞中的 ReadDirectoryChangesW 返回
	}
	return nil
}
