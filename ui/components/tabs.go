package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
	"path/filepath"
)

type Tab struct {
	Path, Title string
	// Draft 保存该标签未写盘的内容快照（编辑器切换标签时存档），
	// 空串表示无未保存草稿。窗口热退出时据此恢复。
	Draft string
	Modified    bool
	Stale       bool // 文件在磁盘上被外部修改，等待用户处理
	Status      FileStatus
}

// Tabs is a reusable document tab strip shared by the editor and agent views.
type Tabs struct {
	Items     []Tab
	Active    int
	Height    int32
	OnSelect  func(path string)
	OnClose   func(path string)
	OnNew     func()             // 新建标签页回调
	OnReorder func(from, to int) // 标签页重新排序回调

	// 横向溢出滚动：ScrollX 为内容左移像素；箭头几何在 Render 时同步。
	ScrollX       int32
	leftArrow     core.Rect
	rightArrow    core.Rect
	leftArrowHot  bool
	rightArrowHot bool
	arrowsVisible bool

	// 右键上下文：OnContextMenu 依据命中的标签动态返回菜单项。
	OnContextMenu func(tab *Tab, x, y int32) []MenuItem
	ContextMenu   *ContextMenu

	// 拖拽状态
	dragging     bool
	dragIndex    int
	dragStartX   int32
	dragStartY   int32
	dragCurrentX int32
	dragCurrentY int32
}

// 横向滚动箭头宽度与滚轮步长（px/档）。
const (
	tabArrowWidth    = 20
	tabScrollStep    = 90
	plusButtonWidth  = 36
)

func NewTabs() *Tabs {
	return &Tabs{Active: -1, Height: 36, ContextMenu: NewContextMenu()}
}

// tabWidth 单个标签的固定宽度（标题长则加宽）。
func tabWidth(title string) int32 {
	if len(title) > 14 {
		return 180
	}
	return 150
}

// tabsTotalWidth 所有标签的总宽度。
func (t *Tabs) tabsTotalWidth() int32 {
	var total int32
	for i := range t.Items {
		total += tabWidth(t.Items[i].Title)
	}
	return total
}

// clampScrollX 把 ScrollX 限制在 [0, 溢出量]。
func (t *Tabs) clampScrollX(availableWidth int32) {
	max := t.tabsTotalWidth() - availableWidth
	if max < 0 {
		max = 0
	}
	if t.ScrollX > max {
		t.ScrollX = max
	}
	if t.ScrollX < 0 {
		t.ScrollX = 0
	}
}

func (t *Tabs) Open(path string) {
	for i := range t.Items {
		if t.Items[i].Path == path {
			t.Active = i
			return
		}
	}
	t.Items = append(t.Items, Tab{Path: path, Title: filepath.Base(path)})
	t.Active = len(t.Items) - 1
}

// SetExternalChange 标记/清除标签页的外部修改警示（⚠）。
func (t *Tabs) SetExternalChange(path string, stale bool) {
	for i := range t.Items {
		if t.Items[i].Path == path {
			t.Items[i].Stale = stale
			return
		}
	}
}

// Close removes a tab from the strip and keeps the active index valid.
// Associated document/session cleanup is owned by the caller.
func (t *Tabs) Close(path string) bool {
	for i := range t.Items {
		if t.Items[i].Path != path {
			continue
		}
		t.Items = append(t.Items[:i], t.Items[i+1:]...)
		switch {
		case len(t.Items) == 0:
			t.Active = -1
		case t.Active > i:
			t.Active--
		case t.Active >= len(t.Items):
			t.Active = len(t.Items) - 1
		}
		return true
	}
	return false
}
func (t *Tabs) SetModified(path string, modified bool) {
	for i := range t.Items {
		if t.Items[i].Path == path {
			t.Items[i].Modified = modified
			if modified {
				t.Items[i].Status = FileStatusModified
			} else if t.Items[i].Status == FileStatusModified {
				t.Items[i].Status = FileStatusNone
			}
		}
	}
}
func (t *Tabs) SetStatus(path string, status FileStatus) {
	for i := range t.Items {
		if t.Items[i].Path == path {
			t.Items[i].Status = status
			t.Items[i].Modified = status == FileStatusModified
		}
	}
}
// IndexOf 返回指定路径标签的下标，不存在返回 -1。
func (t *Tabs) IndexOf(path string) int {
	for i := range t.Items {
		if t.Items[i].Path == path {
			return i
		}
	}
	return -1
}

