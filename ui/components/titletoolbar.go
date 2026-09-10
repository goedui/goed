package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// TitleToolbar 标题栏+工具栏合并组件（类 VSCode）
type TitleToolbar struct {
	Title      string
	OnMinimize func()
	OnMaximize func()
	OnClose    func()
	OnNew      func()
	OnOpen     func()
	OnSave     func()
	OnUndo     func()
	OnRedo     func()

	menuHover   int // 0=none, 1=文件, 2=编辑, 3=选择, 4=查看
	btnHover    int // 按钮悬停索引
	winBtnHover int // -1=none, 0=min, 1=max, 2=close

	width int32
}

func NewTitleToolbar(title string) *TitleToolbar {
	return &TitleToolbar{
		Title:       title,
		menuHover:   0,
		btnHover:    -1,
		winBtnHover: -1,
	}
}

func (t *TitleToolbar) SetWidth(width int32) {
	if width > 0 {
		t.width = width
	}
}

func (t *TitleToolbar) Render(ctx core.DrawingContext, x, y, width int32) {
	t.width = width
	th := theme.GetTheme()
	// 字体阶梯：Caption 档（chrome 层，与菜单栏一致）。
	ctx.SetFont(th.UIFontFamily, th.FontCaption())

	// 背景
	ctx.DrawRect(x, y, width, 40, th.TitleBarBg, true)

	// === 左侧区域 ===
	xPos := x + 8

	// App 图标（16x16 占位符）
	ctx.DrawRect(xPos, y+12, 16, 16, th.StatusBarBg, false)
	xPos += 24

	// 菜单栏
	menus := []string{"文件", "编辑", "选择", "查看"}
	for i, menu := range menus {
		menuWidth := int32(48)

		// 悬停背景
		if t.menuHover == i+1 {
			ctx.DrawRect(xPos, y, menuWidth, 40, th.TitleBarBtnHover, true)
		}

		// 文字
		ctx.DrawText(xPos+12, y+12, menu, th.TitleBarFg)
		xPos += menuWidth
	}

	xPos += 8 // 间距

	// 工具按钮
	buttons := []string{"新建", "打开", "保存", "撤销", "重做"}
	for i, btn := range buttons {
		btnWidth := int32(48)

		// 悬停背景
		if t.btnHover == i {
			ctx.DrawRect(xPos, y, btnWidth, 40, th.TitleBarBtnHover, true)
		}

		// 文字
		textWidth, _ := ctx.MeasureText(btn)
		ctx.DrawText(xPos+(btnWidth-textWidth)/2, y+12, btn, th.TitleBarFg)
		xPos += btnWidth
	}

	// === 右侧区域 ===

	// Windows 三按钮（从右往左）
	btnWidth := int32(46)
	xClose := x + width - btnWidth
	xMax := xClose - btnWidth
	xMin := xMax - btnWidth

	// 关闭按钮（悬停红色）
	if t.winBtnHover == 2 {
		ctx.DrawRect(xClose, y, btnWidth, 40, th.TitleBarCloseHover, true)
	} else {
		ctx.DrawRect(xClose, y, btnWidth, 40, th.TitleBarBg, true)
	}
	DrawCloseIcon(ctx, xClose+btnWidth/2, y+20, th.TitleBarFg)

	// 最大化按钮
	if t.winBtnHover == 1 {
		ctx.DrawRect(xMax, y, btnWidth, 40, th.TitleBarBtnHover, true)
	}
	DrawMaximizeIcon(ctx, xMax+btnWidth/2, y+20, th.TitleBarFg)

	// 最小化按钮
	if t.winBtnHover == 0 {
		ctx.DrawRect(xMin, y, btnWidth, 40, th.TitleBarBtnHover, true)
	}
	DrawMinimizeIcon(ctx, xMin+btnWidth/2, y+20, th.TitleBarFg)

	// 搜索框（在 Windows 按钮左侧）
	searchX := xMin - 128
	searchWidth := int32(120)
	ctx.DrawRect(searchX, y+8, searchWidth, 24, th.SearchBg, true)
	ctx.DrawRect(searchX, y+8, searchWidth, 24, th.SearchBorder, false)
	ctx.DrawText(searchX+8, y+12, "搜索...", th.SearchFg)
}

func (t *TitleToolbar) HandleMouseMove(x, y int32) {
	if y >= 40 {
		t.menuHover = 0
		t.btnHover = -1
		t.winBtnHover = -1
		return
	}

	// 检测菜单悬停
	menuX := int32(32)
	for i := 0; i < 4; i++ {
		if x >= menuX && x < menuX+48 {
			t.menuHover = i + 1
			t.btnHover = -1
			t.winBtnHover = -1
			return
		}
		menuX += 48
	}

	// 检测工具按钮悬停
	btnX := menuX + 8
	for i := 0; i < 5; i++ {
		if x >= btnX && x < btnX+48 {
			t.btnHover = i
			t.menuHover = 0
			t.winBtnHover = -1
			return
		}
		btnX += 48
	}

	// 检测 Windows 按钮
	btnWidth := int32(46)
	xClose := t.width - btnWidth
	xMax := xClose - btnWidth
	xMin := xMax - btnWidth

	if x >= xClose {
		t.winBtnHover = 2 // 关闭
	} else if x >= xMax {
		t.winBtnHover = 1 // 最大化
	} else if x >= xMin {
		t.winBtnHover = 0 // 最小化
	} else {
		t.winBtnHover = -1
	}

	if t.winBtnHover != -1 {
		t.menuHover = 0
		t.btnHover = -1
	}
}

func (t *TitleToolbar) HandleMouseDown(x, y int32) bool {
	if y >= 40 {
		return false
	}
	// A click can arrive before the first hover message, so derive the hit
	// target from the coordinates before dispatching the action.
	t.HandleMouseMove(x, y)

	// Windows 按钮点击
	if t.winBtnHover == 2 && t.OnClose != nil {
		t.OnClose()
		return true
	}
	if t.winBtnHover == 1 && t.OnMaximize != nil {
		t.OnMaximize()
		return true
	}
	if t.winBtnHover == 0 && t.OnMinimize != nil {
		t.OnMinimize()
		return true
	}

	// 工具按钮点击
	if t.btnHover >= 0 {
		switch t.btnHover {
		case 0:
			if t.OnNew != nil {
				t.OnNew()
			}
		case 1:
			if t.OnOpen != nil {
				t.OnOpen()
			}
		case 2:
			if t.OnSave != nil {
				t.OnSave()
			}
		case 3:
			if t.OnUndo != nil {
				t.OnUndo()
			}
		case 4:
			if t.OnRedo != nil {
				t.OnRedo()
			}
		}
		return true
	}

	return false
}

func (t *TitleToolbar) IsDraggable(x, y int32) bool {
	if y >= 40 {
		return false
	}

	// Windows 按钮区域不可拖动
	btnWidth := int32(46)
	if x >= t.width-btnWidth*3 {
		return false
	}

	// 搜索框区域不可拖动
	searchX := t.width - btnWidth*3 - 128
	if x >= searchX && x < searchX+120 {
		return false
	}

	// 工具按钮区域不可拖动
	if x >= 32 && x < 32+4*48 {
		return false
	}
	if x >= 32+4*48+8 && x < 32+4*48+8+5*48 {
		return false
	}

	// 其他区域可拖动
	return true
}
