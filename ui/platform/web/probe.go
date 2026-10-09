//go:build js && wasm && webprobe

package web

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"syscall/js"

	"github.com/goedui/goed/ui/renderer"
)

// RunProbe 在真浏览器里跑两轮自检，把结论写进 #probe：
//
//  1. 渲染后端：自建一块 canvas，逐个绘制入口画下去再按像素读回来；
//  2. 宿主事件：真的起一个 web 宿主，往里灌合成事件，看回调收到了什么。
//
// 为什么要单独一个 wasm 程序：js/wasm 下没有能跑的 go test（需要带 DOM 的环境），
// 而这个后端只有真落到 canvas / DOM 上才有意义 ——「没画出来」和「画错了」在 Go 侧
// 看不出来。windows / linux 靠真实窗口截图，web 这边对应的手段就是它。
//
// 构建：独立 wasm（-tags webprobe），不走 imgen/web 的站点构建。
// 跑法：无头浏览器 dump-dom，读 #probe 的文本。
func RunProbe() {
	runContextProbe()
	// 宿主那一轮跑完就结束（它自己会把帧推完），所以排在最后。
	runHostProbe()
}

var probeLines []string

func probeCheck(name string, ok bool, detail string) {
	mark := "PASS"
	if !ok {
		mark = "FAIL"
	}
	probeLines = append(probeLines, fmt.Sprintf("%s  %-30s %s", mark, name, detail))
}

func probeReport() {
	fails := 0
	for _, ln := range probeLines {
		if strings.HasPrefix(ln, "FAIL") {
			fails++
		}
	}
	text := fmt.Sprintf("webprobe: %d 项 / %d 失败\n", len(probeLines), fails) +
		strings.Join(probeLines, "\n")
	doc := js.Global().Get("document")
	if el := doc.Call("getElementById", "probe"); el.Truthy() {
		el.Set("textContent", text)
	} else if body := doc.Get("body"); body.Truthy() {
		el := doc.Call("createElement", "pre")
		el.Set("id", "probe")
		el.Set("textContent", text)
		body.Call("appendChild", el)
	}
	js.Global().Get("console").Call("log", text)
}

// ===== 第一轮：渲染后端 ================================================

