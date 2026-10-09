//go:build js && wasm

// Package web 是 goed/ui 在浏览器里的宿主：Canvas 2D 渲染后端 + DOM 事件宿主。
// 应用代码只依赖 ui/platform 契约，本包是实现侧。
//
// 对外坐标一律 DIP，进 canvas 之前乘一次 devicePixelRatio（见 resize 里的
// setTransform）；文本折行必须自己算（见 wrapLines）——Canvas 只提供逐串度量。
//
// 选 Canvas 2D 而不是 WebGL / WebGPU：引擎的绘制模型就是一堆即时的 FillRect /
// DrawText / DrawImage（没有顶点也没有着色器），而文本要走浏览器自己的字体栈
// （subpixel 定位、CJK 回退、emoji 彩色字体），自己上 WebGL 得内嵌字体做 shaping。
package web

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strconv"
	"strings"
	"syscall/js"
	"unicode/utf8"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

// 解码后的图片缓存上限。缓存的意义：DrawImage 的 data 是「同一张图的字节」，
// 每帧都会原样传进来，不解码缓存就是每帧重解码一张几 MB 的 PNG。
const (
	maxCachedImages     = 24
	maxCachedImageBytes = 64 << 20
	// maxImagePixels 挡住超大图：解码成 NRGBA 是 4 字节/像素，
	// 一张 1 亿像素的图会直接吃掉 400MB。
	maxImagePixels = 32 << 20
)

// canvasContext 把 Canvas 2D 接到 renderer.Context。
type canvasContext struct {
	canvas js.Value // <canvas> 元素
	ctx    js.Value // CanvasRenderingContext2D

	// scale 是「设备像素 / DIP」，等于 devicePixelRatio。
	scale  float32
	dpi    float32
	width  float32
	height float32
	pixelW int
	pixelH int

	// clips 记录 PushClip 的深度。canvas 的裁剪栈是 save/restore，Clear 时必须先把
	// 残留的裁剪弹干净，否则整屏清不干净（组件漏掉 PopClip 时能看出来）。
	clips int

	// lastFont 避免每帧重复写 ctx.font：一次属性写就要重新解析字体串。
	lastFont string
	fonts    map[string]*fontMetrics

	images     map[uint64]*cachedImage
	imageOrder []uint64
	imageBytes int

	pix *pixelSurface
}

// fontMetrics 是一套字体串的量宽结果。key 见 fontKey。
type fontMetrics struct {
	css string
	// lineH 是自然行高，ascent 是基线到行顶。两者由 fontBoundingBox* 给出。
	// forced 表示这次度量带了强制行距：基线改用行高的 0.8，和 DirectWrite 的
	// UNIFORM baseline=spacing*0.8 对齐，光标才贴在字上。
	lineH  float32
	ascent float32
	forced bool
	// widths 是逐 rune 的步进缓存。折行要按 rune 累加（见 wrapLines），
	// 没有缓存的话一屏中文要几千次 measureText。
	widths map[rune]float32
}

// cachedImage 是一张解码好、放在离屏 canvas 上的图片。
//
// 为什么不用 createImageBitmap：它是异步的（返回 Promise），而
// renderer.Context.DrawImage 是同步调用——异步解码意味着「本帧画不出来，
// 解码完再请求一帧」，调用方每次都要接受一次闪烁。Go 的 image/png、image/jpeg、
// image/gif 都是纯 Go 实现，wasm 下照常工作，同步解码 + putImageData 到离屏
// canvas 之后，drawImage 就能同步拿到缩放结果。
type cachedImage struct {
	canvas js.Value
	w, h   int
	bytes  int
}

// pixelSurface 是 DrawPixels 用的复用缓冲。每帧都新建 Uint8ClampedArray +
// ImageData 会把 GC 压力堆到浏览器上，尺寸不变就复用同一份。
type pixelSurface struct {
	canvas js.Value
	octx   js.Value
	array  js.Value // Uint8ClampedArray
	data   js.Value // ImageData
	w, h   int
}

