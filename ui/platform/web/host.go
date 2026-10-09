//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/goedui/goed/ui/renderer"
)

// Options 描述浏览器宿主。字段与 ui/platform/windows 的 Options 一一对应，
// 由 ui/platform/run_js.go 装配；应用代码看不到这个结构。
type Options struct {
	Title   string
	Width   int
	Height  int
	Dark    bool
	Icon16  []byte
	Icon32  []byte
	Paint   func(ctx renderer.Context)
	Pointer func(x, y float32, down, click bool)
	// RightPointer 报告右键，与 windows 宿主同名同义（见 platform.Options）。
	RightPointer func(x, y float32, down bool)
	Key          func(key int, char rune, down, ctrl, shift, alt bool)
	Wheel        func(x, y, delta float32)
	// Composition 报告输入法组合串（预编辑）。组合串的显示一律由焦点控件自己画：
	// 浏览器没有 IMM32 那种让宿主决定画不画内联组合串的机制，隐藏 textarea 里本就
	// 看不见。返回值只做记录。
	Composition func(text string, cursor int) bool
	// Caret 报告当前焦点的光标矩形（DIP），宿主用它把隐藏 textarea 挪到光标
	// 旁边，输入法候选窗才会贴住光标而不是飘在左上角。
	Caret func() (x, y, w, h float32, ok bool)

	BindRequestFrame func(fn func())
	BindPost         func(fn func(func()))
	FrameUpdate      func(maximized, active bool, hover, pressed int)
	// SetWindowTitle 由平台层注入（web.SetDocumentTitle）。
	SetWindowTitle func(title string)
	// GetWindowSize 由平台层注入，宿主用它把「读画布尺寸」的能力交出去。
	GetWindowSize func(fn func() (w, h int, ok bool))

	// CanvasID 指定要挂载的 <canvas> id。空则用包级 CanvasID。
	CanvasID string
}

// CanvasID 是要挂载的 canvas 元素 id，默认 "app"（imgen/web/index.html 用的就是它）。
// 找不到就退到页面里第一个 <canvas>，再找不到就自己建一个铺满视口的。
// 做成包级变量而不是 platform.Options 的字段：「用哪个 canvas」只有本后端需要。
var CanvasID = "app"

// running 保住当前宿主不被回收：host 与它持有的 js.Func 互相引用，不挂一个
// 包级引用整个环就不可达（js.FuncOf 有全局表兜底，这里再钉一层，也挡住重复 Run）。
var running *host

type host struct {
	opts   Options
	canvas js.Value
	ctx2d  js.Value
	ctx    *canvasContext

	// pending 表示已经有一个 rAF 在排队。js/wasm 单线程，所有 goroutine
	// 都是协作式调度，这里不需要加锁。
	pending bool

	pointerDown bool
	hovering    bool
	composing   bool

	textarea js.Value
	caretX   float32
	caretY   float32

	fnFrame js.Func
	fnSize  js.Func
	fnDown  js.Func
	fnMove  js.Func
	fnUp    js.Func
	fnLeave js.Func
	fnWheel js.Func
	fnCtx   js.Func
	fnKeyDn js.Func
	fnKeyUp js.Func
	fnCompS js.Func
	fnCompU js.Func
	fnCompE js.Func
	fnPaste js.Func
	fnCopy  js.Func
	fnBlur  js.Func
	fnFocus js.Func
	fnFonts js.Func
}

// Run 把引擎挂到页面上的 canvas，然后**永不返回**——浏览器的生命周期由页面
// 决定（关标签页 / 刷新），没有「窗口消息循环退出」这个事件。
//
// 与 windows 后端的差异：
//   - 没有 HWND，platform.HWND 保持 0；
//   - Post 是同步直调（单线程，没有「投递到 UI 线程」这回事）；
//   - 没有 SetWindowSize（画布尺寸由页面布局决定），GetWindowSize 返回画布 CSS 尺寸。
func Run(opts Options) error {
	h, err := start(opts)
	if err != nil {
		return err
	}
	running = h

	// 主 goroutine 挂住。js/wasm 里所有事件回调都由 JS 事件循环驱动，
	// 主 goroutine 一旦返回，go.run 的 Promise 落地、整个实例就结束了。
	select {}
}

