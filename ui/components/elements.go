// Package components contains reusable native hosts for declarative UI.
package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/style"
)

type Measurable interface{ MeasureContent(core.DrawingContext) }

// BoxHost measures natural sizes separately from the bounds assigned by a parent.
type BoxHost struct {
	core.BaseComponent
	Background                                           uint32
	Padding                                              int32
	Style                                                style.Style
	Radius                                               int32
	Border                                               uint32
	selfFlex                                             int32
	paddingTop, paddingRight, paddingBottom, paddingLeft int32
}

func (b *BoxHost) Flex() int32 { return b.selfFlex }
func (b *BoxHost) syncLayout() {
	fl, ok := b.Layout.(*core.FlexLayout)
	if !ok {
		return
	}
	fl.Children = fl.Children[:0]
	for _, c := range b.Children {
		if v, ok := c.(interface{ IsVisible() bool }); ok && !v.IsVisible() {
			continue
		}
		f := int32(0)
		if flex, ok := c.(interface{ Flex() int32 }); ok {
			f = flex.Flex()
		}
		fl.Children = append(fl.Children, core.LayoutNode{Component: c, Flex: f})
	}
}
func (b *BoxHost) measureChildren(ctx core.DrawingContext, width int32) {
	fl, ok := b.Layout.(*core.FlexLayout)
	if !ok {
		return
	}
	var fixed, totalFlex int32
	for _, n := range fl.Children {
		if fl.Direction == core.Row && n.Flex > 0 {
			totalFlex += n.Flex
			continue
		}
		size := core.MeasureComponent(ctx, n.Component, width)
		fixed += size.Width
	}
	if fl.Direction == core.Row && totalFlex > 0 {
		remaining := max(0, width-fixed-fl.Gap*int32(max(0, len(fl.Children)-1)))
		for _, n := range fl.Children {
			if n.Flex > 0 {
				core.MeasureComponent(ctx, n.Component, remaining*n.Flex/totalFlex)
			}
		}
	}
}
func (b *BoxHost) MeasureContent(ctx core.DrawingContext) {
	size := b.MeasureWidth(ctx, b.Bounds.W)
	b.Bounds.W, b.Bounds.H = size.Width, size.Height
}
func (b *BoxHost) MeasureWidth(ctx core.DrawingContext, width int32) core.Size {
	if b.Layout == nil {
		return core.Size{Width: b.Bounds.W, Height: b.Bounds.H}
	}
	b.resolveStyle(width)
	b.syncLayout()
	b.measureChildren(ctx, max(0, width-b.paddingLeft-b.paddingRight))
	size := b.Layout.(*core.FlexLayout).Calculate(core.Constraints{})
	size.Width += b.paddingLeft + b.paddingRight
	size.Height += b.paddingTop + b.paddingBottom
	return size
}
func (b *BoxHost) Render(ctx core.DrawingContext) {
	if !b.Visible {
		return
	}
	b.resolveStyle(b.Bounds.W)
	if b.Layout != nil {
		b.syncLayout()
		content := b.Bounds
		content.X += b.paddingLeft
		content.Y += b.paddingTop
		content.W = max(0, content.W-b.paddingLeft-b.paddingRight)
		content.H = max(0, content.H-b.paddingTop-b.paddingBottom)
		b.measureChildren(ctx, content.W)
		b.Layout.Arrange(content)
	} else {
		for _, c := range b.Children {
			c.SetBounds(b.Bounds)
		}
	}
	if b.Background != 0 {
		ctx.DrawRoundedRect(b.Bounds.X, b.Bounds.Y, b.Bounds.W, b.Bounds.H, b.Radius, b.Background, true)
	}
	if b.Border != 0 {
		ctx.DrawRoundedRect(b.Bounds.X, b.Bounds.Y, b.Bounds.W, b.Bounds.H, b.Radius, b.Border, false)
	}
	for _, c := range b.Children {
		c.Render(ctx)
	}
}
func (b *BoxHost) resolveStyle(width int32) {
	p := max(0, b.Padding)
	b.paddingTop, b.paddingRight, b.paddingBottom, b.paddingLeft = p, p, p, p
	if b.Style.Padding.Top.String() != "" || b.Style.Padding.Right.String() != "" || b.Style.Padding.Bottom.String() != "" || b.Style.Padding.Left.String() != "" {
		b.paddingTop = b.Style.Padding.ResolveTop(b.Bounds.H)
		b.paddingRight = b.Style.Padding.ResolveRight(width)
		b.paddingBottom = b.Style.Padding.ResolveBottom(b.Bounds.H)
		b.paddingLeft = b.Style.Padding.ResolveLeft(width)
	}
	if fl, ok := b.Layout.(*core.FlexLayout); ok {
		if b.Style.Gap.String() != "" {
			fl.Gap = b.Style.Gap.Resolve(width)
		}
		if b.Style.Align != "" {
			fl.AlignItems = alignItemsOf(b.Style.Align)
		}
		if b.Style.Justify != "" {
			fl.JustifyContent = justifyOf(b.Style.Justify)
		}
	}
	if b.Style.Background != "" {
		b.Background = style.ColorHex(b.Style.Background, b.Background)
	}
	if b.Style.HasVisible() {
		b.Visible = b.Style.Visible
	}
}
func styleFromProps(p runtime.Props) style.Style { return style.FromProps(map[string]any(p)) }
func updateBox(b *BoxHost, p runtime.Props) {
	b.Background = p.Color("bg", 0)
	b.Padding = p.Int("padding", 0)
	b.Radius = p.Int("radius", 0)
	b.Border = p.Color("border", 0)
	b.Style = styleFromProps(p)
	b.selfFlex = p.Int("flex", 0)
	b.Visible = p.Bool("visible", true)
	if fl, ok := b.Layout.(*core.FlexLayout); ok {
		fl.Gap = p.Int("gap", 0)
		fl.AlignItems = alignItemsOf(p.Str("align", "stretch"))
		fl.JustifyContent = justifyOf(p.Str("justify", "start"))
	}
}