func newPixelSurface(pw, ph int) *pixelSurface {
	doc := js.Global().Get("document")
	c := doc.Call("createElement", "canvas")
	c.Set("width", pw)
	c.Set("height", ph)
	octx := c.Call("getContext", "2d")
	arr := js.Global().Get("Uint8ClampedArray").New(pw * ph * 4)
	data := js.Global().Get("ImageData").New(arr, pw, ph)
	return &pixelSurface{canvas: c, octx: octx, array: arr, data: data, w: pw, h: ph}
}

func newCanvasContext(canvas, ctx2d js.Value) *canvasContext {
	return &canvasContext{
		canvas: canvas,
		ctx:    ctx2d,
		scale:  1,
		dpi:    96,
		fonts:  map[string]*fontMetrics{},
		images: map[uint64]*cachedImage{},
	}
}

// resize 把 canvas 的 backing store 对齐到 CSS 尺寸 × devicePixelRatio。
//
// width / height 是 CSS 像素（DIP），pixelW / pixelH 是 backing store 的像素数。
// 设置 canvas.width / canvas.height 会把 2D 上下文的**全部状态**（变换、样式、
// 裁剪栈）重置，所以这里必须重新 setTransform。
func (c *canvasContext) resize(pixelW, pixelH int, width, height, dpi float32) {
	if pixelW < 1 {
		pixelW = 1
	}
	if pixelH < 1 {
		pixelH = 1
	}
	if dpi <= 0 {
		dpi = 96
	}
	c.pixelW, c.pixelH = pixelW, pixelH
	c.width, c.height = width, height
	c.dpi = dpi
	c.scale = dpi / 96
	if c.scale <= 0 {
		c.scale = 1
	}
	c.clips = 0
	c.lastFont = ""
	// setTransform 之外还要把上一次的变换基线钉住：save/restore 栈里的
	// 内容已经随 canvas.width 赋值清空了，这里写的就是新基线。
	c.ctx.Call("setTransform", c.scale, 0, 0, c.scale, 0, 0)
	c.ctx.Set("textBaseline", "alphabetic")
	c.ctx.Set("lineJoin", "miter")
	c.ctx.Set("lineCap", "butt")
}

func (c *canvasContext) Width() float32  { return c.width }
func (c *canvasContext) Height() float32 { return c.height }
func (c *canvasContext) DPI() float32    { return c.dpi }

// clearFontCache 丢掉字体度量缓存：webfont 加载完成之后之前量出来的宽度就不作数了，
// 必须重算——否则整屏文字会按回退字体的宽度排版。
func (c *canvasContext) clearFontCache() {
	c.fonts = map[string]*fontMetrics{}
	c.lastFont = ""
}

func (c *canvasContext) Clear(col renderer.Color) {
	// 先把残留的裁剪弹干净：canvas 的裁剪是栈式的，只要有一层没收，
	// 下面这次全屏 fillRect 就只覆盖那一小块。
	for c.clips > 0 {
		c.ctx.Call("restore")
		c.clips--
	}
	c.ctx.Call("setTransform", 1, 0, 0, 1, 0, 0)
	c.ctx.Set("fillStyle", cssColor(col))
	c.ctx.Call("fillRect", 0, 0, c.pixelW, c.pixelH)
	c.ctx.Call("setTransform", c.scale, 0, 0, c.scale, 0, 0)
}

// RotateAt 绕 (cx, cy) 旋转后续绘制（顺时针，y 向下），实现
// renderer.TransformPainter。canvas 的基准变换是 DPI 缩放（见 resize），这里在
// 它之上叠一个「DIP 空间里的旋转」：先转再缩放。canvas 的 setTransform(a,b,c,d,e,f)
// 与 D2D 的行向量矩阵一一对应（a=m11、b=m12、c=m21、d=m22、e=m31、f=m32），
// 所以两端的旋转方向天然一致。
func (c *canvasContext) RotateAt(deg, cx, cy float32) {
	if deg == 0 {
		return
	}
	s := float64(c.scale)
	sn, co := math.Sincos(float64(deg) * math.Pi / 180)
	dx := float64(cx)*(1-co) + float64(cy)*sn
	dy := float64(cy)*(1-co) - float64(cx)*sn
	c.ctx.Call("setTransform", s*co, s*sn, -s*sn, s*co, s*dx, s*dy)
}

