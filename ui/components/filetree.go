package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileTreeEntry is a visible workspace item. The tree deliberately keeps a
// flat visible list so hit-testing and rendering stay cheap for large trees.
type FileTreeEntry struct {
	Name, Path string
	Directory  bool
	Depth      int
	Expanded   bool
	Status     FileStatus
}

type FileStatus uint8

const (
	FileStatusNone FileStatus = iota
	FileStatusModified
	FileStatusUntracked
	FileStatusIgnored
	FileStatusReadOnly
	FileStatusConflict
)

// FileTree is a reusable workspace explorer for editor and agent surfaces.
type FileTree struct {
	Root           string
	Entries        []FileTreeEntry
	Selected       string
	RowHeight      int32
	OnOpen         func(path string)
	OnSelect       func(path string)
	StatusProvider func(path string) FileStatus
	statuses       map[string]FileStatus

	// 右键上下文：OnContextMenu 依据命中条目动态返回菜单项（entry 为 nil
	// 表示空白区）；菜单的展示、命中与关闭由 ContextMenu 托管。
	OnContextMenu func(entry *FileTreeEntry, x, y int32) []MenuItem
	ContextMenu   *ContextMenu

	// 滚动状态：列表顶部被卷去的行数。Render 时同步可视高度。
	ScrollRow       int
	viewportRows    int
	scrollBar       *ScrollBar
	scrollBarActive bool // Render 时依据内容行数决定是否参与命中
}

func NewFileTree(root string) *FileTree {
	t := &FileTree{RowHeight: 28, statuses: make(map[string]FileStatus), scrollBar: NewVScrollBar(0, 0, 100), ContextMenu: NewContextMenu()}
	t.SetRoot(root)
	return t
}

func (t *FileTree) SetRoot(root string) {
	if root == "" {
		root, _ = os.Getwd()
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		root = filepath.Dir(root)
	}
	t.Root = filepath.Clean(root)
	t.Refresh()
}

func (t *FileTree) Refresh() {
	t.Entries = t.Entries[:0]
	if t.Root == "" {
		return
	}
	t.appendDirectory(t.Root, 0, true)
}

// RefreshPreservingExpansion 在保留当前展开状态的前提下重建目录树，
// 供外部变更（watcher）触发的刷新使用，避免用户展开的目录被折叠回去。
func (t *FileTree) RefreshPreservingExpansion() {
	expanded := make(map[string]bool, len(t.Entries))
	for _, entry := range t.Entries {
		if entry.Directory && entry.Expanded {
			expanded[entry.Path] = true
		}
	}
	rootExpanded := expanded[t.Root]
	t.Entries = t.Entries[:0]
	t.appendWithExpansion(t.Root, 0, rootExpanded, expanded)
}

// SetStatus updates every visible occurrence of a path (normally one entry).
func (t *FileTree) SetStatus(path string, status FileStatus) {
	if t.statuses == nil {
		t.statuses = make(map[string]FileStatus)
	}
	if status == FileStatusNone {
		delete(t.statuses, path)
	} else {
		t.statuses[path] = status
	}
	for i := range t.Entries {
		if t.Entries[i].Path == path {
			t.Entries[i].Status = status
		}
	}
}

func (t *FileTree) entryStatus(entry FileTreeEntry) FileStatus {
	if t.StatusProvider != nil {
		if status := t.StatusProvider(entry.Path); status != FileStatusNone {
			return status
		}
	}
	if entry.Status != FileStatusNone {
		return entry.Status
	}
	if !entry.Directory {
		if info, err := os.Stat(entry.Path); err == nil && info.Mode().Perm()&0200 == 0 {
			return FileStatusReadOnly
		}
	}
	return FileStatusNone
}

func (t *FileTree) appendDirectory(path string, depth int, expanded bool) {
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = path
	}
	t.Entries = append(t.Entries, FileTreeEntry{Name: name, Path: path, Directory: true, Depth: depth, Expanded: expanded, Status: t.statuses[path]})
	if !expanded {
		return
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir() != items[j].IsDir() {
			return items[i].IsDir()
		}
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})
	for _, item := range items {
		// Keep the workspace readable by omitting noisy VCS/build directories.
		if item.IsDir() && (item.Name() == ".git" || item.Name() == "node_modules") {
			continue
		}
		child := filepath.Join(path, item.Name())
		t.Entries = append(t.Entries, FileTreeEntry{Name: item.Name(), Path: child, Directory: item.IsDir(), Depth: depth + 1, Expanded: false, Status: t.statuses[child]})
	}
}

