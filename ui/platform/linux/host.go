//go:build linux

package linux

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

/*
#include <stdint.h>
#include <stdlib.h>
int x11_noop_error(void *dpy, void *ev);
int x11_io_error(void *dpy);
void x11_quiet(void *set_error, void *set_io);
void *x11_malloc(size_t n);
void x11_free(void *p);
void x11_image_clear_data(void *img);
*/
import "C"

// Options 描述 X11 宿主窗口，字段对齐 Windows 后端。
type Options struct {
	Title   string
	Width   int
	Height  int
	Dark    bool
	Icon16  []byte
	Icon32  []byte
	Paint   func(ctx renderer.Context)
	Pointer func(x, y float32, down, click bool)
	// RightPointer 报告右键（X11 button 3），与 platform.Options 同名同义。
	RightPointer     func(x, y float32, down bool)
	Key              func(key int, char rune, down, ctrl, shift, alt bool)
	Wheel            func(x, y, delta float32)
	Composition      func(text string, cursor int) bool
	Caret            func() (x, y, w, h float32, ok bool)
	BindRequestFrame func(fn func())
	BindPost         func(fn func(func()))
	BindHWND         func(hwnd uintptr)
	FrameUpdate      func(maximized, active bool, hover, pressed int)
	SetWindowTitle   func(title string)
	ToggleMaximize   func(fn func() bool)
	CloseWindow      func(fn func())
	// Close 返回 false 时不退出。nil 表示照常关。
	Close func() bool
}

const (
	xevSize = 192

	offType       = 0
	offWindow     = 32
	offKeyWin     = 32
	offBtnX       = 64
	offBtnY       = 68
	offBtnRootX   = 72
	offBtnRootY   = 76
	offBtnState   = 80
	offBtnBtn     = 84
	offKeyCode    = 84
	offCfgWin     = 40
	offCfgX       = 48
	offCfgY       = 52
	offCfgW       = 56
	offCfgH       = 60
	offClientType = 40
	offClientFmt  = 48
	offClientData = 56
)

const (
	netWMStateRemove = 0
	netWMStateAdd    = 1
	netWMStateToggle = 2
)

var (
	postedMu   sync.Mutex
	postedSeq  uintptr
	posted     = map[uintptr]func(){}
	activeHost *host
)

type host struct {
	opts         Options
	dpy          uintptr
	screen       int
	root         uintptr
	win          uintptr
	gc           uintptr
	visual       uintptr
	depth        int
	ximage       uintptr
	pixC         unsafe.Pointer
	ctx          *swContext
	pixelW       int
	pixelH       int
	dpi          float32
	maximized    bool
	active       bool
	hover        int
	pressed      int
	pointerDown  bool
	dirty        bool
	quit         bool
	lastClick    int64
	lastHit      int
	rootX, rootY int32

	atomProtocols  uintptr
	atomDelete     uintptr
	atomNetWMState uintptr
	atomMaxVert    uintptr
	atomMaxHorz    uintptr
	atomMoveResize uintptr
	atomMotifHints uintptr
	atomNetWMName  uintptr
	atomUTF8       uintptr
	atomPost       uintptr
	atomFrame      uintptr
}

// Run 创建 DPI 感知窗口，用软件光栅绘制，并阻塞直到关闭。
func Run(opts Options) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := loadX11(); err != nil {
		return err
	}
	if x.XInitThreads != nil {
		xc0(x.XInitThreads)
	}
	if x.XSetErrorHandler != nil || x.XSetIOErrorHandler != nil {
		C.x11_quiet(x.XSetErrorHandler, x.XSetIOErrorHandler)
	}

	if opts.Width <= 0 {
		opts.Width = 800
	}
	if opts.Height <= 0 {
		opts.Height = 600
	}
	if opts.Title == "" {
		opts.Title = "github.com/goedui/goed"
	}

	h := &host{opts: opts, active: true, dpi: 96}
	if err := h.create(); err != nil {
		h.dispose()
		return err
	}
	defer h.dispose()

	activeHost = h
	defer func() { activeHost = nil }()

	if opts.BindRequestFrame != nil {
		opts.BindRequestFrame(h.requestFrame)
	}
	if opts.BindPost != nil {
		opts.BindPost(h.post)
	}
	if opts.BindHWND != nil {
		opts.BindHWND(h.win)
	}
	if opts.ToggleMaximize != nil {
		opts.ToggleMaximize(h.toggleMaximize)
	}
	if opts.CloseWindow != nil {
		opts.CloseWindow(h.closeWindow)
	}
	h.syncFrame()
	h.dirty = true
	return h.loop()
}

