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
	// 视觉效果参数（挂在 Props 上，A==0 / nil 表示不启用）。
	GradientFill = runtime.GradientFill
	ShadowStyle  = runtime.ShadowStyle
)

const (
	Start   = runtime.Start
	Center  = runtime.Center
	End     = runtime.End
	Stretch = runtime.Stretch

	OverflowVisible = runtime.OverflowVisible
	OverflowHidden  = runtime.OverflowHidden
	OverflowAuto    = runtime.OverflowAuto
	OverflowScroll  = runtime.OverflowScroll

	PositionRelative = runtime.PositionRelative
	PositionAbsolute = runtime.PositionAbsolute

	InsetTop    = runtime.InsetTop
	InsetRight  = runtime.InsetRight
	InsetBottom = runtime.InsetBottom
	InsetLeft   = runtime.InsetLeft
	InsetAll    = runtime.InsetAll

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
	KeyP         = runtime.KeyP
	KeyS         = runtime.KeyS
	KeyT         = runtime.KeyT
	KeyV         = runtime.KeyV
	KeyW         = runtime.KeyW
	KeyX         = runtime.KeyX
	KeyY         = runtime.KeyY
	KeyZ         = runtime.KeyZ
	KeyF1        = runtime.KeyF1
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

func Table(parts ...any) *VNode {
	return components.Table(parts...)
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

func ScrollOffset(id string) float32          { return components.ScrollOffset(id) }
func SetScrollOffset(id string, v float32)    { components.SetScrollOffset(id, v) }
func ScrollMax(id string) float32             { return components.ScrollMax(id) }
func ScrollBy(id string, delta float32)       { components.ScrollBy(id, delta) }
func TextareaScrollY(id string) float32       { return components.TextareaScrollY(id) }
func TextareaScrollMax(id string) float32     { return components.TextareaScrollMax(id) }
func SetTextareaScrollY(id string, y float32) { components.SetTextareaScrollY(id, y) }
func PageTextarea(id string, dir float32)     { components.PageTextarea(id, dir) }
func TextareaOffsetY(id string, off int) float32 {
	return components.TextareaOffsetY(id, off)
}
func TextareaLineHeight(id string) float32 { return components.TextareaLineHeight(id) }
func FindByID(nodes []*VNode, id string) *VNode {
	return runtime.FindByID(nodes, id)
}
func LayoutRoot() []*VNode { return renderer.LayoutRoot() }

func Switch(parts ...any) *VNode {
	return components.Switch(parts...)
}

func Checkbox(parts ...any) *VNode {
	return components.Checkbox(parts...)
}

func Empty(parts ...any) *VNode {
	return components.Empty(parts...)
}

func Box(parts ...any) *VNode {
	return components.Box(parts...)
}

func Canvas(parts ...any) *VNode {
	return components.Canvas(parts...)
}

// Glyph 单个矢量图标节点（Icon 名字 + 可选 Size/Color），用于字段标签、行内装饰。
func Glyph(parts ...any) *VNode { return components.Glyph(parts...) }

// RegisterGlyph 登记应用自定义图标（同名覆盖引擎内置）。
func RegisterGlyph(name string, g components.GlyphPaint) { components.RegisterGlyph(name, g) }

// HasGlyph 报告图标名是否可用。
func HasGlyph(name string) bool { return components.HasGlyph(name) }

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

func FocusedCaretRect() (x, y, w, h float32, ok bool) { return components.FocusedCaretRect() }

func MoveCaret(n *VNode, line, col int) { components.MoveCaret(n, line, col) }

func SelectedRange(n *VNode) (from, to int, has bool) { return components.SelectedRange(n) }

func SelectRange(n *VNode, from, to int) { components.SelectRange(n, from, to) }

func SelectAll(n *VNode) { components.SelectAll(n) }

// SelectAllStay 全选，但不滚动视口。
func SelectAllStay(n *VNode) { components.SelectAllStay(n) }

// EditOp 执行标准编辑命令：undo / redo / cut / copy / paste / delete / selectall。
func EditOp(n *VNode, op string) bool { return components.EditOp(n, op) }

// EditOpID 按编辑框 id 执行同一套命令。
func EditOpID(id, op string) bool { return components.EditOpID(id, op) }

// ResetInputState 丢弃某个编辑框的内部状态（切换文件、重建界面时可用）。
func ResetInputState(id string) { components.ResetInputState(id) }

func Popover(parts ...any) *VNode {
	return components.Popover(parts...)
}

func Dialog(parts ...any) *VNode {
	return components.Dialog(parts...)
}

// ContextMenu 右键面板：铺满父级的无色遮罩 + 一张按 atX/atY 摆的菜单卡片。
// 挂在节点（或它的祖先）的 OnContextMenu 回调里，用指针坐标构造它即可。
func ContextMenu(parts ...any) *VNode {
	return components.ContextMenu(parts...)
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

// ToggleMaximize 切换窗口最大化 / 还原。窗口就绪前返回 false。
func ClipboardSet(text string) {
	if platform.ClipboardSet != nil {
		platform.ClipboardSet(text)
	}
}

func ToggleMaximize() bool {
	if platform.ToggleMaximize == nil {
		return false
	}
	return platform.ToggleMaximize()
}

// CloseWindow 关闭窗口，与标题栏关闭按钮同一条路。窗口就绪前无效果。
func CloseWindow() {
	if platform.CloseWindow != nil {
		platform.CloseWindow()
	}
}

// SetTitle 同时更新标题栏显示与窗口标题。编辑器打开 / 另存为后调用，
// 标题里的文件名才会跟着走。
func SetTitle(title string) { platform.SetTitle(title) }

// Frame 返回当前窗口装饰状态（标题、是否最大化、系统按钮悬停）。
// 自绘标题栏的应用每帧读它来画最小化 / 最大化 / 关闭。
func Frame() platform.FrameState { return platform.Frame() }

// CaptionButton 标识自定义标题栏右侧的系统按钮。
type CaptionButton = platform.CaptionButton

const (
	CaptionNone  = platform.CaptionNone
	CaptionMin   = platform.CaptionMin
	CaptionMax   = platform.CaptionMax
	CaptionClose = platform.CaptionClose
)

// CaptionButtons 报告宿主有没有系统标题栏按钮。浏览器里为 false，
// 自绘标题栏时不要再画那三个死按钮。
func CaptionButtons() bool { return platform.CaptionButtons }

// TitleBarHeight / CaptionButtonWidth 是自绘标题栏必须对齐的命中区（DIP）。
// 关闭在宽 - CaptionButtonWidth，最大化再往左一格，最小化再往左一格。
const (
	TitleBarHeight     = renderer.TitleBarHeight
	CaptionButtonWidth = renderer.CaptionButtonWidth
)

// SetCloseGuard 注册关闭前的询问。返回 false 时窗口留着（未保存的修改）。
// 传 nil 表示直接关。标题栏关闭、菜单退出、系统关闭都走这里。
func SetCloseGuard(fn func() bool) { app.SetCloseGuard(fn) }

// RegisterTheme 登记自定义主题，之后可用 CreateApp(...).Theme("name")。
func RegisterTheme(t Theme) { theme.Register(t) }

// LookupTheme 按名称取主题，未知名称回退到默认。
func LookupTheme(name string) Theme { return theme.Lookup(name) }

// SetTheme 运行中切换当前应用主题。
func SetTheme(t Theme) { app.SetTheme(t) }

// CurrentTheme 返回当前应用主题。
func CurrentTheme() Theme { return app.CurrentTheme() }