func runContextProbe() {
	doc := js.Global().Get("document")
	canvas := doc.Call("createElement", "canvas")
	canvas.Set("id", "probe-canvas")
	canvas.Set("width", 200)
	canvas.Set("height", 200)
	ctx2d := canvas.Call("getContext", "2d", willReadFrequently())
	if !ctx2d.Truthy() {
		probeCheck("创建 2D 上下文", false, "getContext(\"2d\") 返回空")
		return
	}
	if body := doc.Get("body"); body.Truthy() {
		body.Call("appendChild", canvas)
	}
	c := newCanvasContext(canvas, ctx2d)
	c.resize(200, 200, 200, 200, 96)

	pix := func(x, y int) [3]int {
		d := ctx2d.Call("getImageData", x, y, 1, 1).Get("data")
		return [3]int{d.Index(0).Int(), d.Index(1).Int(), d.Index(2).Int()}
	}
	near := func(x, y int, want [3]int) bool { return isSame(pix(x, y), want) }
	isWhite := func(p [3]int) bool { return p[0] > 250 && p[1] > 250 && p[2] > 250 }
	isGreen := func(p [3]int) bool { return p[0] < 5 && p[1] > 250 && p[2] < 5 }
	inkIn := func(x0, y0, x1, y1 int) int {
		d := ctx2d.Call("getImageData", x0, y0, x1-x0, y1-y0).Get("data")
		n := 0
		for i := 0; i < d.Length(); i += 4 {
			if d.Index(i).Int() < 128 && d.Index(i+1).Int() < 128 && d.Index(i+2).Int() < 128 {
				n++
			}
		}
		return n
	}

	white := renderer.RGB(255, 255, 255)
	var (
		wWhite = [3]int{255, 255, 255}
		wRed   = [3]int{255, 0, 0}
		wGreen = [3]int{0, 255, 0}
		wBlue  = [3]int{0, 0, 255}
	)

	// ---- 1. FillRoundedRect 切角 ----------------------------------------
	// 判据两条：四角必须是底色（角被切掉了），四边中点必须是填充色
	// （图确实画了、没有整块丢失）。只看其中一条都区分不出「没画」和「没切」。
	c.Clear(white)
	c.FillRoundedRect(20, 20, 160, 160, 24, renderer.RGB(255, 0, 0))
	corners := near(22, 22, wWhite) && near(178, 22, wWhite) && near(22, 178, wWhite) && near(178, 178, wWhite)
	edges := near(100, 21, wRed) && near(100, 179, wRed) && near(21, 100, wRed) && near(179, 100, wRed)
	aaFill, aaFillDetail := scanPartial(pix, isWhite, func(p [3]int) bool { return isSame(p, wRed) })
	probeCheck("FillRoundedRect 切角", corners && edges,
		fmt.Sprintf("角(22,22)=%v 边(100,21)=%v 抗锯齿=%d %s", pix(22, 22), pix(100, 21), aaFill, aaFillDetail))

	// ---- 2. DrawImage：解码 + 缩放 --------------------------------------
	img := solidPNG(64, 64, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	c.Clear(white)
	c.DrawImage(img, 20, 20, 160, 160)
	probeCheck("DrawImage 解码并缩放", near(24, 24, wGreen) && near(176, 176, wGreen) && near(100, 100, wGreen),
		fmt.Sprintf("(24,24)=%v (100,100)=%v", pix(24, 24), pix(100, 100)))

	// ---- 3. DrawImageRounded：圆角图片 ----------------------------------
	c.Clear(white)
	c.DrawImageRounded(img, 20, 20, 160, 160, 24)
	rcorners := near(24, 24, wWhite) && near(176, 24, wWhite) && near(24, 176, wWhite) && near(176, 176, wWhite)
	redges := near(100, 24, wGreen) && near(100, 176, wGreen) && near(24, 100, wGreen) && near(176, 100, wGreen)
	probeCheck("DrawImageRounded 切角", rcorners && redges,
		fmt.Sprintf("角(24,24)=%v 边(100,24)=%v", pix(24, 24), pix(100, 24)))
	// 圆角边缘要抗锯齿：沿第 23 行扫（见 scanPartial 的说明），硬切的话
	// 一个过渡像素都找不到。这条断言同时是「clip 有抗锯齿」的回归哨兵——
	// DrawImageRounded 就是靠 clip 切角的。
	aaImg, aaImgDetail := scanPartial(pix, isWhite, isGreen)
	probeCheck("DrawImageRounded 边缘抗锯齿", aaImg >= 1,
		fmt.Sprintf("过渡像素 %d 个 %s", aaImg, aaImgDetail))

	// ---- 4. 透明像素不泄漏 RGB -------------------------------------------
	// 上游给的「假透明」最常见形态是 alpha=0 但 RGB 里还留着底色（chroma key 的
	// 品红）。解码或合成只要有一步丢了 alpha、或预乘搞反，那层底色就漏出来 ——
	// 用户看到一块全仓库都找不到的纯色，很容易误判成「模型画的」。
	//
	// 三档 alpha 各挡一类 bug：
	//   A=0   丢 alpha / 预乘反 → 露品红；正确是完全看不见（白底）
	//   A=128 按不透明处理     → 饱和品红；正确是与背景混过的中间色
	//   A=255 正常路径         → 必须是原色（否则解码本身就错了）
	c.Clear(white)
	bands := alphaBandsPNG(90, 30, color.NRGBA{R: 250, G: 2, B: 250, A: 255})
	c.DrawImage(bands, 20, 20, 180, 60)
	p0, p128, p255 := pix(50, 50), pix(110, 50), pix(170, 50)
	clearOK := isWhite(p0)
	// 品红(250,2,250) 以 128/255 压在白底上，绿通道 = 2*0.502+255*0.498 ≈ 128。
	halfOK := p128[1] > 118 && p128[1] < 142
	opaqueOK := isSame(p255, [3]int{250, 2, 250})
	probeCheck("透明像素不泄漏 RGB", clearOK && halfOK && opaqueOK,
		fmt.Sprintf("A=0→%v(须白) A=128→%v(绿通道须≈128) A=255→%v", p0, p128, p255))

	// ---- 5. DrawPixels：近邻而不是线性 ----------------------------------
	// 2x2 棋盘放大 80 倍。判据是「边界两侧都是纯色」——线性插值会在交界
	// 处生出一条紫色过渡带，近邻不会。
	buf := make([]byte, 2*2*4)
	put := func(i int, r, g, b byte) {
		buf[i*4], buf[i*4+1], buf[i*4+2], buf[i*4+3] = r, g, b, 255
	}
	put(0, 255, 0, 0)
	put(1, 0, 0, 255)
	put(2, 0, 0, 255)
	put(3, 255, 0, 0)
	c.Clear(white)
	c.DrawPixels(buf, 2, 2, 20, 20, 160, 160)
	quads := near(40, 40, wRed) && near(120, 40, wBlue) && near(40, 120, wBlue) && near(120, 120, wRed)
	pure := true
	for x := 90; x <= 110; x++ {
		p := pix(x, 60)
		if !isSame(p, wRed) && !isSame(p, wBlue) {
			pure = false
		}
	}
	probeCheck("DrawPixels 近邻放大", quads && pure,
		fmt.Sprintf("(40,40)=%v (120,40)=%v 交界纯色=%v", pix(40, 40), pix(120, 40), pure))

	// ---- 6. 文本折行 / 度量 ---------------------------------------------
	// 折行必须与 ui/components 的 inputLayout.build 同一套规则（逐 rune
	// 累加步进），这里量的是「结果确实折了、且每行不超宽」。
	st := renderer.TextStyle{FontFamily: "Segoe UI", FontSize: 13, Color: renderer.RGB(0, 0, 0)}
	text := "这是一段用来验证折行的中文文本，长度需要超过一行宽度才会折成两行显示。"
	_, lineH := c.MeasureText("x", 0, st)
	tw, th := c.MeasureText(text, 160, st)
	lineCount := 0
	if lineH > 0 {
		lineCount = int(th/lineH + 0.5)
	}
	probeCheck("MeasureText 折行", lineH > 0 && lineCount >= 2 && tw <= 161,
		fmt.Sprintf("lineH=%.1f 行数=%d 最宽=%.1f", lineH, lineCount, tw))

	c.Clear(white)
	c.DrawText(text, 20, 20, 160, 0, st)
	lh := int(lineH + 0.5)
	if lh < 1 {
		lh = 1
	}
	ink1 := inkIn(20, 20, 180, 20+lh)
	ink2 := inkIn(20, 20+lh, 180, 20+2*lh)
	probeCheck("DrawText 两行都有字", ink1 > 0 && ink2 > 0, fmt.Sprintf("首行墨点=%d 次行墨点=%d", ink1, ink2))

	// ---- 7. VAlign 居中 --------------------------------------------------
	// button / select / empty / image 都靠后端的 VAlign=Center 把文字摆在控件正中，
	// 传的是完整盒子（y=n.Y、maxH=n.H）。这条把契约钉死：在 [y, y+maxH] 里居中，
	// 不是「在 [y, y+行高] 里居中」，也不是顶对齐。
	//
	// 判据用差分，别猜「墨迹中心应该落在哪」：字形在行盒里天然不对称
	// （fontBoundingBoxAscent 甚至可能大于行高），比墨迹中心会稳定差几个像素。
	// 位移量才是精确值：(maxH - 行高)/2，两个数都取自后端自己的度量。
	//
	// 单列一条是因为编辑框（ui/components/textedit_paint.go）传错过盒子（y=n.Y 配
	// maxH=n.H-2*pad），文字整体偏高 6px —— 调用方的错，但这个契约当时没有任何测试
	// 钉着，错了很久没人发现。
	c.Clear(white)
	vert := renderer.TextStyle{
		FontFamily: "sans-serif", FontSize: 16, Color: renderer.RGB(0, 0, 0), NoWrap: true,
	}
	_, vertLineH := c.MeasureText("Ag", 0, vert)

	const boxY, boxH = 40, 100
	vert.VAlign = renderer.AlignStart
	c.DrawText("Ag", 20, boxY, 160, boxH, vert)
	topTop, _ := inkRows(inkIn, 20, 180, boxY-20, boxY+boxH+20)
	c.Clear(white)
	vert.VAlign = renderer.AlignCenter
	c.DrawText("Ag", 20, boxY, 160, boxH, vert)
	midTop, _ := inkRows(inkIn, 20, 180, boxY-20, boxY+boxH+20)

	wantShift := float64(boxH-vertLineH) / 2
	gotShift := float64(midTop - topTop)
	diff := gotShift - wantShift
	if diff < 0 {
		diff = -diff
	}
	probeCheck("VAlign 居中", topTop >= 0 && midTop >= 0 && diff <= 1,
		fmt.Sprintf("行高 %.1f 应下移 %.1f；顶对齐首行墨迹 y=%d、居中 y=%d，实际下移 %.1f",
			vertLineH, wantShift, topTop, midTop, gotShift))

	// ---- 8. PushClip / PopClip ------------------------------------------
	c.Clear(white)
	c.PushClip(50, 50, 40, 40)
	c.FillRect(0, 0, 200, 200, renderer.RGB(255, 0, 0))
	c.PopClip()
	clipped := near(10, 10, wWhite) && near(70, 70, wRed)
	c.FillRect(0, 0, 10, 10, renderer.RGB(0, 0, 255))
	restored := near(5, 5, wBlue)
	probeCheck("PushClip/PopClip", clipped && restored,
		fmt.Sprintf("裁剪外(10,10)=%v 裁剪内(70,70)=%v 弹栈后(5,5)=%v", pix(10, 10), pix(70, 70), pix(5, 5)))

	// ---- 9. Clear 不受残留裁剪影响 --------------------------------------
	// 组件漏掉 PopClip 时，下一帧的整屏 Clear 必须照样铺满，
	// 否则界面会停在半截旧画面上。
	c.Clear(white)
	c.FillRect(0, 0, 200, 200, renderer.RGB(255, 0, 0))
	c.PushClip(50, 50, 40, 40)
	c.Clear(renderer.RGB(255, 255, 255))
	c.PopClip()
	probeCheck("Clear 忽略残留裁剪", near(10, 10, wWhite) && near(100, 100, wWhite),
		fmt.Sprintf("(10,10)=%v (100,100)=%v", pix(10, 10), pix(100, 100)))

	// ---- 10. DPI 缩放 -----------------------------------------------------
	// 对外坐标是 DIP，进 canvas 之前乘 devicePixelRatio。这里直接给一个
	// 2 倍的后备缓冲：DIP(10,10)-(30,30) 应该落在设备像素 (20,20)-(60,60)。
	c.resize(400, 400, 200, 200, 192)
	c.Clear(white)
	c.FillRect(10, 10, 20, 20, renderer.RGB(255, 0, 0))
	dpiOK := near(18, 18, wWhite) && near(22, 22, wRed) && near(58, 58, wRed) &&
		near(62, 62, wWhite) && c.Width() == 200 && c.DPI() == 192
	probeCheck("DPI 2 倍缩放", dpiOK,
		fmt.Sprintf("外(18,18)=%v 内(22,22)=%v 内(58,58)=%v 外(62,62)=%v Width=%.0f DPI=%.0f",
			pix(18, 18), pix(22, 22), pix(58, 58), pix(62, 62), c.Width(), c.DPI()))
}

// ===== 第二轮：宿主事件 ================================================

// hostProbe 记录宿主回调收到的东西，用来断言「DOM 事件 → 平台回调」这条链路。
type hostProbe struct {
	paints   int
	pointers []string
	keys     []string
	wheels   []string
	comps    []string
	caret    int
}

func (p *hostProbe) last(list []string) string {
	if len(list) == 0 {
		return "(空)"
	}
	return list[len(list)-1]
}

// hostProbeW / hostProbeH 是自检画布的固定 CSS 尺寸。固定下来是为了让
// 「客户端坐标 == 画布内坐标」，合成事件里的坐标才可以直接断言。
const (
	hostProbeW = 400
	hostProbeH = 300
)

func runHostProbe() {
	doc := js.Global().Get("document")
	canvas := doc.Call("createElement", "canvas")
	canvas.Set("id", "host-probe-canvas")
	st := canvas.Get("style")
	st.Set("position", "fixed")
	st.Set("left", "0px")
	st.Set("top", "0px")
	st.Set("width", fmt.Sprintf("%dpx", hostProbeW))
	st.Set("height", fmt.Sprintf("%dpx", hostProbeH))
	st.Set("zIndex", "-1")
	doc.Get("body").Call("appendChild", canvas)

	p := &hostProbe{}
	var errs []string

	// 先接管 requestAnimationFrame，再启宿主——宿主启动时排的那一帧必须落进
	// shim 里。晚一步的话它会走真正的 rAF，在无头环境下要等虚拟时钟快耗尽
	// 才回来（实测 15s 预算里只产出 3 帧，且全在 t≈14.99s）。
	shim := installRAFShim()
	defer shim.restore()

	h, err := start(Options{
		Title:    "webprobe",
		Width:    hostProbeW,
		Height:   hostProbeH,
		CanvasID: "host-probe-canvas",
		Paint: func(ctx renderer.Context) {
			p.paints++
			// 左半红、右半绿：既能证明 Paint 真的跑了，也能证明画布尺寸对。
			ctx.Clear(renderer.RGB(255, 255, 255))
			ctx.FillRect(0, 0, hostProbeW/2, 40, renderer.RGB(255, 0, 0))
			ctx.FillRect(hostProbeW/2, 0, hostProbeW/2, 40, renderer.RGB(0, 128, 0))
		},
		Pointer: func(x, y float32, down, click bool) {
			p.pointers = append(p.pointers, fmt.Sprintf("%.0f,%.0f,%v,%v", x, y, down, click))
		},
		Key: func(key int, char rune, down, ctrl, shift, alt bool) {
			p.keys = append(p.keys, fmt.Sprintf("key=%#x char=%q down=%v ctrl=%v shift=%v alt=%v",
				key, char, down, ctrl, shift, alt))
		},
		Wheel: func(x, y, delta float32) {
			p.wheels = append(p.wheels, fmt.Sprintf("%.0f,%.0f,%.2f", x, y, delta))
		},
		Composition: func(text string, cursor int) bool {
			p.comps = append(p.comps, fmt.Sprintf("%q@%d", text, cursor))
			return true
		},
		Caret: func() (float32, float32, float32, float32, bool) {
			p.caret++
			return 12, 34, 1, 16, true
		},
	})
	if err != nil {
		probeCheck("启动 web 宿主", false, err.Error())
		probeReport()
		return
	}
	// 合成事件是 untrusted 的，出错会被 JS 抛回 wasm（进而 panic），
	// 这里把 window 上的错误接住，避免整页直接白掉、看不到结论。
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		errs = append(errs, args[0].Get("message").String())
		return nil
	})
	js.Global().Call("addEventListener", "error", catch)
	js.Global().Call("addEventListener", "unhandledrejection", catch)

	// 宿主启动时排的第一帧，推完再灌事件。
	shim.flush()
	runHostChecks(p, h, canvas, errs, shim)
}

