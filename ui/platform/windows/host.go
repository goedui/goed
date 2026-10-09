//go:build windows

package windows

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

// Options 描述 Direct2D 宿主窗口。
type Options struct {
	Title   string
	Width   int
	Height  int
	Dark    bool
	Icon16  []byte
	Icon32  []byte
	Paint   func(ctx renderer.Context)
	Pointer func(x, y float32, down, click bool)
	// RightPointer 报告右键（WM_RBUTTONDOWN / UP），与 platform.Options 同名同义。
	RightPointer func(x, y float32, down bool)
	Key          func(key int, char rune, down, ctrl, shift, alt bool)
	Wheel        func(x, y, delta float32)
	// Composition 报告输入法组合串（预编辑）：text 为组合中的文本，
	// cursor 为组合串内的字符偏移。返回 true 表示应用自己绘制组合串。
	Composition func(text string, cursor int) bool
	// Caret 报告当前焦点的光标矩形（客户区 DIP），用来定位输入法候选窗。
	Caret func() (x, y, w, h float32, ok bool)
	// ClientHitTest 报告标题栏区里的某个点是不是应用自己的可点击控件（客户区 DIP）。
	// 非 nil 时标题栏区不再整条判成 HTCAPTION，见 hitTest。nil = 不占用。
	ClientHitTest    func(x, y float32) bool
	// Close 返回 false 时吞掉这次关闭，窗口留着。nil 表示照常关。
	Close            func() bool
	BindRequestFrame func(fn func())
	BindPost         func(fn func(func()))
	BindHWND         func(hwnd uintptr)
	FrameUpdate      func(maximized, active bool, hover, pressed int)
	// 以下几个回调都由平台层注入（宿主不认识平台层）：SetWindowTitle 运行期改标题；
	// SetWindowSize / GetWindowSize / ToggleMaximize 把对应能力交出去，平台层负责
	// 把 fn 投递回 UI 线程、宿主只执行；UIThreadID 建窗后回报宿主线程 id。
	SetWindowTitle func(title string)
	SetWindowSize  func(fn func(w, h int) bool)
	GetWindowSize  func(fn func() (w, h int, ok bool))
	ToggleMaximize func(fn func() bool)
	CloseWindow    func(fn func())
	UIThreadID     func(id uint32)
}

var (
	wndProcCallback = syscall.NewCallback(wndProc)
	hosts           = map[uintptr]*host{}
	creating        *host
	postedMu        sync.Mutex
	postedSeq       uintptr
	posted          = map[uintptr]func(){}
)

type host struct {
	opts Options
	// hwndMu 保护 hwnd：UI 线程在创建 / 销毁时写，后台 goroutine 经
	// requestFrame / post 读（光标闪烁、动画、业务侧跨线程投递）。不加锁就是
	// 「后台读、UI 线程写」，-race 判数据竞争；只有这两条跨 goroutine 路径要锁。
	hwndMu sync.RWMutex
	hwnd   uintptr
	class  *uint16
	title  *uint16
	d2d    com
	dwrite com
	rt     com
	brush  com
	ctx    d2dContext
	pixelW uint32
	pixelH uint32
	// threadID 是建窗的 OS 线程 id（runtime.LockOSThread 钉住的那条）：
	// 消息循环、SetWindowPos、Direct2D 资源都只在这条线程上。
	threadID       uint32
	dpi            float32
	instance       uintptr
	maximized      bool
	active         bool
	hover          int
	pressed        int
	tracking       bool
	clientTracking bool
	pointerDown    bool
	iconSm         uintptr
	iconBg         uintptr
}

