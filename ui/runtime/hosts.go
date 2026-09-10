package runtime

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/style"
	"strconv"
	"strings"
)

// TextHost 文本宿主元素：runtime 内置的最小元素。
type TextHost struct {
	core.BaseComponent
	Text     string
	Color    uint32
	Ellipsis bool
	// measured 记录文本自然尺寸，避免每帧重复测量。
	measuredW, measuredH int32
	measuredText         string
}

// MeasureContent 实现 Measurable：按文本内容测量自然尺寸。
func (t *TextHost) MeasureContent(ctx core.DrawingContext) {
	if t.measuredText == t.Text {
		t.Bounds.W, t.Bounds.H = t.measuredW, t.measuredH
		return
	}
	w, h := ctx.MeasureText(t.Text)
	if w <= 0 {
		w = int32(len([]rune(t.Text))) * 8
	}
	if h <= 0 {
		h = 20
	}
	t.measuredW, t.measuredH, t.measuredText = w, h, t.Text
	t.Bounds.W, t.Bounds.H = w, h
}

// Render 绘制文本。
func (t *TextHost) Render(ctx core.DrawingContext) {
	if !t.Visible {
		return
	}
	if t.Layout != nil {
		t.Layout.Arrange(t.Bounds)
	}
	text := t.Text
	if t.Ellipsis {
		if w, _ := ctx.MeasureText(text); w > t.Bounds.W {
			r := []rune(text)
			for len(r) > 0 {
				r = r[:len(r)-1]
				text = string(r) + "…"
				if w, _ := ctx.MeasureText(text); w <= t.Bounds.W {
					break
				}
			}
		}
	}
	if t.Bounds.W > 0 && t.Bounds.H > 0 {
		core.WithClip(ctx, t.Bounds, func() { ctx.DrawText(t.Bounds.X, t.Bounds.Y, text, t.Color) })
	} else {
		ctx.DrawText(t.Bounds.X, t.Bounds.Y, text, t.Color)
	}
	for _, child := range t.Children {
		if child != nil {
			child.Render(ctx)
		}
	}
}

// textElement "text" 元素实现。
type textElement struct{}

func (textElement) Tag() string { return "text" }

func (textElement) Create(props Props) core.Component {
	color := props.Color("color", 0xFFFFFF)
	if spec := style.FromProps(map[string]any(props)); spec.Color != "" {
		color = parseColor(spec.Color, color)
	}
	if raw, ok := props["style"]; ok {
		if styleMap, ok := raw.(map[string]string); ok {
			if value, exists := styleMap["color"]; exists {
				color = parseColor(value, color)
			}
		}
	}
	host := &TextHost{
		Text:  props.Str("value", ""),
		Color: color,
	}
	host.Visible = props.Bool("visible", true)
	host.Ellipsis = props.Bool("ellipsis", false)
	return host
}

func (textElement) Update(host core.Component, old, new Props) {
	t := host.(*TextHost)
	t.Ellipsis = new.Bool("ellipsis", false)
	if nv := new.Str("value", ""); nv != old.Str("value", "") {
		t.Text = nv
	}
	if nc := new.Color("color", 0xFFFFFF); nc != old.Color("color", 0xFFFFFF) {
		t.Color = nc
	}
	if raw, ok := new["style"]; ok {
		if styleMap, ok := raw.(map[string]string); ok {
			if value, exists := styleMap["color"]; exists {
				t.Color = parseColor(value, t.Color)
			}
		}
	}
	if spec := style.FromProps(map[string]any(new)); spec.Color != "" {
		t.Color = parseColor(spec.Color, t.Color)
	}
	if nv := new.Bool("visible", true); nv != old.Bool("visible", true) {
		t.Visible = nv
	}
}

func parseColor(value string, fallback uint32) uint32 {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "#") {
		if parsed, err := strconv.ParseUint(strings.TrimPrefix(value, "#"), 16, 32); err == nil {
			return uint32(parsed)
		}
	}
	return fallback
}

// ComponentHost 组件的宿主容器：包装 BaseComponent 并携带
// 父布局需要的 flex 信息（来自组件 VNode 的 "flex" prop）。
type ComponentHost struct {
	core.BaseComponent
	flex int32
}

// Flex 实现 flex 探测接口。
func (c *ComponentHost) Flex() int32 { return c.flex }

func (c *ComponentHost) MeasureWidth(ctx core.DrawingContext, width int32) core.Size {
	var size core.Size
	for _, child := range c.Children {
		s := core.MeasureComponent(ctx, child, width)
		size.Width = max(size.Width, s.Width)
		size.Height = max(size.Height, s.Height)
	}
	return size
}

// MeasureContent lets a component participate in its parent's intrinsic
// measurement when it is not flexed.
func (c *ComponentHost) MeasureContent(ctx core.DrawingContext) {
	if c.Bounds.W > 0 || c.Bounds.H > 0 {
		return
	}
	for _, child := range c.Children {
		if measured, ok := child.(interface{ MeasureContent(core.DrawingContext) }); ok {
			measured.MeasureContent(ctx)
		}
		bounds := child.GetBounds()
		if bounds.W > c.Bounds.W {
			c.Bounds.W = bounds.W
		}
		if bounds.H > c.Bounds.H {
			c.Bounds.H = bounds.H
		}
	}
	if c.Bounds.W == 0 && c.Bounds.H == 0 {
		c.Bounds.H = 32
	}
}

// MeasureIntrinsic recomputes a non-flex component's natural size. It is
// intentionally separate from MeasureContent so parent stacks can request a
// fresh measurement even when the host still has bounds from the prior patch.
func (c *ComponentHost) MeasureIntrinsic(ctx core.DrawingContext) {
	if c.flex > 0 || len(c.Children) == 0 {
		return
	}
	child := c.Children[0]
	position := child.GetBounds()
	child.SetBounds(core.Rect{X: position.X, Y: position.Y})
	if measured, ok := child.(interface{ MeasureContent(core.DrawingContext) }); ok {
		measured.MeasureContent(ctx)
	}
	childBounds := child.GetBounds()
	c.Bounds.W = childBounds.W
	c.Bounds.H = childBounds.H
}

// Render forwards the component host's assigned rectangle to its single root
// child. BaseComponent cannot do this because it does not know that a component
// host is a transparent layout boundary.
func (c *ComponentHost) Render(ctx core.DrawingContext) {
	if !c.Visible {
		return
	}
	for _, child := range c.Children {
		if child == nil {
			continue
		}
		child.SetBounds(c.Bounds)
		child.Render(ctx)
	}
}

func init() {
	RegisterElement(textElement{})
}