// rafShim 接管 requestAnimationFrame，把回调攒起来由探针自己推。
//
// 为什么非要这样：无头 Chrome 的 --virtual-time-budget 只在预算快耗尽时突击产出
// 几帧，拿 rAF 当同步点只能得到「等不到帧」或「帧数看运气」。这个后端自检最早的三
// 项假失败（宿主首帧 0 次 Paint、画布全黑、caret 0 次）与后来的「requestFrame 排出
// 新帧：Paint 1 → 1」根子都在这里。两条弯路都别走：用 setTimeout 当「等一会儿」
// （虚拟时钟把定时器快进掉，而 rAF 要等真正的合成帧，断言会在首帧还没来时开跑）、
// 或再加一个「超时就先跑」兜底（同一个竞态换个位置，实测约 1/4 概率假失败）。
//
// 接管之后帧的推进全由探针掌握，宿主注册了几个回调、Paint 调了几次都是确定性的；
// 唯一没覆盖的是「浏览器真的会调 rAF 回调」——那是规范保证的，dist/index.html 的
// 应用级检查（真 rAF + 截图）已经证明帧确实在产。
type rafShim struct {
	pending []js.Value // 攒下的回调，一次 flush 就是浏览器产出一帧
	calls   int        // requestAnimationFrame 被调用次数
	frames  int        // 推过的帧数
	orig    js.Value
}

