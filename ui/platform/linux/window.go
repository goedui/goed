package linux

import (
	"sync"
)

// Window is a small event-loop compatible window abstraction. It intentionally
// has no X11 dependency, making the package safe to import from cross-platform
// code and usable in headless Linux environments.
type Window struct {
	mu              sync.RWMutex
	title           string
	width, height   int32
	visible, closed bool
	done            chan struct{}
	onPaint         func(uintptr)
	onMouseMove     func(int32, int32)
	onMouseDown     func(int32, int32, int)
	onMouseUp       func(int32, int32, int)
	onMouseWheel    func(int32)
	onKeyDown       func(int32)
	onKeyUp         func(int32)
	onChar          func(rune)
	onResize        func(int32, int32)
	onDestroy       func()
	isDragRegion    func(int32, int32) bool
	titleBarHeight  int32
	fullscreen      bool
}

func NewWindow(title string, width, height int32) (*Window, error) {
	return newWindow(title, width, height), nil
}

func NewFrameless(title string, width, height int32) (*Window, error) {
	return newWindow(title, width, height), nil
}

func newWindow(title string, width, height int32) *Window {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return &Window{title: title, width: width, height: height, done: make(chan struct{}), titleBarHeight: 40}
}

func (w *Window) SetOnPaint(fn func(uintptr)) { w.mu.Lock(); w.onPaint = fn; w.mu.Unlock() }
func (w *Window) SetOnMouseMove(fn func(int32, int32)) {
	w.mu.Lock()
	w.onMouseMove = fn
	w.mu.Unlock()
}
func (w *Window) SetOnMouseDown(fn func(int32, int32, int)) {
	w.mu.Lock()
	w.onMouseDown = fn
	w.mu.Unlock()
}
func (w *Window) SetOnMouseUp(fn func(int32, int32, int)) {
	w.mu.Lock()
	w.onMouseUp = fn
	w.mu.Unlock()
}
func (w *Window) SetOnMouseWheel(fn func(int32))    { w.mu.Lock(); w.onMouseWheel = fn; w.mu.Unlock() }
func (w *Window) SetOnKeyDown(fn func(int32))       { w.mu.Lock(); w.onKeyDown = fn; w.mu.Unlock() }
func (w *Window) SetOnKeyUp(fn func(int32))         { w.mu.Lock(); w.onKeyUp = fn; w.mu.Unlock() }
func (w *Window) SetOnChar(fn func(rune))           { w.mu.Lock(); w.onChar = fn; w.mu.Unlock() }
func (w *Window) SetOnResize(fn func(int32, int32)) { w.mu.Lock(); w.onResize = fn; w.mu.Unlock() }
func (w *Window) SetOnDestroy(fn func())            { w.mu.Lock(); w.onDestroy = fn; w.mu.Unlock() }
func (w *Window) SetDragRegion(fn func(int32, int32) bool) {
	w.mu.Lock()
	w.isDragRegion = fn
	w.mu.Unlock()
}
func (w *Window) SetTitleBarHeight(height int32) {
	w.mu.Lock()
	if height >= 0 {
		w.titleBarHeight = height
	}
	w.mu.Unlock()
}

func (w *Window) Show() { w.mu.Lock(); w.visible = true; w.mu.Unlock(); w.Invalidate() }

func (w *Window) Run() error {
	if w == nil {
		return nil
	}
	<-w.done
	return nil
}

func (w *Window) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed, w.visible = true, false
	done, destroy := w.done, w.onDestroy
	w.mu.Unlock()
	close(done)
	if destroy != nil {
		destroy()
	}
}

func (w *Window) Invalidate() {
	if w == nil {
		return
	}
	w.mu.RLock()
	paint, closed := w.onPaint, w.closed
	w.mu.RUnlock()
	if paint != nil && !closed {
		go paint(0)
	}
}

func (w *Window) Minimize()       { w.mu.Lock(); w.visible = false; w.mu.Unlock() }
func (w *Window) ToggleMaximize() {}

func (w *Window) ToggleFullscreen() {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.fullscreen = !w.fullscreen
	w.mu.Unlock()
}

func (w *Window) IsFullscreen() bool {
	if w == nil {
		return false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.fullscreen
}

func (w *Window) ClientSize() (int32, int32) {
	if w == nil {
		return 0, 0
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.width, w.height
}

// Resize lets headless integrations drive the same callback used by native
// window backends.
func (w *Window) Resize(width, height int32) {
	if w == nil {
		return
	}
	w.mu.Lock()
	if width > 0 {
		w.width = width
	}
	if height > 0 {
		w.height = height
	}
	width, height = w.width, w.height
	resize := w.onResize
	w.mu.Unlock()
	if resize != nil {
		resize(width, height)
	}
}