// ResetTransform 回到 DPI 缩放基线（不是恒等矩阵 —— 那会把 DPI 缩放一并丢掉）。
func (c *canvasContext) ResetTransform() {
	c.ctx.Call("setTransform", c.scale, 0, 0, c.scale, 0, 0)
}

func (c *canvasContext) FillRect(x, y, w, h float32, col renderer.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	c.ctx.Set("fillStyle", cssColor(col))
	c.ctx.Call("fillRect", x, y, w, h)
}

func (c *canvasContext) FillRoundedRect(x, y, w, h, r float32, col renderer.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if r <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}
	// 半径超过一半时圆角矩形本身退化，与 linux 后端的钳法保持一致。
	if max := minf(w, h) / 2; r > max {
		r = max
	}
	if r <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}
	c.ctx.Set("fillStyle", cssColor(col))
	roundRectPath(c.ctx, x, y, w, h, r)
	c.ctx.Call("fill")
}

// DrawRoundedRect 用同一条圆角路径描边。线骑在边上，直边和四角同粗。
// 不走点阵拼接：那条路四角会比直边深，圆角幅度也不匀。
func (c *canvasContext) DrawRoundedRect(x, y, w, h, r float32, col renderer.Color, width float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if r <= 0 {
		c.DrawRect(x, y, w, h, col, width)
		return
	}
	if width <= 0 {
		width = 1
	}
	if max := minf(w, h) / 2; r > max {
		r = max
	}
	c.ctx.Set("strokeStyle", cssColor(col))
	c.ctx.Set("lineWidth", width)
	c.ctx.Set("lineJoin", "round")
	roundRectPath(c.ctx, x, y, w, h, r)
	c.ctx.Call("stroke")
}

// DrawRect 走 strokeRect：Direct2D 的 DrawRectangle 是**骑在路径上**描边
// （一半在矩形内、一半在外），canvas 的 stroke 也是同一套语义，两边对得上。
// linux 后端往内填四条边，本来就与 windows 不一致，这里以 windows 为准。
func (c *canvasContext) DrawRect(x, y, w, h float32, col renderer.Color, width float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if width <= 0 {
		width = 1
	}
	c.ctx.Set("strokeStyle", cssColor(col))
	c.ctx.Set("lineWidth", width)
	c.ctx.Call("strokeRect", x, y, w, h)
}

func (c *canvasContext) DrawLine(x1, y1, x2, y2 float32, col renderer.Color, width float32) {
	if width <= 0 {
		width = 1
	}
	c.ctx.Set("strokeStyle", cssColor(col))
	c.ctx.Set("lineWidth", width)
	c.ctx.Call("beginPath")
	c.ctx.Call("moveTo", x1, y1)
	c.ctx.Call("lineTo", x2, y2)
	c.ctx.Call("stroke")
}