// Run 创建 DPI 感知窗口，用 Direct2D 绘制，并阻塞在消息循环直到窗口关闭。
func Run(opts Options) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	enableDPI()

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		return fmt.Errorf("github.com/goedui/goed/ui: CoInitializeEx 失败: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()

	if opts.Width <= 0 {
		opts.Width = 800
	}
	if opts.Height <= 0 {
		opts.Height = 600
	}
	if opts.Title == "" {
		opts.Title = "github.com/goedui/goed"
	}

	h := &host{opts: opts, active: true}
	if err := h.create(); err != nil {
		h.dispose()
		return err
	}
	defer h.dispose()

	if opts.BindRequestFrame != nil {
		opts.BindRequestFrame(h.requestFrame)
	}
	if opts.BindPost != nil {
		opts.BindPost(h.post)
	}
	if opts.BindHWND != nil {
		opts.BindHWND(h.hwnd)
	}
	if opts.SetWindowSize != nil {
		opts.SetWindowSize(h.setClientSize)
	}
	if opts.GetWindowSize != nil {
		opts.GetWindowSize(h.clientSize)
	}
	if opts.ToggleMaximize != nil {
		opts.ToggleMaximize(h.toggleMaximize)
	}
	if opts.CloseWindow != nil {
		opts.CloseWindow(h.closeWindow)
	}
	// 报上 UI 线程 id，平台层才知道哪些调用已在 UI 线程、哪些需要投递回来。
	if opts.UIThreadID != nil {
		opts.UIThreadID(h.threadID)
	}
	h.syncFrame()

	procShowWindow.Call(h.hwnd, swShow)
	procUpdateWindow.Call(h.hwnd)
	return h.loop()
}

func (h *host) create() error {
	var err error
	h.threadID = currentOSThreadID()
	h.d2d, err = d2d1CreateFactory()
	if err != nil {
		return err
	}
	h.dwrite, err = dwriteCreateFactory()
	if err != nil {
		return err
	}
	h.ctx.dwrite = h.dwrite
	// 工厂也交给上下文：圆角图片的遮罩几何体得用工厂建（见 DrawImageRounded）。
	h.ctx.d2d = h.d2d
	h.ctx.locale = userLocale()
	h.ctx.formats = make(map[formatKey]com)

	h.instance = moduleHandle()
	h.class = utf16Ptr("github.com/goedui/goed.ui.d2d.hwnd")
	h.title = utf16Ptr(h.opts.Title)

	wc := wndClassExW{
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   wndProcCallback,
		hInstance:     h.instance,
		hIcon:         loadIcon(idiApplication),
		hCursor:       loadCursor(idcArrow),
		lpszClassName: h.class,
		hIconSm:       loadIcon(idiApplication),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(wc)
	if atom == 0 {
		if err != errorClassAlreadyExists {
			return fmt.Errorf("github.com/goedui/goed/ui: RegisterClassExW 失败: %v", err)
		}
	}

	dpi := systemDPI()
	h.dpi = float32(dpi)
	scale := h.dpi / 96
	clientW := int32(float32(h.opts.Width) * scale)
	clientH := int32(float32(h.opts.Height) * scale)
	if clientW < 1 {
		clientW = 1
	}
	if clientH < 1 {
		clientH = 1
	}

	// 保留 WS_OVERLAPPEDWINDOW（系统仍处理最小化 / 最大化 / 关闭）；视觉非客户区
	// 由 WM_NCCALCSIZE 去掉，改由自定义 TitleBar 绘制。
	style := uint32(wsOverlappedWindow | wsClipChildren | wsClipSiblings)
	winW, winH := clientW, clientH

	screenW, _, _ := procGetSystemMetrics.Call(smCxScreen)
	screenH, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x := (int32(screenW) - winW) / 2
	y := (int32(screenH) - winH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	creating = h
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(h.class)),
		uintptr(unsafe.Pointer(h.title)),
		uintptr(style),
		uintptr(int32(x)),
		uintptr(int32(y)),
		uintptr(winW),
		uintptr(winH),
		0, 0, h.instance, 0,
	)
	creating = nil
	if hwnd == 0 {
		return fmt.Errorf("github.com/goedui/goed/ui: CreateWindowExW 失败: %v", err)
	}
	h.setHandle(hwnd)
	hosts[hwnd] = h
	h.applyDWM()
	h.applyIcons()
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpFrameChanged)
	h.syncClientSize()
	return h.ensureTarget()
}

