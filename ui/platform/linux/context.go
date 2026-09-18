//go:build linux

package linux

import (
	"bytes"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/goedui/goed/ui/renderer"
)

type clipRect struct {
	x0, y0, x1, y1 int
}

type swContext struct {
	pix    []byte
	stride int
	pixelW int
	pixelH int
	dpi    float32
	width  float32
	height float32
	clips  []clipRect
}

func newContext(pixelW, pixelH int, dpi float32) *swContext {
	if pixelW < 1 {
		pixelW = 1
	}
	if pixelH < 1 {
		pixelH = 1
	}
	if dpi <= 0 {
		dpi = 96
	}
	c := &swContext{
		pix:    make([]byte, pixelW*pixelH*4),
		stride: pixelW * 4,
		pixelW: pixelW,
		pixelH: pixelH,
		dpi:    dpi,
	}
	c.width = float32(pixelW) * 96 / dpi
	c.height = float32(pixelH) * 96 / dpi
	c.clips = []clipRect{{0, 0, pixelW, pixelH}}
	return c
}

func (c *swContext) resize(pixelW, pixelH int, dpi float32) {
	if dpi <= 0 {
		dpi = 96
	}
	if pixelW == c.pixelW && pixelH == c.pixelH && dpi == c.dpi && len(c.pix) == pixelW*pixelH*4 {
		c.clips = c.clips[:1]
		c.clips[0] = clipRect{0, 0, pixelW, pixelH}
		return
	}
	*c = *newContext(pixelW, pixelH, dpi)
}

func (c *swContext) Width() float32  { return c.width }
func (c *swContext) Height() float32 { return c.height }
func (c *swContext) DPI() float32    { return c.dpi }

func (c *swContext) toPx(v float32) int {
	if c.dpi <= 0 {
		return int(v + 0.5)
	}
	return int(v*c.dpi/96 + 0.5)
}

func (c *swContext) clip() clipRect {
	if len(c.clips) == 0 {
		return clipRect{0, 0, c.pixelW, c.pixelH}
	}
	return c.clips[len(c.clips)-1]
}

func (c *swContext) Clear(col renderer.Color) {
	r, g, b, a := packBGRA(col)
	for i := 0; i < len(c.pix); i += 4 {
		c.pix[i] = b
		c.pix[i+1] = g
		c.pix[i+2] = r
		c.pix[i+3] = a
	}
}

func (c *swContext) FillRect(x, y, w, h float32, col renderer.Color) {
	c.fillPx(c.toPx(x), c.toPx(y), c.toPx(x+w), c.toPx(y+h), col)
}

func (c *swContext) FillRoundedRect(x, y, w, h, r float32, col renderer.Color) {
	if r <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}
	x0, y0 := c.toPx(x), c.toPx(y)
	x1, y1 := c.toPx(x+w), c.toPx(y+h)
	pr := c.toPx(r)
	maxR := x1 - x0
	if y1-y0 < maxR {
		maxR = y1 - y0
	}
	if pr > maxR/2 {
		pr = maxR / 2
	}
	if pr <= 0 {
		c.fillPx(x0, y0, x1, y1, col)
		return
	}
	cl := c.clip()
	rr, gg, bb, aa := packBGRA(col)
	r2 := pr * pr
	for py := max(y0, cl.y0); py < min(y1, cl.y1); py++ {
		for px := max(x0, cl.x0); px < min(x1, cl.x1); px++ {
			cx := px - x0
			cy := py - y0
			ok := true
			if cx < pr && cy < pr {
				dx, dy := pr-1-cx, pr-1-cy
				ok = dx*dx+dy*dy <= r2
			} else if cx >= (x1-x0)-pr && cy < pr {
				dx, dy := cx-((x1-x0)-pr), pr-1-cy
				ok = dx*dx+dy*dy <= r2
			} else if cx < pr && cy >= (y1-y0)-pr {
				dx, dy := pr-1-cx, cy-((y1-y0)-pr)
				ok = dx*dx+dy*dy <= r2
			} else if cx >= (x1-x0)-pr && cy >= (y1-y0)-pr {
				dx, dy := cx-((x1-x0)-pr), cy-((y1-y0)-pr)
				ok = dx*dx+dy*dy <= r2
			}
			if ok {
				c.blend(px, py, rr, gg, bb, aa)
			}
		}
	}
}

func (c *swContext) DrawRect(x, y, w, h float32, col renderer.Color, width float32) {
	if width <= 0 {
		width = 1
	}
	t := width
	c.FillRect(x, y, w, t, col)
	c.FillRect(x, y+h-t, w, t, col)
	c.FillRect(x, y, t, h, col)
	c.FillRect(x+w-t, y, t, h, col)
}