// DrawText 与 MeasureText 共用同一套折行与字体度量（见 wrapLines / useFont）。
//
// 对齐语义对齐 windows 后端：maxW / maxH 小于等于 0 时按整个画布算；VAlign 按
// **全部行**算起始 y，再按 maxH 限高截行——DirectWrite 是「先排版、再按矩形裁」
// （居中时溢出对称），这里简化成从顶部截。
func (c *canvasContext) DrawText(text string, x, y, maxW, maxH float32, style renderer.TextStyle) {
	if text == "" {
		return
	}
	if maxW <= 0 {
		maxW = c.width
	}
	if maxH <= 0 {
		maxH = c.height
	}
	f := c.useFont(style)
	lines := c.wrapLines(text, maxW, f, style)
	lineH := f.lineH
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	totalH := float32(len(lines)) * lineH
	startY := y
	switch style.VAlign {
	case renderer.AlignCenter:
		startY = y + (maxH-totalH)/2
	case renderer.AlignEnd:
		startY = y + maxH - totalH
	}
	limit := len(lines)
	if !style.NoWrap && maxH > 0 {
		maxLines := int(maxH / lineH)
		if maxLines < 1 {
			maxLines = 1
		}
		if limit > maxLines {
			limit = maxLines
		}
	}
	c.applyTextState(f, style.Color)
	for i := 0; i < limit; i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		sx := x
		switch style.Align {
		case renderer.AlignCenter:
			sx = x + (maxW-c.lineBoxWidth(line, f))/2
		case renderer.AlignEnd:
			sx = x + maxW - c.lineBoxWidth(line, f)
		}
		// textBaseline 用 alphabetic。基线取行盒的 0.8：编辑器强制行距时
		// DirectWrite 也是 baseline = spacing*0.8，光标按这个比例画。
		// 用 fontBoundingBoxAscent 会把字顶顶到行盒顶，光标却按 0.8 停在中间。
		c.ctx.Call("fillText", line, sx, startY+float32(i)*lineH+lineBaseline(lineH, f))
	}
}

func (c *canvasContext) MeasureText(text string, maxW float32, style renderer.TextStyle) (float32, float32) {
	if text == "" {
		return 0, 0
	}
	f := c.useFont(style)
	if maxW <= 0 {
		maxW = 1e6
	}
	lines := c.wrapLines(text, maxW, f, style)
	var maxLine float32
	for _, ln := range lines {
		if w := c.lineBoxWidth(ln, f); w > maxLine {
			maxLine = w
		}
	}
	lineH := f.lineH
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	return maxLine, float32(len(lines)) * lineH
}

// lineBoxWidth 是这一行在布局里需要的宽度。
//
// wrapLines 按单字宽度累加决定断行，画字则用整串 measureText。
// canvas 上这两套数字经常差零点几个像素（kerning / 亚像素取整）。
// 只报整串宽度的话，布局给的盒子会刚好让最后一个字被折到下一行，
// 再被单行高度裁掉——状态栏「Esc 取消」就只剩「Esc 取」。
func (c *canvasContext) lineBoxWidth(line string, f *fontMetrics) float32 {
	w := c.measureLine(line, f)
	var sum float32
	for _, r := range line {
		sum += c.runeWidth(f, r)
	}
	if sum > w {
		w = sum
	}
	return float32(math.Ceil(float64(w)))
}

func (c *canvasContext) DrawImage(data []byte, x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	img := c.imageFor(data)
	if img == nil {
		return
	}
	c.ctx.Set("imageSmoothingEnabled", true)
	c.ctx.Set("imageSmoothingQuality", "high")
	c.ctx.Call("drawImage", img.canvas, x, y, w, h)
}

// DrawImageRounded 画圆角图片，实现 renderer.ImagePainter。
//
// 直接用 canvas 的裁剪路径：建一个圆角矩形路径、clip 上去、再 drawImage。
//
// 为什么不是「离屏画布 + destination-in 遮罩」那套：实测两者画出来的像素
// 一模一样（圆角边缘都有完整的覆盖率过渡，见 ui/platform/web/probe.go 的
// 「DrawImageRounded 边缘抗锯齿」一项），而遮罩那套要多维护一份按尺寸缓存的
// 离屏画布——同一张图在不同尺寸下画几次就是几份全尺寸副本，内存不值这个价。
//
// 依赖 clip 的抗锯齿是这个后端的一个前提：Chrome / Edge / Firefox 现在都做，
// 但它是规范没写死的行为，所以 probe 里那条断言就是它的回归哨兵。
func (c *canvasContext) DrawImageRounded(data []byte, x, y, w, h, r float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if r <= 0 || r*2 > w || r*2 > h {
		c.DrawImage(data, x, y, w, h)
		return
	}
	img := c.imageFor(data)
	if img == nil {
		return
	}
	c.ctx.Call("save")
	roundRectPath(c.ctx, x, y, w, h, r)
	c.ctx.Call("clip")
	c.ctx.Set("imageSmoothingEnabled", true)
	c.ctx.Set("imageSmoothingQuality", "high")
	c.ctx.Call("drawImage", img.canvas, x, y, w, h)
	c.ctx.Call("restore")
	c.lastFont = ""
}

