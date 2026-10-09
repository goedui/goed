package app

import (
	"encoding/binary"
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
	// appTitleBar 为 true 时应用自己画窗口顶部那一条，框架不占那 32 DIP。
	appTitleBar bool
	titleStyle  runtime.Style
	icon        []byte
}

var (
	activeTheme      theme.Theme
	activeInvalidate func()
)

// SetTheme 在运行中的应用里切换主题，并请求下一帧重绘。
func SetTheme(t theme.Theme) {
	if t.FontSize <= 0 {
		t.FontSize = theme.Default.FontSize
	}
	activeTheme = t
	if activeInvalidate != nil {
		activeInvalidate()
	}
}

func CurrentTheme() theme.Theme {
	if activeTheme.FontSize > 0 {
		return activeTheme
	}
	return theme.Default
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

// AppTitleBar 让应用自己画窗口顶部那一条。浏览器把标签条放在最顶就是这么做的。
//
// 开启后框架不再为标题栏预留 32 DIP，也不画自己的标题栏（底色 / 图标 / 标题 /
// 系统按钮都不画），应用树从 y=0 起铺满整个客户区。代价是那三个系统按钮的图形
// 要应用自己画，位置必须对齐宿主保留的命中区：
//
//	x = 宽 - renderer.CaptionButtonWidth   → 关闭
//	x = 宽 - 2*CaptionButtonWidth          → 最大化
//	x = 宽 - 3*CaptionButtonWidth          → 最小化
//
// 高度是 renderer.TitleBarHeight。点击与拖动仍由宿主完成 —— 顶部条的空白处是
// HTCAPTION（拖动、双击最大化），那三块是 HTMINBUTTON / HTMAXBUTTON / HTCLOSE。
// 应用画在那里的**交互控件**要能被点到，靠的是 Run 里自动接上的 ClientHitTest：
// 布局树里有可点节点的地方交回客户区，没有的才留给系统当标题栏。
//
// 默认关闭 —— 绝大多数应用用框架标题栏更省事。
func (a *App) AppTitleBar(v bool) *App {
	if a != nil {
		a.appTitleBar = v
	}
	return a
}

// Icon 设置窗口图标。ico 是标准 ICO（含 16/32 等尺寸）；空则用默认应用图标。
func (a *App) Icon(ico []byte) *App {
	if a != nil {
		a.icon = append([]byte(nil), ico...)
	}
	return a
}

// TitleBarStyle 覆盖框架标题栏的底色、文字色和字体。零值字段仍走主题。
func (a *App) TitleBarStyle(s runtime.Style) *App {
	if a != nil {
		a.titleStyle = s
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

// closeGuard 在窗口关闭前询问应用。返回 false 表示先别关。
var closeGuard func() bool

// SetAfterPaint 注册每帧绘制后的回调。传 nil 表示取消。
// 回调在 UI 线程、布局完成之后执行。
func SetAfterPaint(fn func(renderer.Context)) {
	afterPaint = fn
}

// SetCloseGuard 注册关闭前的询问。返回 false 时窗口留着。传 nil 表示直接关。
func SetCloseGuard(fn func() bool) {
	closeGuard = fn
}

// Run 创建窗口、挂载根组件并进入平台消息循环。必须在主 goroutine 调用。
//
// 消息循环退出（窗口关闭）之后才卸载根实例：注册在 OnUnmounted 里的收尾——
// 保存会话、停掉子进程、取消后台任务——都会在这时跑一次。这一步不能省：
// 根实例平时不会卸载，缺了它那些回调永远不会被执行。
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
	titleBar := runtime.MountWith(components.TitleBar, invalidate, runtime.Props{Style: a.titleStyle})

	activeTheme = a.theme
	activeInvalidate = invalidate
	defer func() {
		activeInvalidate = nil
	}()
	title := a.title
	platform.SetFrame(platform.FrameState{Title: title, Active: true})

	var (
		px, py     float32
		down       bool
		editDrag   bool
		dragTarget *runtime.VNode
		hits       []runtime.Hit
		tree       []*runtime.VNode
		// hitW / hitH 是采集 hits 时的绘制表面尺寸，haveHits 表示已经采过。
		// 命中区必须「在布局之后」采集：节点的 X/Y/W/H 是 PaintTree 写进去的，
		// 早于它采集只会得到一堆 0 宽高的矩形，界面于是点不动。窗口尺寸变化时
		// 也要重采，否则缩放后点击位置会整体错位。
		hitW, hitH float32
		haveHits   bool
	)

	// 自绘标题栏的应用要把「顶部条里哪些点是我的控件」告诉宿主。不说的话宿主会把
	// 整条判成 HTCAPTION（窗口拖动），画在里面的标签条永远点不动 —— 而且症状很隐蔽：
	// 拖动正常、系统按钮正常，只有标签没反应。
	//
	// 判据直接用布局树的命中表：有 OnClick 的节点就是可点控件，其余（含根节点）不算。
	// 读写在同一个 UI 线程上（WM_NCHITTEST 与绘制都由消息循环派发），不必加锁。
	var clientHitTest func(x, y float32) bool
	if a.appTitleBar {
		clientHitTest = func(x, y float32) bool {
			return runtime.HitTest(hits, x, y) != nil
		}
	}
	platform.AppTitleBar = a.appTitleBar

	err := platform.Run(platform.Options{
		Title:  title,
		Width:  a.width,
		Height: a.height,
		Dark:   activeTheme.DarkMode(),
		Icon16: iconImage(a.icon, 16),
		Icon32: iconImage(a.icon, 32),
		// 自绘标题栏时，顶部条里属于应用的控件要交回客户区，见 hitTest。
		ClientHitTest: clientHitTest,
		Close: func() bool {
			if closeGuard == nil {
				return true
			}
			return closeGuard()
		},
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
			renderer.PaintTree(ctx, tree, activeTheme)
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
				dragTarget = nil
				components.EndScrollDrag()
				redraw()
				return
			}
			if !wasDown && isDown {
				dirty = true
				editDrag = false
				dragTarget = nil
				n := runtime.HitNode(tree, x, y)
				if n != nil && n.Attrs != nil {
					if fn, ok := n.Attrs["onPointerDown"].(func()); ok {
						fn()
						editDrag = true
						dragTarget = n
					}
				}
				if !editDrag && runtime.HitTest(hits, x, y) == nil {
					runtime.SetFocus("")
				}
				redraw()
				return
			}
			if wasDown && isDown && dragTarget != nil && dragTarget.Attrs != nil {
				if fn, ok := dragTarget.Attrs["onPointerDrag"].(func()); ok {
					fn()
					dirty = true
				}
			}
			if hoverChanged || downChanged || dirty {
				redraw()
			}
		},
		// 右键：只做一件事 —— 把「右键按下了、按在哪儿」告诉指针下那个节点。
		//
		// 不做冒泡、不自己记状态（菜单开在哪开什么全在节点自己的回调里）。找节点用
		// 与左键同一个 HitNode，所以浮层天然优先：右键面板开着时不会被底下的编辑区抢走。
		RightPointer: func(x, y float32, isDown bool) {
			if !isDown {
				return
			}
			n := runtime.HitNode(tree, x, y)
			if n == nil || n.Attrs == nil {
				return
			}
			fn, ok := n.Attrs["onContextMenu"].(func(float32, float32))
			if !ok || fn == nil {
				return
			}
			fn(x, y)
			dirty = true
			platform.RequestFrame()
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

	// 窗口已经销毁，这里仍在主 goroutine 上。卸载根与标题栏，让 OnUnmounted 跑完。
	inst.Unmount()
	titleBar.Unmount()
	return err
}

func iconImage(ico []byte, want int) []byte {
	if img := icoImage(ico, want); len(img) > 0 {
		return img
	}
	return components.App.WindowsIcon(want)
}

func icoImage(ico []byte, want int) []byte {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return nil
	}
	n := int(binary.LittleEndian.Uint16(ico[4:6]))
	var best []byte
	bestDiff := 1 << 30
	for i := 0; i < n; i++ {
		e := 6 + 16*i
		if e+16 > len(ico) {
			return nil
		}
		w := int(ico[e])
		if w == 0 {
			w = 256
		}
		size := int(binary.LittleEndian.Uint32(ico[e+8:]))
		off := int(binary.LittleEndian.Uint32(ico[e+12:]))
		if size <= 0 || off < 0 || off+size > len(ico) {
			continue
		}
		diff := w - want
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = ico[off : off+size]
		}
	}
	return best
}
