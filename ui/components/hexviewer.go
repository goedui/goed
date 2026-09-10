package components

import (
	"fmt"
	"path/filepath"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// 十六进制预览布局常量：头部/底栏高度、每行字节数。
const (
	hexRowHeight   = 18
	hexBytesPerRow = 16
	hexHeaderH     = 28
	hexFooterH     = 22
	hexPadding     = 12
)

// HexViewer 只读十六进制预览：偏移 / 十六进制 / ASCII 三栏经典布局，
// 等宽字体渲染。支持滚轮滚动、点击选中字节、Shift 范围选择、Ctrl+C 复制。
type HexViewer struct {
	Path      string
	Data      []byte   // 预览数据（可能被截断）
	TotalSize int64    // 磁盘文件完整大小
	Truncated bool     // 是否仅加载部分内容

	FirstRow    int  // 顶部可见行号
	RowHeight   int32
	Selected    int  // 选中字节下标；-1 表示无
	SelectedEnd int  // Shift 范围选择终点

	// 列几何：Render 每帧刷新，命中测试使用最近一次布局。
	charW, hexX, asciiX int32
	viewportRows        int

	// 纵向滚动条：内容超出视口时显示于右缘（几何 Render 时同步）。
	scrollBar       *ScrollBar
	scrollBarActive bool
}

func NewHexViewer() *HexViewer {
	return &HexViewer{RowHeight: hexRowHeight, Selected: -1, SelectedEnd: -1, scrollBar: NewVScrollBar(0, 0, 100)}
}

// Load 载入预览数据并重置滚动/选择状态。
func (v *HexViewer) Load(path string, data []byte, total int64, truncated bool) {
	v.Path = path
	v.Data = data
	v.TotalSize = total
	v.Truncated = truncated
	v.FirstRow = 0
	v.Selected, v.SelectedEnd = -1, -1
}

func (v *HexViewer) Ready() bool { return len(v.Data) > 0 }

func (v *HexViewer) totalRows() int { return (len(v.Data) + hexBytesPerRow - 1) / hexBytesPerRow }

func (v *HexViewer) clampFirstRow() {
	max := v.totalRows() - v.viewportRows
	if max < 0 {
		max = 0
	}
	if v.FirstRow < 0 {
		v.FirstRow = 0
	}
	if v.FirstRow > max {
		v.FirstRow = max
	}
}

// HandleWheel 滚动行（每档 3 行，正值向上，与文件树方向约定一致）。
func (v *HexViewer) HandleWheel(delta int32) {
	rows := delta / 120
	if rows == 0 && delta != 0 {
		rows = delta / absInt(delta)
	}
	v.FirstRow -= int(rows) * 3
	v.clampFirstRow()
}

// HandleMouseDown 处理点击：滚动条条带优先命中；hex/ascii 列命中选中
// 字节，Shift 扩展范围。偏移列与头部点击只消费不选择。返回点击是否属于
// 预览区域。
func (v *HexViewer) HandleMouseDown(x, y, ox, oy, w, h int32, shift bool) bool {
	if x < ox || x >= ox+w || y < oy || y >= oy+h {
		return false
	}
	// 滚动条条带（右缘 12px，几何在 Render 时同步）。
	if v.scrollBarActive && v.scrollBar != nil && x >= v.scrollBar.X {
		if v.scrollBar.HandleMouseDown(x, y) {
			v.FirstRow = int(v.scrollBar.Position)
			return true
		}
	}
	if idx := v.selectionFromPoint(x-ox, y-oy, w, h); idx >= 0 {
		if shift && v.Selected >= 0 {
			v.SelectedEnd = idx
		} else {
			v.Selected, v.SelectedEnd = idx, idx
		}
	}
	return true
}

// HexDragScrollbar 拖拽滚动条 thumb（App 在 MouseMove 中转发），返回是
// 否处于拖拽状态（拖拽期间独占指针）。
func (v *HexViewer) HexDragScrollbar(x, y int32) bool {
	if v.scrollBar != nil && v.scrollBar.dragging {
		v.scrollBar.HandleMouseMove(x, y)
		v.FirstRow = int(v.scrollBar.Position)
		return true
	}
	return false
}

// HexScrollbarReleased 结束滚动条拖拽（App 在 MouseUp 中转发）。
func (v *HexViewer) HexScrollbarReleased() {
	if v.scrollBar != nil {
		v.scrollBar.dragging = false
	}
}

// selectionFromPoint 由预览区域内相对坐标计算字节下标；未命中字节列
// （偏移列、头部、底栏）返回 -1。
func (v *HexViewer) selectionFromPoint(x, y, viewW, viewH int32) int {
	if v.charW <= 0 || y < hexHeaderH || y >= viewH-hexFooterH {
		return -1
	}
	row := v.FirstRow + int((y-hexHeaderH)/v.RowHeight)
	var col int32
	switch {
	case v.asciiX > 0 && x >= v.asciiX:
		col = (x - v.asciiX) / v.charW
		if col >= hexBytesPerRow {
			return -1
		}
	case v.hexX > 0 && x >= v.hexX:
		rel := x - v.hexX
		col = rel / (3 * v.charW)
		if col >= 8 { // 第 8 字节后有一个字符宽的组间距
			col = (rel - v.charW) / (3 * v.charW)
		}
		if col < 0 {
			col = 0
		}
		if col >= hexBytesPerRow {
			col = hexBytesPerRow - 1 // 行尾间隙归到最后一个字节
		}
	default:
		return -1
	}
	idx := row*hexBytesPerRow + int(col)
	if idx >= len(v.Data) {
		return -1
	}
	return idx
}

func (v *HexViewer) HasSelection() bool { return v.Selected >= 0 }

// selectionRange 返回有序选中范围 [lo, hi]（含端点）；无选择返回 false。
func (v *HexViewer) selectionRange() (int, int, bool) {
	if !v.HasSelection() {
		return 0, 0, false
	}
	lo, hi := v.Selected, v.SelectedEnd
	if lo > hi {
		lo, hi = hi, lo
	}
	if hi >= len(v.Data) {
		hi = len(v.Data) - 1
	}
	if lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}

// CopyText 选中范围的十六进制文本（"4D 5A 90" 形式）；无选择返回空串。
func (v *HexViewer) CopyText() string {
	lo, hi, ok := v.selectionRange()
	if !ok {
		return ""
	}
	out := make([]byte, 0, (hi-lo+1)*3)
	for i := lo; i <= hi; i++ {
		if i > lo {
			out = append(out, ' ')
		}
		out = append(out, fmt.Sprintf("%02X", v.Data[i])...)
	}
	return string(out)
}

// SelectionInfo 状态栏摘要：单个字节显示值，范围显示区间。
func (v *HexViewer) SelectionInfo() string {
	lo, hi, ok := v.selectionRange()
	if !ok {
		return ""
	}
	if lo == hi {
		return fmt.Sprintf("0x%08X: 0x%02X", lo, v.Data[lo])
	}
	return fmt.Sprintf("0x%08X - 0x%08X（%d 字节）", lo, hi, hi-lo+1)
}

// hexHumanSize 本地文件大小格式化（与 fileops.FormatSize 同规则，
// 组件层保持零业务依赖）。
func hexHumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func (v *HexViewer) Render(ctx core.DrawingContext, x, y, w, h int32) {
	th := theme.GetTheme()
	ctx.DrawRect(x, y, w, h, th.EditorBg, true)
	if !v.Ready() {
		return
	}
	// 等宽字体（EditorFontFamily = Cascadia Code），字符宽度每帧实测。
	ctx.SetFont(th.EditorFontFamily, th.EditorFontSize)
	cw, _ := ctx.MeasureText("0")
	if cw <= 0 {
		cw = 8
	}
	v.charW = cw
	v.hexX = x + hexPadding + 8*cw + 12
	cellW := 3 * cw
	v.asciiX = v.hexX + int32(hexBytesPerRow)*cellW + cw + 8

	// 头部：文件名 + 完整大小。
	ctx.SetFont(th.UIFontFamily, th.FontBody())
	ctx.DrawText(x+hexPadding, y+8, filepath.Base(v.Path), th.TextPrimary)
	sizeText := hexHumanSize(v.TotalSize)
	if sw, _ := ctx.MeasureText(sizeText); sw > 0 {
		ctx.DrawText(x+w-sw-hexPadding, y+8, sizeText, th.TextSecondary)
	}
	ctx.DrawLine(x, y+hexHeaderH-4, x+w, y+hexHeaderH-4, th.WindowBorder, 1)

	// 行区域裁剪。
	if clip, ok := ctx.(editorClipStack); ok {
		clip.PushClipRect(x, y+hexHeaderH, w, h-hexHeaderH-hexFooterH)
	} else {
		ctx.SetClipRect(x, y+hexHeaderH, w, h-hexHeaderH-hexFooterH)
	}

	ctx.SetFont(th.EditorFontFamily, th.EditorFontSize)
	rowsH := h - hexHeaderH - hexFooterH
	v.viewportRows = int(rowsH / v.RowHeight)
	v.clampFirstRow()
	lo, hi, hasSel := v.selectionRange()

	for row := v.FirstRow; row < v.totalRows(); row++ {
		rowIdx := row - v.FirstRow
		if rowIdx >= v.viewportRows {
			break
		}
		rowY := y + hexHeaderH + int32(rowIdx)*v.RowHeight
		ctx.DrawText(x+hexPadding, rowY+3, fmt.Sprintf("%08X", row*hexBytesPerRow), th.TextSecondary)
		base := row * hexBytesPerRow
		// hex 列：8+8 分组。
		for b := 0; b < hexBytesPerRow; b++ {
			idx := base + b
			if idx >= len(v.Data) {
				break
			}
			cx := v.hexX + int32(b)*cellW
			if b >= 8 {
				cx += cw
			}
			if hasSel && idx >= lo && idx <= hi {
				ctx.DrawRect(cx-2, rowY, 2*cw+3, v.RowHeight, th.EditorSelectionBg, true)
			}
			ctx.DrawText(cx, rowY+3, fmt.Sprintf("%02X", v.Data[idx]), th.EditorFg)
		}
		// ascii 列。
		for b := 0; b < hexBytesPerRow && base+b < len(v.Data); b++ {
			idx := base + b
			c := v.Data[idx]
			if hasSel && idx >= lo && idx <= hi {
				ctx.DrawRect(v.asciiX+int32(b)*cw, rowY, cw+1, v.RowHeight, th.EditorSelectionBg, true)
			}
			ch := byte('.')
			if c >= 0x20 && c < 0x7F {
				ch = c
			}
			ctx.DrawText(v.asciiX+int32(b)*cw, rowY+3, string(rune(ch)), th.TextSecondary)
		}
	}

	if clip, ok := ctx.(editorClipStack); ok {
		clip.PopClip()
	} else {
		ctx.ResetClip()
	}

	// 内容超出视口时在右缘绘制纵向滚动条（宽 12px，行区域高度）。
	v.scrollBarActive = v.scrollBar != nil && v.totalRows() > v.viewportRows && v.viewportRows > 0
	if v.scrollBarActive {
		v.scrollBar.X = x + w - 12
		v.scrollBar.Y = y + hexHeaderH
		v.scrollBar.Width = 12
		v.scrollBar.Height = rowsH
		v.scrollBar.Total = int32(v.totalRows())
		v.scrollBar.Visible = int32(v.viewportRows)
		v.scrollBar.Position = int32(v.FirstRow)
		v.scrollBar.Render(ctx)
	}

	// 底栏：截断提示或选中摘要。
	footerY := y + h - hexFooterH + 4
	ctx.DrawLine(x, footerY-6, x+w, footerY-6, th.WindowBorder, 1)
	ctx.SetFont(th.UIFontFamily, th.FontSecondary())
	switch {
	case hasSel:
		if text := v.SelectionInfo(); text != "" {
			ctx.DrawText(x+hexPadding, footerY, text, th.TextSecondary)
		}
	case v.Truncated:
		ctx.DrawText(x+hexPadding, footerY,
			fmt.Sprintf("仅预览前 %s，文件共 %s", hexHumanSize(int64(len(v.Data))), hexHumanSize(v.TotalSize)),
			th.TextSecondary)
	}
}
