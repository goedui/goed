package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("menubar", renderer.Widget{
		Measure: measureMenuBar,
		Layout:  layoutMenuBar,
		Paint:   paintMenuBar,
	})
}

// Menu 是菜单栏上的一个顶层菜单。
type Menu struct {
	ID    string
	Label string
	Items []MenuItem
}

// MenuItem 是一条下拉项。Sep 为 true 时画分隔线。
//
// 菜单栏的下拉和右键面板（ContextMenu）共用这一种菜单项 —— 两处的排版与
// 「选中哪一项」的判定（menuItemLabel）都是同一份代码。
type MenuItem struct {
	Label    string
	Shortcut string
	Sep      bool
	Checked  bool
	Action   func()
}

func menuIDOf(m Menu) string {
	if m.ID != "" {
		return m.ID
	}
	return m.Label
}

func menusFrom(v any) []Menu {
	switch t := v.(type) {
	case []Menu:
		return t
	case []any:
		out := make([]Menu, 0, len(t))
		for _, x := range t {
			if m, ok := x.(Menu); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// menuItemsFrom 把 Attrs["items"] 归一成 []MenuItem。
//
// MenuBar 和 ContextMenu 共用同一套菜单项（MenuItem / menuItemLabel / 分隔线
// "---"），差别只在菜单从哪儿展开：菜单栏从锚点往下掉，右键面板跟着指针走。
// 归一化放在这儿，两边的「标签怎么拼、动作怎么查」就不会长成两套。
func menuItemsFrom(v any) []MenuItem {
	switch t := v.(type) {
	case []MenuItem:
		return t
	case []any:
		out := make([]MenuItem, 0, len(t))
		for _, x := range t {
			if it, ok := x.(MenuItem); ok {
				out = append(out, it)
			}
		}
		return out
	default:
		return nil
	}
}

func findMenu(menus []Menu, open string) *Menu {
	if open == "" {
		return nil
	}
	for i := range menus {
		if menuIDOf(menus[i]) == open || menus[i].Label == open {
			return &menus[i]
		}
	}
	return nil
}

func menuItemLabel(it MenuItem) string {
	prefix := "  "
	if it.Checked {
		prefix = "✓ "
	}
	label := prefix + it.Label
	if it.Shortcut != "" {
		label += "\t" + it.Shortcut
	}
	return label
}

func attrColor(n *runtime.VNode, key string) (runtime.Color, bool) {
	if n == nil || n.Attrs == nil {
		return runtime.Color{}, false
	}
	c, ok := n.Attrs[key].(runtime.Color)
	return c, ok && renderer.ColorSet(c)
}

// MenuBar 顶层菜单栏：按钮行 + 当前展开的 Popover。
//
// Attrs:
//
//	menus           []Menu
//	open            当前展开菜单的 id 或 label
//	onOpen          func(string) 展开变化（"" 表示收起）
//	popupId         下拉节点 id，默认 <menubar-id>-popup
//	openBackground  展开项底色
//	openColor       展开项字色
func MenuBar(parts ...any) *runtime.VNode {
	n := runtime.H("menubar", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	menus := menusFrom(n.Attrs["menus"])
	open := renderer.AttrString(n, "open")
	leading := append([]*runtime.VNode(nil), n.Children...)
	n.Children = n.Children[:0]
	for _, c := range leading {
		if c != nil {
			n.Children = append(n.Children, c)
		}
	}
	for _, m := range menus {
		m := m
		id := menuIDOf(m)
		selected := open == id || open == m.Label
		bg := n.Style.Background
		fg := n.Style.Color
		if selected {
			if c, ok := attrColor(n, "openBackground"); ok {
				bg = c
			}
			if c, ok := attrColor(n, "openColor"); ok {
				fg = c
			}
		}
		h := n.Style.Height
		if h > 8 {
			h = h - 6
		}
		if h <= 0 {
			h = 24
		}
		btn := Button(runtime.Props{
			Style: runtime.Style{
				Height:     h,
				Radius:     4,
				Background: bg,
				Color:      fg,
				FontSize:   n.Style.FontSize,
				Padding:    10,
			},
			ID: id, Label: m.Label, Variant: "ghost", Selected: selected,
			OnClick: func() {
				next := id
				if open == id || open == m.Label {
					next = ""
				}
				if fn := onString(n, "onOpen"); fn != nil {
					fn(next)
				}
			},
		}, m.Label)
		n.Children = append(n.Children, btn)
	}
	if cur := findMenu(menus, open); cur != nil {
		labels := make([]string, 0, len(cur.Items))
		actions := map[string]func(){}
		for _, it := range cur.Items {
			if it.Sep {
				labels = append(labels, "---")
				continue
			}
			l := menuItemLabel(it)
			labels = append(labels, l)
			if it.Action != nil {
				actions[l] = it.Action
			}
		}
		popupID := renderer.AttrString(n, "popupId")
		if popupID == "" {
			if id := n.ID(); id != "" {
				popupID = id + "-popup"
			}
		}
		n.Children = append(n.Children, Popover(runtime.Props{
			ID: popupID, Anchor: menuIDOf(*cur), AlignX: "start", Items: labels,
			OnClose: func() {
				if fn := onString(n, "onOpen"); fn != nil {
					fn("")
				}
			},
			OnSelect: func(v string) {
				if fn := onString(n, "onOpen"); fn != nil {
					fn("")
				}
				if isPopoverSep(v) {
					return
				}
				if a := actions[v]; a != nil {
					a()
				}
				if fn := onString(n, "onSelect"); fn != nil {
					fn(v)
				}
			},
		}).WithKey(popupID))
	}
	return n
}

func measureMenuBar(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	bar := *n
	bar.Children = visibleMenuBarChildren(n)
	return measureHStack(ctx, &bar, maxW, maxH, style, th)
}

func layoutMenuBar(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	var pop *runtime.VNode
	barKids := make([]*runtime.VNode, 0, len(n.Children))
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Tag == "popover" {
			pop = c
			continue
		}
		barKids = append(barKids, c)
	}
	saved := n.Children
	n.Children = barKids
	layoutHStack(ctx, n, x, y, w, h, style, th)
	n.Children = saved
	if pop != nil {
		// popover 是浮层，上限是画布，不是菜单栏自己的盒子。
		//
		// 曾经把菜单栏的 w/h 当 maxW/maxH 传下去，而 renderer.applySize 会把子节点高度
		// 夹到 maxH —— 菜单栏高 32 DIP，弹出层就被夹成 32：底色只盖住第一项，分隔线那
		// 8 DIP 的空档透出底下的界面。不容易发现是因为菜单项是 ghost 按钮、底色恰好与
		// 弹出层同色，只有空档露馅。
		maxW, maxH := w, h
		if ctx != nil {
			maxW, maxH = ctx.Width(), ctx.Height()
		}
		cw, ch := renderer.MeasureNode(ctx, pop, maxW, maxH, style, th)
		renderer.LayoutNode(ctx, pop, x, y, cw, ch, style, th)
	}
}

func paintMenuBar(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	renderer.PaintBackground(ctx, n)
	for _, c := range n.Children {
		if c == nil || c.Tag == "popover" {
			continue
		}
		renderer.PaintChildren(ctx, &runtime.VNode{Children: []*runtime.VNode{c}}, style, th)
	}
}

func visibleMenuBarChildren(n *runtime.VNode) []*runtime.VNode {
	out := make([]*runtime.VNode, 0, len(n.Children))
	for _, c := range n.Children {
		if c != nil && c.Tag != "popover" {
			out = append(out, c)
		}
	}
	return out
}