// DrawPixels 把一块 NRGBA 近邻缩放着画上去。
//
// 近邻而不是线性：接口注释里写明这块缓冲是「游戏画面、实时可视化」用的，
// 放大时要看得见像素格。DrawImage 那条路则相反，保持线性 + high。
func (c *canvasContext) DrawPixels(pix []byte, pw, ph int, x, y, w, h float32) {
	if pw < 1 || ph < 1 || w <= 0 || h <= 0 {
		return
	}
	need := pw * ph * 4
	if len(pix) < need {
		return
	}
	if c.pix == nil || c.pix.w != pw || c.pix.h != ph {
		c.pix = newPixelSurface(pw, ph)
	}
	js.CopyBytesToJS(c.pix.array, pix[:need])
	c.pix.octx.Call("putImageData", c.pix.data, 0, 0)
	c.ctx.Set("imageSmoothingEnabled", false)
	c.ctx.Call("drawImage", c.pix.canvas, x, y, w, h)
}

func (c *canvasContext) PushClip(x, y, w, h float32) {
	// save/restore 会把 font、textBaseline 一起弹掉。弹完必须清 lastFont，
	// 否则下一笔字按缓存以为字体还在，实际落到浏览器默认的 10px 上。
	c.ctx.Call("save")
	c.ctx.Call("beginPath")
	if w <= 0 || h <= 0 {
		c.ctx.Call("rect", x, y, 0, 0)
	} else {
		c.ctx.Call("rect", x, y, w, h)
	}
	c.ctx.Call("clip")
	c.clips++
	c.lastFont = ""
}

func (c *canvasContext) PopClip() {
	if c.clips <= 0 {
		return
	}
	c.ctx.Call("restore")
	c.clips--
	c.lastFont = ""
}

// applyTextState 在每次画字前把字体状态钉死。
// PushClip 的 save/restore 会清掉 ctx.font，只靠 lastFont 缓存会画出另一套字。
func (c *canvasContext) applyTextState(f *fontMetrics, col renderer.Color) {
	if f != nil {
		c.ctx.Set("font", f.css)
		c.lastFont = f.css
	}
	c.ctx.Set("textBaseline", "alphabetic")
	c.ctx.Set("textAlign", "left")
	c.ctx.Set("fontKerning", "none")
	c.ctx.Set("letterSpacing", "0px")
	c.ctx.Set("fillStyle", cssColor(col))
}

// ===== 字体度量与折行 ==================================================

func fontKey(st renderer.TextStyle) string {
	var b strings.Builder
	b.WriteString(st.FontFamily)
	b.WriteByte(0)
	b.WriteString(strconv.FormatFloat(float64(st.FontSize), 'f', -1, 32))
	b.WriteByte(0)
	b.WriteString(strconv.Itoa(st.Weight))
	if st.Italic {
		b.WriteString("i")
	}
	return b.String()
}

// LayoutText 报告和 DrawText 同一套折行与步进，编辑框用它摆光标。
func (c *canvasContext) LayoutText(text string, maxW float32, style renderer.TextStyle) renderer.TextLayout {
	f := c.useFont(style)
	lineH := f.lineH
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	if maxW <= 0 {
		maxW = 1e6
	}
	lines := c.wrapLines(text, maxW, f, style)
	out := renderer.TextLayout{LineH: lineH, LineCount: len(lines)}
	off := 0
	for li, line := range lines {
		out.Points = append(out.Points, renderer.TextPoint{Off: off, Line: li, X: 0})
		// 前缀整串宽，和 fillText 同一套字距。单字累加会把光标越推越偏。
		runes := []rune(line)
		for i := 1; i <= len(runes); i++ {
			prefix := string(runes[:i])
			off += utf8.RuneLen(runes[i-1])
			out.Points = append(out.Points, renderer.TextPoint{
				Off: off, Line: li, X: c.measureLine(prefix, f),
			})
		}
		if li+1 < len(lines) && off < len(text) && text[off] == '\n' {
			off++
		}
	}
	if len(out.Points) == 0 {
		out.Points = []renderer.TextPoint{{Off: 0, Line: 0, X: 0}}
		out.LineCount = 1
	}
	return out
}