func installRAFShim() *rafShim {
	s := &rafShim{orig: js.Global().Get("requestAnimationFrame")}
	js.Global().Set("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) any {
		s.calls++
		if len(args) > 0 {
			s.pending = append(s.pending, args[0])
		}
		return s.calls
	}))
	return s
}

// flush 把当前攒下的回调跑一遍，相当于浏览器产出一帧。返回跑掉的回调数。
//
// 先取走队列再逐个调用：回调自己可能又排一帧（比如 host.frame 之后的
// requestFrame），那些要留到下一次 flush，否则会无限递归下去。
func (s *rafShim) flush() int {
	q := s.pending
	s.pending = nil
	s.frames++
	ts := js.Global().Get("performance").Call("now")
	for _, cb := range q {
		cb.Invoke(ts)
	}
	return len(q)
}

func (s *rafShim) restore() {
	js.Global().Set("requestAnimationFrame", s.orig)
}

func runHostChecks(p *hostProbe, h *host, canvas js.Value, errs []string, shim *rafShim) {
	doc := js.Global().Get("document")
	win := js.Global()
	dispatch := func(target js.Value, typ string, init map[string]any) {
		o := win.Get("Object").New()
		o.Set("bubbles", true)
		o.Set("cancelable", true)
		for k, v := range init {
			o.Set(k, v)
		}
		target.Call("dispatchEvent", win.Get(eventCtor(typ)).New(typ, o))
	}

	// ---- 9. 首帧与尺寸 ---------------------------------------------------
	// 宿主启动时应当已经排了一帧，探针在 runHostProbe 里把它推完了，
	// 所以这里是「已经画过且只画过一次」的确定性断言。
	probeCheck("宿主启动即排帧", shim.calls >= 1,
		fmt.Sprintf("requestAnimationFrame 调用 %d 次", shim.calls))
	probeCheck("宿主首帧", p.paints == 1, fmt.Sprintf("Paint 调用 %d 次", p.paints))
	first := pixelAt(canvas, 10, 10)
	second := pixelAt(canvas, hostProbeW-10, 10)
	probeCheck("画布尺寸与内容", isSame(first, [3]int{255, 0, 0}) && isSame(second, [3]int{0, 128, 0}),
		fmt.Sprintf("左(10,10)=%v 右(%d,10)=%v", first, hostProbeW-10, second))

	// ---- 10. 指针：按下 / 移动 / 抬起 ------------------------------------
	// 画布贴在 (0,0)、尺寸固定，所以客户端坐标就是画布内坐标。
	dispatch(canvas, "pointerdown", map[string]any{"clientX": 150, "clientY": 30, "button": 0})
	probeCheck("pointerdown 上报", p.last(p.pointers) == "150,30,true,false", p.last(p.pointers))

	// 焦点必须落到隐藏 textarea 上：输入法只在可编辑元素上触发 composition
	// 事件，焦点跑到 canvas 上就等于中文输入废了。
	active := doc.Get("activeElement")
	focused := active.Truthy() && active.Call("hasAttribute", "autocomplete").Bool() &&
		active.Get("tagName").String() == "TEXTAREA"
	probeCheck("pointerdown 交接焦点", focused,
		fmt.Sprintf("activeElement=%s", tagName(active)))

	dispatch(win, "pointermove", map[string]any{"clientX": 200, "clientY": 40})
	probeCheck("pointermove 上报", p.last(p.pointers) == "200,40,true,false", p.last(p.pointers))

	dispatch(win, "pointerup", map[string]any{"clientX": 200, "clientY": 40, "button": 0})
	probeCheck("pointerup 上报（click=抬起前按下）", p.last(p.pointers) == "200,40,false,true", p.last(p.pointers))

	// 抬起之后再移动：down 必须是 false，应用靠它区分悬停与拖拽。
	dispatch(win, "pointermove", map[string]any{"clientX": 210, "clientY": 50})
	probeCheck("pointerup 后 down=false", p.last(p.pointers) == "210,50,false,false", p.last(p.pointers))

	// 移出画布要报 (-1,-1)：控件靠它清 hover。
	dispatch(canvas, "pointerleave", map[string]any{"clientX": 500, "clientY": 500})
	probeCheck("pointerleave 上报 (-1,-1)", p.last(p.pointers) == "-1,-1,false,false", p.last(p.pointers))

	// ---- 11. 滚轮 --------------------------------------------------------
	// deltaMode=0（像素）下一格算 100px，且向上为正（与 windows 的
	// WM_MOUSEWHEEL 一致），所以 deltaY=+100 要报成 -1。
	dispatch(canvas, "wheel", map[string]any{"clientX": 50, "clientY": 20, "deltaY": 100, "deltaMode": 0})
	probeCheck("wheel 像素模式换算成格", p.last(p.wheels) == "50,20,-1.00", p.last(p.wheels))
	dispatch(canvas, "wheel", map[string]any{"clientX": 50, "clientY": 20, "deltaY": -300, "deltaMode": 0})
	probeCheck("wheel 向上滚为负", p.last(p.wheels) == "50,20,3.00", p.last(p.wheels))

	// ---- 12. 键盘 --------------------------------------------------------
	// 可打印键一次带出 VK 与字符（对齐 Windows 的 WM_KEYDOWN + WM_CHAR）。
	dispatch(win, "keydown", map[string]any{"key": "a", "code": "KeyA"})
	probeCheck("keydown 字母带字符", p.last(p.keys) == `key=0x41 char='a' down=true ctrl=false shift=false alt=false`, p.last(p.keys))
	dispatch(win, "keyup", map[string]any{"key": "a", "code": "KeyA"})
	probeCheck("keyup 无字符", p.last(p.keys) == `key=0x41 char='\x00' down=false ctrl=false shift=false alt=false`, p.last(p.keys))

	// 带修饰键时不能带字符：否则 handleEditKey 会把它当文本插进去，
	// Ctrl+A 就变成了「插入字母 a」。
	dispatch(win, "keydown", map[string]any{"key": "a", "code": "KeyA", "ctrlKey": true})
	probeCheck("Ctrl+A 不带字符", strings.Contains(p.last(p.keys), "ctrl=true") && strings.Contains(p.last(p.keys), `char='\x00'`), p.last(p.keys))

	dispatch(win, "keydown", map[string]any{"key": " ", "code": "Space"})
	probeCheck("空格算字符", strings.Contains(p.last(p.keys), `key=0x20 char=' '`), p.last(p.keys))

	dispatch(win, "keydown", map[string]any{"key": "Enter", "code": "Enter"})
	probeCheck("回车走 VK 不带字符", strings.Contains(p.last(p.keys), `key=0xd char='\x00'`), p.last(p.keys))

	// Ctrl+V 刻意不转发：浏览器里同步读剪贴板做不到，粘贴改由 paste 事件
	// 当「上屏文本」送进来。这里断言它确实没进引擎。
	before := len(p.keys)
	dispatch(win, "keydown", map[string]any{"key": "v", "code": "KeyV", "ctrlKey": true})
	probeCheck("Ctrl+V 不转发给引擎", len(p.keys) == before, fmt.Sprintf("keys %d → %d", before, len(p.keys)))

	// ---- 13. 输入法 ------------------------------------------------------
	dispatch(win, "compositionstart", map[string]any{"data": ""})
	dispatch(win, "compositionupdate", map[string]any{"data": "ni"})
	probeCheck("compositionupdate 报组合串", p.last(p.comps) == `"ni"@2`, p.last(p.comps))
	dispatch(win, "compositionend", map[string]any{"data": "你"})
	probeCheck("compositionend 清空组合串", p.last(p.comps) == `""@0`, p.last(p.comps))
	probeCheck("compositionend 上屏字符", p.last(p.keys) == `key=0x0 char='你' down=true ctrl=false shift=false alt=false`, p.last(p.keys))

	// ---- 14. 粘贴 --------------------------------------------------------
	// paste 事件是浏览器里唯一「不需要权限、同步拿到内容」的读法。
	clipboardPaste(win, doc, "粘")
	probeCheck("paste 上屏文本", p.last(p.keys) == `key=0x0 char='粘' down=true ctrl=false shift=false alt=false`, p.last(p.keys))

	// ---- 15. 重绘请求 ----------------------------------------------------
	// 上面那串事件每个都会 requestFrame，先推完把 pending 清掉，
	// 否则下一步测「requestFrame 排不排帧」会被 pending 短路掉。
	eventPaints := p.paints
	shim.flush()
	probeCheck("事件排出的帧真的画了", p.paints > eventPaints,
		fmt.Sprintf("Paint %d → %d", eventPaints, p.paints))

	// 光标闪烁、加载动画这类后台循环全靠 platform.RequestFrame 驱动。
	// 断言它排出的帧数不多不少：每次调用恰好一帧，重复调用不叠加。
	beforeCalls, beforePaint := shim.calls, p.paints
	h.requestFrame()
	probeCheck("requestFrame 排出一帧", shim.calls == beforeCalls+1,
		fmt.Sprintf("rAF %d → %d", beforeCalls, shim.calls))
	mid := shim.calls
	h.requestFrame()
	probeCheck("重复 requestFrame 不重复排帧", shim.calls == mid,
		fmt.Sprintf("rAF %d → %d", mid, shim.calls))
	shim.flush()
	probeCheck("排出的帧真的画了", p.paints == beforePaint+1,
		fmt.Sprintf("Paint %d → %d", beforePaint, p.paints))
	probeCheck("caret 回调被问到", p.caret >= 1, fmt.Sprintf("Caret 调用 %d 次", p.caret))
	probeCheck("合成事件没有抛异常", len(errs) == 0, strings.Join(errs, " | "))
	probeReport()
}

