package windows

import (
	"github.com/goedui/goed/ui/core"
	"syscall"
)

func (g *GDIContext) DrawLine(x1, y1, x2, y2 int32, color uint32, width int32) {
	if g.memDC == 0 {
		return
	}
	pen := g.penFor(PS_SOLID, width, color)
	oldPen := SelectObject(g.memDC, pen)
	MoveToEx(g.memDC, x1, y1)
	LineTo(g.memDC, x2, y2)
	SelectObject(g.memDC, oldPen)
}

func (g *GDIContext) DrawRoundedRect(x, y, w, h, radius int32, color uint32, filled bool) {
	if g.memDC == 0 || w <= 0 || h <= 0 {
		return
	}
	if radius < 0 {
		radius = 0
	}
	if radius*2 > w {
		radius = w / 2
	}
	if radius*2 > h {
		radius = h / 2
	}

	pen := g.penFor(PS_SOLID, 1, color)
	if pen == 0 {
		return
	}
	oldPen := SelectObject(g.memDC, pen)
	var brush syscall.Handle
	var oldBrush syscall.Handle
	if filled {
		brush = g.brushFor(color)
	} else {
		brush = GetStockObject(HOLLOW_BRUSH)
	}
	if brush != 0 {
		oldBrush = SelectObject(g.memDC, brush)
	}
	RoundRect(g.memDC, x, y, x+w, y+h, radius*2, radius*2)
	if brush != 0 {
		SelectObject(g.memDC, oldBrush)
	}
	SelectObject(g.memDC, oldPen)
}

func (g *GDIContext) DrawEllipse(x, y, w, h int32, color uint32, filled bool) {
	if g.memDC == 0 || w <= 0 || h <= 0 {
		return
	}
	pen := g.penFor(PS_SOLID, 1, color)
	if pen == 0 {
		return
	}
	oldPen := SelectObject(g.memDC, pen)
	brush := g.brushFor(color)
	if !filled {
		brush = GetStockObject(HOLLOW_BRUSH)
	}
	oldBrush := SelectObject(g.memDC, brush)
	Ellipse(g.memDC, x, y, x+w, y+h)
	SelectObject(g.memDC, oldBrush)
	SelectObject(g.memDC, oldPen)
}

func (g *GDIContext) DrawPolygon(points []core.Point, color uint32, filled bool) {
	if g.memDC == 0 || len(points) < 2 {
		return
	}
	pts := make([]POINT, len(points))
	for i, p := range points {
		pts[i] = POINT{X: p.X, Y: p.Y}
	}
	pen := g.penFor(PS_SOLID, 1, color)
	oldPen := SelectObject(g.memDC, pen)
	var oldBrush syscall.Handle
	if filled {
		oldBrush = SelectObject(g.memDC, g.brushFor(color))
	} else {
		oldBrush = SelectObject(g.memDC, GetStockObject(HOLLOW_BRUSH))
	}
	Polygon(g.memDC, &pts[0], int32(len(pts)))
	SelectObject(g.memDC, oldBrush)
	SelectObject(g.memDC, oldPen)
}
