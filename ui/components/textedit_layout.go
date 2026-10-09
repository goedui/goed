// 编辑框的布局：把文本切成可视行，并在「字节偏移 ↔ 行列坐标」之间换算。
//
// 光标上下移动、翻页、Home/End、点击落点、选区高亮、光标定位都依赖这里的
// inputLayout —— 它是布局结果，缓存在 editState.layout 里，文本或折行宽度
// 变化时失效。仅参与编辑的核心逻辑用，不含绘制。
package components

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/goedui/goed/ui/renderer"
)

// editorLineStep 是编辑器的行距，收成整数 DIP。
// 文字和光标共用这一格：字体自己的行距带小数（14px Consolas 是 17.78），
// 光标按它累加、文字按自己的行距累加，换一行就差零点几像素，越往下越偏。
func editorLineStep(fontSize, measured float32) float32 {
	if measured <= 0 {
		measured = fontSize * 1.4
	}
	if measured <= 0 {
		return 18
	}
	step := float32(math.Round(float64(measured)))
	if step < 1 {
		step = 1
	}
	return step
}

type layoutPoint struct {
	off  int
	line int
	x    float32
}

type inputLayout struct {
	text       string
	wrapW      float32
	lineH      float32
	points     []layoutPoint
	maxLine    int // 最后一个可视行的下标（0 起）
	large      bool
	visual     bool // 只记可视行起点，不建全量 points
	lineStarts []int
	// runeW 返回单 rune 的绘制宽（像素）。build 时绑到 glyphWidth 缓存上，
	// 稀疏布局靠它把「行内像素横坐标 ↔ 字节偏移」换算回来——layoutPoint.x
	// 两个分支必须同是像素口径，拖选横向滚动与点击落点才不会错位。
	runeW func(rune) float32
}

const largeLayoutBytes = 256 * 1024
const largeLineStride = 256

func layoutOf(ctx renderer.Context, st *editState, wrapW float32, style renderer.TextStyle) *inputLayout {
	return layoutText(ctx, st, st.text(), wrapW, style)
}

func layoutText(ctx renderer.Context, st *editState, text string, wrapW float32, style renderer.TextStyle) *inputLayout {
	// 行高只在缓存未命中时问后端。光标闪烁每帧都会进这里。
	if lay := cachedLayout(st, text, wrapW); lay != nil {
		return lay
	}
	if lay := layoutFromBackend(ctx, text, wrapW, style); lay != nil {
		if st != nil {
			st.layouts[1] = st.layouts[0]
			st.layouts[0] = lay
			st.layout = lay
		}
		return lay
	}
	if st == nil {
		return &inputLayout{text: text, wrapW: wrapW, lineH: editorLineStep(style.FontSize, style.FontSize*1.4)}
	}
	LayoutDebug = append(LayoutDebug, wrapW)
	_, natural := ctx.MeasureText(" ", wrapW, style)
	lineH := editorLineStep(style.FontSize, natural)
	style.LineSpacing = lineH
	if _, snapped := ctx.MeasureText(" ", wrapW, style); snapped > 0 {
		lineH = snapped
	}
	style.LineSpacing = lineH
	if st.glyphLineH != 0 && st.glyphLineH != lineH {
		st.glyphW = nil
	}
	st.glyphLineH = lineH
	if lay := cachedLayout(st, text, wrapW, lineH); lay != nil {
		return lay
	}
	lay := &inputLayout{text: text, wrapW: wrapW, lineH: lineH}
	lay.build(ctx, style, st)
	st.layouts[1] = st.layouts[0]
	st.layouts[0] = lay
	st.layout = lay
	return lay
}

// layoutFromBackend 用和 DrawText 同一套折行摆光标。
// 没有这个接口时退回按单字宽度累加，canvas 上会和画出来的字错开。
func layoutFromBackend(ctx renderer.Context, text string, wrapW float32, style renderer.TextStyle) *inputLayout {
	layouter, ok := ctx.(renderer.TextLayouter)
	if !ok || layouter == nil {
		return nil
	}
	got := layouter.LayoutText(text, wrapW, style)
	if got.LineH <= 0 || len(got.Points) == 0 {
		return nil
	}
	lineH := editorLineStep(style.FontSize, got.LineH)
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	lay := &inputLayout{
		text:    text,
		wrapW:   wrapW,
		lineH:   lineH,
		points:  make([]layoutPoint, len(got.Points)),
		maxLine: got.LineCount - 1,
	}
	if lay.maxLine < 0 {
		lay.maxLine = 0
	}
	for i, p := range got.Points {
		lay.points[i] = layoutPoint{off: p.Off, line: p.Line, x: p.X}
		if p.Line > lay.maxLine {
			lay.maxLine = p.Line
		}
	}
	return lay
}

