package ui

import (
	"github.com/goedui/goed/ui/app"
	"github.com/goedui/goed/ui/components"
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// 应用层只从 github.com/goedui/goed/ui 拿声明式 API。形状对齐 Vue 3 的 h(type, props?, children)：
//
//	var Hello = ui.Component(func(_ *ui.SetupContext) func() *ui.VNode {
//	    count := ui.Ref(0)
//	    return func() *ui.VNode {
//	        return ui.VStack(
//	            ui.Text("点击了 ", count, " 次"),
//	            ui.Button(ui.Props{OnClick: func() { count.Set(count.Get() + 1) }}, "点我"),
//	        )
//	    }
//	})

type (
	VNode           = runtime.VNode
	ComponentDef    = runtime.ComponentDef
	SetupContext    = runtime.SetupContext
	Props           = runtime.Props
	Style           = runtime.Style
	Color           = runtime.Color
	Theme           = theme.Theme
	App             = app.App
	KeyEvent        = runtime.KeyEvent
	KeyEventHandler = runtime.KeyHandler
	Dispatcher      = runtime.Dispatcher
	Menu            = components.Menu
	MenuItem        = components.MenuItem
	Tab             = components.Tab
)

const (
	Start   = runtime.Start
	Center  = runtime.Center
	End     = runtime.End
	Stretch = runtime.Stretch

	KeySpace     = runtime.KeySpace
	KeyEnter     = runtime.KeyEnter
	KeyEscape    = runtime.KeyEscape
	KeyBackspace = runtime.KeyBackspace
	KeyTab       = runtime.KeyTab
	KeyHome      = runtime.KeyHome
	KeyEnd       = runtime.KeyEnd
	KeyLeft      = runtime.KeyLeft
	KeyUp        = runtime.KeyUp
	KeyRight     = runtime.KeyRight
	KeyDown      = runtime.KeyDown
	KeyPageUp    = runtime.KeyPageUp
	KeyPageDown  = runtime.KeyPageDown
	KeyDelete    = runtime.KeyDelete
	KeyA         = runtime.KeyA
	KeyC         = runtime.KeyC
	KeyF         = runtime.KeyF
	KeyL         = runtime.KeyL
	KeyN         = runtime.KeyN
	KeyO         = runtime.KeyO
	KeyS         = runtime.KeyS
	KeyT         = runtime.KeyT
	KeyV         = runtime.KeyV
	KeyW         = runtime.KeyW
	KeyX         = runtime.KeyX
	KeyY         = runtime.KeyY
	KeyZ         = runtime.KeyZ
	KeyF11       = runtime.KeyF11
)

func RGB(r, g, b uint8) Color     { return runtime.RGB(r, g, b) }
func RGBA(r, g, b, a uint8) Color { return runtime.RGBA(r, g, b, a) }

func Component(setup func(ctx *SetupContext) func() *VNode) *ComponentDef {
	return runtime.Component(setup)
}

func Ref[T any](v T) *runtime.Ref[T] {
	return runtime.NewRef(v)
}

func H(typ any, rest ...any) *VNode {
	return runtime.H(typ, rest...)
}

func Text(parts ...any) *VNode {
	return runtime.Text(parts...)
}

func VStack(parts ...any) *VNode {
	return components.VStack(parts...)
}

func HStack(parts ...any) *VNode {
	return components.HStack(parts...)
}

func Button(first any, rest ...any) *VNode {
	return components.Button(first, rest...)
}

func Input(parts ...any) *VNode {
	return components.Input(parts...)
}

func Image(parts ...any) *VNode {
	return components.Image(parts...)
}

func Textarea(parts ...any) *VNode {
	return components.Textarea(parts...)
}

func Scroll(parts ...any) *VNode {
	return components.Scroll(parts...)
}

func Switch(parts ...any) *VNode {
	return components.Switch(parts...)
}

func Empty(parts ...any) *VNode {
	return components.Empty(parts...)
}

func Box(parts ...any) *VNode {
	return components.Box(parts...)
}

func Divider(parts ...any) *VNode {
	return components.Divider(parts...)
}

func Select(parts ...any) *VNode {
	return components.Select(parts...)
}

func MenuBar(parts ...any) *VNode {
	return components.MenuBar(parts...)
}

func Tabs(parts ...any) *VNode {
	return components.Tabs(parts...)
}

func SetSelectOpen(id string, open bool) {
	components.SetSelectOpen(id, open)
}

// ===== 文本编辑 ========================================================
// Input / Textarea 共用一套编辑核心。下面这些入口供应用层做「状态栏行列」、
// 「查找后选中并定位」、「弹出菜单里的剪切 / 复制 / 粘贴 / 撤销」等操作。

// SetFocus 设置焦点控件 id（"" 表示取消）。编辑器用它把焦点交给编辑区。
func SetFocus(id string) { runtime.SetFocus(id) }

// FocusedID 返回当前焦点控件 id。
func FocusedID() string { return runtime.FocusedID() }

func CaretLineCol(n *VNode) (line, col int) { return components.CaretLineCol(n) }

func MoveCaret(n *VNode, line, col int) { components.MoveCaret(n, line, col) }

func SelectedRange(n *VNode) (from, to int, has bool) { return components.SelectedRange(n) }

func SelectRange(n *VNode, from, to int) { components.SelectRange(n, from, to) }

func SelectAll(n *VNode) { components.SelectAll(n) }

// EditOp 执行标准编辑命令：undo / redo / cut / copy / paste / delete / selectall。
func EditOp(n *VNode, op string) bool { return components.EditOp(n, op) }

// ResetInputState 丢弃某个编辑框的内部状态（切换文件、重建界面时可用）。
func ResetInputState(id string) { components.ResetInputState(id) }

func Popover(parts ...any) *VNode {
	return components.Popover(parts...)
}

func Dialog(parts ...any) *VNode {
	return components.Dialog(parts...)
}

func NewDispatcher(post func(func())) *Dispatcher {
	return runtime.NewDispatcher(post)
}

func CreateApp(root *ComponentDef) *App {
	return app.CreateApp(root)
}

// SetAfterPaint 注册每帧绘制完成后的回调（此时布局已生效）。
// 应用层用它做「打开文件后定位光标」这类依赖布局坐标的收尾动作。
func SetAfterPaint(fn func()) {
	if fn == nil {
		app.SetAfterPaint(nil)
		return
	}
	app.SetAfterPaint(func(renderer.Context) { fn() })
}

// SetWindowSize 在运行期把窗口客户区改成 w x h（逻辑 DIP）。
// 窗口就绪前调用返回 false。应用层用它做响应式布局。
func SetWindowSize(w, h int) bool {
	if platform.SetWindowSize == nil {
		return false
	}
	return platform.SetWindowSize(w, h)
}

// WindowSize 返回当前客户区逻辑尺寸（DIP）。窗口就绪前 ok 为 false。
func WindowSize() (w, h int, ok bool) {
	if platform.GetWindowSize == nil {
		return 0, 0, false
	}
	return platform.GetWindowSize()
}

// RegisterTheme 登记自定义主题，之后可用 CreateApp(...).Theme("name")。
func RegisterTheme(t Theme) { theme.Register(t) }

// LookupTheme 按名称取主题，未知名称回退到默认。
func LookupTheme(name string) Theme { return theme.Lookup(name) }
