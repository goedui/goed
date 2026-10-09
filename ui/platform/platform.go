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
	// RightPointer 报告右键，单位 DIP，down 为 true 表示按下；只在按下时回调一次
	// （抬起不通知，浏览器 contextmenu 也是这个时机）。
	//
	// 单独一条通道而不是给 Pointer 加 button 参数：Pointer 是既有公开字段，加参数要
	// 把四个宿主一起改；而右键目前只有「弹上下文菜单」一种用途，nil 就是「不要右键」。
	RightPointer func(x, y float32, down bool)
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
	// ClientHitTest 让宿主在把「标题栏区」的某个点判成 HTCAPTION 之前先问一句
	// 应用：这里是不是应用自己的可点击控件？是就返回 HTCLIENT，点击交给应用。
	//
	// 只有自绘标题栏的应用（见 AppTitleBar）需要它。那类应用把标签条这类交互控件
	// 画进了顶部那一条，而宿主默认把整条判成标题栏（拖动 / 系统按钮），
	// 那些控件的点击就永远到不了应用。nil = 不占用顶部条，行为与以前完全一致。
	ClientHitTest func(x, y float32) bool
	// Close 在窗口真正关闭前询问应用。返回 false 表示先别关。
	// nil 表示直接关。标题栏关闭、菜单退出、系统关闭都走这里。
	Close func() bool
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
// 刻意做成**函数**而不是可赋值的全局变量：宿主在 Run 里绑定它，而引擎里有一批后台
// 循环会周期性调用（光标闪烁、加载动画）—— 变量的话就是「后台 goroutine 读、宿主
// 写」，-race 直接判数据竞争（实测：一个进程里跑两个真实窗口测试就会报）。
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

// ToggleMaximize 切换窗口最大化 / 还原。窗口就绪前为 nil。
// 应用层用它做 F11 全屏。跨线程调用会投递到 UI 线程。
var ToggleMaximize func() bool

// CloseWindow 关闭窗口，与标题栏关闭按钮同一条路。窗口就绪前为 nil。
var CloseWindow func()

// CaptionButtons 报告宿主有没有系统标题栏按钮（最小化 / 最大化 / 关闭）。
//
// Windows / Linux 上这些按钮由宿主做命中测试、交系统执行，所以为 true；
// 浏览器里根本没有「系统窗口」这回事，web 宿主在 Run 里把它置为 false，
// TitleBar 于是不画右侧那三个按钮（见 ui/components/titlebar_paint.go）。
// 留在平台层而不是组件里判断 GOOS：应用可能跑在浏览器里，也可能被嵌进
// 别的东西，只有宿主自己知道有没有系统按钮。
var CaptionButtons = true

// AppTitleBar 报告应用是否自己画窗口顶部那一条。
//
// 为 true 时框架不再为标题栏预留那 32 DIP，也不画自己的标题栏（底色 / 图标 /
// 标题 / 系统按钮都不画），应用树从 y=0 起铺满整个客户区 —— 顶部那一条完全归应用。
//
// 但宿主**仍然**把顶部那一条判成标题栏：空白处的点击走 HTCAPTION（窗口拖动、
// 双击最大化由系统完成），右侧 3×renderer.CaptionButtonWidth 那三块仍然分别是
// 最小化 / 最大化 / 关闭。所以应用要自己画那三个按钮的图形（位置必须与宿主的
// 命中区对齐），并用 Options.ClientHitTest 声明「标签条这些地方是我的控件」。
//
// 与 CaptionButtons 一样留在平台层：只有宿主自己知道有没有「系统窗口」这回事。
var AppTitleBar = false