// cachedLayout 按文本和折行宽找已有布局。
// 不传 lineH 时不比较行高，命中就不必再量一个空格。
func cachedLayout(st *editState, text string, wrapW float32, lineH ...float32) *inputLayout {
	if st == nil {
		return nil
	}
	if layoutReusable(st.layout, text, wrapW, lineH) {
		return st.layout
	}
	for _, cached := range st.layouts {
		if layoutReusable(cached, text, wrapW, lineH) {
			st.layout = cached
			return cached
		}
	}
	return nil
}

func layoutReusable(lay *inputLayout, text string, wrapW float32, lineH []float32) bool {
	if lay == nil || lay.text != text || lay.wrapW != wrapW {
		return false
	}
	if len(lineH) > 0 && lay.lineH != lineH[0] {
		return false
	}
	return true
}

func glyphWidth(ctx renderer.Context, st *editState, r rune, style renderer.TextStyle) float32 {
	if st != nil {
		if st.glyphW == nil {
			st.glyphW = make(map[rune]float32, 128)
		}
		if w, ok := st.glyphW[r]; ok {
			return w
		}
	}
	w, _ := ctx.MeasureText(string(r), 0, style)
	if st != nil {
		st.glyphW[r] = w
	}
	return w
}

// build 按宽度逐字切可视行（满行再折，不在空格 / 连字符处提前断开）。
func (lay *inputLayout) build(ctx renderer.Context, style renderer.TextStyle, st *editState) {
	lay.runeW = func(r rune) float32 { return glyphWidth(ctx, st, r, style) }
	if lay.wrapW <= 0 && len(lay.text) >= largeLayoutBytes {
		lay.buildLarge(ctx, style)
		return
	}
	if utf8.RuneCountInString(lay.text) > maxLayoutRun {
		lay.buildVisualLines(ctx, style, st)
		return
	}
	runes := []rune(lay.text)
	width := func(r rune) float32 {
		return glyphWidth(ctx, st, r, style)
	}
	type span struct{ start, end int }
	lines := []span{}
	start := 0
	var x float32
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			lines = append(lines, span{start, i})
			start = i + 1
			x = 0
			continue
		}
		w := width(r)
		if lay.wrapW > 0 && x+w > lay.wrapW && i > start {
			lines = append(lines, span{start, i})
			start = i
			x = w
			continue
		}
		x += w
	}
	lines = append(lines, span{start, len(runes)})

	lay.maxLine = len(lines) - 1
	byteOff := make([]int, len(runes)+1)
	off := 0
	for i, r := range runes {
		byteOff[i] = off
		off += utf8.RuneLen(r)
	}
	byteOff[len(runes)] = off

	lay.points = make([]layoutPoint, 0, len(runes)+len(lines)+1)
	for li, sp := range lines {
		var lx float32
		lay.points = append(lay.points, layoutPoint{off: byteOff[sp.start], line: li, x: 0})
		for i := sp.start; i < sp.end; i++ {
			lx += width(runes[i])
			lay.points = append(lay.points, layoutPoint{off: byteOff[i+1], line: li, x: lx})
		}
	}
}

func (lay *inputLayout) buildVisualLines(ctx renderer.Context, style renderer.TextStyle, st *editState) {
	lay.visual = true
	lay.lineStarts = []int{0}
	var x float32
	lineStart := 0
	i := 0
	for i < len(lay.text) {
		r, size := utf8.DecodeRuneInString(lay.text[i:])
		if r == '\n' {
			i += size
			lay.lineStarts = append(lay.lineStarts, i)
			lineStart = i
			x = 0
			continue
		}
		w := glyphWidth(ctx, st, r, style)
		if lay.wrapW > 0 && x+w > lay.wrapW && i > lineStart {
			lay.lineStarts = append(lay.lineStarts, i)
			lineStart = i
			x = w
			i += size
			continue
		}
		x += w
		i += size
	}
	if n := len(lay.lineStarts); n > 0 {
		lay.maxLine = n - 1
	}
}

func (lay *inputLayout) buildLarge(ctx renderer.Context, style renderer.TextStyle) {
	_, lay.lineH = ctx.MeasureText(" ", 0, style)
	if lay.lineH <= 0 {
		lay.lineH = style.FontSize * 1.4
	}
	lay.large = true
	lay.lineStarts = []int{0}
	line := 0
	for offset := 0; offset < len(lay.text); offset++ {
		if lay.text[offset] != '\n' {
			continue
		}
		line++
		if line%largeLineStride == 0 && offset+1 < len(lay.text) {
			lay.lineStarts = append(lay.lineStarts, offset+1)
		}
	}
	lay.maxLine = line
}

