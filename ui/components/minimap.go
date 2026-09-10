package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// Minimap 代码缩略图组件
type Minimap struct {
	Width       int32
	ScaleFactor float32 // 每行的像素高度（通常 1-2）

	// 缓存
	needsRedraw bool
	cachedLines []string

	// 纵向拖动状态：按下点相对视口顶部的偏移（minimap 像素）
	grabOffsetPx float32
}

// NewMinimap 创建 Minimap
func NewMinimap() *Minimap {
	return &Minimap{
		Width:       100,
		ScaleFactor: 1.5, // 每行 1.5 像素
		needsRedraw: true,
	}
}

// Render 渲染 Minimap
func (m *Minimap) Render(ctx core.DrawingContext, x, y, height int32, lines []string, scrollLine, visibleLines int, scrollOffset ...int32) {
	th := theme.GetTheme()
	if height <= 0 || m.Width <= 4 {
		return
	}
	ctx.SetClipRect(x, y, m.Width, height)
	defer ctx.ResetClip()

	// 背景
	ctx.DrawRect(x, y, m.Width, height, th.EditorBg, true)

	// 左边框
	ctx.DrawRect(x, y, 1, height, th.WindowBorder, true)

	// 计算总的缩略图高度
	totalMinimapHeight := float32(len(lines)) * m.ScaleFactor

	// 计算缩放比例（如果内容超过高度）
	scale := float32(1.0)
	if totalMinimapHeight > float32(height) {
		scale = float32(height) / totalMinimapHeight
	}

	// 渲染每一行的简化表示
	lastLineY := int32(-1)
	for i, line := range lines {
		// 计算行的 Y 坐标
		lineY := y + int32(float32(i)*m.ScaleFactor*scale+0.5)
		nextY := y + int32(float32(i+1)*m.ScaleFactor*scale+0.5)
		if lineY == lastLineY {
			continue
		}
		lastLineY = lineY
		lineHeight := nextY - lineY

		if lineHeight < 1 {
			lineHeight = 1
		}

		// 根据行内容计算颜色密度
		// 绘制行条
		m.renderCodeLine(ctx, x+4, lineY, m.Width-8, lineHeight, line, th)
	}

	// 渲染可见区域高亮框
	offset := int32(0)
	if len(scrollOffset) > 0 {
		offset = scrollOffset[0]
	}
	viewportStartY := y + int32(float32(scrollLine)*m.ScaleFactor*scale) - int32(float32(offset)/24*m.ScaleFactor*scale)
	viewportHeight := int32(float32(visibleLines) * m.ScaleFactor * scale)
	if viewportStartY < y {
		viewportStartY = y
	}
	if viewportStartY+viewportHeight > y+height {
		viewportHeight = y + height - viewportStartY
	}
	if viewportHeight < 2 {
		viewportHeight = 2
	}

	// 绘制半透明矩形（这里简化为边框）
	m.drawViewportRect(ctx, x+2, viewportStartY, m.Width-4, viewportHeight, th.EditorSelectionBg)
}

// calculateLineDensity 计算行的字符密度
func (m *Minimap) renderCodeLine(ctx core.DrawingContext, x, y, width, height int32, line string, th *theme.Theme) {
	if width <= 0 || height <= 0 || len(line) == 0 {
		return
	}
	runes := []rune(line)
	unit := float32(width) / float32(len(runes))
	start := -1
	for i, r := range runes {
		nonSpace := r != ' ' && r != '\t'
		if nonSpace && start < 0 {
			start = i
		}
		if (!nonSpace || i == len(runes)-1) && start >= 0 {
			end := i
			if nonSpace && i == len(runes)-1 {
				end++
			}
			left := x + int32(float32(start)*unit+0.5)
			right := x + int32(float32(end)*unit+0.5)
			if right <= left {
				right = left + 1
			}
			if right > x+width {
				right = x + width
			}
			color := m.getColorByDensity(float32(end-start)/float32(len(runes)), th)
			ctx.DrawRect(left, y, right-left, height, color, true)
			start = -1
		}
	}
}