// start 把宿主装配好并请求第一帧，但不阻塞。
//
// 从 Run 拆出来是给后端自检用的（ui/platform/web/probe.go）：那里要真的起一个
// 宿主再灌合成事件，看回调到底收到了什么——而 Run 永不返回，之后没法接着做事。
func start(opts Options) (*host, error) {
	if running != nil {
		return nil, fmt.Errorf("github.com/goedui/goed/ui: web 宿主已经启动过了")
	}
	if opts.Paint == nil {
		return nil, fmt.Errorf("github.com/goedui/goed/ui: web 宿主缺少 Paint 回调")
	}
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return nil, fmt.Errorf("github.com/goedui/goed/ui: 没有 document，不能在非浏览器环境启动")
	}
	canvas := findCanvas(doc, opts.CanvasID)
	if !canvas.Truthy() {
		return nil, fmt.Errorf("github.com/goedui/goed/ui: 页面上找不到 <canvas>，也创建不出来")
	}
	ctx2d := canvas.Call("getContext", "2d")
	if !ctx2d.Truthy() {
		return nil, fmt.Errorf("github.com/goedui/goed/ui: canvas.getContext(\"2d\") 返回空")
	}
	h := &host{opts: opts, canvas: canvas, ctx2d: ctx2d}
	h.ctx = newCanvasContext(canvas, ctx2d)

	h.makeTextarea(doc)
	if opts.SetWindowTitle != nil {
		opts.SetWindowTitle(opts.Title)
	}
	if opts.BindRequestFrame != nil {
		opts.BindRequestFrame(h.requestFrame)
	}
	if opts.BindPost != nil {
		// js/wasm 只有一个线程：所有 goroutine 都跑在这条线程上，
		// 「投递到 UI 线程」与直接调用是同一件事（linux 后端的 stub 同理）。
		opts.BindPost(func(fn func()) {
			if fn != nil {
				fn()
			}
		})
	}
	if opts.GetWindowSize != nil {
		opts.GetWindowSize(h.windowSize)
	}
	if opts.FrameUpdate != nil {
		opts.FrameUpdate(false, true, 0, 0)
	}
	h.bind(doc)
	h.resize()
	h.requestFrame()
	return h, nil
}

// findCanvas 按 id → 第一个 canvas → 自己建一个 的顺序找挂载点。
func findCanvas(doc js.Value, id string) js.Value {
	if id == "" {
		id = CanvasID
	}
	if id != "" {
		if el := doc.Call("getElementById", id); el.Truthy() {
			if strings.EqualFold(el.Get("tagName").String(), "CANVAS") {
				return el
			}
		}
	}
	if el := doc.Call("querySelector", "canvas"); el.Truthy() {
		return el
	}
	body := doc.Get("body")
	if !body.Truthy() {
		return js.Null()
	}
	el := doc.Call("createElement", "canvas")
	el.Set("id", "app")
	st := el.Get("style")
	st.Set("display", "block")
	st.Set("width", "100vw")
	st.Set("height", "100vh")
	st.Set("outline", "none")
	body.Call("appendChild", el)
	return el
}

// makeTextarea 建一个 1x1、透明的 textarea 当键盘与输入法的落点。
//
// 必须是真可编辑元素：compositionstart / compositionupdate 只在可编辑元素上触发，
// 只监听 window 的 keydown 拿不到预编辑串，中文 / 日文输入就用不了。pointer-events:none
// 让它接不到点击（命中测试归 canvas），焦点由 canvas 的 pointerdown 主动给它。
func (h *host) makeTextarea(doc js.Value) {
	ta := doc.Call("createElement", "textarea")
	ta.Set("autocomplete", "off")
	ta.Set("autocapitalize", "off")
	ta.Set("autocorrect", "off")
	ta.Set("spellcheck", false)
	ta.Set("wrap", "off")
	ta.Set("tabIndex", -1)
	st := ta.Get("style")
	st.Set("position", "fixed")
	st.Set("left", "0px")
	st.Set("top", "0px")
	st.Set("width", "1px")
	st.Set("height", "1px")
	st.Set("padding", "0")
	st.Set("margin", "0")
	st.Set("border", "0")
	st.Set("outline", "none")
	st.Set("resize", "none")
	st.Set("overflow", "hidden")
	st.Set("whiteSpace", "pre")
	st.Set("background", "transparent")
	// 不用 opacity:0——部分浏览器会因此不给元素开输入法；透明文字 + 1px 见方
	// 效果一样，输入法照常工作。
	st.Set("color", "transparent")
	st.Set("caretColor", "transparent")
	st.Set("fontSize", "16px")
	st.Set("pointerEvents", "none")
	st.Set("zIndex", "1")
	doc.Get("body").Call("appendChild", ta)
	h.textarea = ta
	ta.Call("focus")
}