func (lay *inputLayout) lineStart(line int) int {
	if line <= 0 || len(lay.lineStarts) == 0 {
		return 0
	}
	if line > lay.maxLine {
		line = lay.maxLine
	}
	if lay.visual {
		if line >= len(lay.lineStarts) {
			return len(lay.text)
		}
		return lay.lineStarts[line]
	}
	base := (line / largeLineStride) * largeLineStride
	index := line / largeLineStride
	if index >= len(lay.lineStarts) {
		index = len(lay.lineStarts) - 1
		base = index * largeLineStride
	}
	offset := lay.lineStarts[index]
	for current := base; current < line; current++ {
		next := strings.IndexByte(lay.text[offset:], '\n')
		if next < 0 {
			return len(lay.text)
		}
		offset += next + 1
	}
	return offset
}

func (lay *inputLayout) lineForOffset(offset int) int {
	if offset <= 0 {
		return 0
	}
	if offset >= len(lay.text) {
		return lay.maxLine
	}
	if lay.visual {
		low, high := 0, len(lay.lineStarts)
		for low+1 < high {
			mid := (low + high) / 2
			if lay.lineStarts[mid] <= offset {
				low = mid
			} else {
				high = mid
			}
		}
		return low
	}
	low, high := 0, len(lay.lineStarts)
	for low+1 < high {
		mid := (low + high) / 2
		if lay.lineStarts[mid] <= offset {
			low = mid
		} else {
			high = mid
		}
	}
	line := low * largeLineStride
	for line < lay.maxLine {
		next := strings.IndexByte(lay.text[lay.lineStart(line):], '\n')
		if next < 0 || lay.lineStart(line)+next+1 > offset {
			break
		}
		line++
	}
	return line
}

func (lay *inputLayout) sparse() bool {
	return lay != nil && (lay.large || lay.visual)
}

// widthOf 返回单个 rune 的绘制宽。runeW 未绑定时退回「一字节 1」，
// 与旧的 off-start 字节差口径一致，不会更离谱。
func (lay *inputLayout) widthOf(r rune) float32 {
	if lay.runeW != nil {
		return lay.runeW(r)
	}
	return float32(utf8.RuneLen(r))
}

// offsetToX 返回可视行内 [start, off) 的像素宽。
func (lay *inputLayout) offsetToX(start, off int) float32 {
	var x float32
	for i := start; i < off && i < len(lay.text); {
		r, size := utf8.DecodeRuneInString(lay.text[i:])
		x += lay.widthOf(r)
		i += size
	}
	return x
}

// xToOffset 把行内的像素横坐标换算成最近的 rune 边界：
// 点中字符左半边落在它前面，右半边落在它后面。超出行末就钳到行末。
func (lay *inputLayout) xToOffset(start, end int, x float32) int {
	if x <= 0 {
		return start
	}
	var acc float32
	for i := start; i < end && i < len(lay.text); {
		r, size := utf8.DecodeRuneInString(lay.text[i:])
		w := lay.widthOf(r)
		if acc+w/2 >= x {
			return i
		}
		acc += w
		if acc >= x {
			return i + size
		}
		i += size
	}
	return end
}

func (lay *inputLayout) lineEnd(line int) int {
	end := len(lay.text)
	if line < lay.maxLine {
		end = lay.lineStart(line + 1)
		if end > 0 && lay.text[end-1] == '\n' {
			end--
		}
	}
	return end
}

func (lay *inputLayout) pointAt(off int) layoutPoint {
	if lay != nil && lay.sparse() {
		line := lay.lineForOffset(off)
		start := lay.lineStart(line)
		end := lay.lineEnd(line)
		if off < start {
			off = start
		}
		if off > end {
			off = end
		}
		return layoutPoint{off: off, line: line, x: lay.offsetToX(start, off)}
	}
	if len(lay.points) == 0 {
		return layoutPoint{}
	}
	best := lay.points[0]
	for _, p := range lay.points {
		if p.off > off {
			break
		}
		best = p
	}
	return best
}

func (lay *inputLayout) lineByteStart(line int) int {
	if lay == nil {
		return 0
	}
	if lay.sparse() {
		return lay.lineStart(line)
	}
	if line <= 0 {
		if len(lay.points) == 0 {
			return 0
		}
		return lay.points[0].off
	}
	if line > lay.maxLine {
		return len(lay.text)
	}
	for _, p := range lay.points {
		if p.line == line {
			return p.off
		}
	}
	return len(lay.text)
}

