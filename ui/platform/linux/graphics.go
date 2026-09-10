package linux

import (
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"github.com/goedui/goed/ui/core"
)

// GDIContext is the platform-neutral software drawing surface used by the
// Linux compatibility backend. A native X11/Cairo surface can replace the
// pixel buffer later without changing callers.
type GDIContext struct {
	width, height int32
	clearColor    uint32
	fontName      string
	fontSize      int32
	clip          image.Rectangle
	clipStack     []image.Rectangle
	pixels        []uint32
}

func NewGDIContext(_ uintptr, width, height int32) *GDIContext {
	g := &GDIContext{clearColor: 0x001F1F1F, fontName: "Sans", fontSize: 14}
	g.Resize(0, width, height)
	return g
}

func (g *GDIContext) BeginDraw() {
	if g == nil {
		return
	}
	for i := range g.pixels {
		g.pixels[i] = g.clearColor
	}
}

func (g *GDIContext) EndDraw() {}

func (g *GDIContext) SetBackgroundColor(c uint32) {
	if g != nil {
		g.clearColor = c
	}
}

func (g *GDIContext) DrawRect(x, y, w, h int32, c uint32, filled bool) {
	if filled {
		g.fillRect(x, y, w, h, c)
		return
	}
	g.DrawLine(x, y, x+w-1, y, c, 1)
	g.DrawLine(x, y+h-1, x+w-1, y+h-1, c, 1)
	g.DrawLine(x, y, x, y+h-1, c, 1)
	g.DrawLine(x+w-1, y, x+w-1, y+h-1, c, 1)
}

