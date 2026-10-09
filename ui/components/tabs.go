package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("tabs", renderer.Widget{
		Measure: measureTabs,
		Layout:  layoutTabs,
		Paint:   paintTabs,
	})
}

// Tab 是标签栏上的一项。Icon 非空时画在文字左侧（选中胶囊上的 sparkle）。
type Tab struct {
	ID       string
	Label    string
	Icon     string
	Closable bool
	Disabled bool
	CloseID  string
}

func tabIDOf(t Tab) string {
	if t.ID != "" {
		return t.ID
	}
	return t.Label
}

func tabCloseID(t Tab) string {
	if t.CloseID != "" {
		return t.CloseID
	}
	return tabIDOf(t) + "-close"
}

func tabsFrom(v any) []Tab {
	switch t := v.(type) {
	case []Tab:
		return t
	case []any:
		out := make([]Tab, 0, len(t))
		for _, x := range t {
			if tab, ok := x.(Tab); ok {
				out = append(out, tab)
			}
		}
		return out
	default:
		return nil
	}
}

// Tabs 水平标签栏。
//
// Attrs:
//
//	items      []Tab
//	value      当前选中 id
//	onChange   func(string)
//	onClose    func(string)  关闭按钮；参数是 tab id
//	variant    "pill" 胶囊；默认矩形
//	颜色：selectedBackground/selectedColor 选中项，itemBackground/itemColor/itemBorder
//	未选中项，closeColor 关闭按钮；缺省都取主题
func Tabs(parts ...any) *runtime.VNode {
	n := runtime.H("tabs", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	trailing := append([]*runtime.VNode(nil), n.Children...)
	n.Children = n.Children[:0]

	items := tabsFrom(n.Attrs["items"])
	value := renderer.AttrString(n, "value")
	pill := renderer.AttrString(n, "variant") == "pill"
	barDisabled := !renderer.Enabled(n)

	itemH := n.Style.Height
	if itemH > 0 && n.Style.Padding > 0 {
		itemH = itemH - 2*n.Style.Padding
	}
	if itemH <= 0 {
		if pill {
			itemH = 34
		} else {
			itemH = 28
		}
	}
	radius := n.Style.Radius
	if pill && radius <= 0 {
		radius = itemH / 2
	}
	if radius <= 0 {
		radius = 4
	}
	pad := float32(10)
	if pill {
		pad = 14
	}

	for _, t := range items {
		t := t
		id := tabIDOf(t)
		selected := value == id
		disabled := barDisabled || t.Disabled
		bg, fg, border := tabColors(n, selected, pill)
		btn := Button(runtime.Props{
			Style: runtime.Style{
				Height:     itemH,
				Radius:     radius,
				Background: bg,
				Color:      fg,
				Border:     border,
				FontSize:   n.Style.FontSize,
				Padding:    pad,
			},
			Icon: t.Icon,
			ID: id, Label: t.Label, Variant: "ghost", Selected: selected, Disabled: disabled,
			OnClick: func() {
				if disabled || id == value {
					return
				}
				if fn := onString(n, "onChange"); fn != nil {
					fn(id)
				}
			},
		}, t.Label)
		n.Children = append(n.Children, btn)
		if t.Closable {
			closeFg := fg
			if c, ok := attrColor(n, "closeColor"); ok {
				closeFg = c
			}
			n.Children = append(n.Children, Button(runtime.Props{
				Style: runtime.Style{
					Height:     itemH,
					Width:      itemH,
					Radius:     radius,
					Background: bg,
					Color:      closeFg,
					FontSize:   n.Style.FontSize,
				},
				ID: tabCloseID(t), Label: "×", Variant: "ghost", Disabled: disabled,
				OnClick: func() {
					if disabled {
						return
					}
					if fn := onString(n, "onClose"); fn != nil {
						fn(id)
					}
				},
			}, "×"))
		}
	}
	for _, c := range trailing {
		if c != nil {
			n.Children = append(n.Children, c)
		}
	}
	if n.Style.Gap == 0 {
		if pill {
			n.Style.Gap = 8
		} else {
			n.Style.Gap = 4
		}
	}
	if n.Style.Align == "" {
		n.Style.Align = runtime.Center
	}
	return n
}

func tabColors(n *runtime.VNode, selected, pill bool) (bg, fg, border runtime.Color) {
	if selected {
		if c, ok := attrColor(n, "selectedBackground"); ok {
			bg = c
		}
		if c, ok := attrColor(n, "selectedColor"); ok {
			fg = c
		}
		return bg, fg, runtime.Color{}
	}
	if c, ok := attrColor(n, "itemBackground"); ok {
		bg = c
	} else {
		bg = n.Style.Background
	}
	if c, ok := attrColor(n, "itemColor"); ok {
		fg = c
	} else {
		fg = n.Style.Color
	}
	if pill {
		if c, ok := attrColor(n, "itemBorder"); ok {
			border = c
		}
	}
	return bg, fg, border
}

func measureTabs(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	return measureHStack(ctx, n, maxW, maxH, style, th)
}

func layoutTabs(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	layoutHStack(ctx, n, x, y, w, h, style, th)
}

func paintTabs(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	paintHStack(ctx, n, style, th)
}
