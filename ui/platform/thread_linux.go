//go:build linux

package platform

import (
	"runtime"
	"sync/atomic"
)

// uiGoroutineID 是宿主事件循环所在 goroutine 的 id。0 表示宿主还没起来。
//
// X11 的 Xlib 只在单线程下安全，宿主用 runtime.LockOSThread 把事件循环钉住。
// 这里没法像 Windows 那样拿 OS 线程 id（没有稳定的公开接口），退而求其次：
// 宿主在事件循环开始时登记自己的 goroutine id，调用方比对 goroutine id。
// 判错的代价只是多投递一次，不会死锁。
//
// 目前 Linux 宿主还没暴露「运行期改窗口尺寸」的能力，所以 SetWindowSize
// 保持 nil，应用层调用会直接拿到 false。
var uiGoroutineID atomic.Uint64

// SetUIGoroutine 由宿主登记事件循环所在的 goroutine id。
func SetUIGoroutine(id uint64) { uiGoroutineID.Store(id) }

func onUIThread() bool {
	id := uiGoroutineID.Load()
	return id != 0 && id == currentGoroutineID()
}

// currentGoroutineID 从 runtime 栈头解析出当前 goroutine id。
// 用的是非稳定接口，但只用于「判断是否同一条线程」；取不到就返回 0
// （等于「不在 UI 线程」，最坏情况是多投递一次，无害）。
func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	const prefix = "goroutine "
	if n < len(prefix) {
		return 0
	}
	s := string(buf[len(prefix):n])
	var id uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}