// SetWindowTitle 更新 X11 窗口标题。HWND 是 Window id。
func SetWindowTitle(hwnd uintptr, title string) {
	if activeHost != nil && activeHost.win == hwnd {
		activeHost.setTitle(title)
	}
}

func (h *host) create() error {
	dpy := xc1(x.XOpenDisplay, 0)
	if dpy == 0 {
		return fmt.Errorf("github.com/goedui/goed/ui: 无法打开 X11 DISPLAY（需要 X11 或 XWayland）")
	}
	h.dpy = dpy
	h.screen = int(xc1(x.XDefaultScreen, dpy))
	h.root = xc2(x.XRootWindow, dpy, uintptr(h.screen))
	h.visual = xc2(x.XDefaultVisual, dpy, uintptr(h.screen))
	h.depth = int(xc2(x.XDefaultDepth, dpy, uintptr(h.screen)))
	if h.depth < 24 {
		return fmt.Errorf("github.com/goedui/goed/ui: 需要 24/32 位 TrueColor 显示，当前 depth=%d", h.depth)
	}

	h.dpi = h.readDPI()
	scale := h.dpi / 96
	winW := int32(float32(h.opts.Width) * scale)
	winH := int32(float32(h.opts.Height) * scale)
	if winW < 1 {
		winW = 1
	}
	if winH < 1 {
		winH = 1
	}

	screenW := int32(xc2(x.XDisplayWidth, dpy, uintptr(h.screen)))
	screenH := int32(xc2(x.XDisplayHeight, dpy, uintptr(h.screen)))
	x0 := (screenW - winW) / 2
	y0 := (screenH - winH) / 2
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	black := xc2(x.XBlackPixel, dpy, uintptr(h.screen))
	h.win = xc9(
		x.XCreateSimpleWindow,
		dpy, h.root,
		uintptr(uint32(x0)), uintptr(uint32(y0)),
		uintptr(uint32(winW)), uintptr(uint32(winH)),
		0, black, black,
	)
	if h.win == 0 {
		return fmt.Errorf("github.com/goedui/goed/ui: XCreateSimpleWindow 失败")
	}

	mask := uintptr(xKeyPressMask | xKeyReleaseMask | xButtonPressMask | xButtonReleaseMask |
		xPointerMotionMask | xExposureMask | xStructureNotifyMask | xFocusChangeMask)
	xc3(x.XSelectInput, dpy, h.win, mask)

	h.internAtoms()
	h.stripDecorations()
	h.setProtocols()
	h.setTitle(h.opts.Title)

	h.gc = xc4(x.XCreateGC, dpy, h.win, 0, 0)
	h.pixelW, h.pixelH = int(winW), int(winH)
	h.ctx = newContext(h.pixelW, h.pixelH, h.dpi)
	if err := h.ensureImage(); err != nil {
		return err
	}

	xc2(x.XMapWindow, dpy, h.win)
	xc1(x.XFlush, dpy)
	return nil
}

func (h *host) internAtoms() {
	h.atomProtocols = internAtom(h.dpy, "WM_PROTOCOLS", false)
	h.atomDelete = internAtom(h.dpy, "WM_DELETE_WINDOW", false)
	h.atomNetWMState = internAtom(h.dpy, "_NET_WM_STATE", false)
	h.atomMaxVert = internAtom(h.dpy, "_NET_WM_STATE_MAXIMIZED_VERT", false)
	h.atomMaxHorz = internAtom(h.dpy, "_NET_WM_STATE_MAXIMIZED_HORZ", false)
	h.atomMoveResize = internAtom(h.dpy, "_NET_WM_MOVERESIZE", false)
	h.atomMotifHints = internAtom(h.dpy, "_MOTIF_WM_HINTS", false)
	h.atomNetWMName = internAtom(h.dpy, "_NET_WM_NAME", false)
	h.atomUTF8 = internAtom(h.dpy, "UTF8_STRING", false)
	h.atomPost = internAtom(h.dpy, "_github.com/goedui/goed_POST", false)
	h.atomFrame = internAtom(h.dpy, "_github.com/goedui/goed_FRAME", false)
}

