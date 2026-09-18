package platform

import (
	"sync"

	"github.com/goedui/goed/ui/renderer"
)

// Options 描述要创建的宿主窗口。
type Options struct {
	Title  string
	Width  int
	Height int
	Dark   bool
	// Icon16 / Icon32 是 BITMAPINFOHEADER + XOR + AND 的 Windows 图标位图。
	Icon16 []byte
	Icon32 []byte
	Paint  func(ctx renderer.Context)
	// Pointer 报告客户区指针。单位 DIP。click 为 true 表示一次完整单击（左键抬起）。
	Pointer func(x, y float32, down, click bool)
	// Key 报告按键。char 非 0 表示文本输入。
	Key func(key int, char rune, down, ctrl, shift, alt bool)
	// Wheel 报告滚轮。delta 以“格”为单位（120 像素单位 / 120）。
	Wheel func(x, y, delta float32)
	// Composition 报告输入法组合串（预编辑）：text 是组合中的文本，
	// cursor 是组合串内的字符偏移。返回 true 表示应用自己绘制组合串，
	// 宿主不必再让系统绘制内联组合窗（候选窗仍由系统显示）。
	Composition func(text string, cursor int) bool
	// Caret 报告当前焦点的光标矩形（客户区 DIP），宿主用它把输入法候选窗
	// 贴到光标旁边。ok 为 false 表示暂时没有可用位置。
	Caret func() (x, y, w, h float32, ok bool)
}

var (
	requestFrameMu sync.RWMutex
	requestFrame   func()
)

// bindRequestFrame 由宿主在窗口就绪时调用一次（一个进程只有一个宿主）。
func bindRequestFrame(fn func()) {
	requestFrameMu.Lock()
	requestFrame = fn
	requestFrameMu.Unlock()
}

// RequestFrame 在当前 UI 线程请求一次重绘。
//
// 刻意做成**函数**而不是可赋值的全局变量：宿主在 Run 里绑定这个钩子
// （见 run_windows.go / run_linux.go），而引擎里有一批**后台循环**会周期性
// 调用它——光标闪烁（components/textedit_state.go）与加载动画（components/empty.go）。
// 变量的话就是「后台 goroutine 读、宿主写」，-race 直接判数据竞争
// （实测：一个进程里跑两个真实窗口测试就会报）。
// 做成函数 + 带锁绑定之后，调用方没有「忘记同步」的写法可用。
func RequestFrame() {
	requestFrameMu.RLock()
	fn := requestFrame
	requestFrameMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// Post 把函数投递到 UI 线程。非 Windows 平台为同步执行。
var Post func(func())

// HWND 是当前窗口句柄。非 Windows 为 0。
var HWND uintptr

// SetWindowSize 在运行期把窗口客户区改成 w x h（逻辑 DIP）。
//
// 可以从任意 goroutine 调用：非 UI 线程的调用会被投递到 UI 线程执行，
// 此时返回 true 只表示「已受理」，不代表尺寸已经改好；想确认结果请读
// GetWindowSize。窗口就绪前为 nil。
//
// 之所以要投递，是因为 SetWindowPos 对「别的线程拥有的窗口」是同步调用：
// 它会等属主线程处理完消息才返回，属主线程若正好阻塞在别处就会死锁。
var SetWindowSize func(w, h int) bool

// GetWindowSize 返回当前客户区逻辑尺寸（DIP）。窗口就绪前为 nil。
var GetWindowSize func() (w, h int, ok bool)