func (t *FileTree) Render(ctx core.DrawingContext, x, y, width, height int32) {
	th := theme.GetTheme()
	ctx.DrawRect(x, y, width, height, th.EditorGutterBg, true)
	// 字体阶梯（字体规范.md 第 4 节）：头部 Body、目录路径 Secondary。
	ctx.SetFont(th.UIFontFamily, th.FontBody())
	ctx.DrawText(x+18, y+14, "资源管理器", th.TextPrimary)
	ctx.SetFont(th.UIFontFamily, th.FontSecondary())
	rootLabel := filepath.Base(t.Root)
	if rootLabel == "." || rootLabel == "" {
		rootLabel = t.Root
	}
	ctx.DrawText(x+18, y+40, rootLabel, th.TextSecondary)
	listY := y + 58
	// 列表区高度（头部 58px 之后的剩余空间）。
	listH := height - 58
	if listH < 0 {
		listH = 0
	}
	t.viewportRows = int(listH / t.RowHeight)
	t.clampScroll()

	if clip, ok := ctx.(editorClipStack); ok {
		clip.PushClipRect(x, listY, width, listH)
	} else {
		ctx.SetClipRect(x, listY, width, listH)
	}

	for i, entry := range t.Entries {
		rowY := listY + int32(i-t.ScrollRow)*t.RowHeight
		if rowY+t.RowHeight < listY || rowY > listY+listH {
			continue
		}
		if entry.Path == t.Selected {
			ctx.DrawRect(x+6, rowY, width-12, t.RowHeight, th.MenuItemHoverBg, true)
		}
		indent := int32(entry.Depth) * 16
		// 折叠标记：几何三角（与 foldgutter 同风格）。
		if entry.Directory {
			dir := ChevronRight
			if entry.Expanded {
				dir = ChevronDown
			}
			DrawChevron(ctx, x+18+indent, rowY+t.RowHeight/2, dir, th.TextSecondary)
		} else {
			// 文件行：小实心点占位，保持缩进对齐。
			DrawDotIcon(ctx, x+18+indent, rowY+t.RowHeight/2, th.TextDisabled)
		}
		ctx.SetFont(th.UIFontFamily, th.FontSecondary())
		label := entry.Name
		if entry.Directory {
			label += "/"
		}
		ctx.DrawText(x+32+indent, rowY+6, label, th.TextPrimary)
		status := t.entryStatus(entry)
		if status != FileStatusNone {
			switch status {
			case FileStatusModified:
				DrawDotIcon(ctx, x+width-20, rowY+t.RowHeight/2, th.SyntaxString)
			case FileStatusIgnored:
				// 忽略：更小的空心圆。
				ctx.DrawEllipse(x+width-22, rowY+t.RowHeight/2-2, 4, 4, th.TextDisabled, false)
			case FileStatusReadOnly:
				// 只读：小空心矩形（锁的抽象）。
				ctx.DrawRect(x+width-23, rowY+t.RowHeight/2-3, 6, 6, th.TextSecondary, false)
			case FileStatusConflict:
				DrawCloseIcon(ctx, x+width-20, rowY+t.RowHeight/2, th.TitleBarCloseHover)
			default:
				DrawDotIcon(ctx, x+width-20, rowY+t.RowHeight/2, th.SyntaxKeyword)
			}
		}
	}
	// 结束列表裁剪后绘制边框与滚动条。
	if clip, ok := ctx.(editorClipStack); ok {
		clip.PopClip()
	} else {
		ctx.ResetClip()
	}
	ctx.DrawLine(x+width-1, y, x+width-1, y+height, th.WindowBorder, 1)

	// 内容超出视口时在右缘绘制纵向滚动条（宽 12px）。
	t.scrollBarActive = t.scrollBar != nil && len(t.Entries) > t.viewportRows && t.viewportRows > 0
	if t.scrollBarActive {
		t.scrollBar.X = x + width - 12
		t.scrollBar.Y = listY
		t.scrollBar.Width = 12
		t.scrollBar.Height = listH
		t.scrollBar.Total = int32(len(t.Entries))
		t.scrollBar.Visible = int32(t.viewportRows)
		t.scrollBar.Position = int32(t.ScrollRow)
		t.scrollBar.Render(ctx)
	}
}

// clampScroll 把 ScrollRow 限制在 [0, len(Entries)-viewportRows]。
func (t *FileTree) clampScroll() {
	max := len(t.Entries) - t.viewportRows
	if max < 0 {
		max = 0
	}
	if t.ScrollRow < 0 {
		t.ScrollRow = 0
	}
	if t.ScrollRow > max {
		t.ScrollRow = max
	}
}

// HandleWheel 处理文件树区域内的滚轮事件（delta 同编辑器：120/档，
// 负值向下）。每档滚动 1 行。
func (t *FileTree) HandleWheel(delta int32) {
	rows := delta / 120
	if rows == 0 && delta != 0 {
		rows = delta / absInt(delta) // 高分辨率滚轮：不足一档按方向滚 1 行
	}
	t.ScrollRow -= int(rows)
	t.clampScroll()
}