func (h *host) stripDecorations() {
	// flags = MWM_HINTS_DECORATIONS (1<<1)，decorations = 0；format=32 在
	// 64 位 Xlib 里按 long 数组写。
	hints := [5]uint64{2, 0, 0, 0, 0}
	xc8(
		x.XChangeProperty,
		h.dpy, h.win, h.atomMotifHints, h.atomMotifHints,
		32, xPropModeReplace,
		uintptr(unsafe.Pointer(&hints[0])), 5,
	)
	runtime.KeepAlive(hints)
}

func (h *host) setProtocols() {
	proto := h.atomDelete
	xc4(x.XSetWMProtocols, h.dpy, h.win, uintptr(unsafe.Pointer(&proto)), 1)
	runtime.KeepAlive(proto)
}

func (h *host) setTitle(title string) {
	if h.dpy == 0 || h.win == 0 {
		return
	}
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	xc3(x.XStoreName, h.dpy, h.win, uintptr(unsafe.Pointer(ctitle)))
	xc8(
		x.XChangeProperty,
		h.dpy, h.win, h.atomNetWMName, h.atomUTF8,
		8, xPropModeReplace,
		uintptr(unsafe.Pointer(ctitle)), uintptr(len(title)),
	)
}

func (h *host) readDPI() float32 {
	wpx := float32(int(xc2(x.XDisplayWidth, h.dpy, uintptr(h.screen))))
	wmm := float32(int(xc2(x.XDisplayWidthMM, h.dpy, uintptr(h.screen))))
	if wmm <= 0 || wpx <= 0 {
		return 96
	}
	dpi := wpx * 25.4 / wmm
	if dpi < 72 {
		return 96
	}
	if dpi > 384 {
		return 384
	}
	return dpi
}

func (h *host) loop() error {
	var ev [xevSize]byte
	for !h.quit {
		for xc1(x.XPending, h.dpy) != 0 {
			xc2(x.XNextEvent, h.dpy, uintptr(unsafe.Pointer(&ev[0])))
			runtime.KeepAlive(ev)
			h.handle(&ev)
			if h.quit {
				return nil
			}
		}
		if h.dirty {
			h.paint()
			h.dirty = false
		}
		xc2(x.XNextEvent, h.dpy, uintptr(unsafe.Pointer(&ev[0])))
		runtime.KeepAlive(ev)
		h.handle(&ev)
	}
	return nil
}

func (h *host) handle(ev *[xevSize]byte) {
	typ := int(*(*int32)(unsafe.Pointer(&ev[offType])))
	switch typ {
	case xExpose:
		h.dirty = true
	case xConfigureNotify:
		w := int(*(*int32)(unsafe.Pointer(&ev[offCfgW])))
		ht := int(*(*int32)(unsafe.Pointer(&ev[offCfgH])))
		if w > 0 && ht > 0 && (w != h.pixelW || ht != h.pixelH) {
			h.pixelW, h.pixelH = w, ht
			h.ctx.resize(w, ht, h.dpi)
			h.destroyImage()
			_ = h.ensureImage()
			h.dirty = true
		}
	case xClientMessage:
		msg := *(*uintptr)(unsafe.Pointer(&ev[offClientType]))
		data0 := *(*uintptr)(unsafe.Pointer(&ev[offClientData]))
		if msg == h.atomProtocols && data0 == h.atomDelete {
			if h.opts.Close != nil && !h.opts.Close() {
				return
			}
			h.quit = true
			return
		}
		if msg == h.atomPost {
			h.runPosted(data0)
			return
		}
		if msg == h.atomFrame {
			h.dirty = true
		}
	case xFocusIn:
		h.active = true
		h.syncFrame()
		h.dirty = true
	case xFocusOut:
		h.active = false
		h.syncFrame()
		h.dirty = true
	case xDestroyNotify:
		h.quit = true
	case xMotionNotify:
		h.onMotion(ev)
	case xButtonPress:
		h.onButton(ev, true)
	case xButtonRelease:
		h.onButton(ev, false)
	case xLeaveNotify:
		if h.hover != 0 || h.pressed != 0 {
			h.setHover(0)
			h.setPressed(0)
		}
		h.pointerDown = false
		h.emitPointer(-1, -1, false, false)
	case xKeyPress:
		h.onKey(ev, true)
	case xKeyRelease:
		h.onKey(ev, false)
	}
}