// ===== 帧循环 ==========================================================

// requestFrame 请求一次重绘。可以来自任意 goroutine（光标闪烁、加载动画这类
// 后台循环都这么调），单线程所以只是个布尔标记。
func (h *host) requestFrame() {
	if h.pending {
		return
	}
	h.pending = true
	js.Global().Call("requestAnimationFrame", h.fnFrame)
}

func (h *host) frame(js.Value, []js.Value) any {
	h.pending = false
	h.resize()
	h.opts.Paint(h.ctx)
	h.syncCaret()
	return nil
}

// resize 把画布尺寸对齐到当前的 CSS 尺寸与 devicePixelRatio。
//
// 每帧都查一遍而不是只在 resize 事件里做：设备像素比会随窗口拖到另一块屏幕而变，
// 而那个变化不一定伴随 resize 事件；查一次只是两次属性读。
func (h *host) resize() {
	cssW := h.canvas.Get("clientWidth").Float()
	cssH := h.canvas.Get("clientHeight").Float()
	if cssW < 1 {
		cssW = float64(h.opts.Width)
	}
	if cssH < 1 {
		cssH = float64(h.opts.Height)
	}
	if cssW < 1 {
		cssW = 800
	}
	if cssH < 1 {
		cssH = 600
	}
	dpr := js.Global().Get("devicePixelRatio").Float()
	if dpr <= 0 {
		dpr = 1
	}
	pw := int(cssW*dpr + 0.5)
	ph := int(cssH*dpr + 0.5)
	if h.canvas.Get("width").Int() != pw {
		h.canvas.Set("width", pw)
	}
	if h.canvas.Get("height").Int() != ph {
		h.canvas.Set("height", ph)
	}
	h.ctx.resize(pw, ph, float32(cssW), float32(cssH), float32(96*dpr))
}

func (h *host) windowSize() (int, int, bool) {
	w := h.canvas.Get("clientWidth").Float()
	ht := h.canvas.Get("clientHeight").Float()
	if w < 1 || ht < 1 {
		return 0, 0, false
	}
	return int(w + 0.5), int(ht + 0.5), true
}

// syncCaret 把隐藏 textarea 挪到光标下方，输入法候选窗才会贴着光标。
func (h *host) syncCaret() {
	if h.opts.Caret == nil {
		return
	}
	x, y, _, ch, ok := h.opts.Caret()
	if !ok {
		return
	}
	if x == h.caretX && y == h.caretY {
		return
	}
	h.caretX, h.caretY = x, y
	st := h.textarea.Get("style")
	st.Set("left", fmt.Sprintf("%dpx", int(x)))
	st.Set("top", fmt.Sprintf("%dpx", int(y+ch)))
}

// ===== 事件绑定 ========================================================

