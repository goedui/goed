package app

import (
	"fmt"

	"github.com/goedui/goed/ui/components"
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// App 是应用程序句柄。CreateApp 之后用链式方法配置，最后 Run 进入消息循环。
type App struct {
	root   *runtime.ComponentDef
	title  string
	width  int
	height int
	theme  theme.Theme
	// continuous 为 true 时根组件每帧都重建（标题栏本来如此），供游戏、
	// 动画这类「状态每帧都变」的场景使用。事件驱动的界面应保持 false。
	continuous bool
}

// CreateApp 以根组件创建应用。
func CreateApp(root *runtime.ComponentDef) *App {
	return &App{
		root:   root,
		title:  "github.com/goedui/goed",
		width:  800,
		height: 600,
		theme:  theme.Default,
	}
}

// Title 设置窗口标题。
func (a *App) Title(title string) *App {
	if a != nil {
		a.title = title
	}
	return a
}

// Size 设置客户区逻辑尺寸（DIP，96 DIP = 1 英寸）。高 DPI 下按监视器缩放。
func (a *App) Size(width, height int) *App {
	if a != nil {
		a.width = width
		a.height = height
	}
	return a
}

// Theme 按名称设置主题。内置 light / dark；自定义主题先 theme.Register。
func (a *App) Theme(name string) *App {
	if a != nil {
		a.theme = theme.Lookup(name)
	}
	return a
}

// UseTheme 直接使用一套主题，不必先登记名称。
func (a *App) UseTheme(t theme.Theme) *App {
	if a != nil {
		if t.FontSize <= 0 {
			t.FontSize = theme.Default.FontSize
		}
		a.theme = t
	}
	return a
}

// Continuous 开启连续帧模式：根组件每帧重建，配合 platform.RequestFrame 或
// 定时器实现游戏循环、实时动画。事件驱动的普通界面不要开。
func (a *App) Continuous(v bool) *App {
	if a != nil {
		a.continuous = v
	}
	return a
}

// testPaint 是给同包测试用的绘制钩子：每次绘制后调用，用来在真实窗口里
// 断言绘制 / 命中状态。生产代码不设置它。
var testPaint func(renderer.Context)

// afterPaint 在每帧布局 / 绘制完成后调用。应用层用它做需要布局结果才能
// 完成的收尾动作，例如「打开文件后把光标定位到文首」——光标的像素坐标
// 依赖本帧布局，早于绘制调用只会拿到上一帧的坐标。
var afterPaint func(renderer.Context)

// SetAfterPaint 注册每帧绘制后的回调。传 nil 表示取消。
// 回调在 UI 线程、布局完成之后执行。
func SetAfterPaint(fn func(renderer.Context)) {
	afterPaint = fn
}

// Run 创建窗口、挂载根组件并进入平台消息循环。必须在主 goroutine 调用。
func (a *App) Run() error {
	if a == nil {
		return fmt.Errorf("github.com/goedui/goed/ui: App 为 nil")
	}
	if a.root == nil {
		return fmt.Errorf("github.com/goedui/goed/ui: 根组件为 nil")
	}
	if a.width <= 0 {
		a.width = 800
	}
	if a.height <= 0 {
		a.height = 600
	}
	if a.title == "" {
		a.title = "github.com/goedui/goed"
	}

	var dirty = true
	invalidate := func() {
		dirty = true
		platform.RequestFrame()
	}
	inst := runtime.Mount(a.root, invalidate)
	titleBar := runtime.Mount(components.TitleBar, invalidate)

	theme := a.theme
	title := a.title
	platform.SetFrame(platform.FrameState{Title: title, Active: true})

	var (
		px, py float32
		down   bool
		editDrag bool
		hits   []runtime.Hit
		tree   []*runtime.VNode
		// hitW / hitH 是采集 hits 时的绘制表面尺寸，haveHits 表示已经采过。
		// 命中区必须「在布局之后」采集：节点的 X/Y/W/H 是 PaintTree 写进去的，
		// 早于它采集只会得到一堆 0 宽高的矩形，界面于是点不动。窗口尺寸变化时
		// 也要重采，否则缩放后点击位置会整体错位。
		hitW, hitH float32
		haveHits   bool
	)

	return platform.Run(platform.Options{
		Title:  title,
		Width:  a.width,
		Height: a.height,
		Dark:   theme.DarkMode(),
		Icon16: components.App.WindowsIcon(16),
		Icon32: components.App.WindowsIcon(32),
		Paint: func(ctx renderer.Context) {
			renderer.SetPointer(px, py, down)
			w, h := ctx.Width(), ctx.Height()
			rebuilt := a.continuous || dirty || tree == nil
			// 标题栏每帧都要重建：hover / pressed 由宿主在 WM_NCMOUSEMOVE 里
			// 直接 requestFrame，不经过 dirty 标记，若只在 dirty 时重渲染，
			// 画的还是上一帧的 VNode，关闭按钮的红色 hover 永远出不来。
			tb := titleBar.Render()
			var nodes []*runtime.VNode
			if rebuilt {
				nodes = inst.Render()
				dirty = false
			} else {
				// 保留上次的应用树；从 tree 里摘出标题栏之后的部分。
				// 标题栏节点固定带 Attrs["hover"]（数字），用它识别而不是
				// 引入 renderer 的内部判断。
				nodes = tree
				if len(nodes) > 0 && nodes[0].Tag == "titlebar" {
					nodes = nodes[1:]
				}
			}
			if cap(tree) < len(tb)+len(nodes) {
				tree = make([]*runtime.VNode, 0, len(tb)+len(nodes))
			} else {
				tree = tree[:0]
			}
			tree = append(tree, tb...)
			tree = append(tree, nodes...)
			renderer.PaintTree(ctx, tree, theme)
			if rebuilt || !haveHits || w != hitW || h != hitH {
				hits = runtime.CollectHits(tree)
				hitW, hitH = w, h
				haveHits = true
			}
			if afterPaint != nil {
				afterPaint(ctx)
			}
			if testPaint != nil {
				testPaint(ctx)
			}
		},
		Pointer: func(x, y float32, isDown, click bool) {
			hoverChanged := runtime.HoverChanged(hits, px, py, x, y)
			downChanged := down != isDown
			wasDown := down
			px, py, down = x, y, isDown
			renderer.SetPointer(px, py, down)
			redraw := func() {
				platform.RequestFrame()
			}
			if click {
				dirty = true
				if fn := runtime.HitTest(hits, x, y); fn != nil {
					fn()
				} else if !editDrag {
					runtime.SetFocus("")
				}
				editDrag = false
				redraw()
				return
			}
			if !wasDown && isDown {
				dirty = true
				editDrag = false
				n := runtime.HitNode(tree, x, y)
				if n != nil && n.Attrs != nil {
					if fn, ok := n.Attrs["onPointerDown"].(func()); ok {
						fn()
						editDrag = true
					}
				}
				if !editDrag && runtime.HitTest(hits, x, y) == nil {
					runtime.SetFocus("")
				}
				redraw()
				return
			}
			if wasDown && isDown {
				if n := runtime.FindByID(tree, runtime.FocusedID()); n != nil {
					if fn, ok := n.Attrs["onPointerDrag"].(func()); ok {
						fn()
						dirty = true
					}
				}
			}
			if hoverChanged || downChanged || dirty {
				redraw()
			}
		},
		Key: func(key int, char rune, isDown, ctrl, shift, alt bool) {
			if !isDown && char == 0 {
				// 纯 keyup 原本直接丢弃；游戏「按住移动」需要它维护按键状态，
				// 分发给 onKeyUp，不影响只认 keydown 的输入框 / 文本编辑。
				runtime.DispatchKeyUp(tree, runtime.KeyEvent{
					Key:   key,
					Ctrl:  ctrl,
					Shift: shift,
					Alt:   alt,
				})
				return
			}
			dirty = true
			runtime.DispatchKey(tree, runtime.KeyEvent{
				Key:   key,
				Char:  char,
				Ctrl:  ctrl,
				Shift: shift,
				Alt:   alt,
				Down:  isDown,
			})
		},
		Wheel: func(x, y, delta float32) {
			dirty = true
			components.HandleWheel(tree, x, y, delta)
		},
		// 输入法组合串交给焦点控件自己绘制（返回 true 时系统不再画内联组合窗）。
		Composition: func(text string, cursor int) bool {
			return runtime.DispatchComposition(tree, text, cursor)
		},
		// 光标矩形由输入框在绘制时记下来，宿主用它贴住输入法候选窗。
		Caret: func() (x, y, w, h float32, ok bool) {
			return components.FocusedCaretRect()
		},
	})
}
