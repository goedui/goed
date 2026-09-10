package components

import (
	"github.com/goedui/goed/ui/composables"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// FoldGutter 折叠侧边栏组件
type FoldGutter struct {
	Width   int32
	Manager *composables.FoldingManager

	// 事件回调
	OnToggleFold func(line int)
}

// NewFoldGutter 创建折叠侧边栏
func NewFoldGutter(manager *composables.FoldingManager) *FoldGutter {
	return &FoldGutter{
		Width:   20,
		Manager: manager,
	}
}

// Render 渲染折叠侧边栏
func (fg *FoldGutter) Render(ctx core.DrawingContext, x, y int32, height int32, scrollLine int, lineHeight int32, visibleLines int) {
	visible := make([]int, 0, visibleLines)
	for i := 0; i < visibleLines; i++ {
		visible = append(visible, scrollLine+i)
	}
	fg.RenderVisible(ctx, x, y, height, visible, lineHeight, 0)
}

// RenderVisible renders exactly the already-resolved visible document lines.
// The editor supplies this slice so folded regions and scrolling do not force
// the gutter to rescan all preceding lines.
func (fg *FoldGutter) RenderVisible(ctx core.DrawingContext, x, y int32, height int32, visible []int, lineHeight, pixelOffset int32) {
	th := theme.GetTheme()

	// 背景
	ctx.DrawRect(x, y, fg.Width, height, th.EditorBg, true)

	// 右边框
	ctx.DrawRect(x+fg.Width-1, y, 1, height, th.WindowBorder, true)

	// 渲染折叠图标（统一走 icons.go 的 DrawChevron）
	for i, lineNum := range visible {
		region := fg.Manager.GetFoldRegionAt(lineNum)

		if region != nil {
			iconY := y + int32(i)*lineHeight - pixelOffset + lineHeight/2 - 4
			iconX := x + fg.Width/2 - 4

			dir := ChevronDown
			if region.Folded {
				dir = ChevronRight
			}
			DrawChevron(ctx, iconX+4, iconY+4, dir, th.EditorFg)
		}
	}
}

// HandleClick 处理点击事件
func (fg *FoldGutter) HandleClick(x, y int32, scrollLine int, lineHeight int32) {
	// 计算点击的行号
	lineNum := scrollLine + int(y/lineHeight)

	// 检查是否点击了折叠区域
	region := fg.Manager.GetFoldRegionAt(lineNum)
	if region != nil {
		fg.Manager.ToggleFold(lineNum)

		// 触发回调
		if fg.OnToggleFold != nil {
			fg.OnToggleFold(lineNum)
		}
	}
}

// HandleClickVisible 在编辑器提供的可见行窗口内解析点击行。窗口与
// RenderVisible 的几何一致（含像素偏移），避免折叠/滚动后重扫全文档。
func (fg *FoldGutter) HandleClickVisible(visible []int, y int32, lineHeight int32, pixelOffset int32) {
	if lineHeight <= 0 || len(visible) == 0 {
		return
	}
	row := int((y + pixelOffset) / lineHeight)
	if row < 0 {
		row = 0
	}
	if row >= len(visible) {
		row = len(visible) - 1
	}
	lineNum := visible[row]

	region := fg.Manager.GetFoldRegionAt(lineNum)
	if region != nil {
		fg.Manager.ToggleFold(lineNum)
		if fg.OnToggleFold != nil {
			fg.OnToggleFold(lineNum)
		}
	}
}
