// 编辑框的布局：把文本切成可视行，并在「字节偏移 ↔ 行列坐标」之间换算。
//
// 光标上下移动、翻页、Home/End、点击落点、选区高亮、光标定位都依赖这里的
// inputLayout —— 它是布局结果，缓存在 editState.layout 里，文本或折行宽度
// 变化时失效。仅参与编辑的核心逻辑用，不含绘制。
package components

import (
	"strings"
	"unicode/utf8"

	"github.com/goedui/goed/ui/renderer"
)

type layoutPoint struct {
	off  int
	line int
	x    float32
}

type inputLayout struct {
	text    string
	wrapW   float32
	lineH   float32
	points  []layoutPoint
	maxLine int // 最后一个可视行的下标（0 起）
}

func layoutOf(ctx renderer.Context, st *editState, wrapW float32, style renderer.TextStyle) *inputLayout {
	return layoutText(ctx, st, st.text(), wrapW, style)
}

func layoutText(ctx renderer.Context, st *editState, text string, wrapW float32, style renderer.TextStyle) *inputLayout {
	lineH := float32(0)
	_, lh := ctx.MeasureText(" ", wrapW, style)
	lineH = lh
	if lineH <= 0 {
		lineH = style.FontSize * 1.4
	}
	if st.layout != nil && st.layout.text == text && st.layout.wrapW == wrapW && st.layout.lineH == lineH {
		return st.layout
	}
	lay := &inputLayout{text: text, wrapW: wrapW, lineH: lineH}
	lay.build(ctx, style)
	st.layout = lay
	return lay
}

// build 按宽度逐字切可视行（满行再折，不在空格 / 连字符处提前断开）。
func (lay *inputLayout) build(ctx renderer.Context, style renderer.TextStyle) {
	runes := []rune(lay.text)
	if len(runes) > maxLayoutRun {
		runes = runes[:maxLayoutRun]
	}
	width := func(r rune) float32 {
		w, _ := ctx.MeasureText(string(r), 0, style)
		return w
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

func (lay *inputLayout) pointAt(off int) layoutPoint {
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

// pointAtLineEnd 返回某可视行末尾的边界点。
func (lay *inputLayout) pointAtLineEnd(line int) layoutPoint {
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
	if st.layout != nil && len(st.layout.points) > 0 {
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
