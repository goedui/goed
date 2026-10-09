package components

import (
	"math"

	"github.com/goedui/goed/ui/renderer"
)

// GlyphPaint 是 24×24 设计网格上的单色矢量图标绘制函数。fg 是笔画色，x/y/size
// 是目标 DIP 矩形（内部按 size/24 缩放），底色不画（由调用方决定）。
// 组件节点同名（glyph_widget.go 的 Glyph），所以绘制函数带 Paint 后缀。
type GlyphPaint func(ctx renderer.Context, x, y, size float32, fg renderer.Color)

// glyphs 是引擎级通用图标集。应用层通过 Props.Icon 传名字消费（按钮、标签、
// empty 都认）；新图标在 init 前注册即可，不改消费方。
var glyphs = map[string]GlyphPaint{}

// RegisterGlyph 登记图标绘制函数，同名覆盖。供应用扩展自定义图标。
func RegisterGlyph(name string, g GlyphPaint) {
	if name == "" || g == nil {
		return
	}
	glyphs[name] = g
}

// PaintGlyph 按名字画图标，返回是否画了（名字不存在返回 false，调用方降级）。
func PaintGlyph(ctx renderer.Context, name string, x, y, size float32, fg renderer.Color) bool {
	g := glyphs[name]
	if g == nil || ctx == nil || size <= 0 {
		return false
	}
	g(ctx, x, y, size, fg)
	return true
}

// HasGlyph 报告图标名是否已登记。
func HasGlyph(name string) bool { _, ok := glyphs[name]; return ok }

func init() {
	RegisterGlyph("edit", paintGlyphEdit)
	RegisterGlyph("link", paintGlyphLink)
	RegisterGlyph("shield", paintGlyphShield)
	RegisterGlyph("box", paintGlyphBox)
	RegisterGlyph("user", paintGlyphUser)
	RegisterGlyph("save", paintGlyphSave)
	RegisterGlyph("trash", paintGlyphTrash)
	RegisterGlyph("eye", paintGlyphEye)
	RegisterGlyph("refresh", paintGlyphBolt)
	RegisterGlyph("sparkle", paintGlyphSparkle)
	RegisterGlyph("upload", paintGlyphUpload)
	RegisterGlyph("image", paintGlyphImage)
	RegisterGlyph("chevron", paintGlyphChevron)
}

// gx/gy 把 24 网格坐标换到目标矩形。线宽随尺寸缩放，16px 下约 1.4px。
func gx(x, size, u float32) float32 { return x + size*u/24 }
func gy(y, size, u float32) float32 { return y + size*u/24 }

func strokeW(size float32) float32 {
	w := size * 1.6 / 24
	if w < 1.15 {
		return 1.15
	}
	return w
}

func stroke(ctx renderer.Context, x, y, size float32, fg renderer.Color, pts ...float32) {
	strokePath(ctx, x, y, size, fg, false, pts...)
}

func strokePath(ctx renderer.Context, x, y, size float32, fg renderer.Color, closed bool, pts ...float32) {
	if len(pts) < 4 {
		return
	}
	path := make([]renderer.StrokePoint, 0, len(pts)/2)
	for i := 0; i+1 < len(pts); i += 2 {
		path = append(path, renderer.StrokePoint{X: gx(x, size, pts[i]), Y: gy(y, size, pts[i+1])})
	}
	if renderer.DrawRoundStroke(ctx, path, strokeW(size), fg, closed) {
		return
	}
	for i := 1; i < len(path); i++ {
		ctx.DrawLine(path[i-1].X, path[i-1].Y, path[i].X, path[i].Y, fg, strokeW(size))
	}
	if closed && len(path) > 2 {
		ctx.DrawLine(path[len(path)-1].X, path[len(path)-1].Y, path[0].X, path[0].Y, fg, strokeW(size))
	}
}

func strokeClosed(ctx renderer.Context, x, y, size float32, fg renderer.Color, pts ...float32) {
	strokePath(ctx, x, y, size, fg, true, pts...)
}