func (h *host) applyDWM() {
	if h.hwnd == 0 {
		return
	}
	// 1px 顶边扩展，保留 DWM 阴影，同时让客户区盖住系统标题栏。
	m := margins{cyTopHeight: 1}
	procDwmExtendFrameIntoClientArea.Call(h.hwnd, uintptr(unsafe.Pointer(&m)))
	runtime.KeepAlive(m)

	dark := int32(0)
	if h.opts.Dark {
		dark = 1
	}
	procDwmSetWindowAttribute.Call(h.hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&dark)), 4)
	procDwmSetWindowAttribute.Call(h.hwnd, dwmwaUseImmersiveDarkMode19, uintptr(unsafe.Pointer(&dark)), 4)
	pref := int32(dwmwcpRound)
	procDwmSetWindowAttribute.Call(h.hwnd, dwmwaWindowCornerPreference, uintptr(unsafe.Pointer(&pref)), 4)
	runtime.KeepAlive(dark)
	runtime.KeepAlive(pref)
}

func (h *host) loop() error {
	var m winMsg
	for {
		ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("github.com/goedui/goed/ui: GetMessageW 失败: %v", err)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
}

// handleID 读窗口句柄，**可跨 goroutine 调用**（见 host.hwndMu）：requestFrame /
// post 会被后台循环和别的线程调到，那时 UI 线程可能正在销毁窗口。
func (h *host) handleID() uintptr {
	h.hwndMu.RLock()
	v := h.hwnd
	h.hwndMu.RUnlock()
	return v
}

// setHandle 写窗口句柄：只在创建与销毁时调用（都在 UI 线程上）。
func (h *host) setHandle(v uintptr) {
	h.hwndMu.Lock()
	h.hwnd = v
	h.hwndMu.Unlock()
}

func (h *host) requestFrame() {
	if id := h.handleID(); id != 0 {
		procInvalidateRect.Call(id, 0, 0)
	}
}

func (h *host) post(fn func()) {
	id := h.handleID()
	if fn == nil || id == 0 {
		return
	}
	postedMu.Lock()
	seq := postedSeq
	postedSeq++
	posted[seq] = fn
	postedMu.Unlock()
	procPostMessageW.Call(id, wmAppPost, uintptr(seq), 0)
}

func (h *host) syncFrame() {
	if h.opts.FrameUpdate != nil {
		h.opts.FrameUpdate(h.maximized, h.active, h.hover, h.pressed)
	}
}

func (h *host) setHover(v int) {
	if h.hover == v {
		return
	}
	h.hover = v
	h.syncFrame()
	h.requestFrame()
}

func (h *host) setPressed(v int) {
	if h.pressed == v {
		return
	}
	h.pressed = v
	h.syncFrame()
	h.requestFrame()
}

func (h *host) titleBarPx() int32 {
	return dipToPx(renderer.TitleBarHeight, h.dpi)
}

func (h *host) captionBtnPx() int32 {
	return dipToPx(renderer.CaptionButtonWidth, h.dpi)
}

func (h *host) borderPx() int32 {
	return resizeBorderPx(uint32(h.dpi + 0.5))
}

func (h *host) hitFromLParam(lParam uintptr) uintptr {
	pt := point{x: signedLOWORD(lParam), y: signedHIWORD(lParam)}
	procScreenToClient.Call(h.hwnd, uintptr(unsafe.Pointer(&pt)))
	dpi := h.dpi
	if dpi <= 0 {
		dpi = 96
	}
	return hitTest(pt.x, pt.y, int32(h.pixelW), int32(h.pixelH), h.borderPx(), h.titleBarPx(), h.captionBtnPx(), h.maximized, h.opts.ClientHitTest, dpi)
}

func (h *host) clientDIP(lParam uintptr) (float32, float32) {
	dpi := h.dpi
	if dpi <= 0 {
		dpi = 96
	}
	return float32(signedLOWORD(lParam)) * 96 / dpi, float32(signedHIWORD(lParam)) * 96 / dpi
}

// clientDIPFromScreen 把「屏幕坐标」的 lParam 换成客户区 DIP。
// WM_MOUSEWHEEL 的 lParam 是屏幕坐标，与鼠标移动 / 按键消息不同；直接当客户区用
// 会让命中落到错位 —— 症状是「滚十几下才响应一次」（窗口贴着屏幕原点时才碰巧对上）。
func (h *host) clientDIPFromScreen(lParam uintptr) (float32, float32) {
	origin := point{} // 客户区 (0,0) 对应的屏幕坐标
	procClientToScreen.Call(h.hwnd, uintptr(unsafe.Pointer(&origin)))
	runtime.KeepAlive(origin)
	return wheelClientDIP(signedLOWORD(lParam), signedHIWORD(lParam), origin.x, origin.y, h.dpi)
}

// wheelClientDIP 由滚轮消息的屏幕坐标 + 客户区原点 + DPI 算出客户区 DIP；
// 单独抽出来是为了能测（把屏幕坐标当客户区用正是「滚轮时灵时不灵」的根因）。
func wheelClientDIP(screenX, screenY, originX, originY int32, dpi float32) (float32, float32) {
	if dpi <= 0 {
		dpi = 96
	}
	return float32(screenX-originX) * 96 / dpi, float32(screenY-originY) * 96 / dpi
}

func (h *host) emitPointer(x, y float32, down, click bool) {
	if h.opts.Pointer != nil {
		h.opts.Pointer(x, y, down, click)
	}
	if click || down {
		h.requestFrame()
	}
}

// emitRightPointer 上报右键。
//
// 刻意**不动 h.pointerDown**：左键按住拖动（编辑器拖选）与右键是两条互不相干的通道，
// 右键按下不能把左键的按下状态搅了，否则拖选中途点右键会当场断掉；也不 SetCapture
// —— 菜单开在按下那一刻，不需要后续移动事件。
func (h *host) emitRightPointer(x, y float32, down bool) {
	if h.opts.RightPointer != nil {
		h.opts.RightPointer(x, y, down)
	}
	if down {
		h.requestFrame()
	}
}

func (h *host) emitKey(key int, char rune, down bool) {
	if h.opts.Key == nil {
		return
	}
	ctrl := keyDown(vkControl)
	shift := keyDown(vkShift)
	alt := keyDown(vkMenu)
	h.opts.Key(key, char, down, ctrl, shift, alt)
	h.requestFrame()
}

func (h *host) trackClientLeave() {
	if h.clientTracking {
		return
	}
	tme := trackMouseEvent{
		dwFlags:   tmeLeave,
		hwndTrack: h.hwnd,
	}
	tme.cbSize = uint32(unsafe.Sizeof(tme))
	procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
	runtime.KeepAlive(tme)
	h.clientTracking = true
}

func (h *host) trackNCLeave() {
	if h.tracking {
		return
	}
	tme := trackMouseEvent{
		dwFlags:   tmeLeave | tmeNonClient,
		hwndTrack: h.hwnd,
	}
	tme.cbSize = uint32(unsafe.Sizeof(tme))
	procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
	runtime.KeepAlive(tme)
	h.tracking = true
}

// clientSize 返回当前客户区逻辑尺寸（DIP）。
func (h *host) clientSize() (int, int, bool) {
	if h.hwnd == 0 {
		return 0, 0, false
	}
	dpi := h.dpi
	if dpi <= 0 {
		dpi = 96
	}
	w := int(float32(h.pixelW)*96/dpi + 0.5)
	ht := int(float32(h.pixelH)*96/dpi + 0.5)
	if w < 1 || ht < 1 {
		return 0, 0, false
	}
	return w, ht, true
}

// dipToPixels 把逻辑尺寸（DIP）换算成物理像素。dpi <= 0 时按 96 处理；
// 单独抽出来是为了能测（高 DPI 下少乘一次缩放，窗口就小一圈）。
func dipToPixels(dip float32, dpi float32) int32 {
	if dpi <= 0 {
		dpi = 96
	}
	v := dip * dpi / 96
	if v < 1 {
		return 1
	}
	return int32(v + 0.5)
}

// pixelsToDIP 把物理像素换算回逻辑尺寸（DIP）。
func pixelsToDIP(px uint32, dpi float32) int32 {
	if dpi <= 0 {
		dpi = 96
	}
	return int32(float32(px)*96/dpi + 0.5)
}

// fitToScreen 把窗口左上角夹到屏幕内，保证 w x h 的窗口至少有一部分可见。
// 缩小窗口时若左上角在右下边，不夹的话窗口会直接跑到屏幕外看不见。
func fitToScreen(x, y, w, h, screenW, screenH int32) (int32, int32) {
	if maxX := screenW - w; x > maxX {
		x = maxX
	}
	if maxY := screenH - h; y > maxY {
		y = maxY
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// setClientSize 把客户区改成逻辑尺寸 w x h（DIP），窗口左上角保持不动。
//
// 用「客户区矩形」反推窗口矩形，而不是直接 SetWindowPos 到某个猜测尺寸：非客户区
// 大小由 WM_NCCALCSIZE 自己决定（自定义标题栏把它去掉了），按客户区算才不差标题栏高度。
func (h *host) setClientSize(w, ht int) bool {
	if h.hwnd == 0 || w <= 0 || ht <= 0 {
		return false
	}
	dpi := windowDPI(h.hwnd)
	pxW := dipToPixels(float32(w), float32(dpi))
	pxH := dipToPixels(float32(ht), float32(dpi))
	var win, cli rect
	if r, _, _ := procGetWindowRect.Call(h.hwnd, uintptr(unsafe.Pointer(&win))); r == 0 {
		return false
	}
	if r, _, _ := procGetClientRect.Call(h.hwnd, uintptr(unsafe.Pointer(&cli))); r == 0 {
		return false
	}
	// 非客户区尺寸 = 窗口矩形 - 客户区矩形，缩放时保持不变。
	nw := (win.right - win.left) - (cli.right - cli.left)
	nh := (win.bottom - win.top) - (cli.bottom - cli.top)
	// 屏幕边缘不越界，免得缩小后再也看不见窗口。
	sw, _, _ := procGetSystemMetrics.Call(smCxScreen)
	sh, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x, y := fitToScreen(win.left, win.top, pxW+nw, pxH+nh, int32(sw), int32(sh))
	r, _, _ := procSetWindowPos.Call(
		h.hwnd, 0,
		uintptr(x), uintptr(y),
		uintptr(pxW+nw), uintptr(pxH+nh),
		swpNoZOrder|swpNoActivate,
	)
	if r == 0 {
		return false
	}
	// WM_SIZE 里已经跟过一遍；这里重采是因为 DWM 的边框重算可能晚一拍，
	// 保证 ctx 尺寸与 rt 一致。
	h.syncClientSize()
	h.requestFrame()
	return true
}

func (h *host) syncClientSize() {
	if h.hwnd == 0 {
		return
	}
	var rc rect
	procGetClientRect.Call(h.hwnd, uintptr(unsafe.Pointer(&rc)))
	w := uint32(rc.right - rc.left)
	ht := uint32(rc.bottom - rc.top)
	if w == 0 || ht == 0 {
		return
	}
	oldDPI := h.dpi
	h.pixelW, h.pixelH = w, ht
	h.dpi = float32(windowDPI(h.hwnd))
	if h.dpi <= 0 {
		h.dpi = 96
	}
	h.ctx.width = float32(w) * 96 / h.dpi
	h.ctx.height = float32(ht) * 96 / h.dpi
	h.ctx.dpi = h.dpi
	zoomed, _, _ := procIsZoomed.Call(h.hwnd)
	h.maximized = zoomed != 0
	h.syncFrame()
	if h.rt == 0 {
		return
	}
	if oldDPI != h.dpi {
		h.releaseTarget()
		return
	}
	size := d2d1SizeU{width: w, height: ht}
	hr, _, _ := syscall.SyscallN(
		h.rt.slot(idxResize),
		uintptr(h.rt),
		uintptr(unsafe.Pointer(&size)),
	)
	runtime.KeepAlive(size)
	if hresult(hr).failed() {
		h.releaseTarget()
	}
}

func (h *host) ensureTarget() error {
	if h.rt != 0 {
		return nil
	}
	if h.hwnd == 0 || h.pixelW == 0 || h.pixelH == 0 {
		return fmt.Errorf("github.com/goedui/goed/ui: 窗口尚未就绪")
	}
	rt, err := createHwndRenderTarget(h.d2d, h.hwnd, h.pixelW, h.pixelH, h.dpi)
	if err != nil {
		return err
	}
	brush, err := createSolidColorBrush(rt)
	if err != nil {
		release(rt)
		return err
	}
	h.rt = rt
	h.brush = brush
	h.ctx.rt = rt
	h.ctx.brush = brush
	return nil
}

func (h *host) paint() {
	if err := h.ensureTarget(); err != nil {
		return
	}
	syscall.SyscallN(h.rt.slot(idxBeginDraw), uintptr(h.rt))
	syscall.SyscallN(h.rt.slot(idxSetTextAntialiasMode), uintptr(h.rt), uintptr(d2d1TextAntialiasClearType))

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Fprintf(os.Stderr, "github.com/goedui/goed/ui: 绘制失败: %v\n", rec)
			}
		}()
		if h.opts.Paint != nil {
			h.opts.Paint(&h.ctx)
		}
	}()

	hr, _, _ := syscall.SyscallN(h.rt.slot(idxEndDraw), uintptr(h.rt), 0, 0)
	if hresult(hr).failed() {
		h.releaseTarget()
		if h.hwnd != 0 {
			procInvalidateRect.Call(h.hwnd, 0, 0)
		}
	}
}