// useFont 取字体度量并把 ctx.font 切过去。折行、量宽、画字都依赖
// ctx.font 与传入的 style 一致——不一致的话量出来的宽度和画出来的对不上。
func (c *canvasContext) useFont(st renderer.TextStyle) *fontMetrics {
	key := fontKey(st)
	f, ok := c.fonts[key]
	if !ok {
		f = c.buildFont(st)
		c.fonts[key] = f
	}
	f.forced = st.LineSpacing > 0
	if c.lastFont != f.css {
		c.ctx.Set("font", f.css)
		c.lastFont = f.css
	}
	return f
}

func (c *canvasContext) buildFont(st renderer.TextStyle) *fontMetrics {
	css := fontCSS(st)
	f := &fontMetrics{css: css, widths: map[rune]float32{}}
	c.ctx.Set("font", css)
	c.lastFont = css
	size := st.FontSize
	if size <= 0 {
		size = theme.Default.FontSize
	}
	m := c.ctx.Call("measureText", "Mg")
	asc, okA := jsNumber(m.Get("fontBoundingBoxAscent"))
	desc, okD := jsNumber(m.Get("fontBoundingBoxDescent"))
	if !okA || asc <= 0 {
		// fontBoundingBox* 是较新的属性。取不到就退回「em 框」的常见比例，
		// 行高会略偏，但量宽/画字仍然一致，不会错位。
		asc = float64(size) * 0.8
	}
	if !okD || desc <= 0 {
		desc = float64(size) * 0.2
	}
	f.ascent = float32(asc)
	f.lineH = float32(asc + desc)
	if f.lineH <= 0 {
		f.lineH = size * 1.2
	}
	return f
}

// FontMetrics 报告行盒度量。强制行距时基线是行高的 0.8，和 DirectWrite
// UNIFORM 一致；否则用字体自己的 fontBoundingBoxAscent。
func (c *canvasContext) FontMetrics(style renderer.TextStyle) (renderer.FontMetrics, bool) {
	f := c.useFont(style)
	lineH := f.lineH
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	ascent := f.ascent
	if style.LineSpacing > 0 {
		ascent = lineH * 0.8
	}
	if ascent > lineH {
		ascent = lineH
	}
	if ascent < 0 {
		ascent = 0
	}
	return renderer.FontMetrics{Ascent: ascent, Descent: lineH - ascent}, f.lineH > 0
}

func (c *canvasContext) measureLine(line string, f *fontMetrics) float32 {
	if line == "" {
		return 0
	}
	// PushClip 的 save/restore 会把 ctx.font 弹回默认 10px。
	// 不重写的话，光标按 10px 量宽，字却按 14px 画。
	if f != nil && c.lastFont != f.css {
		c.ctx.Set("font", f.css)
		c.lastFont = f.css
	}
	return float32(c.ctx.Call("measureText", line).Get("width").Float())
}

// runeWidth 是编辑框光标用的步进。canvas 的 measureText(单字) 和
// measureText(整串) 差在字距上：密码点、拉丁字母会越积越偏。
// 用「前缀整串宽之差」才和 fillText 画出来的位置一致。
func (c *canvasContext) runeWidth(f *fontMetrics, r rune) float32 {
	if w, ok := f.widths[r]; ok {
		return w
	}
	w := c.advanceOf(f, string(r))
	f.widths[r] = w
	return w
}