func (m *Minimap) calculateLineDensity(line string) float32 {
	if len(line) == 0 {
		return 0
	}

	// 简单计算：非空格字符的比例
	nonSpace := 0
	for _, ch := range line {
		if ch != ' ' && ch != '\t' {
			nonSpace++
		}
	}

	// 返回 0.0 - 1.0 的密度
	density := float32(nonSpace) / float32(len(line))
	if density > 1.0 {
		density = 1.0
	}

	return density
}

// getColorByDensity 根据密度返回行条颜色：在编辑器背景色与前景色之间按
// 密度插值。旧实现按固定系数缩放前景亮度，在 Light 等前景为纯黑的主题下
// 0×系数恒为 0，所有行条退化成死黑；插值对任意明暗主题都成立。
func (m *Minimap) getColorByDensity(density float32, th *theme.Theme) uint32 {
	fr := int32((th.EditorFg >> 16) & 0xFF)
	fgc := int32((th.EditorFg >> 8) & 0xFF)
	fb := int32(th.EditorFg & 0xFF)
	br := int32((th.EditorBg >> 16) & 0xFF)
	bgc := int32((th.EditorBg >> 8) & 0xFF)
	bb := int32(th.EditorBg & 0xFF)

	// 密度 0 → 30% 前景（稀疏代码仍可辨识），密度 1 → 90% 前景。
	alpha := 0.30 + density*0.60

	r := br + int32(float32(fr-br)*alpha)
	g := bgc + int32(float32(fgc-bgc)*alpha)
	b := bb + int32(float32(fb-bb)*alpha)

	return (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

// drawViewportRect 绘制可见区域矩形
func (m *Minimap) drawViewportRect(ctx core.DrawingContext, x, y, w, h int32, color uint32) {
	// 绘制四条边
	ctx.DrawRect(x, y, w, 2, color, true)     // 上
	ctx.DrawRect(x, y+h-2, w, 2, color, true) // 下
	ctx.DrawRect(x, y, 2, h, color, true)     // 左
	ctx.DrawRect(x+w-2, y, 2, h, color, true) // 右
}

// scaleFor 计算缩略图纵向缩放比例（内容超出可视高度时整体压缩）
func (m *Minimap) scaleFor(minimapHeight int32, totalLines int) float32 {
	total := float32(totalLines) * m.ScaleFactor
	if total > float32(minimapHeight) {
		return float32(minimapHeight) / total
	}
	return 1.0
}

// clampScroll 将滚动行号限制在 [0, totalLines-visibleLines]
func clampScroll(v, totalLines, visibleLines int) int {
	max := totalLines - visibleLines
	if max < 0 {
		max = 0
	}
	if v < 0 {
		v = 0
	}
	if v > max {
		v = max
	}
	return v
}

// HandleMouseDown 处理按下：点在视口内则原地开始拖动；
// 点在视口外则先以点击行为中心跳转（VSCode 行为）再开始拖动。
// 返回应生效的 scrollLine，由调用方应用。
func (m *Minimap) HandleMouseDown(relY, minimapHeight int32, totalLines, visibleLines, scrollLine int) int {
	rowPx := m.ScaleFactor * m.scaleFor(minimapHeight, totalLines)
	if rowPx <= 0 {
		return clampScroll(scrollLine, totalLines, visibleLines)
	}
	y := float32(relY)
	viewportTop := float32(scrollLine) * rowPx
	viewportH := float32(visibleLines) * rowPx

	if y < viewportTop || y >= viewportTop+viewportH {
		line := int(y/rowPx + 0.5)
		scrollLine = clampScroll(line-visibleLines/2, totalLines, visibleLines)
	}
	m.grabOffsetPx = y - float32(scrollLine)*rowPx
	return scrollLine
}

// HandleDragMove 拖动中：把鼠标纵坐标换算成 scrollLine
func (m *Minimap) HandleDragMove(relY, minimapHeight int32, totalLines, visibleLines int) int {
	rowPx := m.ScaleFactor * m.scaleFor(minimapHeight, totalLines)
	if rowPx <= 0 {
		return 0
	}
	line := int((float32(relY)-m.grabOffsetPx)/rowPx + 0.5)
	return clampScroll(line, totalLines, visibleLines)
}

// MarkDirty 标记需要重绘
func (m *Minimap) MarkDirty() {
	m.needsRedraw = true
}