func (h *host) onMotion(ev *[xevSize]byte) {
	px := int(*(*int32)(unsafe.Pointer(&ev[offBtnX])))
	py := int(*(*int32)(unsafe.Pointer(&ev[offBtnY])))
	h.rootX = *(*int32)(unsafe.Pointer(&ev[offBtnRootX]))
	h.rootY = *(*int32)(unsafe.Pointer(&ev[offBtnRootY]))
	hit := h.hitAt(px, py)
	cap := hitToCaption(hit)
	if cap != 0 || hit == hitCaption || hitToMoveResize(hit) >= 0 && hit != hitClient {
		h.setHover(cap)
		return
	}
	if h.hover != 0 || h.pressed != 0 {
		h.setHover(0)
		h.setPressed(0)
	}
	x, y := h.toDIP(px, py)
	h.emitPointer(x, y, h.pointerDown, false)
}

func (h *host) onButton(ev *[xevSize]byte, down bool) {
	px := int(*(*int32)(unsafe.Pointer(&ev[offBtnX])))
	py := int(*(*int32)(unsafe.Pointer(&ev[offBtnY])))
	btn := int(*(*uint32)(unsafe.Pointer(&ev[offBtnBtn])))
	h.rootX = *(*int32)(unsafe.Pointer(&ev[offBtnRootX]))
	h.rootY = *(*int32)(unsafe.Pointer(&ev[offBtnRootY]))

	if btn == 4 || btn == 5 {
		if down && h.opts.Wheel != nil {
			delta := float32(1)
			if btn == 5 {
				delta = -1
			}
			x, y := h.toDIP(px, py)
			h.opts.Wheel(x, y, delta)
			h.dirty = true
		}
		return
	}
	if btn == 3 {
		// X11 的右键是 button 3。和 windows / web 一样只上报，不碰 pointerDown
		// （左键拖选是另一条通道）。
		x, y := h.toDIP(px, py)
		h.emitRightPointer(x, y, down)
		return
	}
	if btn != 1 {
		return
	}

	hit := h.hitAt(px, py)
	cap := hitToCaption(hit)
	if down {
		if cap != 0 {
			h.setPressed(cap)
			return
		}
		dir := hitToMoveResize(hit)
		if dir >= 0 {
			if hit == hitCaption {
				h.maybeDoubleClickMax()
			}
			h.beginMoveResize(dir)
			return
		}
		h.pointerDown = true
		x, y := h.toDIP(px, py)
		h.emitPointer(x, y, true, false)
		return
	}

	pressed := h.pressed
	h.setPressed(0)
	if cap != 0 && cap == pressed {
		h.runCaption(cap)
		return
	}
	wasDown := h.pointerDown
	h.pointerDown = false
	x, y := h.toDIP(px, py)
	h.emitPointer(x, y, false, wasDown)
}

func (h *host) maybeDoubleClickMax() {
	now := nsec()
	if now-h.lastClick < 400000000 && h.lastHit == hitCaption {
		h.runCaption(2)
		h.lastClick = 0
		return
	}
	h.lastClick = now
	h.lastHit = hitCaption
}

func nsec() int64 {
	return time.Now().UnixNano()
}

func (h *host) onKey(ev *[xevSize]byte, down bool) {
	if h.opts.Key == nil {
		return
	}
	state := *(*uint32)(unsafe.Pointer(&ev[offBtnState]))
	ctrl := state&controlMask != 0
	shift := state&shiftMask != 0
	alt := state&mod1Mask != 0

	var keysym uint64
	var buf [8]byte
	n := int(xc5(
		x.XLookupString,
		uintptr(unsafe.Pointer(&ev[0])),
		uintptr(unsafe.Pointer(&buf[0])),
		8,
		uintptr(unsafe.Pointer(&keysym)),
		0,
	))
	runtime.KeepAlive(ev)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(keysym)
	key, _ := mapKeySym(keysym)
	h.opts.Key(key, 0, down, ctrl, shift, alt)
	if down && n > 0 {
		s := string(buf[:n])
		for _, r := range s {
			if r >= 0x20 || r == '\t' || r == '\r' || r == '\n' {
				h.opts.Key(0, r, true, ctrl, shift, alt)
			}
		}
	}
	h.dirty = true
}