func (c *canvasContext) advanceOf(f *fontMetrics, s string) float32 {
	if s == "" {
		return 0
	}
	// 夹在同一对字母中间量，字距和单独量一个字符不一样。
	// "H"+s+"H" 的增量才是这段文字在句子里的步进。
	full := c.measureLine("H"+s+"H", f)
	side := c.measureLine("HH", f)
	w := full - side
	if w < 0 {
		w = 0
	}
	return w
}

// lineBaseline 是这一行的基线，相对行顶。
// 一律用行高的 0.8：编辑器光标、DirectWrite UNIFORM 都是这个比例。
// 自然行距时 fontBoundingBoxAscent 经常大于行高，字会画出格子、和光标错开。
func lineBaseline(lineH float32, f *fontMetrics) float32 {
	if lineH > 0 {
		return lineH * 0.8
	}
	if f != nil && f.ascent > 0 {
		return f.ascent
	}
	return 0
}

// wrapLines 按宽度折行。**必须与编辑框自己的排版一致**：ui/components 的
// inputLayout.build 是「逐 rune 累加步进、超过 wrapW 就断行、不在空格处提前
// 断开」，这里用同一套规则，光标落点 / 选区高亮才会和画出来的字对齐。
//
// 反过来如果这里改用 measureText(整行) 去试宽，得到的断点会和编辑框算的
// 差一两个字——那个差异在点选文本时非常明显（点在第 3 个字符上，光标落在
// 第 5 个字符后面）。
//
// \n 是硬换行，NoWrap 只关掉软折行、不关硬换行。
func (c *canvasContext) wrapLines(text string, maxW float32, f *fontMetrics, st renderer.TextStyle) []string {
	runes := []rune(text)
	noWrap := st.NoWrap || maxW <= 0
	lines := make([]string, 0, 4)
	start := 0
	var x float32
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			lines = append(lines, string(runes[start:i]))
			start = i + 1
			x = 0
			continue
		}
		w := c.runeWidth(f, r)
		if !noWrap && x+w > maxW && i > start {
			lines = append(lines, string(runes[start:i]))
			start = i
			x = w
			continue
		}
		x += w
	}
	lines = append(lines, string(runes[start:]))
	return lines
}

// ===== 图片 ============================================================

func (c *canvasContext) imageFor(data []byte) *cachedImage {
	if len(data) == 0 {
		return nil
	}
	key := hashBytes(data)
	if img, ok := c.images[key]; ok {
		return img
	}
	img := decodeImage(data)
	if img == nil {
		return nil
	}
	c.remember(key, img)
	return img
}

func (c *canvasContext) remember(key uint64, img *cachedImage) {
	c.images[key] = img
	c.imageOrder = append(c.imageOrder, key)
	c.imageBytes += img.bytes
	for len(c.imageOrder) > maxCachedImages || (c.imageBytes > maxCachedImageBytes && len(c.imageOrder) > 1) {
		old := c.imageOrder[0]
		c.imageOrder = c.imageOrder[1:]
		if victim, ok := c.images[old]; ok {
			c.imageBytes -= victim.bytes
			delete(c.images, old)
		}
	}
}

// decodeImage 同步解码并贴到离屏 canvas 上。任何一步失败都返回 nil
// （画不出来可以，画错不行）。
func decodeImage(data []byte) *cachedImage {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 || w*h > maxImagePixels {
		return nil
	}
	// 统一转成 NRGBA：ImageData 要的正是「非预乘 RGBA」，与 image.NRGBA
	// 的内存布局逐字节一致，可以直接 memcpy。
	rgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)

	doc := js.Global().Get("document")
	off := doc.Call("createElement", "canvas")
	off.Set("width", w)
	off.Set("height", h)
	octx := off.Call("getContext", "2d")
	arr := js.Global().Get("Uint8ClampedArray").New(len(rgba.Pix))
	js.CopyBytesToJS(arr, rgba.Pix)
	octx.Call("putImageData", js.Global().Get("ImageData").New(arr, w, h), 0, 0)
	return &cachedImage{canvas: off, w: w, h: h, bytes: len(rgba.Pix)}
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}

