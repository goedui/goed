//go:build linux

package linux

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/goedui/goed/ui/renderer"
)

// 系统无衬线：Inter / Ubuntu 覆盖拉丁，Noto Sans CJK / Droid Fallback 覆盖中文。
// 字库交给系统装的 libfreetype 解析（见 ft.go）；解析失败或库缺失时退回内嵌 8×8 点阵。

var fontFiles = []struct {
	path string
	cjk  bool
}{
	{"/usr/share/fonts/opentype/inter/Inter-Regular.otf", false},
	{"/usr/share/fonts/opentype/inter/Inter-Bold.otf", false},
	{"/usr/share/fonts/truetype/ubuntu/Ubuntu-R.ttf", false},
	{"/usr/share/fonts/truetype/ubuntu/Ubuntu-B.ttf", false},
	{"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", false},
	{"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", false},
	{"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", true},
	{"/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc", true},
	{"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf", true},
	{"/usr/share/fonts/truetype/arphic/uming.ttc", true},
}

type sysFont struct {
	ft   *ftFace
	bold bool
	cjk  bool
}

type faceKey struct {
	px   int
	bold bool
}

// sizedFace 是某个像素字号下的一对 face（拉丁 + CJK），字号记在这里，
// 画字时按需把 FT_Face 切过去。
type sizedFace struct {
	px     int
	latin  *sysFont
	cjk    *sysFont
	ascent int
	height int
}

var (
	fontOnce   sync.Once
	sysFonts   []sysFont
	faceMu     sync.Mutex
	faceCache  = map[faceKey]*sizedFace{}
	bitmapOnly bool
)

func loadSysFonts() {
	fontOnce.Do(func() {
		for _, spec := range fontFiles {
			f, err := openFTFace(spec.path, 0)
			if err != nil {
				continue
			}
			sysFonts = append(sysFonts, sysFont{
				ft:   f,
				bold: strings.Contains(spec.path, "Bold") || strings.Contains(spec.path, "-B."),
				cjk:  spec.cjk,
			})
		}
		bitmapOnly = len(sysFonts) == 0
	})
}

func faceFor(style renderer.TextStyle, dpi float32) *sizedFace {
	loadSysFonts()
	if bitmapOnly {
		return nil
	}
	px := int(style.FontSize*dpi/96 + 0.5)
	if px < 8 {
		px = 8
	}
	if px > 96 {
		px = 96
	}
	key := faceKey{px: px, bold: style.Weight >= 600}
	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faceCache[key]; ok {
		return f
	}
	sf := buildFace(key)
	if sf != nil {
		faceCache[key] = sf
	}
	return sf
}

func buildFace(key faceKey) *sizedFace {
	// 同字族里优先精确匹配粗细，没有粗体面就用同族正体顶替。
	pick := func(wantCJK, wantBold bool) *sysFont {
		var fallback *sysFont
		for i := range sysFonts {
			f := &sysFonts[i]
			if f.cjk != wantCJK {
				continue
			}
			if f.bold == wantBold {
				return f
			}
			if fallback == nil {
				fallback = f
			}
		}
		return fallback
	}
	latin := pick(false, key.bold)
	cjk := pick(true, key.bold)
	primary := latin
	if primary == nil {
		primary = cjk
	}
	if primary == nil {
		return nil
	}
	ascent, height, err := primary.ft.setSize(key.px)
	if err != nil {
		return nil
	}
	if height < 1 {
		height = key.px
	}
	return &sizedFace{px: key.px, latin: latin, cjk: cjk, ascent: ascent, height: height}
}

func (sf *sizedFace) pick(r rune) *sysFont {
	if sf == nil {
		return nil
	}
	if wantsCJK(r) && sf.cjk != nil {
		return sf.cjk
	}
	if sf.latin != nil {
		return sf.latin
	}
	return sf.cjk
}

func wantsCJK(r rune) bool {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF, r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0xFE30 && r <= 0xFE4F, r >= 0x20000 && r <= 0x2FA1F:
		return true
	case r >= 0x3000 && r <= 0x303F:
		return true
	default:
		return false
	}
}

func (sf *sizedFace) advance(r rune) int {
	f := sf.pick(r)
	if f == nil {
		return 0
	}
	return f.ft.advance(r, sf.px)
}

func measureSys(text string, maxW float32, style renderer.TextStyle, dpi float32) (w, h float32, ok bool) {
	sf := faceFor(style, dpi)
	if sf == nil {
		return 0, 0, false
	}
	lines := wrapSys(text, maxW, style, dpi, sf)
	var maxLine int
	for _, ln := range lines {
		if aw := lineAdvance(sf, ln); aw > maxLine {
			maxLine = aw
		}
	}
	scale := float32(1)
	if dpi > 0 {
		scale = 96 / dpi
	}
	return float32(maxLine) * scale, float32(len(lines)*sf.height) * scale, true
}

