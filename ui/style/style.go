// Package style 提供 CSS 风格的行内样式体系。
//
// 设计目标：让声明式 UI 拥有和 Web 一样的布局表达力。
//
//	// 宽高支持四种单位：
//	style.Size("100vw")   // 视口宽度的 100%
//	style.Size("50vh")    // 视口高度的 50%
//	style.Size("120px")   // 固定像素
//	style.Size("50%")     // 父容器的 50%
//
//	// 一段式声明（对应 CSS 行内样式）：
//	style.Parse(map[string]string{
//	    "width":     "100vw",
//	    "height":    "100vh",
//	    "padding":   "32px",
//	    "align":     "center",   // 交叉轴居中
//	    "justify":   "center",   // 主轴居中
//	    "gap":       "16px",
//	})
//
// 视口尺寸（vw/vh 的基准）由宿主在每帧布局前通过 SetViewport 注入。
package style

import (
	"fmt"
	"strconv"
	"strings"
)

// Viewport 视口尺寸（vw/vh 的基准），由宿主每帧更新。
var viewport struct {
	w, h int32
}

// SetViewport 设置视口尺寸，窗口 resize 时调用。
func SetViewport(width, height int32) {
	viewport.w, viewport.h = width, height
}

// Viewport 返回当前视口尺寸。
func Viewport() (width, height int32) {
	return viewport.w, viewport.h
}

// Value 一个可解析的尺寸值：延迟求值，视口变化时自动跟随。
type Value struct {
	raw     string
	kind    valueKind
	num     float64
	percent bool // "%" 单位
}

type valueKind int

const (
	kindInvalid valueKind = iota
	kindPx
	kindVW
	kindVH
	kindPercent
)