func strokeArc(ctx renderer.Context, x, y, size, cx, cy, r, a0, a1 float32, fg renderer.Color) {
	// 用两段三次贝塞尔逼近圆弧。后端没有圆头路径时退回短折线。
	if a1 < a0 {
		a0, a1 = a1, a0
	}
	span := a1 - a0
	if span > math.Pi*2 {
		span = math.Pi * 2
	}
	n := int(span/(math.Pi/2) + 0.999)
	if n < 1 {
		n = 1
	}
	path := make([]renderer.StrokePoint, 0, n+1)
	path = append(path, renderer.StrokePoint{
		X: gx(x, size, cx+r*float32(math.Cos(float64(a0)))),
		Y: gy(y, size, cy+r*float32(math.Sin(float64(a0)))),
	})
	for i := 0; i < n; i++ {
		t0 := a0 + span*float32(i)/float32(n)
		t1 := a0 + span*float32(i+1)/float32(n)
		k := float32(math.Tan(float64(t1-t0)/2)) * 4 / 3
		path = append(path, renderer.StrokePoint{
			Curve: true,
			C1X:   gx(x, size, cx+r*(float32(math.Cos(float64(t0)))-k*float32(math.Sin(float64(t0))))),
			C1Y:   gy(y, size, cy+r*(float32(math.Sin(float64(t0)))+k*float32(math.Cos(float64(t0))))),
			C2X:   gx(x, size, cx+r*(float32(math.Cos(float64(t1)))+k*float32(math.Sin(float64(t1))))),
			C2Y:   gy(y, size, cy+r*(float32(math.Sin(float64(t1)))-k*float32(math.Cos(float64(t1))))),
			X:     gx(x, size, cx+r*float32(math.Cos(float64(t1)))),
			Y:     gy(y, size, cy+r*float32(math.Sin(float64(t1)))),
		})
	}
	if renderer.DrawRoundStroke(ctx, path, strokeW(size), fg, false) {
		return
	}
	flat := flattenStroke(path)
	w := strokeW(size)
	for i := 1; i < len(flat); i++ {
		ctx.DrawLine(flat[i-1].X, flat[i-1].Y, flat[i].X, flat[i].Y, fg, w)
	}
}

func flattenStroke(path []renderer.StrokePoint) []renderer.StrokePoint {
	if len(path) == 0 {
		return nil
	}
	out := []renderer.StrokePoint{path[0]}
	for i := 1; i < len(path); i++ {
		p := path[i]
		if !p.Curve {
			out = append(out, p)
			continue
		}
		p0 := out[len(out)-1]
		for s := 1; s <= 8; s++ {
			t := float32(s) / 8
			out = append(out, renderer.StrokePoint{
				X: cubic(p0.X, p.C1X, p.C2X, p.X, t),
				Y: cubic(p0.Y, p.C1Y, p.C2Y, p.Y, t),
			})
		}
	}
	return out
}

func cubic(a, b, c, d, t float32) float32 {
	u := 1 - t
	return u*u*u*a + 3*u*u*t*b + 3*u*t*t*c + t*t*t*d
}

func dot(ctx renderer.Context, x, y, size, cx, cy, d float32, fg renderer.Color) {
	ctx.FillRoundedRect(gx(x, size, cx-d/2), gy(y, size, cy-d/2), size*d/24, size*d/24, size*d/48, fg)
}

// paintGlyphEdit 铅笔：斜笔身 + 笔尖，底部一条被编辑的线。
func paintGlyphEdit(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 14.5, 4.5, 19.5, 9.5, 8.5, 20.5, 3.5, 20.5, 3.5, 15.5, 14.5, 4.5)
	stroke(ctx, x, y, size, fg, 12.2, 6.8, 17.2, 11.8)
	stroke(ctx, x, y, size, fg, 3, 21.5, 9, 21.5)
}

// paintGlyphLink 两个开口链环。
func paintGlyphLink(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	strokeArc(ctx, x, y, size, 9, 15, 3.4, math.Pi*0.15, math.Pi*1.7, fg)
	strokeArc(ctx, x, y, size, 15, 9, 3.4, math.Pi*1.15, math.Pi*2.7, fg)
	stroke(ctx, x, y, size, fg, 9.6, 12.2, 14.4, 11.8)
}

// paintGlyphShield 盾牌轮廓 + 中线勾。
func paintGlyphShield(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg,
		12, 3.2, 19.2, 6, 19.2, 12, 12, 20.6, 4.8, 12, 4.8, 6, 12, 3.2)
	stroke(ctx, x, y, size, fg, 9.2, 11.4, 11.1, 13.4, 15.2, 8.6)
}

