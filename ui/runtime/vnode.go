package runtime

import (
	"fmt"
	"strings"
)

// Kind 表示虚拟节点类型。
type Kind uint8

const (
	KindText Kind = iota
	KindElement
	KindComponent
)

// Attrs 是节点内部属性表。控件构造函数和引擎读取用，不出现在公开 Props 里。
type Attrs map[string]any

// Props 是引擎标签的公开属性。Style 保持嵌套，把布局/外观和组件语义分开。
//
//	ui.Button(ui.Props{Style: ui.Style{Width: 80}, OnClick: fn}, "点我")
//	ui.Select(ui.Props{ID: "size", Value: size, Items: choices, OnChange: fn})
type Props struct {
	Style Style
	Key   string

	ID, Label, Placeholder, Variant, Title, Hint, Icon, Name, Badge string
	Message, Path, Open, PopupId, Anchor, AlignX, Shortcut, TextAlign string

	Disabled, Selected, Password, Compact, Multiline, Readonly bool
	Nowrap, Borderless, Dashed, Vertical, CloseOnEnter, Modal, Focus bool

	OnClick       func()
	Value         any
	OnChange      any
	OnInput       func(string)
	OnSubmit      func(string)
	OnKeyDown     KeyHandler
	OnClose       any
	OnOpen        any
	OnSelect      any
	OnEmpty       func()
	OnEnter       func()
	OnWheel       any
	OnPaint       any
	OnPointerDown func()
	Items         any
	Menus         any
	Data          []byte
	State         any
	Dispatcher    any
	Term          any
	ContentWidth  float32

	SelectedBackground, SelectedColor                 Color
	ItemBackground, ItemColor, ItemBorder, CloseColor Color
	OpenBackground, OpenColor                         Color
	OnBG, OffBG, Knob                                 any
}

// VNode 是渲染函数产出的虚拟节点。引擎对它做布局与绘制，而不是直接操作 HWND。
type VNode struct {
	Kind       Kind
	Tag        string
	Text       string
	Props      Props
	Attrs      Attrs
	Style      Style
	Children   []*VNode
	Component  *ComponentDef
	OnClick    func()
	X, Y, W, H float32
	resolved   bool
}

func applyProps(n *VNode, p Props) *VNode {
	if n == nil {
		return nil
	}
	n.Props = p
	n.Style = p.Style
	n.OnClick = p.OnClick
	put := func(k string, v any) {
		if n.Attrs == nil {
			n.Attrs = Attrs{}
		}
		n.Attrs[k] = v
	}
	putStr := func(k, v string) {
		if v != "" {
			put(k, v)
		}
	}
	putBool := func(k string, v bool) {
		if v {
			put(k, true)
		}
	}
	putAny := func(k string, v any) {
		if v != nil {
			put(k, v)
		}
	}
	putColor := func(k string, c Color) {
		if c.A != 0 {
			put(k, c)
		}
	}

	putStr("id", p.ID)
	putStr("label", p.Label)
	putStr("placeholder", p.Placeholder)
	putStr("variant", p.Variant)
	putStr("title", p.Title)
	putStr("hint", p.Hint)
	putStr("icon", p.Icon)
	putStr("name", p.Name)
	putStr("badge", p.Badge)
	putStr("message", p.Message)
	putStr("path", p.Path)
	putStr("open", p.Open)
	putStr("popupId", p.PopupId)
	putStr("anchor", p.Anchor)
	putStr("alignX", p.AlignX)
	putStr("shortcut", p.Shortcut)
	putStr("align", p.TextAlign)

	putBool("disabled", p.Disabled)
	putBool("selected", p.Selected)
	putBool("password", p.Password)
	putBool("compact", p.Compact)
	putBool("multiline", p.Multiline)
	putBool("readonly", p.Readonly)
	putBool("nowrap", p.Nowrap)
	putBool("borderless", p.Borderless)
	putBool("dashed", p.Dashed)
	putBool("vertical", p.Vertical)
	putBool("closeOnEnter", p.CloseOnEnter)
	putBool("focus", p.Focus)
	if p.Modal {
		put("dismiss", false)
	}

	putAny("value", p.Value)
	putAny("onChange", p.OnChange)
	putAny("onClose", p.OnClose)
	putAny("onOpen", p.OnOpen)
	putAny("onSelect", p.OnSelect)
	putAny("onWheel", p.OnWheel)
	putAny("onPaint", p.OnPaint)
	putAny("items", p.Items)
	putAny("menus", p.Menus)
	putAny("state", p.State)
	putAny("dispatcher", p.Dispatcher)
	putAny("term", p.Term)
	putAny("onbg", p.OnBG)
	putAny("offbg", p.OffBG)
	putAny("knob", p.Knob)
	if p.OnInput != nil {
		put("onInput", p.OnInput)
	}
	if p.OnSubmit != nil {
		put("onSubmit", p.OnSubmit)
	}
	if p.OnKeyDown != nil {
		put("onKeyDown", p.OnKeyDown)
	}
	if p.OnEmpty != nil {
		put("onEmpty", p.OnEmpty)
	}
	if p.OnEnter != nil {
		put("onEnter", p.OnEnter)
	}
	if p.OnPointerDown != nil {
		put("onPointerDown", p.OnPointerDown)
	}
	if p.ContentWidth != 0 {
		put("width", p.ContentWidth)
	}
	if len(p.Data) > 0 {
		put("data", p.Data)
	}

	putColor("selectedBackground", p.SelectedBackground)
	putColor("selectedColor", p.SelectedColor)
	putColor("itemBackground", p.ItemBackground)
	putColor("itemColor", p.ItemColor)
	putColor("itemBorder", p.ItemBorder)
	putColor("closeColor", p.CloseColor)
	putColor("openBackground", p.OpenBackground)
	putColor("openColor", p.OpenColor)
	return n
}