// Size 解析尺寸字符串："120px" / "50vw" / "30vh" / "50%" / "42"（默认 px）。
// 解析失败返回零值 Value（Resolve 时得 0）。
func Size(s string) Value {
	v := Value{raw: s}
	s = strings.TrimSpace(strings.ToLower(s))
	switch {
	case strings.HasSuffix(s, "px"):
		v.kind, v.num = kindPx, parseFloat(s[:len(s)-2])
	case strings.HasSuffix(s, "vw"):
		v.kind, v.num = kindVW, parseFloat(s[:len(s)-2])
	case strings.HasSuffix(s, "vh"):
		v.kind, v.num = kindVH, parseFloat(s[:len(s)-2])
	case strings.HasSuffix(s, "%"):
		v.kind, v.num, v.percent = kindPercent, parseFloat(s[:len(s)-1]), true
	default:
		v.kind, v.num = kindPx, parseFloat(s)
	}
	return v
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

// Resolve 把尺寸值解析为像素。
// parentSize 用于 "%" 单位；vw/vh 用全局视口。
func (v Value) Resolve(parentSize int32) int32 {
	switch v.kind {
	case kindPx:
		return int32(v.num)
	case kindVW:
		return int32(v.num * float64(viewport.w) / 100)
	case kindVH:
		return int32(v.num * float64(viewport.h) / 100)
	case kindPercent:
		return int32(v.num * float64(parentSize) / 100)
	default:
		return 0
	}
}

// String 返回原始声明，便于调试。
func (v Value) String() string { return v.raw }

// Edges 四边尺寸（margin/padding 用）。
// 支持一到四个值，语义与 CSS 一致：
//
//	"8px"            → 上下左右全是 8
//	"8px 16px"       → 上下 8，左右 16
//	"8px 16px 24px"  → 上 8，左右 16，下 24
//	"8px 16px 24px 32px" → 上 右 下 左（顺时针）
type Edges struct {
	Top, Right, Bottom, Left Value
}

// ParseEdges 解析四边值。
func ParseEdges(s string) Edges {
	parts := strings.Fields(s)
	var vals []Value
	for _, p := range parts {
		vals = append(vals, Size(p))
	}
	switch len(vals) {
	case 1:
		return Edges{vals[0], vals[0], vals[0], vals[0]}
	case 2:
		return Edges{vals[0], vals[1], vals[0], vals[1]}
	case 3:
		return Edges{vals[0], vals[1], vals[2], vals[1]}
	case 4:
		return Edges{vals[0], vals[1], vals[2], vals[3]}
	default:
		return Edges{}
	}
}

// ResolveTop 等解析四边为像素。parentSize 是百分比基准
// （水平边用父宽，垂直边用父高，由调用方分别传入）。
func (e Edges) ResolveTop(parentH int32) int32    { return e.Top.Resolve(parentH) }
func (e Edges) ResolveBottom(parentH int32) int32 { return e.Bottom.Resolve(parentH) }
func (e Edges) ResolveLeft(parentW int32) int32   { return e.Left.Resolve(parentW) }
func (e Edges) ResolveRight(parentW int32) int32  { return e.Right.Resolve(parentW) }

// Style 行内样式：声明式组件的布局与外观描述。
// 零值即"未指定"，布局器按默认行为处理。
type Style struct {
	// 尺寸
	Width  Value // "100vw" / "200px" / "50%"；未指定 = 自动
	Height Value

	// 间距
	Margin  Edges // 外边距（相对父容器）
	Padding Edges // 内边距（压缩内容区）

	// Flex 布局
	Flex      float64 // 弹性因子（0 = 固定，>0 = 占据剩余空间的比例）
	Gap       Value   // 子组件间距
	Align     string  // 交叉轴对齐：start/center/end/stretch
	Justify   string  // 主轴对齐：start/center/end/between/around
	Direction string  // flex 方向：row/column（容器）

	// 外观
	Background string // "#RRGGBB" 或 "#RRGGBBAA"
	Color      string // 文本色
	Visible    bool   // false = 隐藏（占位）
	hasVisible bool
}

// HasVisible 报告 Visible 是否被显式设置。
func (s Style) HasVisible() bool { return s.hasVisible }

// Parse 从字符串 map 构造样式（对应 CSS 行内风格的键值对）。
//
//	style.Parse(map[string]string{
//	    "width": "100vw", "height": "100vh",
//	    "padding": "32px", "align": "center", "justify": "center",
//	})
func Parse(m map[string]string) Style {
	s := Style{Visible: true, hasVisible: false}
	for k, v := range m {
		switch k {
		case "width":
			s.Width = Size(v)
		case "height":
			s.Height = Size(v)
		case "margin":
			s.Margin = ParseEdges(v)
		case "padding":
			s.Padding = ParseEdges(v)
		case "flex":
			s.Flex = parseFloat(v)
		case "gap":
			s.Gap = Size(v)
		case "align":
			s.Align = v
		case "justify":
			s.Justify = v
		case "direction", "flex-direction":
			s.Direction = v
		case "background", "background-color":
			s.Background = v
		case "color":
			s.Color = v
		case "visible":
			s.Visible = v != "false" && v != "0" && v != "hidden"
			s.hasVisible = true
		default:
			// 未知键忽略（宽容处理，与 CSS 一致）。
		}
	}
	return s
}

// ColorHex 解析 "#RRGGBB" 或 "#RRGGBBAA" 为 uint32（0xRRGGBB）。
// 解析失败返回 def。
func ColorHex(s string, def uint32) uint32 {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 && len(s) != 8 {
		return def
	}
	rgb, err := strconv.ParseUint(s[:6], 16, 32)
	if err != nil {
		return def
	}
	return uint32(rgb)
}

// FromProps 从 runtime.Props 中提取 "style" 键（map[string]string 或 Style）。
// 返回零值 Style 表示未提供。
func FromProps(props map[string]any) Style {
	switch v := props["style"].(type) {
	case Style:
		return v
	case map[string]string:
		return Parse(v)
	case nil:
		return Style{Visible: true}
	default:
		fmt.Println("style: 不支持的 style 类型，已忽略")
		return Style{Visible: true}
	}
}