func (c *swContext) DrawLine(x1, y1, x2, y2 float32, col renderer.Color, width float32) {
	if width <= 0 {
		width = 1
	}
	px1, py1 := c.toPx(x1), c.toPx(y1)
	px2, py2 := c.toPx(x2), c.toPx(y2)
	if px1 == px2 || py1 == py2 {
		if px1 > px2 {
			px1, px2 = px2, px1
		}
		if py1 > py2 {
			py1, py2 = py2, py1
		}
		t := c.toPx(width)
		if t < 1 {
			t = 1
		}
		if px1 == px2 {
			c.fillPx(px1, py1, px1+t, py2+t, col)
			return
		}
		c.fillPx(px1, py1, px2+t, py1+t, col)
		return
	}
	c.bresenham(px1, py1, px2, py2, col)
}

func (c *swContext) DrawText(text string, x, y, maxW, maxH float32, style renderer.TextStyle) {
	if text == "" {
		return
	}
	if c.drawSysText(text, x, y, maxW, maxH, style) {
		return
	}
	cw, ch := glyphSize(style.FontSize)
	if cw < 1 || ch < 1 {
		return
	}
	lines := wrapText(text, maxW, style, cw)
	totalH := float32(len(lines) * ch)
	startY := y
	switch style.VAlign {
	case renderer.AlignCenter:
		startY = y + (maxH-totalH)/2
	case renderer.AlignEnd:
		startY = y + maxH - totalH
	}
	limit := len(lines)
	if maxH > 0 && !style.NoWrap {
		maxLines := int(maxH / float32(ch))
		if maxLines < 1 {
			maxLines = 1
		}
		if limit > maxLines {
			limit = maxLines
		}
	}
	for i := 0; i < limit; i++ {
		line := lines[i]
		lw := float32(len([]rune(line)) * cw)
		startX := x
		switch style.Align {
		case renderer.AlignCenter:
			startX = x + (maxW-lw)/2
		case renderer.AlignEnd:
			startX = x + maxW - lw
		}
		c.blitLine(line, c.toPx(startX), c.toPx(startY)+i*ch, cw, ch, style.Color)
	}
}

func (c *swContext) MeasureText(text string, maxW float32, style renderer.TextStyle) (float32, float32) {
	if text == "" {
		return 0, 0
	}
	if w, h, ok := measureSys(text, maxW, style, c.dpi); ok {
		return w, h
	}
	cw, ch := glyphSize(style.FontSize)
	if style.NoWrap || maxW <= 0 {
		n := len([]rune(text))
		return float32(n * cw), float32(ch)
	}
	lines := wrapText(text, maxW, style, cw)
	maxLine := 0
	for _, ln := range lines {
		if n := len([]rune(ln)); n > maxLine {
			maxLine = n
		}
	}
	return float32(maxLine * cw), float32(len(lines) * ch)
}

func (c *swContext) DrawImage(data []byte, x, y, w, h float32) {
	if len(data) == 0 || w <= 0 || h <= 0 {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	dx0, dy0 := c.toPx(x), c.toPx(y)
	dx1, dy1 := c.toPx(x+w), c.toPx(y+h)
	dw, dh := dx1-dx0, dy1-dy0
	if dw < 1 || dh < 1 {
		return
	}
	sw, sh := rgba.Bounds().Dx(), rgba.Bounds().Dy()
	if sw < 1 || sh < 1 {
		return
	}
	cl := c.clip()
	for py := max(dy0, cl.y0); py < min(dy1, cl.y1); py++ {
		sy := (py - dy0) * sh / dh
		if sy >= sh {
			sy = sh - 1
		}
		for px := max(dx0, cl.x0); px < min(dx1, cl.x1); px++ {
			sx := (px - dx0) * sw / dw
			if sx >= sw {
				sx = sw - 1
			}
			off := sy*rgba.Stride + sx*4
			c.blend(px, py, rgba.Pix[off+0], rgba.Pix[off+1], rgba.Pix[off+2], rgba.Pix[off+3])
		}
	}
}

func (c *swContext) DrawPixels(pix []byte, pw, ph int, x, y, w, h float32) {
	needed := pw * ph * 4
	if len(pix) < needed || pw < 1 || ph < 1 || w <= 0 || h <= 0 {
		return
	}
	dx0, dy0 := c.toPx(x), c.toPx(y)
	dx1, dy1 := c.toPx(x+w), c.toPx(y+h)
	dw, dh := dx1-dx0, dy1-dy0
	if dw < 1 || dh < 1 {
		return
	}
	cl := c.clip()
	for py := max(dy0, cl.y0); py < min(dy1, cl.y1); py++ {
		sy := (py - dy0) * ph / dh
		if sy >= ph {
			sy = ph - 1
		}
		for px := max(dx0, cl.x0); px < min(dx1, cl.x1); px++ {
			sx := (px - dx0) * pw / dw
			if sx >= pw {
				sx = pw - 1
			}
			off := (sy*pw + sx) * 4
			c.blend(px, py, pix[off], pix[off+1], pix[off+2], pix[off+3])
		}
	}
}

func (c *swContext) PushClip(x, y, w, h float32) {
	n := clipRect{c.toPx(x), c.toPx(y), c.toPx(x + w), c.toPx(y + h)}
	cur := c.clip()
	if n.x0 < cur.x0 {
		n.x0 = cur.x0
	}
	if n.y0 < cur.y0 {
		n.y0 = cur.y0
	}
	if n.x1 > cur.x1 {
		n.x1 = cur.x1
	}
	if n.y1 > cur.y1 {
		n.y1 = cur.y1
	}
	c.clips = append(c.clips, n)
}

func (c *swContext) PopClip() {
	if len(c.clips) > 1 {
		c.clips = c.clips[:len(c.clips)-1]
	}
}

func (c *swContext) fillPx(x0, y0, x1, y1 int, col renderer.Color) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	cl := c.clip()
	x0 = max(x0, cl.x0)
	y0 = max(y0, cl.y0)
	x1 = min(x1, cl.x1)
	y1 = min(y1, cl.y1)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	r, g, b, a := packBGRA(col)
	if a == 255 {
		for y := y0; y < y1; y++ {
			off := y*c.stride + x0*4
			for x := x0; x < x1; x++ {
				c.pix[off] = b
				c.pix[off+1] = g
				c.pix[off+2] = r
				c.pix[off+3] = a
				off += 4
			}
		}
		return
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.blend(x, y, r, g, b, a)
		}
	}
}