func (t *Tabs) ActivePath() string {
	if t.Active >= 0 && t.Active < len(t.Items) {
		return t.Items[t.Active].Path
	}
	return ""
}

func (t *Tabs) Render(ctx core.DrawingContext, x, y, width int32) {
	th := theme.GetTheme()
	ctx.DrawRect(x, y, width, t.Height, th.WindowBg, true)
	ctx.DrawRect(x, y+t.Height-1, width, 1, th.WindowBorder, true)

	// 字体阶梯：Tabs 档（字体规范.md 第 4 节）。
	ctx.SetFont(th.UIFontFamily, th.FontTabs())

	// 为"+"按钮预留空间
	availableWidth := width - plusButtonWidth

	// 横向溢出：夹紧滚动量并启用箭头与裁剪。
	t.clampScrollX(availableWidth)
	t.arrowsVisible = t.tabsTotalWidth() > availableWidth

	// 标签绘制区裁剪（避免滚出条带的标签覆盖"+"按钮或箭头外区域）。
	if t.arrowsVisible {
		if clip, ok := ctx.(editorClipStack); ok {
			clip.PushClipRect(x, y, availableWidth, t.Height)
		} else {
			ctx.SetClipRect(x, y, availableWidth, t.Height)
		}
	}

	cursor := x - t.ScrollX
	for i, item := range t.Items {
		w := tabWidth(item.Title)
		if cursor+w < x || cursor > x+availableWidth {
			cursor += w
			continue
		}
		if i == t.Active {
			ctx.DrawRect(cursor, y, w, t.Height-1, th.EditorCurrentLineBg, true)
			ctx.DrawRect(cursor, y+t.Height-2, w, 2, th.ButtonBg, true)
		}
		ctx.DrawText(cursor+12, y+10, item.Title, th.TextPrimary)
		statusBg := th.WindowBg
		if i == t.Active {
			statusBg = th.EditorCurrentLineBg
		}
		ctx.DrawRect(cursor+w-48, y+2, 48, t.Height-4, statusBg, true)

		// 标签尾部分区布局（互不重叠）：
		//   [状态位 w-32] [关闭钮 w-14]
		// 状态标记画在状态位；关闭钮固定在右缘，两者间隔 18px。
		if item.Stale {
			// 外部修改警示优先于未保存圆点展示（实心圆，冲突色）。
			DrawDotIcon(ctx, cursor+w-32, y+t.Height/2, th.TabStatusConflict)
		} else if item.Status == FileStatusReadOnly {
			// 只读锁定：小空心矩形（语义化几何标记）。
			ctx.DrawRect(cursor+w-35, y+t.Height/2-3, 6, 6, th.TextSecondary, false)
		} else if item.Modified {
			DrawDotIcon(ctx, cursor+w-32, y+t.Height/2, th.TabStatusModified)
		}
		// git 状态语义点与未保存点共用状态位（renderStatusDot 内部判重）。
		t.renderStatusDot(ctx, cursor, y, w, item)
		DrawCloseIcon(ctx, cursor+w-14, y+t.Height/2, th.TextSecondary)
		cursor += w
	}

	if t.arrowsVisible {
		if clip, ok := ctx.(editorClipStack); ok {
			clip.PopClip()
		} else {
			ctx.ResetClip()
		}
	}

	// 溢出时的左右滚动箭头（覆盖条带两端，最后绘制保证可见）。
	if t.arrowsVisible {
		if t.ScrollX > 0 {
			t.leftArrow = core.Rect{X: x, Y: y, W: tabArrowWidth, H: t.Height - 1}
			bg := th.WindowBg
			if t.leftArrowHot {
				bg = th.MenuItemHoverBg
			}
			ctx.DrawRect(t.leftArrow.X, t.leftArrow.Y, t.leftArrow.W, t.leftArrow.H, bg, true)
			DrawChevron(ctx, t.leftArrow.X+t.leftArrow.W/2, y+t.Height/2, ChevronLeft, th.TextPrimary)
		}
		if t.ScrollX < t.tabsTotalWidth()-availableWidth {
			t.rightArrow = core.Rect{X: x + availableWidth - tabArrowWidth, Y: y, W: tabArrowWidth, H: t.Height - 1}
			bg := th.WindowBg
			if t.rightArrowHot {
				bg = th.MenuItemHoverBg
			}
			ctx.DrawRect(t.rightArrow.X, t.rightArrow.Y, t.rightArrow.W, t.rightArrow.H, bg, true)
			DrawChevron(ctx, t.rightArrow.X+t.rightArrow.W/2, y+t.Height/2, ChevronRight, th.TextPrimary)
		}
	}

	// 绘制"+"按钮
	if t.OnNew != nil {
		plusX := x + availableWidth
		plusY := y
		// 绘制按钮背景（悬停时高亮）
		ctx.DrawRect(plusX, plusY, plusButtonWidth, t.Height-1, th.WindowBg, true)
		// "+"几何十字
		DrawPlusIcon(ctx, plusX+plusButtonWidth/2, y+t.Height/2, th.TextPrimary)
	}
}

