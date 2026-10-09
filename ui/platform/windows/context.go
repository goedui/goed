//go:build windows

package windows

import (
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

type formatKey struct {
	family  string
	size    float32
	spacing float32
	weight  int
	italic  bool
	align   renderer.Align
	valign renderer.Align
	nowrap bool
}

// d2dContext 把 Direct2D / DirectWrite 接到 renderer.Context。
type d2dContext struct {
	rt    com
	brush com
	// d2d 是 ID2D1Factory：画圆角图片要它建遮罩几何体（见 DrawImageRounded）。
	// 为 0 时圆角退化成直角，不会出错。
	d2d         com
	dwrite      com
	// stroke 是圆头线帽的 ID2D1StrokeStyle，懒创建，跟渲染目标一起释放。
	stroke com
	locale      []uint16
	formats     map[formatKey]com
	formatOrder []formatKey
	// families 缓存「font-family 列表 -> 挑中的族名」，避免每个新字号都把整条
	// 列表拿去查一遍系统字体集合（见 pickFamily）。
	families map[string]string
	images      map[uint64]cachedBitmap
	imageOrder  []uint64
	imageBytes  int
	shadows     map[shadowKey]com
	scratch     []byte
	width       float32
	height      float32
	dpi         float32
}

func (c *d2dContext) Width() float32  { return c.width }
func (c *d2dContext) Height() float32 { return c.height }
func (c *d2dContext) DPI() float32    { return c.dpi }

func (c *d2dContext) Clear(col renderer.Color) {
	if c.rt == 0 {
		return
	}
	color := d2d1ColorF{col.R, col.G, col.B, col.A}
	syscall.SyscallN(c.rt.slot(idxClear), uintptr(c.rt), uintptr(unsafe.Pointer(&color)))
	runtime.KeepAlive(color)
}

func (c *d2dContext) FillRect(x, y, w, h float32, col renderer.Color) {
	if c.rt == 0 || c.brush == 0 || w <= 0 || h <= 0 {
		return
	}
	c.setBrush(col)
	rc := d2d1RectF{x, y, x + w, y + h}
	syscall.SyscallN(
		c.rt.slot(idxFillRectangle),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(c.brush),
	)
	runtime.KeepAlive(rc)
}

func (c *d2dContext) DrawLine(x1, y1, x2, y2 float32, col renderer.Color, width float32) {
	if c.rt == 0 || c.brush == 0 {
		return
	}
	if width <= 0 {
		width = 1
	}
	c.setBrush(col)
	p0 := uintptr(uint64(math.Float32bits(x1)) | uint64(math.Float32bits(y1))<<32)
	p1 := uintptr(uint64(math.Float32bits(x2)) | uint64(math.Float32bits(y2))<<32)
	syscall.SyscallN(
		c.rt.slot(idxDrawLine),
		uintptr(c.rt),
		p0,
		p1,
		uintptr(c.brush),
		uintptr(math.Float32bits(width)),
		0,
	)
}

func (c *d2dContext) DrawText(text string, x, y, maxW, maxH float32, style renderer.TextStyle) {
	if c.rt == 0 || c.brush == 0 || text == "" {
		return
	}
	format := c.textFormat(style, text)
	if format == 0 {
		return
	}
	if maxW <= 0 {
		maxW = c.width
	}
	if maxH <= 0 {
		maxH = c.height
	}
	c.setBrush(style.Color)
	buf, n := utf16Buf(text)
	if n == 0 {
		return
	}
	rc := d2d1RectF{x, y, x + maxW, y + maxH}
	opts := uintptr(d2dDrawTextClip | d2dDrawTextEnableColorFont)
	syscall.SyscallN(
		c.rt.slot(idxDrawText),
		uintptr(c.rt),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(n),
		uintptr(format),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(c.brush),
		opts,
		0,
	)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(rc)
}

// FontMetrics 报告字体在给定字号下的纵向度量。
//
// 做法是拿一个非空字符串建一次文本布局，再用 **GetLineMetrics** 读第一行的
// height / baseline。不能只看 TextLayout 的 GetMetrics：那一层给的是整个布局
// 外框（多行时是所有行的并集），拿不到单行的基线 —— 而基线正是排版要的那个数。
//
// 取 "Hg" 而不是单个汉字：拉丁字母有稳定的上升部与下降部，量出来的
// height/baseline 最接近字体的设计值；单个汉字（如「国」）在多数中文字体里
// 上下都留白，会把行高量小。
func (c *d2dContext) FontMetrics(style renderer.TextStyle) (renderer.FontMetrics, bool) {
	if c.dwrite == 0 {
		return renderer.FontMetrics{}, false
	}
	format := c.textFormat(style, "Hg")
	if format == 0 {
		return renderer.FontMetrics{}, false
	}
	buf, n := utf16Buf("Hg")
	if n == 0 {
		return renderer.FontMetrics{}, false
	}
	layout, err := createTextLayout(c.dwrite, &buf[0], n, format, 1e6, 1e6)
	if err != nil {
		return renderer.FontMetrics{}, false
	}
	defer release(layout)
	var lm dwriteLineMetrics
	var count uint32
	hr, _, _ := syscall.SyscallN(
		layout.slot(idxGetLineMetrics),
		uintptr(layout),
		uintptr(unsafe.Pointer(&lm)),
		1,
		uintptr(unsafe.Pointer(&count)),
	)
	if hresult(hr).failed() || count == 0 || lm.height <= 0 {
		return renderer.FontMetrics{}, false
	}
	// baseline 是「基线到行盒顶」，descent 就是剩下的那一段。
	// height 已经含了 lineGap，所以 lineGap 单独报 0 —— 再报一遍会被重复加进行高。
	return renderer.FontMetrics{
		Ascent:  lm.baseline,
		Descent: lm.height - lm.baseline,
	}, true
}

func (c *d2dContext) MeasureText(text string, maxW float32, style renderer.TextStyle) (float32, float32) {
	if c.dwrite == 0 || text == "" {
		return 0, 0
	}
	format := c.textFormat(style, text)
	if format == 0 {
		return 0, style.FontSize
	}
	if maxW <= 0 {
		maxW = 1e6
	}
	buf, n := utf16Buf(text)
	if n == 0 {
		return 0, 0
	}
	layout, err := createTextLayout(c.dwrite, &buf[0], n, format, maxW, 1e6)
	if err != nil {
		return float32(n) * style.FontSize * 0.5, style.FontSize * 1.25
	}
	defer release(layout)
	m, err := textLayoutMetrics(layout)
	if err != nil {
		return float32(n) * style.FontSize * 0.5, style.FontSize * 1.25
	}
	return m.widthIncludingTrailingWhitespace, m.height
}

func (c *d2dContext) setBrush(col renderer.Color) {
	color := d2d1ColorF{col.R, col.G, col.B, col.A}
	syscall.SyscallN(c.brush.slot(idxBrushSetColor), uintptr(c.brush), uintptr(unsafe.Pointer(&color)))
	runtime.KeepAlive(color)
}

// systemFonts 是进程内共享的 IDWriteFontCollection。
//
// GetSystemFontCollection 返回的是跨线程只读对象，取一次就够 —— 每建一个
// textFormat 都去取一遍是白花花的开销。
var (
	systemFontsOnce sync.Once
	systemFonts     com
)

func systemFontCollection(dwrite com) com {
	systemFontsOnce.Do(func() {
		var coll com
		hr, _, _ := syscall.SyscallN(
			dwrite.slot(idxGetSystemFontCollection),
			uintptr(dwrite),
			uintptr(unsafe.Pointer(&coll)),
			0, // checkForUpdates
		)
		if hresult(hr).failed() || coll == 0 {
			return
		}
		systemFonts = coll
	})
	return systemFonts
}

// familyInstalled 报告族名在系统字体集合里存不存在。
func familyInstalled(dwrite com, name string) bool {
	coll := systemFontCollection(dwrite)
	if coll == 0 || name == "" {
		return false
	}
	namePtr := utf16Ptr(name)
	var idx uint32
	var exists int32
	hr, _, _ := syscall.SyscallN(
		coll.slot(idxFindFamilyName),
		uintptr(coll),
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&idx)),
		uintptr(unsafe.Pointer(&exists)),
	)
	runtime.KeepAlive(namePtr)
	if hresult(hr).failed() {
		return false
	}
	return exists != 0
}