func (h *host) bind(doc js.Value) {
	win := js.Global()
	h.fnFrame = js.FuncOf(h.frame)
	h.fnSize = js.FuncOf(func(js.Value, []js.Value) any {
		h.requestFrame()
		return nil
	})
	h.fnDown = js.FuncOf(h.onPointerDown)
	h.fnMove = js.FuncOf(h.onPointerMove)
	h.fnUp = js.FuncOf(h.onPointerUp)
	h.fnLeave = js.FuncOf(func(js.Value, []js.Value) any {
		// 拖拽中离开画布不算「移出」：Windows 在捕获状态下也照样继续上报客户区外的
		// 坐标，控件靠这个把拖到边界外的选区接住。
		if h.pointerDown {
			return nil
		}
		if h.hovering {
			h.hovering = false
			h.emitPointer(-1, -1, false, false)
		}
		return nil
	})
	h.fnWheel = js.FuncOf(h.onWheel)
	h.fnKeyDn = js.FuncOf(h.onKeyDown)
	h.fnKeyUp = js.FuncOf(h.onKeyUp)
	h.fnCompS = js.FuncOf(h.onCompositionStart)
	h.fnCompU = js.FuncOf(h.onCompositionUpdate)
	h.fnCompE = js.FuncOf(h.onCompositionEnd)
	h.fnPaste = js.FuncOf(h.onPaste)
	h.fnCopy = js.FuncOf(func(this js.Value, args []js.Value) any {
		// 编辑框的 Ctrl+C 走 platform.ClipboardSet；浏览器随后还会拿「隐藏 textarea
		// 里的选区」覆盖一次剪贴板——那份选区是空的，不拦下来就把刚写进去的内容抹掉。
		args[0].Call("preventDefault")
		return nil
	})
	// 右键菜单：浏览器自己的那份要拦掉，画布上的右键归应用（见 onPointerDown）。
	h.fnCtx = js.FuncOf(func(this js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		return nil
	})
	h.fnBlur = js.FuncOf(func(js.Value, []js.Value) any {
		if h.opts.FrameUpdate != nil {
			h.opts.FrameUpdate(false, false, 0, 0)
		}
		h.requestFrame()
		return nil
	})
	h.fnFocus = js.FuncOf(func(js.Value, []js.Value) any {
		h.textarea.Call("focus")
		if h.opts.FrameUpdate != nil {
			h.opts.FrameUpdate(false, true, 0, 0)
		}
		h.requestFrame()
		return nil
	})
	h.fnFonts = js.FuncOf(func(js.Value, []js.Value) any {
		// 自定义字体（webfont）加载完成前量出来的宽度是回退字体的宽度，字体一到
		// 就把度量缓存丢掉重算，否则整屏文字会按错的宽度排版。
		h.ctx.clearFontCache()
		h.requestFrame()
		return nil
	})

	h.listen(h.canvas, "pointerdown", h.fnDown, false)
	h.listen(h.canvas, "pointerleave", h.fnLeave, false)
	// move / up 挂 window 而不是 canvas：拖到画布外面（选一段文字、拖滚动条）
	// 时事件还得继续来。挂 canvas 就得用 setPointerCapture，而那个调用在
	// pointerId 失效时会抛异常，抛进 wasm 就是整个程序挂掉。
	h.listen(win, "pointermove", h.fnMove, true)
	h.listen(win, "pointerup", h.fnUp, true)
	h.listen(win, "pointercancel", h.fnUp, true)
	h.listen(h.canvas, "wheel", h.fnWheel, false)
	h.listen(h.canvas, "contextmenu", h.fnCtx, false)
	// 键盘挂 window：焦点在隐藏 textarea 上时事件冒泡到这里，焦点万一丢了
	// （用户点了别处、浏览器吞了一次 focus）也还能接住。
	h.listen(win, "keydown", h.fnKeyDn, false)
	h.listen(win, "keyup", h.fnKeyUp, true)
	h.listen(win, "compositionstart", h.fnCompS, false)
	h.listen(win, "compositionupdate", h.fnCompU, true)
	h.listen(win, "compositionend", h.fnCompE, false)
	h.listen(win, "paste", h.fnPaste, false)
	h.listen(win, "copy", h.fnCopy, false)
	h.listen(win, "cut", h.fnCopy, false)
	h.listen(win, "blur", h.fnBlur, true)
	h.listen(win, "focus", h.fnFocus, true)

	// 画布尺寸变化（窗口缩放、侧栏折叠、CSS 变化）统一走 ResizeObserver；
	// 没有它就退回 window 的 resize 事件。
	if ro := js.Global().Get("ResizeObserver"); ro.Truthy() {
		ro.New(h.fnSize).Call("observe", h.canvas)
	} else {
		h.listen(win, "resize", h.fnSize, true)
	}
	if fonts := doc.Get("fonts"); fonts.Truthy() {
		if ready := fonts.Get("ready"); ready.Truthy() {
			ready.Call("then", h.fnFonts)
		}
	}
}

// listen 统一带 options 注册：passive:false 才允许 preventDefault
// （wheel 在 Chrome 里默认是 passive，不显式关掉 preventDefault 会被忽略）。
func (h *host) listen(target js.Value, name string, fn js.Func, passive bool) {
	opts := js.Global().Get("Object").New()
	opts.Set("passive", passive)
	target.Call("addEventListener", name, fn, opts)
}