func wrapSys(text string, maxW float32, style renderer.TextStyle, dpi float32, sf *sizedFace) []string {
	if text == "" {
		return []string{""}
	}
	if style.NoWrap || maxW <= 0 {
		return splitNL(text)
	}
	maxPx := int(maxW * dpi / 96)
	if maxPx < 1 {
		maxPx = 1
	}
	var lines []string
	var cur []rune
	curW := 0
	flush := func() {
		lines = append(lines, string(cur))
		cur = cur[:0]
		curW = 0
	}
	for _, r := range text {
		if r == '\n' {
			flush()
			continue
		}
		adv := sf.advance(r)
		if adv < 1 {
			adv = sf.height / 2
			if adv < 1 {
				adv = 8
			}
		}
		if len(cur) > 0 && curW+adv > maxPx {
			flush()
		}
		cur = append(cur, r)
		curW += adv
	}
	if len(cur) > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

func splitNL(text string) []string {
	var lines []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	lines = append(lines, text[start:])
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func lineAdvance(sf *sizedFace, s string) int {
	n := 0
	for _, r := range s {
		adv := sf.advance(r)
		if adv < 1 {
			adv = sf.height / 2
		}
		n += adv
	}
	return n
}

func (c *swContext) drawSysText(text string, x, y, maxW, maxH float32, style renderer.TextStyle) bool {
	sf := faceFor(style, c.dpi)
	if sf == nil {
		return false
	}
	lines := wrapSys(text, maxW, style, c.dpi, sf)
	lineH := float32(sf.height) * 96 / c.dpi
	if style.LineSpacing > 0 {
		lineH = style.LineSpacing
	}
	if lineH <= 0 {
		lineH = style.FontSize
	}
	totalH := lineH * float32(len(lines))
	startY := y
	switch style.VAlign {
	case renderer.AlignCenter:
		startY = y + (maxH-totalH)/2
	case renderer.AlignEnd:
		startY = y + maxH - totalH
	}
	limit := len(lines)
	if maxH > 0 && !style.NoWrap {
		maxLines := int(maxH / lineH)
		if maxLines < 1 {
			maxLines = 1
		}
		if limit > maxLines {
			limit = maxLines
		}
	}
	for i := 0; i < limit; i++ {
		line := lines[i]
		lw := float32(lineAdvance(sf, line)) * 96 / c.dpi
		startX := x
		switch style.Align {
		case renderer.AlignCenter:
			startX = x + (maxW-lw)/2
		case renderer.AlignEnd:
			startX = x + maxW - lw
		}
		c.blitSysLine(sf, line, c.toPx(startX), c.toPx(startY)+i*sf.height, style.Color)
	}
	return true
}

func (c *swContext) blitSysLine(sf *sizedFace, text string, x, y int, col renderer.Color) {
	sr, sg, sb, sa := packBGRA(col)
	cl := c.clip()
	penX := x
	// y 是行顶，字形位图按基线摆放，所以先下移到基线。
	baseline := y + sf.ascent
	for _, r := range text {
		f := sf.pick(r)
		if f == nil {
			continue
		}
		g, ok := f.ft.glyph(r, sf.px)
		if ok {
			c.blitGlyph(g, penX+g.left, baseline-g.top, sr, sg, sb, sa, cl)
		}
		switch {
		case ok && g.adv > 0:
			penX += g.adv
		default:
			penX += sf.height / 2
		}
	}
}

// blitGlyph 把字形位图按颜色混合到 (dx,dy)——位图左上角，不是基线。
// 位图是 8 位灰度（FT_PIXEL_MODE_GRAY）或 1 位单色，pitch 允许为负（行倒序）。
func (c *swContext) blitGlyph(g ftGlyph, dx, dy int, sr, sg, sb, sa byte, cl clipRect) {
	if g.buf == nil || g.w < 1 || g.h < 1 || g.pitch == 0 {
		return
	}
	if dx >= cl.x1 || dy >= cl.y1 || dx+g.w <= cl.x0 || dy+g.h <= cl.y0 {
		return
	}
	stride := g.pitch
	if stride < 0 {
		stride = -stride
	}
	for gy := 0; gy < g.h; gy++ {
		py := dy + gy
		if py < cl.y0 || py >= cl.y1 {
			continue
		}
		row := gy
		if g.pitch < 0 {
			row = g.h - 1 - gy
		}
		src := unsafe.Slice((*byte)(unsafe.Add(g.buf, row*stride)), stride)
		for gx := 0; gx < g.w; gx++ {
			px := dx + gx
			if px < cl.x0 || px >= cl.x1 {
				continue
			}
			var ma byte
			if g.mode == ftPixelMono {
				if src[gx>>3]&(0x80>>(uint(gx)&7)) == 0 {
					continue
				}
				ma = 255
			} else {
				ma = src[gx]
				if ma == 0 {
					continue
				}
			}
			a := byte((uint32(sa) * uint32(ma)) / 255)
			if a == 0 {
				continue
			}
			c.blend(px, py, sr, sg, sb, a)
		}
	}
}