// pointAtLineEnd 返回某可视行末尾的边界点。
func (lay *inputLayout) pointAtLineEnd(line int) layoutPoint {
	if lay != nil && lay.sparse() {
		start := lay.lineStart(line)
		end := lay.lineEnd(line)
		return layoutPoint{off: end, line: line, x: lay.offsetToX(start, end)}
	}
	var best layoutPoint
	found := false
	for _, p := range lay.points {
		if p.line != line {
			continue
		}
		if !found || p.off >= best.off {
			best = p
			found = true
		}
	}
	return best
}

func (lay *inputLayout) pointAtXY(x, y float32) layoutPoint {
	if lay != nil && lay.sparse() {
		line := 0
		if lay.lineH > 0 {
			line = int(y / lay.lineH)
		}
		if line < 0 {
			line = 0
		}
		if line > lay.maxLine {
			line = lay.maxLine
		}
		start := lay.lineStart(line)
		end := lay.lineEnd(line)
		off := lay.xToOffset(start, end, x)
		return layoutPoint{off: off, line: line, x: lay.offsetToX(start, off)}
	}
	if len(lay.points) == 0 {
		return layoutPoint{}
	}
	line := 0
	if lay.lineH > 0 {
		line = int(y / lay.lineH)
	}
	best := lay.points[0]
	bestDist := float32(1e18)
	for _, p := range lay.points {
		d := (p.x - x) * (p.x - x)
		if p.line != line {
			d += 1e7
		}
		if d < bestDist {
			bestDist = d
			best = p
		}
	}
	return best
}

// ===== 行列 ↔ 字节偏移 ==================================================
//
// 下面几个方法供「状态栏显示行列」「按行列定位光标」使用。布局尚未建立时
// 统一退化为按硬换行切分，因此打开文件后立刻定位不必等第一帧绘制。

// moveToLineCol 把光标移到 0 基的 (行, 列)，列按 rune 计。
func (st *editState) moveToLineCol(line, col int) {
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	off := 0
	if st.layout != nil && st.layout.sparse() {
		if line > st.layout.maxLine {
			line = st.layout.maxLine
		}
		off = st.layout.lineStart(line)
		end := st.layout.lineEnd(line)
		for i := 0; i < col; i++ {
			next := st.nextRuneOffset(off)
			if next == off || next > end {
				break
			}
			off = next
		}
	} else if st.layout != nil && len(st.layout.points) > 0 {
		if line > st.layout.maxLine {
			line = st.layout.maxLine
		}
		for _, p := range st.layout.points {
			if p.line == line {
				off = p.off
				break
			}
		}
		for i := 0; i < col; i++ {
			next := st.nextRuneOffset(off)
			if next == off || st.layout.pointAt(next).line != line {
				break
			}
			off = next
		}
	} else {
		lines := strings.Split(st.value, "\n")
		if line >= len(lines) {
			line = len(lines) - 1
		}
		for i := 0; i < line; i++ {
			off += len(lines[i]) + 1
		}
		runes := []rune(lines[line])
		if col > len(runes) {
			col = len(runes)
		}
		off += len(string(runes[:col]))
	}
	st.caret = clampCaret(st.value, off)
	st.anchor = st.caret
	st.scrollSync = true
}

func (st *editState) nextRuneOffset(off int) int {
	if off >= len(st.value) {
		return off
	}
	_, size := utf8.DecodeRuneInString(st.value[off:])
	return off + size
}

// caretLineCol 返回光标的 0 基 (行, 列)；列按 rune 计，软换行也算一行。
func (st *editState) caretLineCol() (line, col int) {
	at := st.caretOffset()
	if st.layout != nil && st.layout.sparse() {
		line = st.layout.lineForOffset(at)
		start := st.layout.lineStart(line)
		from := clampCaret(st.value, start)
		to := clampCaret(st.value, at)
		if to > from {
			col = utf8.RuneCountInString(st.value[from:to])
		}
		return line, col
	}
	if st.layout != nil && len(st.layout.points) > 0 {
		p := st.layout.pointAt(at)
		line = p.line
		start := 0
		for _, q := range st.layout.points {
			if q.line == line {
				start = q.off
				break
			}
		}
		from := clampCaret(st.value, start)
		to := clampCaret(st.value, at)
		if to > from {
			col = utf8.RuneCountInString(st.value[from:to])
		}
		return line, col
	}
	text := st.value
	at = clampCaret(text, at)
	line = strings.Count(text[:at], "\n")
	start := strings.LastIndexByte(text[:at], '\n') + 1
	col = utf8.RuneCountInString(text[start:at])
	return line, col
}