// fallthroughProps 把父级传到组件上的 style / onClick 落到单根上，对应 Vue 的 attribute fallthrough。
func fallthroughProps(host *VNode, roots []*VNode) {
	if host == nil || len(roots) != 1 || roots[0] == nil {
		return
	}
	root := roots[0]
	mergeStyle(&root.Style, host.Style)
	if root.OnClick == nil {
		root.OnClick = host.OnClick
	}
}

func mergeStyle(dst *Style, src Style) {
	if dst == nil {
		return
	}
	if dst.Flex == 0 {
		dst.Flex = src.Flex
	}
	if dst.Width == 0 {
		dst.Width = src.Width
	}
	if dst.Height == 0 {
		dst.Height = src.Height
	}
	if dst.MinWidth == 0 {
		dst.MinWidth = src.MinWidth
	}
	if dst.MinHeight == 0 {
		dst.MinHeight = src.MinHeight
	}
	if dst.MaxWidth == 0 {
		dst.MaxWidth = src.MaxWidth
	}
	if dst.MaxHeight == 0 {
		dst.MaxHeight = src.MaxHeight
	}
	if dst.Padding == 0 {
		dst.Padding = src.Padding
	}
	if dst.Gap == 0 {
		dst.Gap = src.Gap
	}
	if dst.FontSize == 0 {
		dst.FontSize = src.FontSize
	}
	if dst.FontFamily == "" {
		dst.FontFamily = src.FontFamily
	}
	if dst.Weight == 0 {
		dst.Weight = src.Weight
	}
	if dst.Align == "" {
		dst.Align = src.Align
	}
	if dst.Justify == "" {
		dst.Justify = src.Justify
	}
	if dst.Color.A == 0 {
		dst.Color = src.Color
	}
	if dst.Background.A == 0 {
		dst.Background = src.Background
	}
	if dst.Border.A == 0 {
		dst.Border = src.Border
	}
	if dst.Radius == 0 {
		dst.Radius = src.Radius
	}
}

// WithKey 给节点设 reconciliation key，链式调用。
func (n *VNode) WithKey(k string) *VNode {
	if n != nil {
		n.Props.Key = k
	}
	return n
}

// ID 读取 Attrs["id"]。
func (n *VNode) ID() string {
	if n == nil || n.Attrs == nil {
		return ""
	}
	s, _ := n.Attrs["id"].(string)
	return s
}

// Text 创建文本节点。第一个参数若是 Props，则当作 h('text', props, children)。
func Text(parts ...any) *VNode {
	var p Props
	if len(parts) > 0 {
		if pp, ok := parts[0].(Props); ok {
			p = pp
			parts = parts[1:]
		}
	}
	return applyProps(&VNode{Kind: KindText, Text: joinText(parts)}, p)
}

func joinText(parts []any) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		if s, ok := parts[0].(string); ok {
			return s
		}
	}
	var b strings.Builder
	for _, p := range parts {
		fmt.Fprint(&b, p)
	}
	return b.String()
}

// splitProps 把 h() 的其余参数拆成 props 和 children。
// 第一个参数若是 Props 则当作属性，否则全部是子节点。没有 props 就不用写。
func splitProps(parts []any) (Props, []*VNode) {
	var p Props
	start := 0
	if len(parts) > 0 {
		if pp, ok := parts[0].(Props); ok {
			p = pp
			start = 1
		}
	}
	kids := make([]*VNode, 0, len(parts)-start)
	for _, part := range parts[start:] {
		switch v := part.(type) {
		case nil:
		case *VNode:
			kids = append(kids, v)
		case []*VNode:
			kids = append(kids, v...)
		default:
			kids = append(kids, Text(v))
		}
	}
	return p, kids
}

// H 对应 Vue 3 的 h(type, props?, children...)。
// type 可以是标签名（"vstack"）或自定义组件（*ComponentDef）。
// 没有 props 时直接 H(Counter) 或 H("vstack", child)。
func H(typ any, rest ...any) *VNode {
	props, children := splitProps(rest)
	switch t := typ.(type) {
	case string:
		return applyProps(&VNode{
			Kind:     KindElement,
			Tag:      t,
			Children: children,
		}, props)
	case *ComponentDef:
		return C(t, props, children...)
	default:
		tag := ""
		if t != nil {
			tag = fmt.Sprint(t)
		}
		return applyProps(&VNode{
			Kind:     KindElement,
			Tag:      tag,
			Children: children,
		}, props)
	}
}

// Tag 创建带内部属性的元素，供 titlebar 等引擎控件使用。
func Tag(tag string, attrs Attrs, children ...*VNode) *VNode {
	return &VNode{
		Kind:     KindElement,
		Tag:      tag,
		Attrs:    attrs,
		Children: children,
	}
}

// C 把组件定义包装成可挂到树上的 VNode。等价于 H(def, props, children)。
func C(def *ComponentDef, props Props, children ...*VNode) *VNode {
	return applyProps(&VNode{
		Kind:      KindComponent,
		Children:  children,
		Component: def,
	}, props)
}
