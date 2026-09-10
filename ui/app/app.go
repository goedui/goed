// Package app provides the smallest application shell for declarative Goed UI
// programs. Platform and rendering details stay here so examples can focus on
// components and state.
package app

import (
	"errors"
	stdruntime "runtime"
	"syscall"

	"github.com/goedui/goed/ui/components"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/platform/windows"
	"github.com/goedui/goed/ui/reactive"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// RunOptions controls the application shell around the root component.
type RunOptions struct {
	// HideTitleBar removes the default titlebar from the root layout. This is
	// useful when the application renders its own top bar.
	HideTitleBar bool
	// Theme selects the initial appearance; nil preserves the default dark theme.
	Theme *theme.Theme
	// ClearType sharpens text on opaque native UI surfaces.
	ClearType bool
}

// Application is a configured native Goed application.
type Application struct {
	title     string
	width     int32
	height    int32
	component *runtime.Def
	options   RunOptions
	props     runtime.Props
}

// CreateApp creates an application builder with sensible desktop defaults.
// The root component is supplied first so the common case remains concise.
func CreateApp(components ...*runtime.Def) *Application {
	a := &Application{title: "Goed", width: 800, height: 600}
	if len(components) > 0 {
		a.component = components[0]
	}
	return a
}

// Mount sets the root component. It enables the explicit builder form:
// CreateApp().Mount(App).Title("My App").Run().
func (a *Application) Mount(component *runtime.Def) *Application {
	if a != nil {
		a.component = component
	}
	return a
}

// Title sets the native window title.
func (a *Application) Title(title string) *Application {
	if a != nil && title != "" {
		a.title = title
	}
	return a
}

// Size sets the initial native window size.
func (a *Application) Size(width, height int32) *Application {
	if a != nil {
		if width > 0 {
			a.width = width
		}
		if height > 0 {
			a.height = height
		}
	}
	return a
}

// HideTitleBar controls whether the default titlebar is mounted.
func (a *Application) HideTitleBar(hide bool) *Application {
	if a != nil {
		a.options.HideTitleBar = hide
	}
	return a
}

// Prop forwards an extra prop to the mounted root component, letting callers
// parameterize the declarative root (e.g. an initial file path).
func (a *Application) Prop(key string, value any) *Application {
	if a != nil {
		if a.props == nil {
			a.props = runtime.Props{}
		}
		a.props[key] = value
	}
	return a
}

// Run starts the configured application and blocks until its window closes.
func (a *Application) Run() error {
	if a == nil || a.component == nil {
		return errors.New("app: a root component is required")
	}
	return run(a.title, a.width, a.height, a.component, a.options, a.props)
}

// Run creates a themed native window with the default titlebar, mounts the
// component, and enters its event loop.
func Run(title string, width, height int32, component *runtime.Def) error {
	return CreateApp(component).Title(title).Size(width, height).Run()
}

// RunWithOptions is Run with explicit shell settings.
func RunWithOptions(title string, width, height int32, component *runtime.Def, opts RunOptions) error {
	return CreateApp(component).Title(title).Size(width, height).WithOptions(opts).Run()
}

// WithOptions applies shell settings to the application builder.
func (a *Application) WithOptions(opts RunOptions) *Application {
	if a != nil {
		a.options = opts
	}
	return a
}

func run(title string, width, height int32, component *runtime.Def, opts RunOptions, props runtime.Props) error {
	// Win32 windows and common dialogs are thread-affine. Keep creation,
	// message dispatch, and callbacks on the same OS thread; in particular,
	// GetSaveFileNameW can fail or terminate the process when invoked after the
	// goroutine migrates to another thread.
	stdruntime.LockOSThread()
	defer stdruntime.UnlockOSThread()

	win, err := windows.NewFrameless(title, width, height)
	if err != nil {
		return err
	}
	if opts.Theme != nil {
		theme.SetTheme(opts.Theme)
	} else {
		theme.SetTheme(theme.Dark())
	}

	root := core.NewBaseComponent()
	dispatcher := runtime.NewDispatcher(win.Invalidate)
	defer dispatcher.Close()
	renderer := runtime.NewRenderer(root)
	var rootBox *components.BoxHost
	var paintCtx *windows.GDIContext
	clientW, clientH := width, height
	showTitleBar := !opts.HideTitleBar

	// Ref.Set schedules an effect; the native window performs the actual flush
	// on its UI thread before the next paint.
	reactive.SetFlushCallback(func() {
		reactive.Flush()
		win.Invalidate()
	})

	renderer.Mount(func() []*runtime.VNode {
		children := make([]*runtime.VNode, 0, 2)
		if showTitleBar {
			children = append(children, runtime.H("titlebar", runtime.Props{
				"title":      title,
				"onMinimize": func() { win.Minimize() },
				"onMaximize": func() { win.ToggleMaximize() },
				"onClose":    func() { win.Close() },
			}))
		}
		rootProps := runtime.Props{"flex": 1}
		rootProps["window"], rootProps["dispatcher"] = win, dispatcher
		for k, v := range props {
			if _, exists := rootProps[k]; !exists {
				rootProps[k] = v
			}
		}
		children = append(children, runtime.C(component, rootProps))
		return []*runtime.VNode{runtime.H("box", runtime.Props{
			"bg": theme.GetTheme().WindowBg, "direction": "column",
		}, children...)}
	})
	if len(root.Children) > 0 {
		rootBox, _ = root.Children[0].(*components.BoxHost)
	}

	if showTitleBar {
		win.SetTitleBarHeight(components.TitleBarHeight)
		win.SetDragRegion(func(x, y int32) bool {
			return !(y >= 0 && y < components.TitleBarHeight && x >= clientW-components.TitleBarButtonWidth*3)
		})
	} else {
		var drag func(int32, int32) bool
		runtime.WalkTree(root, func(c core.Component) bool {
			if provider, ok := c.(interface{ IsDragRegion(int32, int32) bool }); ok {
				drag = provider.IsDragRegion
				return true
			}
			return false
		})
		// HideTitleBar only hides the default titlebar VNode. A custom shell
		// still needs a native hit-test band so HTCAPTION can start dragging.
		if drag != nil {
			win.SetTitleBarHeight(components.TitleBarHeight)
		} else {
			win.SetTitleBarHeight(0)
		}
		win.SetDragRegion(drag)
	}
	// Configure mounted native surfaces after the window exists.
	configureHost := func(c core.Component) {
		if provider, ok := c.(interface {
			ConfigureWindow(*windows.FramelessWindow)
		}); ok {
			provider.ConfigureWindow(win)
		}
	}
	runtime.WalkTree(root, func(c core.Component) bool { configureHost(c); return false })
	renderer.OnHostMounted = configureHost
	win.SetOnMouseMove(func(x, y int32) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventMouseMove, X: x, Y: y})
		win.Invalidate()
	})
	win.SetOnMouseLeave(func() {
		runtime.DispatchEvent(root, core.Event{Type: core.EventMouseLeave})
		win.Invalidate()
	})
	win.SetOnMouseDown(func(x, y int32, button int) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventMouseDown, X: x, Y: y, Button: int32(button), Ctrl: windows.IsKeyDown(windows.VK_CONTROL), Shift: windows.IsKeyDown(windows.VK_SHIFT)})
		win.Invalidate()
	})
	win.SetOnMouseUp(func(x, y int32, button int) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventMouseUp, X: x, Y: y, Button: int32(button), Ctrl: windows.IsKeyDown(windows.VK_CONTROL), Shift: windows.IsKeyDown(windows.VK_SHIFT)})
		win.Invalidate()
	})
	win.SetOnMouseWheel(func(delta, x, y int32) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventMouseWheel, Delta: delta, X: x, Y: y})
		win.Invalidate()
	})
	win.SetOnKeyDown(func(key int32) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: key, Ctrl: windows.IsKeyDown(windows.VK_CONTROL), Shift: windows.IsKeyDown(windows.VK_SHIFT)})
		win.Invalidate()
	})
	win.SetOnKeyUp(func(key int32) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventKeyUp, Key: key, Ctrl: windows.IsKeyDown(windows.VK_CONTROL), Shift: windows.IsKeyDown(windows.VK_SHIFT)})
		win.Invalidate()
	})
	win.SetOnChar(func(char rune) {
		runtime.DispatchEvent(root, core.Event{Type: core.EventKeyChar, Char: char, Ctrl: windows.IsKeyDown(windows.VK_CONTROL), Shift: windows.IsKeyDown(windows.VK_SHIFT)})
		win.Invalidate()
	})
	win.SetOnResize(func(width, height int32) {
		clientW, clientH = width, height
		win.Invalidate()
	})
	win.SetOnDestroy(func() {
		renderer.Unmount()
		if paintCtx != nil {
			paintCtx.Cleanup()
			paintCtx = nil
		}
	})
	win.SetOnPaint(func(hdc syscall.Handle) {
		dispatcher.Drain()
		reactive.Flush()
		if width, height := win.ClientSize(); width > 0 && height > 0 {
			clientW, clientH = width, height
		}
		if paintCtx == nil {
			paintCtx = windows.NewGDIContext(hdc, clientW, clientH)
		} else {
			paintCtx.SetTarget(hdc)
			paintCtx.Resize(hdc, clientW, clientH)
		}
		paintCtx.SetBackgroundColor(theme.GetTheme().WindowBg)
		paintCtx.SetClearType(opts.ClearType)
		paintCtx.SetFont(theme.GetTheme().UIFontFamily, theme.GetTheme().UIFontSize)
		paintCtx.ResetClip()
		paintCtx.BeginDraw()
		root.SetBounds(core.Rect{W: clientW, H: clientH})
		if rootBox != nil {
			rootBox.SetBounds(core.Rect{W: clientW, H: clientH})
		}
		root.Render(paintCtx)
		paintCtx.EndDraw()
	})

	win.Show()
	return win.Run()
}