func (t *Tabs) renderStatusDot(ctx core.DrawingContext, x, y, width int32, item Tab) {
	// 外层已画过未保存/警示/只读标记时跳过，保证状态位只有一个圆。
	if item.Stale || item.Modified || item.Status == FileStatusReadOnly {
		return
	}
	status := item.Status
	th := theme.GetTheme()
	var color uint32
	switch status {
	case FileStatusModified:
		color = th.TabStatusModified
	case FileStatusUntracked:
		color = th.TabStatusUntracked
	case FileStatusIgnored:
		color = th.TabStatusIgnored
	case FileStatusReadOnly:
		color = th.TabStatusReadOnly
	case FileStatusConflict:
		color = th.TabStatusConflict
	default:
		return
	}
	// 与未保存/警示点共用状态位（w-32），同一标签只画一个状态圆。
	// git 语义状态优先级高于未保存点，因此先于调用方的 Modified 圆判断。
	ctx.DrawRoundedRect(x+width-36, y+t.Height/2-3, 6, 6, 3, color, true)
}

func (t *Tabs) HandleMouseDown(x, y, originX, originY, width int32) bool {
	if x < originX || x >= originX+width || y < originY || y >= originY+t.Height {
		return false
	}

	availableWidth := width - plusButtonWidth

	// 溢出时滚动箭头优先命中（箭头覆盖条带两端）。
	if t.arrowsVisible && contains(t.leftArrow, x, y) {
		t.ScrollX -= tabScrollStep * 2
		t.clampScrollX(availableWidth)
		return true
	}
	if t.arrowsVisible && contains(t.rightArrow, x, y) {
		t.ScrollX += tabScrollStep * 2
		t.clampScrollX(availableWidth)
		return true
	}

	// 检查是否点击"+"按钮
	plusX := originX + availableWidth
	if t.OnNew != nil && x >= plusX && x < plusX+plusButtonWidth {
		t.OnNew()
		return true
	}

	cursor := originX - t.ScrollX
	for i, item := range t.Items {
		w := tabWidth(item.Title)
		if x >= cursor && x < cursor+w {
			if x >= cursor+w-30 && t.OnClose != nil {
				t.OnClose(item.Path)
			} else {
				// 开始拖拽
				t.dragging = true
				t.dragIndex = i
				t.dragStartX = x
				t.dragStartY = y
				t.dragCurrentX = x
				t.dragCurrentY = y

				t.Active = i
				if t.OnSelect != nil {
					t.OnSelect(item.Path)
				}
			}
			return true
		}
		cursor += w
	}
	return true
}