// fontFamily 是按精确名字取到的 IDWriteFontFamily，供字形覆盖探测复用。
var (
	fontFamiliesMu sync.Mutex
	fontFamilies   = map[string]com{}
	glyphCoverage  = map[string]bool{}
)

// findFontFamily 按族名取 IDWriteFontFamily。取不到返回 0。
func findFontFamily(dwrite com, name string) com {
	coll := systemFontCollection(dwrite)
	if coll == 0 || name == "" {
		return 0
	}
	fontFamiliesMu.Lock()
	got, ok := fontFamilies[name]
	fontFamiliesMu.Unlock()
	if ok {
		return got
	}
	namePtr := utf16Ptr(name)
	var idx uint32
	var exists int32
	hr, _, _ := syscall.SyscallN(
		coll.slot(idxFindFamilyName),
		uintptr(coll),
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&idx)),
		uintptr(unsafe.Pointer(&exists)),
	)
	runtime.KeepAlive(namePtr)
	if hresult(hr).failed() || exists == 0 {
		return 0
	}
	var fam com
	hr, _, _ = syscall.SyscallN(
		coll.slot(idxGetFontFamily),
		uintptr(coll),
		uintptr(idx),
		uintptr(unsafe.Pointer(&fam)),
	)
	if hresult(hr).failed() || fam == 0 {
		return 0
	}
	fontFamiliesMu.Lock()
	// 并发时后到的先用已经存下的那个，避免泄漏。
	if prev, dup := fontFamilies[name]; dup {
		fontFamiliesMu.Unlock()
		release(fam)
		return prev
	}
	fontFamilies[name] = fam
	fontFamiliesMu.Unlock()
	return fam
}