func absInt(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// HandleMouseDown returns true when the click belongs to the tree.
func (t *FileTree) HandleMouseDown(x, y, originX, originY, width, height int32) bool {
	if x < originX || x >= originX+width || y < originY || y >= originY+height {
		return false
	}
	// 滚动条带命中（右缘 12px，几何在 Render 时同步；未渲染时不参与命中）。
	if t.scrollBarActive && t.scrollBar != nil && x >= t.scrollBar.X {
		if t.scrollBar.HandleMouseDown(x, y) {
			t.ScrollRow = int(t.scrollBar.Position)
			return true
		}
	}
	index := int((y-originY-58)/t.RowHeight) + t.ScrollRow
	if index < 0 || index >= len(t.Entries) {
		return true
	}
	entry := t.Entries[index]
	t.Selected = entry.Path
	if entry.Directory {
		t.Toggle(entry.Path)
	} else if t.OnOpen != nil {
		t.OnOpen(entry.Path)
	}
	if t.OnSelect != nil {
		t.OnSelect(entry.Path)
	}
	return true
}

// HandleRightClick 处理右键：命中条目时仅选中并请求上下文菜单项（不展开、
// 不打开）；列表下方空白区域 entry 为 nil。菜单项由 OnContextMenu 动态构建，
// 展示位置经 ShowClamped 防越界。返回点击是否属于树区域。
func (t *FileTree) HandleRightClick(x, y, originX, originY, width, height, viewW, viewH int32) bool {
	if x < originX || x >= originX+width || y < originY || y >= originY+height {
		return false
	}
	// 滚动条条带：右键无动作，仅消费。
	if t.scrollBarActive && t.scrollBar != nil && x >= t.scrollBar.X {
		return true
	}
	var entry *FileTreeEntry
	if index := int((y-originY-58)/t.RowHeight) + t.ScrollRow; y-originY >= 58 && index >= 0 && index < len(t.Entries) {
		entry = &t.Entries[index]
		t.Selected = entry.Path
	}
	if t.OnContextMenu != nil && t.ContextMenu != nil {
		if items := t.OnContextMenu(entry, x, y); len(items) > 0 {
			t.ContextMenu.ShowClamped(x, y, viewW, viewH, items)
		}
	}
	return true
}

// TreeDragScrollbar 拖拽滚动条 thumb（editor 在 MouseMove 中转发），
// 返回是否处于拖拽状态（拖拽期间独占指针）。
func (t *FileTree) TreeDragScrollbar(x, y int32) bool {
	if t.scrollBar != nil && t.scrollBar.dragging {
		t.scrollBar.HandleMouseMove(x, y)
		t.ScrollRow = int(t.scrollBar.Position)
		return true
	}
	return false
}

// TreeScrollbarReleased 结束滚动条拖拽（editor 在 MouseUp 中转发）。
func (t *FileTree) TreeScrollbarReleased() {
	if t.scrollBar != nil {
		t.scrollBar.dragging = false
	}
}

// Toggle 切换目录展开状态（左键折叠标记与右键"打开"共用）。
func (t *FileTree) Toggle(path string) {
	// Rebuild while preserving expanded directories except for the clicked one.
	expanded := map[string]bool{}
	for _, entry := range t.Entries {
		if entry.Directory && entry.Expanded {
			expanded[entry.Path] = true
		}
	}
	if expanded[path] {
		delete(expanded, path)
	} else {
		expanded[path] = true
	}
	rootExpanded := expanded[t.Root]
	t.Entries = t.Entries[:0]
	t.appendWithExpansion(t.Root, 0, rootExpanded, expanded)
}

func (t *FileTree) appendWithExpansion(path string, depth int, isExpanded bool, expanded map[string]bool) {
	name := filepath.Base(path)
	if name == "." || name == "" {
		name = path
	}
	t.Entries = append(t.Entries, FileTreeEntry{Name: name, Path: path, Directory: true, Depth: depth, Expanded: isExpanded, Status: t.statuses[path]})
	if !isExpanded {
		return
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir() != items[j].IsDir() {
			return items[i].IsDir()
		}
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})
	for _, item := range items {
		if item.IsDir() && (item.Name() == ".git" || item.Name() == "node_modules") {
			continue
		}
		child := filepath.Join(path, item.Name())
		if item.IsDir() {
			t.appendWithExpansion(child, depth+1, expanded[child], expanded)
		} else {
			t.Entries = append(t.Entries, FileTreeEntry{Name: item.Name(), Path: child, Depth: depth + 1, Status: t.statuses[child]})
		}
	}
}