func (c *swContext) blend(x, y int, sr, sg, sb, sa byte) {
	if x < 0 || y < 0 || x >= c.pixelW || y >= c.pixelH {
		return
	}
	off := y*c.stride + x*4
	if sa == 255 {
		c.pix[off] = sb
		c.pix[off+1] = sg
		c.pix[off+2] = sr
		c.pix[off+3] = 255
		return
	}
	if sa == 0 {
		return
	}
	inv := uint32(255 - sa)
	c.pix[off] = byte((uint32(sb)*uint32(sa) + uint32(c.pix[off])*inv) / 255)
	c.pix[off+1] = byte((uint32(sg)*uint32(sa) + uint32(c.pix[off+1])*inv) / 255)
	c.pix[off+2] = byte((uint32(sr)*uint32(sa) + uint32(c.pix[off+2])*inv) / 255)
	c.pix[off+3] = 255
}

func (c *swContext) bresenham(x0, y0, x1, y1 int, col renderer.Color) {
	r, g, b, a := packBGRA(col)
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	cl := c.clip()
	for {
		if x0 >= cl.x0 && x0 < cl.x1 && y0 >= cl.y0 && y0 < cl.y1 {
			c.blend(x0, y0, r, g, b, a)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *swContext) blitLine(text string, x, y, cw, ch int, col renderer.Color) {
	r, g, b, a := packBGRA(col)
	cl := c.clip()
	scale := cw / glyphW
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, ru := range text {
		bits, ok := glyphBits(ru)
		for row := 0; row < glyphH; row++ {
			var rowBits byte
			if ok {
				rowBits = bits[row]
			} else if row == 0 || row == glyphH-1 {
				rowBits = 0x7E
			} else {
				rowBits = 0x42
			}
			for colx := 0; colx < glyphW; colx++ {
				if rowBits&(1<<colx) == 0 {
					continue
				}
				px0 := cx + colx*scale
				py0 := y + row*scale
				for oy := 0; oy < scale; oy++ {
					py := py0 + oy
					if py < cl.y0 || py >= cl.y1 {
						continue
					}
					for ox := 0; ox < scale; ox++ {
						px := px0 + ox
						if px < cl.x0 || px >= cl.x1 {
							continue
						}
						c.blend(px, py, r, g, b, a)
					}
				}
			}
		}
		cx += cw
	}
}

func wrapText(text string, maxW float32, style renderer.TextStyle, cellW int) []string {
	runes := []rune(text)
	if style.NoWrap || maxW <= 0 || cellW <= 0 {
		return []string{text}
	}
	maxChars := int(maxW) / cellW
	if maxChars < 1 {
		maxChars = 1
	}
	var lines []string
	for len(runes) > 0 {
		n := maxChars
		if n > len(runes) {
			n = len(runes)
		}
		chunk := runes[:n]
		if i := lastIndexRune(chunk, '\n'); i >= 0 {
			lines = append(lines, string(chunk[:i]))
			runes = runes[i+1:]
			continue
		}
		lines = append(lines, string(chunk))
		runes = runes[n:]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func lastIndexRune(runes []rune, r rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == r {
			return i
		}
	}
	return -1
}

func packBGRA(col renderer.Color) (r, g, b, a byte) {
	r = clamp8(col.R * 255)
	g = clamp8(col.G * 255)
	b = clamp8(col.B * 255)
	a = clamp8(col.A * 255)
	return
}

func clamp8(v float32) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dipToPx(v, dpi float32) int {
	if dpi <= 0 {
		dpi = 96
	}
	n := int(v*dpi/96 + 0.5)
	if n < 1 && v > 0 {
		return 1
	}
	return n
}