// paintGlyphBox 立方体：顶面菱形 + 左右侧面。
func paintGlyphBox(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 12, 3.2, 20.2, 7.4, 12, 11.6, 3.8, 7.4, 12, 3.2)
	stroke(ctx, x, y, size, fg, 12, 11.6, 12, 20.8)
	stroke(ctx, x, y, size, fg, 3.8, 7.4, 3.8, 16.6, 12, 20.8)
	stroke(ctx, x, y, size, fg, 20.2, 7.4, 20.2, 16.6, 12, 20.8)
}

// paintGlyphUser 头像：圆头 + 肩弧。
func paintGlyphUser(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	strokeArc(ctx, x, y, size, 12, 8.2, 3.3, 0, math.Pi*2, fg)
	strokeArc(ctx, x, y, size, 12, 18.6, 6.4, math.Pi*1.15, math.Pi*1.85, fg)
}

// paintGlyphSave 软盘轮廓 + 标签窗。
func paintGlyphSave(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	strokeClosed(ctx, x, y, size, fg, 5, 4, 16, 4, 20, 8, 20, 20, 4, 20, 4, 8)
	stroke(ctx, x, y, size, fg, 8, 4, 8, 9, 15, 9, 15, 4)
	strokeClosed(ctx, x, y, size, fg, 8, 13, 16, 13, 16, 20, 8, 20)
}

// paintGlyphTrash 垃圾桶：提手、盖、收腰桶身。
func paintGlyphTrash(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 9, 5, 9, 3.4, 15, 3.4, 15, 5)
	stroke(ctx, x, y, size, fg, 4.5, 5, 19.5, 5)
	stroke(ctx, x, y, size, fg, 6.2, 5, 7.4, 20, 16.6, 20, 17.8, 5)
	stroke(ctx, x, y, size, fg, 10, 9, 10.6, 17)
	stroke(ctx, x, y, size, fg, 14, 9, 13.4, 17)
}

// paintGlyphEye 杏仁眼 + 瞳孔。
func paintGlyphEye(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 2.4, 12, 7, 7.2, 12, 6.2, 17, 7.2, 21.6, 12, 17, 16.8, 12, 17.8, 7, 16.8, 2.4, 12)
	strokeArc(ctx, x, y, size, 12, 12, 2.1, 0, math.Pi*2, fg)
}

// paintGlyphBolt 闪电，刷新按钮用。
func paintGlyphBolt(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 13.2, 2.6, 6.4, 13.2, 11.2, 13.2, 9.6, 21.4, 17.6, 10.2, 12.4, 10.2, 13.2, 2.6)
}

// paintGlyphSparkle 一大一小两颗四角星。
func paintGlyphSparkle(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 10, 3, 11.3, 8.7, 17, 10, 11.3, 11.3, 10, 17, 8.7, 11.3, 3, 10, 8.7, 8.7, 10, 3)
	stroke(ctx, x, y, size, fg, 17.4, 15.2, 18, 17.2, 20, 17.8, 18, 18.4, 17.4, 20.4, 16.8, 18.4, 14.8, 17.8, 16.8, 17.2, 17.4, 15.2)
}

// paintGlyphUpload 托盘 + 向上箭头。
func paintGlyphUpload(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 5, 14, 5, 19, 19, 19, 19, 14)
	stroke(ctx, x, y, size, fg, 12, 15, 12, 4.5)
	stroke(ctx, x, y, size, fg, 7.6, 8.6, 12, 4.2, 16.4, 8.6)
}

// paintGlyphImage 圆角相框 + 太阳 + 山。
func paintGlyphImage(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	strokeClosed(ctx, x, y, size, fg, 4, 5.5, 20, 5.5, 20, 18.5, 4, 18.5)
	dot(ctx, x, y, size, 8.4, 9.2, 2.2, fg)
	stroke(ctx, x, y, size, fg, 4.4, 16.2, 9.2, 11.6, 13.2, 15.2, 16.2, 12.6, 19.6, 15.4)
}

// paintGlyphChevron 向下的小尖角，下拉按钮用。
func paintGlyphChevron(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
	stroke(ctx, x, y, size, fg, 6, 9.5, 12, 15.5, 18, 9.5)
}
