package platform

// CaptionButton 标识自定义标题栏右侧的系统按钮。
type CaptionButton int

const (
	CaptionNone CaptionButton = iota
	CaptionMin
	CaptionMax
	CaptionClose
)

// FrameState 是当前窗口装饰的即时状态，供 TitleBar 在每帧读取。
// 这里的 Frame 指窗口边框（标题、激活、最大化、三按钮），不是浏览器。
type FrameState struct {
	Title     string
	Maximized bool
	Active    bool
	Hover     CaptionButton
	Pressed   CaptionButton
}

var frameState FrameState

// SetWindowTitle 更新窗口标题（标题栏 / 任务栏 / Alt+Tab）。非 Windows 为 no-op。
var SetWindowTitle func(title string)

// Frame 返回当前窗口装饰状态。仅在 UI 线程读取。
func Frame() FrameState {
	return frameState
}

// SetFrame 由平台层在命中测试 / 尺寸变化时更新。
func SetFrame(s FrameState) {
	frameState = s
}

// SetTitle 同时更新标题栏显示与窗口标题。编辑器这类程序在「新建 / 打开 /
// 另存为 / 内容变更」后调用，标题里的文件名与修改标记才会跟着走。
func SetTitle(title string) {
	st := frameState
	st.Title = title
	frameState = st
	if SetWindowTitle != nil {
		SetWindowTitle(title)
	}
}