func (g *GDIContext) DrawLine(x1, y1, x2, y2 int32, c uint32, width int32) {
	if g == nil || width < 1 {
		width = 1
	}
	dx, sx := abs32(x2-x1), int32(1)
	if x1 > x2 {
		sx = -1
	}
	dy, sy := -abs32(y2-y1), int32(1)
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy
	for {
		g.fillRect(x1-width/2, y1-width/2, width, width, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func (g *GDIContext) DrawRoundedRect(x, y, w, h, _ int32, c uint32, filled bool) {
	// The software fallback keeps the geometry stable; native backends can
	// provide true anti-aliased rounded corners later.
	g.DrawRect(x, y, w, h, c, filled)
}

func (g *GDIContext) DrawEllipse(x, y, w, h int32, c uint32, filled bool) {
	if g == nil || w <= 0 || h <= 0 {
		return
	}
	rx, ry := float64(w)/2, float64(h)/2
	cx := float64(x) + rx
	for yy := y; yy < y+h; yy++ {
		dy := (float64(yy) + 0.5 - (float64(y) + ry)) / ry
		t := 1 - dy*dy
		if t < 0 {
			continue
		}
		half := int32(rx * math.Sqrt(t))
		left := int32(cx) - half
		if filled {
			g.fillRect(left, yy, half*2, 1, c)
			continue
		}
		// 轮廓：每行在左右边缘画 1px，并在上下边缘补上整行
		if yy == y || yy == y+h-1 {
			g.fillRect(left, yy, half*2, 1, c)
		} else {
			g.fillRect(left, yy, 1, 1, c)
			g.fillRect(left+half*2-1, yy, 1, 1, c)
		}
	}
}

func (g *GDIContext) DrawPolygon(points []core.Point, c uint32, filled bool) {
	if g == nil || len(points) < 2 {
		return
	}
	if !filled {
		for i := range points {
			a, b := points[i], points[(i+1)%len(points)]
			g.DrawLine(a.X, a.Y, b.X, b.Y, c, 1)
		}
		return
	}
	// 偶奇规则扫描线填充
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		minY, maxY = min32(minY, p.Y), max32(maxY, p.Y)
	}
	n := len(points)
	for yy := minY; yy <= maxY; yy++ {
		xs := make([]float64, 0, 2)
		for i := 0; i < n; i++ {
			a, b := points[i], points[(i+1)%n]
			if (a.Y <= yy && b.Y > yy) || (b.Y <= yy && a.Y > yy) {
				t := float64(yy-a.Y) / float64(b.Y-a.Y)
				xs = append(xs, float64(a.X)+t*float64(b.X-a.X))
			}
		}
		for i := 0; i < len(xs)-1; i += 2 {
			x0, x1 := int32(math.Ceil(xs[i])), int32(math.Floor(xs[i+1]))
			if x1 >= x0 {
				g.fillRect(x0, yy, x1-x0+1, 1, c)
			}
		}
	}
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (g *GDIContext) DrawText(_, _ int32, _ string, _ uint32) {}

func (g *GDIContext) MeasureText(text string) (int32, int32) {
	if g == nil {
		return 0, 0
	}
	size := g.fontSize
	if size <= 0 {
		size = 14
	}
	return int32(utf8.RuneCountInString(text)) * size * 3 / 5, size
}

func (g *GDIContext) SetFont(name string, size int32) {
	if g != nil {
		if name != "" {
			g.fontName = name
		}
		if size > 0 {
			g.fontSize = size
		}
	}
}
func (g *GDIContext) SetBkMode(_ bool) {}

func (g *GDIContext) SetClipRect(x, y, w, h int32) {
	if g == nil {
		return
	}
	// 硬性设置：丢弃已有的嵌套层级。
	g.clipStack = nil
	g.clip = image.Rect(int(x), int(y), int(x+w), int(y+h)).Intersect(image.Rect(0, 0, int(g.width), int(g.height)))
}

// PushClipRect intersects the requested rectangle with the active clip and
// remembers the previous level, so PopClip can restore it.
func (g *GDIContext) PushClipRect(x, y, w, h int32) {
	if g == nil {
		return
	}
	base := g.clip
	if base.Empty() {
		// 没有裁剪时以整个画布为基准。
		base = image.Rect(0, 0, int(g.width), int(g.height))
	}
	g.clipStack = append(g.clipStack, g.clip)
	g.clip = base.Intersect(image.Rect(int(x), int(y), int(x+w), int(y+h)))
	if g.clip.Empty() {
		// 空裁剪用画布外的一小块区域表示，避免退化成“不裁剪”。
		g.clip = image.Rect(-2, -2, -1, -1)
	}
}

func (g *GDIContext) PopClip() {
	if g == nil || len(g.clipStack) == 0 {
		return
	}
	g.clip = g.clipStack[len(g.clipStack)-1]
	g.clipStack = g.clipStack[:len(g.clipStack)-1]
}

func (g *GDIContext) ResetClip() {
	if g != nil {
		g.clip = image.Rectangle{}
		g.clipStack = nil
	}
}
func (g *GDIContext) SetTarget(_ uintptr) {}
func (g *GDIContext) Resize(_ uintptr, width, height int32) {
	if g == nil {
		return
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	g.width, g.height = width, height
	g.pixels = make([]uint32, int64(width)*int64(height))
}
func (g *GDIContext) Cleanup() {
	if g != nil {
		g.pixels = nil
	}
}

func (g *GDIContext) fillRect(x, y, w, h int32, c uint32) {
	if g == nil || w <= 0 || h <= 0 {
		return
	}
	r := image.Rect(int(x), int(y), int(x+w), int(y+h)).Intersect(image.Rect(0, 0, int(g.width), int(g.height)))
	if !g.clip.Empty() {
		r = r.Intersect(g.clip)
	}
	for yy := r.Min.Y; yy < r.Max.Y; yy++ {
		for xx := r.Min.X; xx < r.Max.X; xx++ {
			g.pixels[yy*int(g.width)+xx] = c
		}
	}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// Color converts the packed RGB values used by UI components into a standard
// Go color for integrations that expose the software surface.
func (g *GDIContext) Color(c uint32) color.RGBA {
	return color.RGBA{R: uint8(c), G: uint8(c >> 8), B: uint8(c >> 16), A: 0xff}
}