func (h *host) closeWindow() {
	if h.opts.Close != nil && !h.opts.Close() {
		return
	}
	h.quit = true
}

func (h *host) toggleMaximize() bool {
	h.runCaption(2)
	return true
}

func (h *host) runCaption(btn int) {
	switch btn {
	case 1:
		xc3(x.XIconifyWindow, h.dpy, h.win, uintptr(h.screen))
	case 2:
		action := uintptr(netWMStateAdd)
		if h.maximized {
			action = netWMStateRemove
		}
		h.sendNetWMState(action, h.atomMaxVert, h.atomMaxHorz)
		h.maximized = !h.maximized
		h.syncFrame()
		h.dirty = true
	case 3:
		h.closeWindow()
	}
}

func (h *host) beginMoveResize(dir int) {
	xc2(x.XUngrabPointer, h.dpy, 0)
	h.sendClient(h.root, h.atomMoveResize, true,
		uintptr(uint32(h.rootX)),
		uintptr(uint32(h.rootY)),
		uintptr(dir),
		1,
		1,
	)
}

func (h *host) sendNetWMState(action, a1, a2 uintptr) {
	h.sendClient(h.root, h.atomNetWMState, true, action, a1, a2, 1, 0)
}

func (h *host) sendClient(dest, msg uintptr, toRoot bool, d0, d1, d2, d3, d4 uintptr) {
	var ev [xevSize]byte
	*(*int32)(unsafe.Pointer(&ev[offType])) = xClientMessage
	*(*uintptr)(unsafe.Pointer(&ev[offWindow])) = h.win
	*(*uintptr)(unsafe.Pointer(&ev[offClientType])) = msg
	*(*int32)(unsafe.Pointer(&ev[offClientFmt])) = 32
	*(*uintptr)(unsafe.Pointer(&ev[offClientData])) = d0
	*(*uintptr)(unsafe.Pointer(&ev[offClientData+8])) = d1
	*(*uintptr)(unsafe.Pointer(&ev[offClientData+16])) = d2
	*(*uintptr)(unsafe.Pointer(&ev[offClientData+24])) = d3
	*(*uintptr)(unsafe.Pointer(&ev[offClientData+32])) = d4
	mask := uintptr(0)
	if toRoot {
		mask = xSubstructureNotify | xSubstructureRedirect
	}
	xc5(x.XSendEvent, h.dpy, dest, 0, mask, uintptr(unsafe.Pointer(&ev[0])))
	runtime.KeepAlive(ev)
	xc1(x.XFlush, h.dpy)
}

func (h *host) sendSelf(atom, data uintptr) {
	h.sendClient(h.win, atom, false, data, 0, 0, 0, 0)
}

func (h *host) requestFrame() {
	h.dirty = true
	if h.dpy != 0 && h.win != 0 {
		h.sendSelf(h.atomFrame, 0)
	}
}

func (h *host) post(fn func()) {
	if fn == nil || h.win == 0 {
		return
	}
	postedMu.Lock()
	id := postedSeq
	postedSeq++
	posted[id] = fn
	postedMu.Unlock()
	h.sendSelf(h.atomPost, id)
}

func (h *host) runPosted(id uintptr) {
	postedMu.Lock()
	fn := posted[id]
	delete(posted, id)
	postedMu.Unlock()
	if fn != nil {
		fn()
		h.dirty = true
	}
}

func (h *host) hitAt(px, py int) int {
	return hitTest(px, py, h.pixelW, h.pixelH, h.borderPx(), h.titleBarPx(), h.captionBtnPx(), h.maximized)
}

func (h *host) titleBarPx() int {
	return dipToPx(renderer.TitleBarHeight, h.dpi)
}

func (h *host) captionBtnPx() int {
	return dipToPx(renderer.CaptionButtonWidth, h.dpi)
}

