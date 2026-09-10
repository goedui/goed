package components

import "github.com/goedui/goed/ui/core"

// 共享几何图标绘制（见 图标资源.md 第 4/5 节）。
// 约定：所有图标以中心点 (cx, cy) 定位，1px 线宽，颜色由调用方
// 从主题前景色中选择，本文件不做任何主题查询。

// DrawCloseIcon 绘制 8×8 关闭图标（两条对角线）。
// 逐像素绘制而非 DrawLine：GDI 线段端点包含规则会让斜线视觉上
// 一端宽一端窄；逐像素保证 X 完美对称，且尺寸更精致。
func DrawCloseIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	const half int32 = 4
	for i := -half; i <= half; i++ {
		// 主对角线 ↘ 与副对角线 ↙，共用交点 (cx, cy)。
		ctx.DrawRect(cx+i, cy+i, 1, 1, color, true)
		ctx.DrawRect(cx+i, cy-i, 1, 1, color, true)
	}
}

// DrawMinimizeIcon 绘制最小化图标（水平居中线，10×1，位于中心 cy）。
func DrawMinimizeIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawRect(cx-5, cy, 10, 1, color, true)
}

// DrawMaximizeIcon 绘制最大化图标（10×10 空心矩形，中心对称）。
func DrawMaximizeIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawRect(cx-5, cy-5, 11, 11, color, false)
}

// DrawRestoreIcon 绘制还原图标（两个错位空心矩形）。
func DrawRestoreIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawRect(cx-5, cy-3, 8, 8, color, false)
	ctx.DrawLine(cx-3, cy-5, cx+5, cy-5, color, 1)
	ctx.DrawLine(cx+5, cy-5, cx+5, cy+3, color, 1)
}

// ChevronDir 折叠箭头方向。
type ChevronDir int

const (
	ChevronRight ChevronDir = iota
	ChevronDown
	ChevronUp
	ChevronLeft
)

// DrawChevron 绘制实心小三角折叠箭头（9×8 基准，居中于 cx,cy）。
func DrawChevron(ctx core.DrawingContext, cx, cy int32, dir ChevronDir, color uint32) {
	pts := make([]core.Point, 0, 3)
	switch dir {
	case ChevronRight:
		pts = append(pts, core.Point{X: cx - 3, Y: cy - 4}, core.Point{X: cx - 3, Y: cy + 4}, core.Point{X: cx + 4, Y: cy})
	case ChevronDown:
		pts = append(pts, core.Point{X: cx - 4, Y: cy - 3}, core.Point{X: cx + 4, Y: cy - 3}, core.Point{X: cx, Y: cy + 4})
	case ChevronUp:
		pts = append(pts, core.Point{X: cx - 4, Y: cy + 3}, core.Point{X: cx + 4, Y: cy + 3}, core.Point{X: cx, Y: cy - 4})
	case ChevronLeft:
		pts = append(pts, core.Point{X: cx + 3, Y: cy - 4}, core.Point{X: cx + 3, Y: cy + 4}, core.Point{X: cx - 4, Y: cy})
	}
	ctx.DrawPolygon(pts, color, true)
}

// DrawNavUpIcon 绘制"上一处"折线箭头（chevron 形 ∧，非文字箭头）。
func DrawNavUpIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawLine(cx-4, cy+2, cx, cy-2, color, 1)
	ctx.DrawLine(cx, cy-2, cx+4, cy+2, color, 1)
}

// DrawNavDownIcon 绘制"下一处"折线箭头（chevron 形 ∨）。
func DrawNavDownIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawLine(cx-4, cy-2, cx, cy+2, color, 1)
	ctx.DrawLine(cx, cy+2, cx+4, cy-2, color, 1)
}

// DrawPlusIcon 绘制新增十字（10×10）。
func DrawPlusIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawLine(cx-5, cy, cx+5, cy, color, 1)
	ctx.DrawLine(cx, cy-5, cx, cy+5, color, 1)
}

// DrawDotIcon 绘制实心圆状态标记（半径 3）。
func DrawDotIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawEllipse(cx-3, cy-3, 6, 6, color, true)
}

// DrawReplaceOneIcon 绘制替换单个图标：小矩形（被替换文本）+ 向右入箭头。
func DrawReplaceOneIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawRect(cx-5, cy-3, 6, 6, color, false)
	ctx.DrawLine(cx+1, cy, cx+5, cy, color, 1)
	ctx.DrawLine(cx+3, cy-2, cx+5, cy, color, 1)
	ctx.DrawLine(cx+3, cy+2, cx+5, cy, color, 1)
}

// DrawReplaceAllIcon 绘制全部替换图标：双矩形（多处文本）+ 箭头。
func DrawReplaceAllIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawRect(cx-6, cy-3, 5, 6, color, false)
	ctx.DrawRect(cx-1, cy-3, 5, 6, color, false)
	ctx.DrawLine(cx+4, cy, cx+6, cy, color, 1)
	ctx.DrawLine(cx+5, cy-1, cx+6, cy, color, 1)
	ctx.DrawLine(cx+5, cy+1, cx+6, cy, color, 1)
}

// DrawSearchIcon 绘制搜索放大镜（圆环 + 45° 手柄）。
func DrawSearchIcon(ctx core.DrawingContext, cx, cy int32, color uint32) {
	ctx.DrawEllipse(cx-4, cy-4, 8, 8, color, false)
	ctx.DrawLine(cx+3, cy+3, cx+6, cy+6, color, 1)
}