// foreignEditable 报告事件目标是不是「别人的」输入控件：页面外壳里若有搜索框
// 之类的真 DOM 输入，不该被我们抢走键盘。
//
// target 不一定是元素（键盘事件派发到 window 时 target 就是 window，刚加载完
// 可能是 document / body），它们没有 tagName / isContentEditable，而 syscall/js
// 对 undefined 调 Bool() 会 panic——一次 panic 整个 wasm 实例就退出，所以每个
// 属性都要先判类型。（probe.go 的宿主自检第一次跑就撞上了。）
func (h *host) foreignEditable(e js.Value) bool {
	t := e.Get("target")
	if !t.Truthy() || t.Type() != js.TypeObject || !h.textarea.Truthy() || t.Equal(h.textarea) {
		return false
	}
	if tag := t.Get("tagName"); tag.Type() == js.TypeString {
		switch strings.ToUpper(tag.String()) {
		case "INPUT", "TEXTAREA", "SELECT":
			return true
		}
	}
	if v := t.Get("isContentEditable"); v.Type() == js.TypeBoolean {
		return v.Bool()
	}
	return false
}

// ===== 指针 ============================================================

func (h *host) emitPointer(x, y float32, down, click bool) {
	if h.opts.Pointer != nil {
		h.opts.Pointer(x, y, down, click)
	}
	if click || down {
		h.requestFrame()
	}
}

// emitRightPointer 上报右键，与 windows 宿主对齐：只在按下时请求重绘，
// 也不碰 h.pointerDown（左键拖选是另一条通道）。
func (h *host) emitRightPointer(x, y float32, down bool) {
	if h.opts.RightPointer != nil {
		h.opts.RightPointer(x, y, down)
	}
	if down {
		h.requestFrame()
	}
}

// pointerPos 把客户端坐标换成画布内的 DIP 坐标，并报告是否落在画布内。
func (h *host) pointerPos(e js.Value) (float32, float32, bool) {
	r := h.canvas.Call("getBoundingClientRect")
	x := e.Get("clientX").Float() - r.Get("left").Float()
	y := e.Get("clientY").Float() - r.Get("top").Float()
	w := r.Get("width").Float()
	ht := r.Get("height").Float()
	inside := x >= 0 && y >= 0 && x <= w && y <= ht
	return float32(x), float32(y), inside
}

func (h *host) onPointerDown(this js.Value, args []js.Value) any {
	e := args[0]
	if btn := e.Get("button").Int(); btn == 2 {
		// 右键走单独一条通道：不改 h.pointerDown、不抢焦点，也不进左键那条路。
		e.Call("preventDefault")
		x, y, _ := h.pointerPos(e)
		h.emitRightPointer(x, y, true)
		return nil
	} else if btn != 0 {
		return nil
	}
	// 取消 pointerdown 会连带取消后续的 mousedown，canvas 就不会抢走
	// textarea 的焦点——否则每点一次鼠标，输入法就失效一次。
	e.Call("preventDefault")
	h.textarea.Call("focus")
	x, y, _ := h.pointerPos(e)
	h.pointerDown = true
	h.hovering = true
	h.emitPointer(x, y, true, false)
	return nil
}

func (h *host) onPointerMove(this js.Value, args []js.Value) any {
	e := args[0]
	x, y, inside := h.pointerPos(e)
	if !inside && !h.pointerDown {
		if h.hovering {
			h.hovering = false
			h.emitPointer(-1, -1, false, false)
		}
		return nil
	}
	h.hovering = true
	h.emitPointer(x, y, h.pointerDown, false)
	return nil
}

func (h *host) onPointerUp(this js.Value, args []js.Value) any {
	e := args[0]
	if e.Get("button").Int() == 2 {
		x, y, _ := h.pointerPos(e)
		h.emitRightPointer(x, y, false)
		return nil
	}
	if !h.pointerDown {
		return nil
	}
	if e.Get("button").Int() != 0 && e.Get("type").String() != "pointercancel" {
		return nil
	}
	x, y, _ := h.pointerPos(e)
	// 与 windows 对齐：抬起时 down=false，click=抬起前是否按着。
	h.pointerDown = false
	click := e.Get("type").String() != "pointercancel"
	h.emitPointer(x, y, false, click)
	return nil
}

