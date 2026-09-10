package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// ContextMenu 右键上下文菜单组件
type ContextMenu struct {
	Items      []MenuItem
	X, Y       int32  // 菜单位置
	Width      int32  // 菜单宽度
	ItemHeight int32  // 菜单项高度
	IsVisible  bool   // 是否显示
	hoveredIdx int    // 悬停项索引
	OnClose    func() // 关闭回调
}

// NewContextMenu 创建上下文菜单
func NewContextMenu() *ContextMenu {
	return &ContextMenu{
		Items:      []MenuItem{},
		Width:      200,
		ItemHeight: 26,
		IsVisible:  false,
		hoveredIdx: -1,
	}
}

// Show 显示菜单
func (cm *ContextMenu) Show(x, y int32, items []MenuItem) {
	cm.X = x
	cm.Y = y
	cm.Items = items
	cm.IsVisible = true
	cm.hoveredIdx = -1
}

// ShowClamped 显示菜单，并在超出窗口边界（boundW/boundH 为客户区尺寸）时
// 向内收纳位置，供树右缘、底缘等弹出场景使用。
func (cm *ContextMenu) ShowClamped(x, y, boundW, boundH int32, items []MenuItem) {
	cm.Show(x, y, items)
	height := int32(len(items)) * cm.ItemHeight
	if cm.X+cm.Width > boundW {
		cm.X = boundW - cm.Width
	}
	if cm.Y+height > boundH {
		cm.Y = boundH - height
	}
	if cm.X < 0 {
		cm.X = 0
	}
	if cm.Y < 0 {
		cm.Y = 0
	}
}

// Hide 隐藏菜单
func (cm *ContextMenu) Hide() {
	cm.IsVisible = false
	cm.hoveredIdx = -1
	if cm.OnClose != nil {
		cm.OnClose()
	}
}

// Render 渲染上下文菜单
func (cm *ContextMenu) Render(ctx core.DrawingContext) {
	if !cm.IsVisible || len(cm.Items) == 0 {
		return
	}

	th := theme.GetTheme()

	// 计算菜单高度
	height := int32(len(cm.Items)) * cm.ItemHeight

	// 背景 - VSCode Dark+ 菜单背景
	ctx.DrawRect(cm.X, cm.Y, cm.Width, height, th.MenuDropdownBg, true)

	// 边框
	borderColor := th.MenuBorder
	ctx.DrawRect(cm.X, cm.Y, cm.Width, 1, borderColor, true)          // 上
	ctx.DrawRect(cm.X, cm.Y+height-1, cm.Width, 1, borderColor, true) // 下
	ctx.DrawRect(cm.X, cm.Y, 1, height, borderColor, true)            // 左
	ctx.DrawRect(cm.X+cm.Width-1, cm.Y, 1, height, borderColor, true) // 右

	// 字体阶梯：Body 档（字体规范.md 第 4 节）。
	ctx.SetFont(th.UIFontFamily, th.FontBody())

	// 渲染菜单项
	for i, item := range cm.Items {
		itemY := cm.Y + int32(i)*cm.ItemHeight

		if item.Separator {
			// 分隔线
			ctx.DrawRect(cm.X+8, itemY+cm.ItemHeight/2, cm.Width-16, 1, th.MenuSeparator, true)
			continue
		}

		// 悬停背景
		if i == cm.hoveredIdx && item.Enabled {
			ctx.DrawRect(cm.X+1, itemY+1, cm.Width-2, cm.ItemHeight-2, th.MenuItemHoverBg, true)
		}

		// 文字颜色
		textColor := th.MenuFg
		if !item.Enabled {
			textColor = th.MenuDisabledFg
		}

		// 标签
		textY := itemY + (cm.ItemHeight-14)/2
		ctx.DrawText(cm.X+12, textY, item.Label, textColor)

		// 快捷键提示（右对齐）
		if item.Shortcut != "" {
			shortcutX := cm.X + cm.Width - int32(len(item.Shortcut)*7) - 12
			ctx.DrawText(shortcutX, textY, item.Shortcut, th.MenuShortcutFg)
		}
	}
}

// RenderAt draws the menu in the coordinate space of its owning component.
// Hit testing remains local while nested editors can paint at screen position.
func (cm *ContextMenu) RenderAt(ctx core.DrawingContext, offsetX, offsetY int32) {
	if offsetX == 0 && offsetY == 0 {
		cm.Render(ctx)
		return
	}
	x, y := cm.X, cm.Y
	cm.X, cm.Y = x+offsetX, y+offsetY
	cm.Render(ctx)
	cm.X, cm.Y = x, y
}

// HandleMouseMove 处理鼠标移动
func (cm *ContextMenu) HandleMouseMove(x, y int32) bool {
	if !cm.IsVisible {
		return false
	}

	oldHovered := cm.hoveredIdx
	cm.hoveredIdx = -1

	// 检查是否在菜单范围内
	height := int32(len(cm.Items)) * cm.ItemHeight
	if x >= cm.X && x < cm.X+cm.Width && y >= cm.Y && y < cm.Y+height {
		itemIndex := int((y - cm.Y) / cm.ItemHeight)
		if itemIndex >= 0 && itemIndex < len(cm.Items) {
			// 不悬停分隔线
			if !cm.Items[itemIndex].Separator {
				cm.hoveredIdx = itemIndex
			}
		}
	}

	return oldHovered != cm.hoveredIdx
}

// HandleMouseDown 处理鼠标点击
func (cm *ContextMenu) HandleMouseDown(x, y int32) bool {
	if !cm.IsVisible {
		return false
	}

	// 检查是否在菜单范围内
	height := int32(len(cm.Items)) * cm.ItemHeight
	if x >= cm.X && x < cm.X+cm.Width && y >= cm.Y && y < cm.Y+height {
		itemIndex := int((y - cm.Y) / cm.ItemHeight)
		if itemIndex >= 0 && itemIndex < len(cm.Items) {
			item := cm.Items[itemIndex]
			if !item.Separator && item.Enabled && item.OnClick != nil {
				item.OnClick()
			}
		}
		cm.Hide()
		return true
	}

	// 点击菜单外部，关闭菜单
	cm.Hide()
	return false
}

// IsHit 判断坐标是否在菜单内
func (cm *ContextMenu) IsHit(x, y int32) bool {
	if !cm.IsVisible {
		return false
	}

	height := int32(len(cm.Items)) * cm.ItemHeight
	return x >= cm.X && x < cm.X+cm.Width && y >= cm.Y && y < cm.Y+height
}