// HandleWheel 处理标签条带内的滚轮事件：横向滚动（正值向左，与纵向列表
// 的"滚轮上滚看顶部"方向约定一致）。上界由 Render 每帧夹紧。
func (t *Tabs) HandleWheel(delta int32) {
	rows := delta / 120
	if rows == 0 && delta != 0 {
		rows = delta / abs(delta)
	}
	t.ScrollX -= rows * tabScrollStep
	if t.ScrollX < 0 {
		t.ScrollX = 0
	}
}

// HandleRightClick 处理标签条带内的右键：命中标签时请求上下文菜单项并
// 经 ShowClamped 防越界展示；空白区/加号区不弹菜单但消费点击。命中测试
// 与 Render 相同的滚动偏移。
func (t *Tabs) HandleRightClick(x, y, originX, originY, width, viewW, viewH int32) bool {
	if x < originX || x >= originX+width || y < originY || y >= originY+t.Height {
		return false
	}
	var tab *Tab
	cursor := originX - t.ScrollX
	for i := range t.Items {
		w := tabWidth(t.Items[i].Title)
		if x >= cursor && x < cursor+w {
			tab = &t.Items[i]
			break
		}
		cursor += w
	}
	if tab != nil && t.OnContextMenu != nil && t.ContextMenu != nil {
		if items := t.OnContextMenu(tab, x, y); len(items) > 0 {
			t.ContextMenu.ShowClamped(x, y, viewW, viewH, items)
		}
	}
	return true
}

// HandleMouseMove 处理鼠标移动（用于拖拽与滚动箭头悬停高亮）
func (t *Tabs) HandleMouseMove(x, y, originX, originY, width int32) bool {
	if !t.dragging {
		// 滚动箭头悬停高亮（几何在 Render 时同步）。
		t.clampScrollX(width - plusButtonWidth)
		oldLeft, oldRight := t.leftArrowHot, t.rightArrowHot
		t.leftArrowHot = t.arrowsVisible && contains(t.leftArrow, x, y)
		t.rightArrowHot = t.arrowsVisible && contains(t.rightArrow, x, y)
		return oldLeft != t.leftArrowHot || oldRight != t.rightArrowHot
	}

	t.dragCurrentX = x
	t.dragCurrentY = y

	// 检测拖拽距离是否超过阈值
	dragThreshold := int32(5)
	if abs(x-t.dragStartX) > dragThreshold || abs(y-t.dragStartY) > dragThreshold {
		// 计算目标位置（与 Render/HandleMouseDown 相同的滚动偏移）
		cursor := originX - t.ScrollX
		targetIndex := -1

		for i := range t.Items {
			w := tabWidth(t.Items[i].Title)

			if x >= cursor && x < cursor+w {
				targetIndex = i
				break
			}
			cursor += w
		}

		// 如果找到目标位置且与当前位置不同，执行重排序
		if targetIndex >= 0 && targetIndex != t.dragIndex {
			// 交换标签页
			temp := t.Items[t.dragIndex]
			t.Items = append(t.Items[:t.dragIndex], t.Items[t.dragIndex+1:]...)

			if targetIndex > t.dragIndex {
				targetIndex--
			}

			// 插入到新位置
			t.Items = append(t.Items[:targetIndex], append([]Tab{temp}, t.Items[targetIndex:]...)...)

			// 更新活动索引
			if t.Active == t.dragIndex {
				t.Active = targetIndex
			} else if t.dragIndex < t.Active && targetIndex >= t.Active {
				t.Active--
			} else if t.dragIndex > t.Active && targetIndex <= t.Active {
				t.Active++
			}

			// 通知重排序
			if t.OnReorder != nil {
				t.OnReorder(t.dragIndex, targetIndex)
			}

			t.dragIndex = targetIndex
		}
	}

	return true
}

// HandleMouseUp 处理鼠标释放（结束拖拽）
func (t *Tabs) HandleMouseUp(x, y int32) bool {
	if !t.dragging {
		return false
	}

	t.dragging = false
	t.dragIndex = -1
	return true
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