func (h *host) onWheel(this js.Value, args []js.Value) any {
	e := args[0]
	e.Call("preventDefault")
	if h.opts.Wheel == nil {
		return nil
	}
	x, y, _ := h.pointerPos(e)
	d := e.Get("deltaY").Float()
	var notches float64
	switch e.Get("deltaMode").Int() {
	case 1: // 行
		notches = d / 3
	case 2: // 页
		notches = d
	default: // 像素：Chrome 滚轮一格 100px
		notches = d / 100
	}
	if notches == 0 {
		return nil
	}
	// windows 的 delta 以「格」为单位、向上为正；浏览器的 deltaY 向下为正。
	h.opts.Wheel(x, y, float32(-notches))
	h.requestFrame()
	return nil
}

// ===== 键盘与输入法 ====================================================

func (h *host) emitKey(key int, char rune, down, ctrl, shift, alt bool) {
	if h.opts.Key == nil {
		return
	}
	h.opts.Key(key, char, down, ctrl, shift, alt)
}

func (h *host) onKeyDown(this js.Value, args []js.Value) any {
	e := args[0]
	if h.foreignEditable(e) {
		return nil
	}
	// 输入法组合中的按键全部交给 IME：keyCode 229 是「正在组合」的
	// 历史约定，Chrome 起手那一下 isComposing 还是 false，只看它不够。
	if h.composing || e.Get("isComposing").Bool() || e.Get("keyCode").Int() == 229 {
		return nil
	}
	ctrl := e.Get("ctrlKey").Bool() || e.Get("metaKey").Bool()
	shift := e.Get("shiftKey").Bool()
	alt := e.Get("altKey").Bool()
	key := e.Get("key").String()
	code := e.Get("code").String()

	// Ctrl+V 不转发：编辑框收到 Ctrl+V 会去读 platform.ClipboardGet，而浏览器里
	// 同步读剪贴板做不到（navigator.clipboard 全是 Promise）。交给下面的 paste 事件
	// 当「上屏文本」送进去——那条路既不弹权限框，也不用猜用户什么时候按的 Ctrl。
	if ctrl && !alt && (key == "v" || key == "V") {
		return nil
	}
	char := rune(0)
	if !ctrl && !alt {
		char = charFromKey(key)
	}
	h.emitKey(vkFromCode(code), char, true, ctrl, shift, alt)
	h.requestFrame()
	if !letBrowserHandle(key, ctrl, shift) {
		e.Call("preventDefault")
	}
	return nil
}

func (h *host) onKeyUp(this js.Value, args []js.Value) any {
	e := args[0]
	if h.foreignEditable(e) || h.composing {
		return nil
	}
	h.emitKey(vkFromCode(e.Get("code").String()), 0, false,
		e.Get("ctrlKey").Bool() || e.Get("metaKey").Bool(),
		e.Get("shiftKey").Bool(), e.Get("altKey").Bool())
	return nil
}

func (h *host) onCompositionStart(this js.Value, args []js.Value) any {
	h.composing = true
	if h.opts.Composition != nil {
		h.opts.Composition("", 0)
	}
	h.requestFrame()
	return nil
}

func (h *host) onCompositionUpdate(this js.Value, args []js.Value) any {
	text := args[0].Get("data").String()
	if h.opts.Composition != nil {
		// 浏览器不给组合串内的光标位置，只能按「光标在末尾」报。
		h.opts.Composition(text, len([]rune(text)))
	}
	h.requestFrame()
	return nil
}

func (h *host) onCompositionEnd(this js.Value, args []js.Value) any {
	h.composing = false
	if h.opts.Composition != nil {
		h.opts.Composition("", 0)
	}
	h.commitText(args[0].Get("data").String())
	h.textarea.Set("value", "")
	h.requestFrame()
	return nil
}

func (h *host) onPaste(this js.Value, args []js.Value) any {
	e := args[0]
	if h.foreignEditable(e) {
		return nil
	}
	e.Call("preventDefault")
	cd := e.Get("clipboardData")
	if !cd.Truthy() {
		return nil
	}
	text := cd.Call("getData", "text/plain").String()
	if text == "" {
		text = cd.Call("getData", "text").String()
	}
	if text == "" {
		return nil
	}
	// paste 是浏览器里唯一「不需要权限、同步拿到内容」的剪贴板读法，
	// 顺手把内容记进缓存，给还会走 platform.ClipboardGet 的调用方用。
	clipCache = text
	h.commitText(text)
	h.requestFrame()
	return nil
}

