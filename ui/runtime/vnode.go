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

	ID, Label, Placeholder, Variant, Title, Hint, Icon, Name, Badge   string
	Message, Path, Open, PopupId, Anchor, AlignX, Shortcut, TextAlign string

	Disabled, Selected, Password, Compact, Multiline, Readonly       bool
	Nowrap, Borderless, Dashed, Vertical, CloseOnEnter, Modal, Focus bool
	WideBar                                                          bool

	OnClick       func()
	Value         any
	OnChange      any
	OnInput       func(string)
	OnSubmit      func(string)
	OnBlur        func()
	OnKeyDown     KeyHandler
	OnClose       any
	OnOpen        any
	OnSelect      any
	OnEmpty       func()
	OnEnter       func()
	OnWheel       any
	// OnScroll 在编辑框纵向滚动偏移变化时回调（滚轮、光标跟随、外部设置都会
	// 经绘制出口统一通知），供「编辑区滚动 → 预览同步」这类联动使用。
	OnScroll  func(float32)
	OnPaint   any
	OnPointerDown func()
	// OnContextMenu 在指针按下右键、且按在这个节点上时回调，x/y 是客户区指针位置
	// （DIP）。
	//
	// 和 OnClick 一样「谁在最上面就归谁」，不做冒泡 —— 冒泡会让「右键点在编辑区」和
	// 「右键点在编辑区的空白处」分不清，而这两处想要的菜单不一样。
	//
	// 带坐标是因为菜单要长在指针那儿：右键面板只有坐标这一个锚点（下拉菜单有个按钮
	// 可以当锚），宿主拿不到坐标就得自己再问「指针在哪」，凭空多一份要跟事件对齐的状态。
	OnContextMenu func(x, y float32)
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

	// Gradient 让节点背景走线性渐变填充（按钮、强调底）。nil 表示不用渐变，
	// 走 Style.Background 纯色。方向由 Vertical 决定，端点色 A==0 视为未设置。
	Gradient *GradientFill
	// Shadow 给节点背景加圆角投影。nil 表示无阴影。Color A==0 时按黑色低透明度画。
	Shadow *ShadowStyle
	// Accent 是 empty 这类展示节点的「强调」开关：徽章底/图标改用主题强调色，
	// 并允许画装饰元素。零值与现状一致。
	Accent bool

	// AtX / AtY 是浮层要长出来的位置（DIP），目前只有右键面板（ctxmenu）用。
	//
	// 之所以做成 Props 字段而不是让宿主塞 Attrs：位置是这个控件的公开契约
	// ——右键面板只有「按坐标摆」一种用法，宿主必须说得出摆哪儿。
	AtX, AtY float32
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
	parent     *VNode
	// meas 跨帧复用 Measure 结果。滚动不重建树时避免重复 MeasureText。
	meas [2]measMemo
}

// measMemo 记住某次 Measure 的约束与结果。两个槽覆盖 HStack「自然宽 / 分配宽」两遍。
type measMemo struct {
	maxW, maxH, w, h                float32
	width, height, flex, pad, gap   float32
	minW, minH, maxWidth, maxHeight float32
	nSize, pSize                    float32
	nWeight, pWeight, childN        int
	nFamily, pFamily, text          string
	ok                              bool
}

func vnodeMeasureText(n *VNode) string {
	if n == nil {
		return ""
	}
	if n.Text != "" {
		return n.Text
	}
	if n.Attrs != nil {
		if s, ok := n.Attrs["value"].(string); ok {
			return s
		}
	}
	return ""
}

func (n *VNode) measureMatches(m *measMemo, maxW, maxH float32, family string, fontSize float32, weight int) bool {
	if n == nil || m == nil || !m.ok {
		return false
	}
	s := n.Style
	return m.maxW == maxW && m.maxH == maxH &&
		m.width == s.Width && m.height == s.Height &&
		m.flex == s.Flex && m.pad == s.Padding && m.gap == s.Gap &&
		m.minW == s.MinWidth && m.minH == s.MinHeight &&
		m.maxWidth == s.MaxWidth && m.maxHeight == s.MaxHeight &&
		m.nSize == s.FontSize && m.nWeight == s.Weight && m.nFamily == s.FontFamily &&
		m.pSize == fontSize && m.pWeight == weight && m.pFamily == family &&
		m.childN == len(n.Children) && m.text == vnodeMeasureText(n)
}

// LookupMeasure 若本节点在相同约束下已经量过，返回缓存尺寸。
func (n *VNode) LookupMeasure(maxW, maxH float32, family string, fontSize float32, weight int) (float32, float32, bool) {
	if n == nil {
		return 0, 0, false
	}
	for i := range n.meas {
		if n.measureMatches(&n.meas[i], maxW, maxH, family, fontSize, weight) {
			return n.meas[i].w, n.meas[i].h, true
		}
	}
	return 0, 0, false
}

// StoreMeasure 记下本次测量，供后续帧复用。
func (n *VNode) StoreMeasure(maxW, maxH float32, family string, fontSize float32, weight int, w, h float32) {
	if n == nil {
		return
	}
	s := n.Style
	n.meas[1] = n.meas[0]
	n.meas[0] = measMemo{
		maxW: maxW, maxH: maxH, w: w, h: h,
		width: s.Width, height: s.Height, flex: s.Flex, pad: s.Padding, gap: s.Gap,
		minW: s.MinWidth, minH: s.MinHeight, maxWidth: s.MaxWidth, maxHeight: s.MaxHeight,
		nSize: s.FontSize, pSize: fontSize, nWeight: s.Weight, pWeight: weight,
		childN: len(n.Children), nFamily: s.FontFamily, pFamily: family,
		text: vnodeMeasureText(n), ok: true,
	}
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
	putBool("widebar", p.WideBar)
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
	putAny("gradient", p.Gradient)
	putAny("shadow", p.Shadow)
	putBool("accent", p.Accent)
	if p.OnInput != nil {
		put("onInput", p.OnInput)
	}
	if p.OnSubmit != nil {
		put("onSubmit", p.OnSubmit)
	}
	if p.OnBlur != nil {
		put("onBlur", p.OnBlur)
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
	if p.OnScroll != nil {
		put("onScroll", p.OnScroll)
	}
	if p.OnPointerDown != nil {
		put("onPointerDown", p.OnPointerDown)
	}
	if p.OnContextMenu != nil {
		put("onContextMenu", p.OnContextMenu)
	}
	if p.ContentWidth != 0 {
		put("width", p.ContentWidth)
	}
	// 0 不写：面板坐标的 0 就是原点，读出来缺省也是 0，两种情况等价，
	// 而每个节点都多两个键不划算。
	if p.AtX != 0 {
		put("atX", p.AtX)
	}
	if p.AtY != 0 {
		put("atY", p.AtY)
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
	if dst.Margin == 0 {
		dst.Margin = src.Margin
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
	if dst.Overflow == "" {
		dst.Overflow = src.Overflow
	}
	if dst.OverflowX == "" {
		dst.OverflowX = src.OverflowX
	}
	if dst.OverflowY == "" {
		dst.OverflowY = src.OverflowY
	}
	if dst.Position == "" {
		dst.Position = src.Position
		dst.Top, dst.Right, dst.Bottom, dst.Left = src.Top, src.Right, src.Bottom, src.Left
		dst.Inset = src.Inset
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
