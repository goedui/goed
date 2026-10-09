//go:build windows

package platform

import (
	"sync/atomic"
	"syscall"
)

// uiThreadID 是窗口属主线程（也就是消息循环所在线程）的 id。
// 0 表示宿主还没起来。SetWindowPos 这类调用必须在它上面发。
var uiThreadID atomic.Uint32

var procGetCurrentThreadId = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")

func currentThreadID() uint32 {
	id, _, _ := procGetCurrentThreadId.Call()
	return uint32(id)
}

// SetUIThread 由宿主在窗口创建后调用，登记 UI 线程 id。
func SetUIThread(id uint32) { uiThreadID.Store(id) }

// onUIThread 报告当前是否跑在 UI 线程上。宿主未就绪时保守地返回 false，
// 这样调用会被投递出去而不是就地执行。
func onUIThread() bool {
	id := uiThreadID.Load()
	return id != 0 && id == currentThreadID()
}

// requestClientSize 把「改客户区尺寸」投递到 UI 线程执行。
//
// SetWindowPos 跨线程时是同步调用：它会等窗口属主线程处理完 WM_WINDOWPOSCHANGING
// 才返回。如果属主线程此刻没在跑消息循环（比如刚好阻塞在别处），调用方会一直
// 卡住。所以这里统一投递，调用方永远不直接碰窗口。
//
// 返回值语义：true 表示「已受理或已执行」，false 表示宿主还没准备好。
// 想确认尺寸真的改了，读 GetWindowSize。
func requestClientSize(fn func(w, h int) bool, w, h int) bool {
	if fn == nil {
		return false
	}
	if onUIThread() {
		return fn(w, h)
	}
	if Post == nil {
		// 宿主还没绑 Post（启动早期）。这时候窗口属主线程正在建窗，
		// 就地调用反而会死锁，直接拒绝更安全。
		return false
	}
	Post(func() { fn(w, h) })
	return true
}

func requestClose(fn func()) {
	if fn == nil {
		return
	}
	if onUIThread() {
		fn()
		return
	}
	if Post != nil {
		Post(fn)
	}
}

func requestToggleMaximize(fn func() bool) bool {
	if fn == nil {
		return false
	}
	if onUIThread() {
		return fn()
	}
	if Post == nil {
		return false
	}
	Post(func() { fn() })
	return true
}