// eventCtor 把事件类型映射到对应的构造函数名。
func eventCtor(typ string) string {
	switch typ {
	case "pointerdown", "pointermove", "pointerup", "pointerleave":
		return "PointerEvent"
	case "wheel":
		return "WheelEvent"
	case "compositionstart", "compositionupdate", "compositionend":
		return "CompositionEvent"
	default:
		return "KeyboardEvent"
	}
}

func pixelAt(canvas js.Value, x, y int) [3]int {
	d := canvas.Call("getContext", "2d").Call("getImageData", x, y, 1, 1).Get("data")
	return [3]int{d.Index(0).Int(), d.Index(1).Int(), d.Index(2).Int()}
}

func willReadFrequently() js.Value {
	o := js.Global().Get("Object").New()
	o.Set("willReadFrequently", true)
	return o
}

func tagName(v js.Value) string {
	if !v.Truthy() {
		return "(无)"
	}
	return v.Get("tagName").String()
}

// clipboardPaste 造一个带 clipboardData 的 paste 事件。
//
// 别用 new Event("paste", {clipboardData: dt})：Event 构造函数只认 bubbles /
// cancelable / composed，别的字段直接丢掉，宿主读到的 clipboardData 是 undefined，
// onPaste 第二个 if 就返回了 —— 探针会报「粘贴没上屏」，而问题在探针这边。
//
// 正规做法是 ClipboardEvent（init 字典里有 clipboardData）；个别浏览器不给这个构造
// 函数、或给了但忽略该字段，再用 defineProperty 补一刀。
func clipboardPaste(win, doc js.Value, text string) {
	dt := newDataTransfer(win, text)
	o := win.Get("Object").New()
	o.Set("bubbles", true)
	o.Set("cancelable", true)
	o.Set("clipboardData", dt)
	var e js.Value
	if ctor := win.Get("ClipboardEvent"); ctor.Truthy() {
		e = ctor.New("paste", o)
	}
	if !e.Truthy() {
		e = win.Get("Event").New("paste", o)
	}
	if !e.Get("clipboardData").Truthy() {
		defineValue(win, e, "clipboardData", dt)
	}
	doc.Get("body").Call("dispatchEvent", e)
}