// ===== 工具 ============================================================

// roundRectPath 用 arcTo 画圆角矩形路径。刻意不用 ctx.roundRect：
// 那个方法 Chrome 99 才有，arcTo 是初版 canvas 就有的。
func roundRectPath(ctx js.Value, x, y, w, h, r float32) {
	ctx.Call("beginPath")
	ctx.Call("moveTo", x+r, y)
	ctx.Call("lineTo", x+w-r, y)
	ctx.Call("arcTo", x+w, y, x+w, y+r, r)
	ctx.Call("lineTo", x+w, y+h-r)
	ctx.Call("arcTo", x+w, y+h, x+w-r, y+h, r)
	ctx.Call("lineTo", x+r, y+h)
	ctx.Call("arcTo", x, y+h, x, y+h-r, r)
	ctx.Call("lineTo", x, y+r)
	ctx.Call("arcTo", x, y, x+r, y, r)
	ctx.Call("closePath")
}

// fontCSS 组装 canvas 的 font 串（font-style font-weight font-size font-family）。
func fontCSS(st renderer.TextStyle) string {
	family := st.FontFamily
	if family == "" {
		family = theme.Default.FontFamily
	}
	size := st.FontSize
	if size <= 0 {
		size = theme.Default.FontSize
	}
	// 与 windows 后端一样收成 0.5 DIP 一档，避免 13.000001 这类键把度量缓存撑爆。
	size = float32(int(size*2+0.5)) / 2
	weight := st.Weight
	if weight <= 0 {
		weight = 400
	}
	var b strings.Builder
	if st.Italic {
		b.WriteString("italic ")
	}
	b.WriteString(strconv.Itoa(weight))
	b.WriteByte(' ')
	b.WriteString(strconv.FormatFloat(float64(size), 'f', -1, 32))
	b.WriteString("px ")
	b.WriteString(quoteFamily(family))
	// 主题里的字体名是 Windows 的（Segoe UI）。浏览器上必须挂一串回退，
	// 否则换台机器就掉到 serif 了。回退串不影响度量一致性：量宽与画字
	// 用的是同一个 font 串，浏览器自己会选同一套字体。
	b.WriteString(`, "Segoe UI", system-ui, -apple-system, "Helvetica Neue", Arial, "Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif`)
	return b.String()
}

// quoteFamily 给带空格的字体名加引号。已经写成字体列表（含逗号）的原样返回。
func quoteFamily(name string) string {
	if name == "" || strings.ContainsAny(name, ",") {
		return name
	}
	if strings.ContainsAny(name, "\"'") {
		return name
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == ' ' || ch == '-' || ch == '_' || (ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			continue
		}
		// 非 ASCII（中文家族名）也要引起来。
		return `"` + name + `"`
	}
	if strings.ContainsAny(name, " -") {
		return `"` + name + `"`
	}
	return name
}

func cssColor(c renderer.Color) string {
	r := clamp8(c.R * 255)
	g := clamp8(c.G * 255)
	b := clamp8(c.B * 255)
	a := c.A
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	var sb strings.Builder
	sb.Grow(24)
	if a >= 1 {
		sb.WriteString("rgb(")
		sb.WriteString(strconv.Itoa(int(r)))
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(int(g)))
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(int(b)))
		sb.WriteByte(')')
		return sb.String()
	}
	sb.WriteString("rgba(")
	sb.WriteString(strconv.Itoa(int(r)))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(int(g)))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(int(b)))
	sb.WriteByte(',')
	sb.WriteString(strconv.FormatFloat(float64(a), 'f', 3, 32))
	sb.WriteByte(')')
	return sb.String()
}

func clamp8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// jsNumber 把 js.Value 读成数字。属性不存在时 Get 返回 undefined，
// 而 Value.Float() 对非 number 会 panic，所以必须先判类型。
func jsNumber(v js.Value) (float64, bool) {
	if v.Type() != js.TypeNumber {
		return 0, false
	}
	return v.Float(), true
}