// commitText 把一段上屏文本按字符逐个交给引擎，走与 WM_CHAR 同一条路（key=0、
// char 非 0、down=true）。
//
// 逐字符而不是整串：platform.Options 的 Key 回调只有单个 rune 的口子，引擎侧
// （components.handleEditKey）也按单字符插入。粘几百个字符没问题，真要粘一整篇几万字的
// 文档，这条路的代价会显出来。
func (h *host) commitText(text string) {
	if text == "" || h.opts.Key == nil {
		return
	}
	for _, r := range text {
		if r == '\r' {
			continue
		}
		if r == '\n' {
			// 换行按回车送：编辑框里 '\n' 走的是「插入字符」那条路，
			// 而 char < 32 会被当成控制字符丢掉，退化成什么都不发生。
			h.opts.Key(vkEnter, 0, true, false, false, false)
			continue
		}
		h.opts.Key(0, r, true, false, false, false)
	}
}

// charFromKey 从 KeyboardEvent.key 取文本字符。返回 0 表示「不是文本输入」。
func charFromKey(key string) rune {
	r := []rune(key)
	if len(r) != 1 || r[0] < 32 {
		return 0
	}
	return r[0]
}

// letBrowserHandle 决定哪些按键不拦。
//
// 拦得太狠会让刷新 / 开发者工具都按不出来（canvas 应用很容易掉进这个坑），放得太松
// 又会让 Tab 把焦点移走、空格滚动页面。
func letBrowserHandle(key string, ctrl, shift bool) bool {
	switch key {
	case "F5", "F11", "F12":
		return true
	}
	if ctrl {
		switch key {
		case "r", "R": // 刷新
			return true
		case "I", "J": // 开发者工具（Ctrl+Shift+I / J）
			return shift
		}
	}
	return false
}

// ===== 虚拟键 ==========================================================

const (
	vkEnter = 0x0D
)

var codeKeys = map[string]int{
	"Backspace": 0x08, "Tab": 0x09, "Enter": 0x0D, "NumpadEnter": 0x0D,
	"ShiftLeft": 0x10, "ShiftRight": 0x10,
	"ControlLeft": 0x11, "ControlRight": 0x11,
	"AltLeft": 0x12, "AltRight": 0x12,
	"CapsLock": 0x14, "Escape": 0x1B, "Space": 0x20,
	"PageUp": 0x21, "PageDown": 0x22, "End": 0x23, "Home": 0x24,
	"ArrowLeft": 0x25, "ArrowUp": 0x26, "ArrowRight": 0x27, "ArrowDown": 0x28,
	"Insert": 0x2D, "Delete": 0x2E,
	"Minus": 0xBD, "Equal": 0xBB, "BracketLeft": 0xDB, "BracketRight": 0xDD,
	"Backslash": 0xDC, "Semicolon": 0xBA, "Quote": 0xDE, "Backquote": 0xC0,
	"Comma": 0xBC, "Period": 0xBE, "Slash": 0xBF,
}

// vkFromCode 把 KeyboardEvent.code（物理键位）换成 Windows VK 号。
//
// 用 code 而不是 key：ui/runtime 的 KeyA..KeyZ 与 VK_* 对齐，快捷键要的是「按了
// 哪个键」而非「打出了什么字」——Ctrl+Shift+1 在美式键盘上 key 是 "!"，用 key
// 推出的 VK 就不是 VK_1；用 code 也天然免疫输入法 / 大小写状态。
func vkFromCode(code string) int {
	if v, ok := codeKeys[code]; ok {
		return v
	}
	switch {
	case len(code) == 4 && strings.HasPrefix(code, "Key"):
		// KeyA..KeyZ → 0x41..0x5A（正好是 'A'..'Z' 的 ASCII）
		return int(code[3])
	case len(code) == 6 && strings.HasPrefix(code, "Digit"):
		return int(code[5])
	case len(code) == 7 && strings.HasPrefix(code, "Numpad"):
		if c := code[6]; c >= '0' && c <= '9' {
			return 0x60 + int(c-'0')
		}
	case strings.HasPrefix(code, "F"):
		n := 0
		for i := 1; i < len(code); i++ {
			if code[i] < '0' || code[i] > '9' {
				return 0
			}
			n = n*10 + int(code[i]-'0')
		}
		if n >= 1 && n <= 24 {
			return 0x70 + n - 1
		}
	}
	return 0
}