func (h *host) releaseTarget() {
	h.ctx.releaseStroke()
	h.ctx.releaseShadows()
	h.ctx.releaseImages()
	if h.brush != 0 {
		release(h.brush)
		h.brush = 0
		h.ctx.brush = 0
	}
	if h.rt != 0 {
		release(h.rt)
		h.rt = 0
		h.ctx.rt = 0
	}
}

func (h *host) dispose() {
	h.releaseTarget()
	h.releaseIcons()
	h.ctx.releaseFormats()
	if h.dwrite != 0 {
		release(h.dwrite)
		h.dwrite = 0
		h.ctx.dwrite = 0
	}
	if h.d2d != 0 {
		release(h.d2d)
		h.d2d = 0
	}
	if hwnd := h.handleID(); hwnd != 0 {
		h.setHandle(0)
		delete(hosts, hwnd)
		procDestroyWindow.Call(hwnd)
	}
}

func (h *host) handleNcCalcSize(wParam, lParam uintptr) uintptr {
	if lParam == 0 {
		return 0
	}
	var r *rect
	if wParam != 0 {
		r = &(*nccalcsizeParams)(unsafe.Pointer(lParam)).rgrc[0]
	} else {
		r = (*rect)(unsafe.Pointer(lParam))
	}
	zoomed, _, _ := procIsZoomed.Call(h.hwnd)
	if zoomed != 0 {
		b := h.borderPx()
		r.left += b
		r.top += b
		r.right -= b
		r.bottom -= b
	}
	return 0
}

