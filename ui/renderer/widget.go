package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// Widget 是标签对应的度量 / 布局 / 绘制。由 ui/components 在 init 里注册。
// renderer 不认识具体控件，只按 tag 查找。
type Widget struct {
	Measure func(ctx Context, n *runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (w, h float32)
	Layout  func(ctx Context, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme)
	Paint   func(ctx Context, n *runtime.VNode, style TextStyle, th theme.Theme)
}

var widgets = map[string]Widget{}

// Register 注册一个标签。后注册的覆盖先注册的。
func Register(tag string, w Widget) {
	if tag == "" {
		return
	}
	widgets[tag] = w
}

// Lookup 查找已注册控件。
func Lookup(tag string) (Widget, bool) {
	w, ok := widgets[tag]
	return w, ok
}

// MeasureNode 度量任意子节点，供控件布局子树。
func MeasureNode(ctx Context, n *runtime.VNode, maxW, maxH float32, style TextStyle, th theme.Theme) (float32, float32) {
	return measureNode(ctx, n, maxW, maxH, style, th)
}

// LayoutNode 放置任意子节点。
func LayoutNode(ctx Context, n *runtime.VNode, x, y, w, h float32, style TextStyle, th theme.Theme) {
	layoutNode(ctx, n, x, y, w, h, style, th)
}

// ApplyStyle 把行内 style 叠到文本样式上。
func ApplyStyle(base TextStyle, s runtime.Style) TextStyle {
	return applyStyle(base, s)
}

// ColorFrom 把 runtime.Color 转成绘制颜色。
func ColorFrom(c runtime.Color) Color { return colorFrom(c) }

// ColorSet 报告颜色是否显式设置（A>0）。
func ColorSet(c runtime.Color) bool { return colorSet(c) }

// PaintBackground 若节点有 background 则填充。
func PaintBackground(ctx Context, n *runtime.VNode) { paintBackground(ctx, n) }



// PointerIn 报告指针是否落在 DIP 矩形内。
func PointerIn(x, y, w, h float32) bool { return pointerIn(x, y, w, h) }

// PointerDown 报告左键是否按下。
func PointerDown() bool { return pointerDown }

// Enabled 报告节点是否可交互。disabled=true 或 enabled=false 时不可用。
func Enabled(n *runtime.VNode) bool {
	if n == nil || n.Attrs == nil {
		return true
	}
	if v, ok := n.Attrs["enabled"].(bool); ok {
		return v
	}
	if v, ok := n.Attrs["disabled"].(bool); ok {
		return !v
	}
	return true
}

// AttrString 读取 Attrs 字符串字段。
func AttrString(n *runtime.VNode, key string) string {
	if n == nil || n.Attrs == nil {
		return ""
	}
	s, _ := n.Attrs[key].(string)
	return s
}

// AttrBool 读取 Attrs 布尔字段。
func AttrBool(n *runtime.VNode, key string) bool {
	if n == nil || n.Attrs == nil {
		return false
	}
	v, _ := n.Attrs[key].(bool)
	return v
}

// AttrInt 读取 Attrs 整数字段。
func AttrInt(n *runtime.VNode, key string) int {
	if n == nil || n.Attrs == nil {
		return 0
	}
	switch v := n.Attrs[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	default:
		return 0
	}
}

// Attr 读取任意 Attrs 字段。
func Attr(n *runtime.VNode, key string) any {
	if n == nil || n.Attrs == nil {
		return nil
	}
	return n.Attrs[key]
}

// AttrBytes 读取 []byte 字段。
func AttrBytes(n *runtime.VNode, key string) []byte {
	if n == nil || n.Attrs == nil {
		return nil
	}
	b, _ := n.Attrs[key].([]byte)
	return b
}

// Contains 报告点是否落在节点布局矩形内。
func Contains(n *runtime.VNode, x, y float32) bool {
	if n == nil {
		return false
	}
	return x >= n.X && y >= n.Y && x < n.X+n.W && y < n.Y+n.H
}