func (h *host) borderPx() int {
	n := dipToPx(8, h.dpi)
	if n < 4 {
		return 4
	}
	return n
}

func (h *host) toDIP(px, py int) (float32, float32) {
	dpi := h.dpi
	if dpi <= 0 {
		dpi = 96
	}
	return float32(px) * 96 / dpi, float32(py) * 96 / dpi
}

func (h *host) emitPointer(x, y float32, down, click bool) {
	if h.opts.Pointer != nil {
		h.opts.Pointer(x, y, down, click)
	}
	if click || down {
		h.dirty = true
	}
}

// emitRightPointer 上报右键，与 windows / web 宿主对齐（见 platform.Options）。
func (h *host) emitRightPointer(x, y float32, down bool) {
	if h.opts.RightPointer != nil {
		h.opts.RightPointer(x, y, down)
	}
	if down {
		h.dirty = true
	}
}

func (h *host) setHover(v int) {
	if h.hover == v {
		return
	}
	h.hover = v
	h.syncFrame()
	h.dirty = true
}

func (h *host) setPressed(v int) {
	if h.pressed == v {
		return
	}
	h.pressed = v
	h.syncFrame()
	h.dirty = true
}

func (h *host) syncFrame() {
	if h.opts.FrameUpdate != nil {
		h.opts.FrameUpdate(h.maximized, h.active, h.hover, h.pressed)
	}
}

func (h *host) paint() {
	if h.ctx == nil {
		return
	}
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Fprintf(os.Stderr, "github.com/goedui/goed/ui: 绘制失败: %v\n", rec)
			}
		}()
		if h.opts.Paint != nil {
			h.opts.Paint(h.ctx)
		}
	}()
	h.present()
}

func (h *host) ensureImage() error {
	if h.ximage != 0 && h.pixC != nil {
		return nil
	}
	h.destroyImage()
	n := h.pixelW * h.pixelH * 4
	if n < 4 {
		n = 4
	}
	h.pixC = C.x11_malloc(C.size_t(n))
	if h.pixC == nil {
		return fmt.Errorf("github.com/goedui/goed/ui: 无法分配绘制缓冲")
	}
	img := xc10(
		x.XCreateImage,
		h.dpy, h.visual, uintptr(h.depth),
		zPixmap, 0,
		uintptr(h.pixC),
		uintptr(h.pixelW), uintptr(h.pixelH),
		32, uintptr(h.pixelW*4),
	)
	if img == 0 {
		C.x11_free(h.pixC)
		h.pixC = nil
		return fmt.Errorf("github.com/goedui/goed/ui: XCreateImage 失败")
	}
	h.ximage = img
	return nil
}

func (h *host) destroyImage() {
	if h.ximage != 0 {
		C.x11_image_clear_data(unsafe.Pointer(h.ximage))
		if x.XDestroyImage != nil {
			xc1(x.XDestroyImage, h.ximage)
		}
		h.ximage = 0
	}
	if h.pixC != nil {
		C.x11_free(h.pixC)
		h.pixC = nil
	}
}

func (h *host) present() {
	if h.ximage == 0 || h.gc == 0 || h.pixC == nil || h.ctx == nil {
		return
	}
	n := h.pixelW * h.pixelH * 4
	if n > len(h.ctx.pix) {
		n = len(h.ctx.pix)
	}
	if n > 0 {
		copy(unsafe.Slice((*byte)(h.pixC), n), h.ctx.pix[:n])
	}
	xc10(
		x.XPutImage,
		h.dpy, h.win, h.gc, h.ximage,
		0, 0, 0, 0,
		uintptr(h.pixelW), uintptr(h.pixelH),
	)
	xc1(x.XFlush, h.dpy)
}

func (h *host) dispose() {
	h.destroyImage()
	if h.gc != 0 && x.XFreeGC != nil {
		xc2(x.XFreeGC, h.dpy, h.gc)
		h.gc = 0
	}
	if h.win != 0 {
		xc2(x.XDestroyWindow, h.dpy, h.win)
		h.win = 0
	}
	if h.dpy != 0 {
		xc1(x.XCloseDisplay, h.dpy)
		h.dpy = 0
	}
}