func (h *host) closeWindow() {
	if h.hwnd != 0 {
		procPostMessageW.Call(h.hwnd, wmClose, 0, 0)
	}
}

func (h *host) toggleMaximize() bool {
	if h.hwnd == 0 {
		return false
	}
	h.runCaption(2)
	return true
}

func (h *host) runCaption(btn int) {
	switch btn {
	case 1:
		procShowWindow.Call(h.hwnd, swMinimize)
	case 2:
		if h.maximized {
			procShowWindow.Call(h.hwnd, swRestore)
		} else {
			procShowWindow.Call(h.hwnd, swMaximize)
		}
	case 3:
		procPostMessageW.Call(h.hwnd, wmClose, 0, 0)
	}
}

func (h *host) handle(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmNcCalcSize:
		return h.handleNcCalcSize(wParam, lParam)
	case wmNcHitTest:
		return h.hitFromLParam(lParam)
	case wmNcActivate:
		h.active = wParam != 0
		h.syncFrame()
		h.requestFrame()
		return 1
	case wmNcPaint:
		return 0
	case wmNcMouseMove:
		h.trackNCLeave()
		h.setHover(hitToCaption(h.hitFromLParam(lParam)))
		return 0
	case wmNcLButtonDown:
		cap := hitToCaption(wParam)
		if cap != 0 {
			h.setPressed(cap)
			return 0
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	case wmNcLButtonUp:
		cap := hitToCaption(h.hitFromLParam(lParam))
		pressed := h.pressed
		h.setPressed(0)
		if cap != 0 && cap == pressed {
			h.runCaption(cap)
			return 0
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	case wmNcLButtonDblClk:
		if wParam == htCaption {
			h.runCaption(2)
			return 0
		}
		return 0
	case wmNcMouseLeave:
		h.tracking = false
		h.setHover(0)
		h.setPressed(0)
		return 0
	case wmMouseMove:
		if h.hover != 0 || h.pressed != 0 {
			h.setHover(0)
			h.setPressed(0)
		}
		h.trackClientLeave()
		x, y := h.clientDIP(lParam)
		h.emitPointer(x, y, h.pointerDown, false)
		return 0
	case wmLButtonDown:
		h.pointerDown = true
		procSetCapture.Call(hwnd)
		x, y := h.clientDIP(lParam)
		h.emitPointer(x, y, true, false)
		return 0
	case wmLButtonUp:
		wasDown := h.pointerDown
		h.pointerDown = false
		procReleaseCapture.Call()
		x, y := h.clientDIP(lParam)
		h.emitPointer(x, y, false, wasDown)
		return 0
	case wmRButtonDown:
		x, y := h.clientDIP(lParam)
		h.emitRightPointer(x, y, true)
		return 0
	case wmRButtonUp:
		x, y := h.clientDIP(lParam)
		h.emitRightPointer(x, y, false)
		return 0
	case wmMouseLeave:
		h.clientTracking = false
		h.pointerDown = false
		h.emitPointer(-1, -1, false, false)
		return 0
	case wmMouseWheel:
		x, y := h.clientDIPFromScreen(lParam)
		if h.opts.Wheel != nil {
			delta := float32(int16(wParam>>16)) / 120
			h.opts.Wheel(x, y, delta)
			h.requestFrame()
		}
		return 0
	case wmKeyDown, wmSysKeyDown:
		h.emitKey(int(wParam), 0, true)
		return 0
	case wmKeyUp, wmSysKeyUp:
		h.emitKey(int(wParam), 0, false)
		return 0
	case wmImeSetContext:
		// 应用自己画预编辑串时，让系统别画内联组合窗（候选窗照常显示）。
		if h.opts.Composition != nil {
			lParam &^= iscShowUICompositionWindow
		}
	case wmImeStartComposition:
		if h.syncComposition(0) {
			return 0
		}
	case wmImeComposition:
		if h.syncComposition(lParam) {
			return 0
		}
		// 带 GCS_RESULTSTR 的那次继续交给 DefWindowProc：
		// 它生成 WM_IME_CHAR → WM_CHAR，上屏文本走正常输入通路。
	case wmImeEndComposition:
		h.endComposition()
		return 0
	case wmChar:
		h.emitKey(0, rune(wParam), true)
		return 0
	case wmAppPost:
		postedMu.Lock()
		fn := posted[wParam]
		delete(posted, wParam)
		postedMu.Unlock()
		if fn != nil {
			fn()
			h.requestFrame()
		}
		return 0
	case wmActivate:
		h.active = wParam != 0
		h.syncFrame()
		h.requestFrame()
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSize:
		if wParam != sizeMinimized {
			h.maximized = wParam == sizeMaximized
			h.syncClientSize()
			h.requestFrame()
		}
		return 0
	case wmDpiChanged:
		if lParam != 0 {
			r := (*rect)(unsafe.Pointer(lParam))
			procSetWindowPos.Call(
				hwnd,
				0,
				uintptr(r.left),
				uintptr(r.top),
				uintptr(r.right-r.left),
				uintptr(r.bottom-r.top),
				swpNoZOrder|swpNoActivate,
			)
		}
		h.syncClientSize()
		h.applyDWM()
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case wmPaint:
		var ps [80]byte
		procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		h.paint()
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		return 0
	case wmClose:
		if h.opts.Close != nil && !h.opts.Close() {
			return 0
		}
	case wmDestroy:
		h.releaseTarget()
		delete(hosts, hwnd)
		// 走 setHandle：这步之后后台循环（光标闪烁 / 动画）还会再调一两次
		// requestFrame，读的就是这个字段。
		h.setHandle(0)
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	h := hosts[hwnd]
	if h == nil && creating != nil {
		creating.hwnd = hwnd
		hosts[hwnd] = creating
		h = creating
	}
	if h != nil {
		return h.handle(hwnd, uint32(msg), wParam, lParam)
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}