// newDataTransfer 优先用真的 DataTransfer（浏览器里可以 new，也能 setData），
// 拿不到就拼一个只有 getData 的对象——宿主只调 getData 这一个方法。
func newDataTransfer(win js.Value, text string) js.Value {
	if ctor := win.Get("DataTransfer"); ctor.Truthy() {
		if dt := ctor.New(); dt.Truthy() {
			dt.Call("setData", "text/plain", text)
			return dt
		}
	}
	dt := win.Get("Object").New()
	dt.Set("getData", js.FuncOf(func(this js.Value, args []js.Value) any {
		return text
	}))
	return dt
}

// defineValue 给一个已有对象补一个属性。用在「构造函数吃掉了 init 字段」的
// 场景：clipboardData 在 Event 上本来不存在，defineProperty 加的是普通自有
// 属性，宿主读起来和真的没区别。
func defineValue(win, obj js.Value, key string, val js.Value) {
	desc := win.Get("Object").New()
	desc.Set("value", val)
	win.Get("Object").Call("defineProperty", obj, key, desc)
}

func solidPNG(w, h int, col color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = col.R, col.G, col.B, col.A
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func isSame(p, want [3]int) bool {
	return iabs(p[0]-want[0]) <= 2 && iabs(p[1]-want[1]) <= 2 && iabs(p[2]-want[2]) <= 2
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// inkRows 找出 [y0,y1) 里有墨迹的首行与末行（都没有则返回 -1, -1）。
//
// 逐行调 getImageData 而不是一次取一大块：调用次数多一点，但「哪一行有墨迹」
// 直接就是答案，不用在字节数组里自己数行。
func inkRows(inkIn func(x0, y0, x1, y1 int) int, x0, x1, y0, y1 int) (int, int) {
	top, bot := -1, -1
	for y := y0; y < y1; y++ {
		if inkIn(x0, y, x1, y+1) > 0 {
			if top < 0 {
				top = y
			}
			bot = y
		}
	}
	return top, bot
}

// alphaBandsPNG 造一张「同一颜色、三档 alpha 横向并排」的图（左→右 A=0 / 128 / 255），
// 用来验证透明像素不会把底下的 RGB 泄漏出来（见「透明像素不泄漏 RGB」那条检查）。
func alphaBandsPNG(w, h int, col color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	band := w / 3
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			switch {
			case x < band:
				a = 0
			case x < 2*band:
				a = 128
			}
			img.SetNRGBA(x, y, color.NRGBA{R: col.R, G: col.G, B: col.B, A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// scanPartial 沿圆角上沿那一行扫一段，数出既不是底色也不是目标色的过渡像素
// （覆盖率在 0 与 1 之间），并把它们的值一并带出来。
//
// 扫的是 y=23、x∈[24,48]：矩形从 (20,20) 起、半径 24，第 23 行的圆弧边界落在
// x≈32.4，正好把一个像素劈成两半。不能沿对角线扫 —— 圆弧在对角线上几乎垂直于扫描
// 方向，覆盖率直接从 0 跳到 0.998，取整后还是纯色，硬边和抗锯齿读数一模一样。
func scanPartial(pix func(x, y int) [3]int, isBG, isFG func([3]int) bool) (int, string) {
	n := 0
	var detail []string
	for x := 24; x < 48; x++ {
		p := pix(x, 23)
		if !isBG(p) && !isFG(p) {
			n++
			detail = append(detail, fmt.Sprintf("(%d,23)=%v", x, p))
		}
	}
	return n, strings.Join(detail, " ")
}