// familyCoversText 报告某个字体族的常规字面能不能画出这段文字里的每一个字。
//
// 这是「font-family 按顺序取第一个装了的」在**中文字**上的关键一刀：本机装了
// Arial，但 Arial 没有汉字字形 —— 只查「族在不在」会让整页中文落到 Arial 上，
// 量宽与光栅都退回系统兜底字体，跟参考图对不上。
//
// 判定用 HasCharacter（在 IDWriteFont 上，直接问 cmap）。字面取 GetFirstMatchingFont
// 的常规权重 —— 页面极少整段用粗体，而粗体字面的覆盖范围与常规基本一致。
// 结果按「族名 + 文字片段」缓存：整页里同一串文字会被问很多次。
func familyCoversText(dwrite com, name, text string) bool {
	fam := findFontFamily(dwrite, name)
	if fam == 0 || text == "" {
		return false
	}
	fontFamiliesMu.Lock()
	if v, ok := glyphCoverage[name+"\x00"+text]; ok {
		fontFamiliesMu.Unlock()
		return v
	}
	fontFamiliesMu.Unlock()

	var font com
	hr, _, _ := syscall.SyscallN(
		fam.slot(idxGetFirstMatchingFont),
		uintptr(fam),
		uintptr(dwriteFontWeightNormal),
		uintptr(dwriteFontStretchNormal),
		uintptr(dwriteFontStyleNormal),
		uintptr(unsafe.Pointer(&font)),
	)
	if hresult(hr).failed() || font == 0 {
		return false
	}
	defer release(font)
	ok := true
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r < 0x20 {
			continue
		}
		var has int32
		hr, _, _ := syscall.SyscallN(
			font.slot(idxFontHasCharacter),
			uintptr(font),
			uintptr(r),
			uintptr(unsafe.Pointer(&has)),
		)
		if hresult(hr).failed() || has == 0 {
			ok = false
			break
		}
	}
	fontFamiliesMu.Lock()
	if len(glyphCoverage) > 512 {
		glyphCoverage = make(map[string]bool, 512)
	}
	glyphCoverage[name+"\x00"+text] = ok
	fontFamiliesMu.Unlock()
	return ok
}

// pickFamily 从 CSS 的 font-family 列表里挑出第一个**画得出这段文字**的族。
//
// CreateTextFormat 只认**一个**族名，而且族不存在时**不报错** —— 它会静默换成
// 系统默认字体，度量与绘制一起变。所以「取列表第一个」是不行的。
//
// 两道筛选都要有：
//  1. 族装没装（familyInstalled）—— `"PingFang SC","Helvetica Neue"` 两台机器上
//     一个是「没装」一个是「装了」；
//  2. 装了能不能画出这段字（familyCoversText）—— 本机装了 Arial，但 Arial 没有
//     汉字，`Arial,sans-serif` 的中文段落必须继续往下走到主题字体上去。
//
// 这也正是浏览器「按顺序取第一个装了的」那条规则的实现位置：CSS 层只负责把
// 通用族换成具体族名（见 browser/src/css 的 resolveFamily），能不能用只有
// 平台层知道。
func pickFamily(dwrite com, list, fallback, text string) string {
	for _, f := range strings.Split(list, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !familyInstalled(dwrite, f) {
			continue
		}
		if text != "" && !familyCoversText(dwrite, f, text) {
			continue
		}
		return f
	}
	return fallback
}

