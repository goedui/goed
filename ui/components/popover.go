package components

import (
	"strings"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// splitMenuText 把「标签\tshortcut」拆成两段；没有制表符时 shortcut 为空。
func splitMenuText(s string) (label, shortcut string) {
	if idx := strings.IndexByte(s, '\t'); idx >= 0 {
		return s[:idx], s[idx+1:]
	}
	return s, ""
}

func init() {
	renderer.Register("popover", renderer.Widget{
		Measure: measurePopover,
		Layout:  layoutPopover,
		Paint:   paintPopover,
	})
}

// Popover 锚定下拉。Attrs: anchor, items []string, onSelect, onClose。
// items 里 "---" / "-" 画成分隔线。
func Popover(parts ...any) *runtime.VNode {
	n := runtime.H("popover", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		if e.Key == runtime.KeyEscape {
			if fn := onVoid(n, "onClose"); fn != nil {
				fn()
			}
			return true
		}
		if e.Key == runtime.KeyEnter || e.Key == runtime.KeyDown {
			items := stringList(renderer.Attr(n, "items"))
			idx := firstPopoverItem(items)
			if idx < 0 {
				return true
			}
			if e.Key == runtime.KeyDown {
				if next := nextPopoverItem(items, idx); next >= 0 {
					idx = next
				}
			}
			if fn := onString(n, "onSelect"); fn != nil {
				fn(items[idx])
			}
			return true
		}
		return false
	})
	return n
}

func popoverRow(style renderer.TextStyle) float32 {
	row := style.FontSize*1.4 + 12
	if row < 28 {
		row = 28
	}
	return row
}

func isPopoverSep(s string) bool {
	return s == "---" || s == "-"
}

func firstPopoverItem(items []string) int {
	for i, s := range items {
		if !isPopoverSep(s) {
			return i
		}
	}
	return -1
}

func nextPopoverItem(items []string, from int) int {
	for i := from + 1; i < len(items); i++ {
		if !isPopoverSep(items[i]) {
			return i
		}
	}
	return -1
}

func measurePopover(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	items := stringList(renderer.Attr(n, "items"))
	var w float32 = 160
	row := popoverRow(style)
	var h float32
	for _, s := range items {
		if isPopoverSep(s) {
			h += 8
			continue
		}
		label, shortcut := splitMenuText(s)
		lw, _ := ctx.MeasureText(label, 400, style)
		sw, _ := ctx.MeasureText(shortcut, 400, style)
		need := lw + sw + 24
		if shortcut != "" {
			need += 28
		}
		if need > w {
			w = need
		}
		h += row
	}
	if h < row {
		h = row
	}
	_ = maxW
	_ = maxH
	_ = th
	_ = n
	return w, h
}

func layoutPopover(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	anchorID := renderer.AttrString(n, "anchor")
	ax, ay, aw, ah := x, y, w, float32(0)
	if anchorID != "" {
		if a := runtime.FindByID(renderer.LayoutRoot(), anchorID); a != nil {
			ax, ay, aw, ah = a.X, a.Y, a.W, a.H
		}
	}
	px := ax
	switch renderer.AttrString(n, "alignX") {
	case "center":
		px = ax + (aw-w)/2
	case "end":
		px = ax + aw - w
	}
	py := ay + ah + 4
	if ctx != nil {
		if px+w > ctx.Width() {
			px = ctx.Width() - w
		}
		if py+h > ctx.Height() {
			py = ay - h - 4
		}
	}
	if px < 0 {
		px = 0
	}
	if py < 0 {
		py = 0
	}
	_ = aw
	n.X, n.Y, n.W, n.H = px, py, w, h
	items := stringList(renderer.Attr(n, "items"))
	row := popoverRow(style)
	n.Children = n.Children[:0]
	cy := py
	for _, item := range items {
		item := item
		if isPopoverSep(item) {
			child := runtime.H("box", runtime.Props{
				Style: runtime.Style{Width: w - 16, Height: 1, Background: th.Separator},
			})
			renderer.LayoutNode(ctx, child, px+8, cy+3.5, w-16, 1, style, th)
			n.Children = append(n.Children, child)
			cy += 8
			continue
		}
		label, shortcut := splitMenuText(item)
		child := runtime.H("button", runtime.Props{
			Style: runtime.Style{Width: w, Height: row},
			OnClick: func() {
				if fn := onString(n, "onSelect"); fn != nil {
					fn(item)
				}
			},
			Label: label, Variant: "ghost", TextAlign: "start", Shortcut: shortcut,
		})
		renderer.LayoutNode(ctx, child, px, cy, w, row, style, th)
		n.Children = append(n.Children, child)
		cy += row
	}
}

func paintPopover(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	bg := renderer.ColorFrom(th.Background)
	if bg.A == 0 {
		bg = renderer.RGB(255, 255, 255)
	}
	if th.Dark {
		bg = renderer.ColorFrom(th.Titlebar)
	}
	border := renderer.ColorFrom(th.Separator)
	if border.A == 0 {
		border = renderer.RGB(200, 200, 200)
	}
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, 6, bg)
	ctx.DrawRect(n.X, n.Y, n.W, n.H, border, 1)
	renderer.PaintChildren(ctx, n, style, th)
}