// ===== 剪贴板 ==========================================================

// clipCache 是剪贴板的本地副本：ClipboardGet 是同步签名，而浏览器里读剪贴板全是
// Promise，两者对不上。凡是能同步拿到内容的时机（paste 事件、我们自己写进去的）
// 就更新缓存，读取方拿到的是「最近一次见到或写过的内容」；真正的读路径是 paste
// 事件（见 onPaste），缓存只给还走 ClipboardGet 的调用方兜底。
var (
	clipCache    string
	pendingCopy  string
	copyResolved = js.FuncOf(func(js.Value, []js.Value) any { return nil })
	copyRejected = js.FuncOf(func(js.Value, []js.Value) any {
		legacyCopy(pendingCopy)
		return nil
	})
)

// ClipboardText 读剪贴板文本（本地缓存）。实现 platform.ClipboardGet。
func ClipboardText() string { return clipCache }

// SetClipboardText 写剪贴板文本。实现 platform.ClipboardSet。
func SetClipboardText(text string) {
	clipCache = text
	pendingCopy = text
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return
	}
	cb := nav.Get("clipboard")
	if !cb.Truthy() {
		legacyCopy(text)
		return
	}
	// writeText 在「非安全上下文」或文档没焦点时会 reject，退回 execCommand('copy')。
	// 两个 js.Func 都是包级预建的：每次调用新建一个会一路泄漏 JS 函数对象。
	cb.Call("writeText", text).Call("then", copyResolved, copyRejected)
}

// legacyCopy 用一次临时 textarea + execCommand('copy') 兜底。
func legacyCopy(text string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() || text == "" {
		return
	}
	ta := doc.Call("createElement", "textarea")
	ta.Set("value", text)
	st := ta.Get("style")
	st.Set("position", "fixed")
	st.Set("left", "-9999px")
	st.Set("top", "0px")
	doc.Get("body").Call("appendChild", ta)
	ta.Call("select")
	doc.Call("execCommand", "copy")
	ta.Call("remove")
}

// SetDocumentTitle 更新浏览器标签页标题。实现 platform.SetWindowTitle。
func SetDocumentTitle(title string) {
	if title == "" {
		return
	}
	doc := js.Global().Get("document")
	if doc.Truthy() {
		doc.Set("title", title)
	}
}

// DownloadBytes 触发一次浏览器下载，实现 platform.SaveBytes（web 下「不弹对话框直接
// 保存」的那条路）。
//
// 用 <a download> 而不是 File System Access API（showSaveFilePicker）：后者只有
// Chromium 系有，而且每次都要用户手势。mime 写进 Blob 类型（浏览器据此决定「打开还是
// 下载」以及关联程序），name 作文件名；同源 blob 不受 <a download> 的跨域限制。
func DownloadBytes(name, mime string, data []byte) error {
	if len(data) == 0 {
		return errors.New("web: 没有要保存的内容")
	}
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return errors.New("web: 页面里没有 document")
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	if name == "" {
		name = "download"
	}

	// []byte → Uint8Array → Blob。CopyBytesToJS 直接把 Go 的切片拷进 JS 堆，
	// 不用先转成字符串（图片二进制走字符串会经历一次 UTF-8 编解码，体积和内容
	// 都可能出问题）。
	u8 := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(u8, data)

	opts := js.Global().Get("Object").New()
	opts.Set("type", mime)
	blob := js.Global().Get("Blob").New(js.Global().Get("Array").Call("of", u8), opts)

	url := js.Global().Get("URL").Call("createObjectURL", blob)

	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	a.Get("style").Set("display", "none")
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	a.Call("remove")

	// 不能立刻 revoke：下载是异步起步的，同任务里把 blob 收掉会让下载拿不到数据
	// （Firefox 上稳定复现）。等一秒再释放。
	go func() {
		time.Sleep(time.Second)
		js.Global().Get("URL").Call("revokeObjectURL", url)
	}()
	return nil
}