func (c *d2dContext) textFormat(style renderer.TextStyle, text string) com {
	family := style.FontFamily
	if family == "" {
		family = theme.Default.FontFamily
	}
	if strings.Contains(family, ",") {
		// 只按「列表 + 文字片段」解析一次：同一串 font-family 在整页里出现成百上千次。
		// 带文字是因为要判字形覆盖（Arial 装是装了，但没有汉字）。
		if c.families == nil {
			c.families = make(map[string]string, 8)
		}
		key := family + "\x00" + text
		picked, ok := c.families[key]
		if !ok {
			picked = pickFamily(c.dwrite, family, theme.Default.FontFamily, text)
			c.families[key] = picked
		}
		family = picked
	}
	size := quantizeFontSize(style.FontSize)
	if size <= 0 {
		size = theme.Default.FontSize
	}
	weight := style.Weight
	if weight <= 0 {
		weight = dwriteFontWeightNormal
	}
	key := formatKey{
		family:  family,
		size:    size,
		spacing: quantizeFontSize(style.LineSpacing),
		weight:  weight,
		italic:  style.Italic,
		align:   style.Align,
		valign:  style.VAlign,
		nowrap:  style.NoWrap,
	}
	if c.formats == nil {
		c.formats = make(map[formatKey]com)
	}
	if f, ok := c.formats[key]; ok {
		c.formatOrder = touchKey(c.formatOrder, key)
		return f
	}
	// locale **必须非 NULL**：DirectWrite 的 CreateTextFormat 对 localeName 传
	// NULL 会直接返回 E_INVALIDARG（0x80070057）。这条坑很安静 ——
	// 失败只体现为「format == 0」，而调用方大多把 0 当成「没字体」退回按字号
	// 估宽，于是整页文字的量宽与光栅全部走估算路径，看起来只是「字有点飘」，
	// 不会有人想到是 locale 的空指针。
	locale := []uint16{0}
	if len(c.locale) > 0 {
		locale = c.locale
	}
	// 斜体走 CreateTextFormat 的 fontStyle 参数，而不是给画布加倾斜矩阵：DirectWrite
	// 会优先取字体自带的 italic 面、没有才合成（DWRITE_FONT_SIMULATIONS_OBLIQUE），
	// 度量与光栅都自动跟上；SetTransform 倾斜则要自己反推画字原点与裁剪矩形。
	fontStyle := dwriteFontStyleNormal
	if style.Italic {
		fontStyle = dwriteFontStyleItalic
	}
	loc := &locale[0]
	if os.Getenv("TFTRACE") != "" {
		println("TF family=", family, "len=", len(c.locale), "size=", int(size), "valign=", int(style.VAlign))
	}
	f, err := createTextFormat(c.dwrite, utf16Ptr(family), size, weight, fontStyle, loc)
	if err != nil && family != theme.Default.FontFamily {
		f, err = createTextFormat(c.dwrite, utf16Ptr(theme.Default.FontFamily), size, weight, fontStyle, loc)
	}
	if err != nil {
		f, err = createTextFormat(c.dwrite, utf16Ptr("Segoe UI"), size, weight, fontStyle, loc)
	}
	if err != nil {
		return 0
	}
	// 三种水平对齐都要落到 IDWriteTextFormat 上。此前只处理了 AlignCenter，于是
	// AlignEnd 被静默忽略、文本仍从左画起 —— 右对齐的快捷键会直接压在标签上。
	switch style.Align {
	case renderer.AlignCenter:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignCenter))
	case renderer.AlignEnd:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignTrailing))
	default:
		syscall.SyscallN(f.slot(idxSetTextAlignment), uintptr(f), uintptr(dwriteTextAlignLeading))
	}
	switch style.VAlign {
	case renderer.AlignCenter:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphCenter))
	case renderer.AlignEnd:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphFar))
	default:
		syscall.SyscallN(f.slot(idxSetParagraphAlignment), uintptr(f), uintptr(dwriteParagraphNear))
	}
	if style.NoWrap {
		syscall.SyscallN(f.slot(idxSetWordWrapping), uintptr(f), uintptr(dwriteWordWrappingNoWrap))
	}
	// UNIFORM：每一行都是同一个行距，基线在行盒里的比例固定。
	// 编辑器的光标按这个行距摆，换多少行都不会和文字拉开。
	if key.spacing > 0 {
		syscall.SyscallN(f.slot(idxSetLineSpacing), uintptr(f),
			uintptr(dwriteLineSpacingUniform),
			uintptr(math.Float32bits(key.spacing)),
			uintptr(math.Float32bits(key.spacing*0.8)),
		)
	}
	c.rememberFormat(key, f)
	return f
}

func (c *d2dContext) releaseFormats() {
	for k, f := range c.formats {
		release(f)
		delete(c.formats, k)
	}
	c.formatOrder = nil
}