type boxElement struct{}

func (boxElement) Tag() string { return "box" }
func (boxElement) Create(p runtime.Props) core.Component {
	b := &BoxHost{BaseComponent: *core.NewBaseComponent()}
	if dir := p.Str("direction", ""); dir != "" {
		b.Layout = core.NewFlexLayout(directionOf(dir))
	}
	updateBox(b, p)
	return b
}
func (boxElement) Update(c core.Component, old, p runtime.Props) {
	b := c.(*BoxHost)
	if old.Str("direction", "") != p.Str("direction", "") {
		b.Layout = nil
		if dir := p.Str("direction", ""); dir != "" {
			b.Layout = core.NewFlexLayout(directionOf(dir))
		}
	}
	updateBox(b, p)
}

type stackElement struct {
	tag string
	dir core.FlexDirection
}

func (s stackElement) Tag() string { return s.tag }
func (s stackElement) Create(p runtime.Props) core.Component {
	b := &BoxHost{BaseComponent: *core.NewBaseComponent()}
	b.Layout = core.NewFlexLayout(s.dir)
	updateBox(b, p)
	return b
}
func (s stackElement) Update(c core.Component, _, p runtime.Props) { updateBox(c.(*BoxHost), p) }
func directionOf(s string) core.FlexDirection {
	if s == "row" {
		return core.Row
	}
	return core.Column
}
func alignItemsOf(s string) core.AlignItems {
	switch s {
	case "start":
		return core.AlignStart
	case "center":
		return core.AlignCenter
	case "end":
		return core.AlignEnd
	}
	return core.AlignStretch
}
func justifyOf(s string) core.JustifyContent {
	switch s {
	case "center":
		return core.JustifyCenter
	case "end":
		return core.JustifyEnd
	case "between":
		return core.JustifySpaceBetween
	case "around":
		return core.JustifySpaceAround
	}
	return core.JustifyStart
}
func init() {
	runtime.RegisterElement(boxElement{})
	runtime.RegisterElement(stackElement{tag: "vstack", dir: core.Column})
	runtime.RegisterElement(stackElement{tag: "hstack", dir: core.Row})
	runtime.RegisterElement(buttonElement{})
	runtime.RegisterElement(inputElement{})
}
