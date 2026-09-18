//go:build linux

package linux

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/goedui/goed/ui/renderer"
)

// 系统无衬线：Inter / Ubuntu 覆盖拉丁；Noto Sans CJK / Droid Fallback 覆盖中文。
// 解析失败时退回内嵌 8×8 点阵。

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
	ot   *opentype.Font
	bold bool
	cjk  bool
}

type faceKey struct {
	px   int
	bold bool
}

type sizedFace struct {
	latin  font.Face
	cjk    font.Face
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
			parsed := parseFonts(spec.path, spec.cjk)
			sysFonts = append(sysFonts, parsed...)
		}
		bitmapOnly = len(sysFonts) == 0
	})
}

func parseFonts(path string, cjk bool) []sysFont {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	bold := strings.Contains(path, "Bold") || strings.Contains(path, "-B.")
	if col, err := opentype.ParseCollection(data); err == nil && col.NumFonts() > 0 {
		f, err := col.Font(0)
		if err != nil {
			return nil
		}
		return []sysFont{{ot: f, bold: bold, cjk: cjk}}
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil
	}
	return []sysFont{{ot: f, bold: bold, cjk: cjk}}
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
	opts := &opentype.FaceOptions{Size: float64(key.px), DPI: 72, Hinting: font.HintingFull}
	var latin, cjk font.Face
	pick := func(wantCJK, wantBold bool) font.Face {
		var fallback font.Face
		for _, f := range sysFonts {
			if f.cjk != wantCJK || f.ot == nil {
				continue
			}
			face, err := opentype.NewFace(f.ot, opts)
			if err != nil {
				continue
			}
			if f.bold == wantBold {
				return face
			}
			if fallback == nil {
				fallback = face
			}
		}
		return fallback
	}
	latin = pick(false, key.bold)
	cjk = pick(true, key.bold)
	primary := latin
	if primary == nil {
		primary = cjk
	}
	if primary == nil {
		return nil
	}
	m := primary.Metrics()
	h := m.Height.Ceil()
	if h < 1 {
		h = key.px
	}
	return &sizedFace{latin: latin, cjk: cjk, ascent: m.Ascent.Ceil(), height: h}
}

func (sf *sizedFace) pick(r rune) font.Face {
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
	face := sf.pick(r)
	if face == nil {
		return 0
	}
	adv, ok := face.GlyphAdvance(r)
	if !ok {
		return 0
	}
	return adv.Ceil()
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
	dot := fixed.P(x, y+sf.ascent)
	sr, sg, sb, sa := packBGRA(col)
	cl := c.clip()
	for _, r := range text {
		face := sf.pick(r)
		if face == nil {
			continue
		}
		dr, mask, maskp, adv, ok := face.Glyph(dot, r)
		if ok && mask != nil {
			minX := max(dr.Min.X, cl.x0)
			minY := max(dr.Min.Y, cl.y0)
			maxX := min(dr.Max.X, cl.x1)
			maxY := min(dr.Max.Y, cl.y1)
			for py := minY; py < maxY; py++ {
				for px := minX; px < maxX; px++ {
					_, _, _, ma := mask.At(maskp.X+px-dr.Min.X, maskp.Y+py-dr.Min.Y).RGBA()
					if ma == 0 {
						continue
					}
					a := byte((uint32(sa) * (ma >> 8)) / 255)
					if a == 0 {
						continue
					}
					c.blend(px, py, sr, sg, sb, a)
				}
			}
		}
		if adv > 0 {
			dot.X += adv
		} else {
			dot.X += fixed.I(sf.height / 2)
		}
	}
}
