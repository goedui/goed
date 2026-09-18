package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("canvas", renderer.Widget{
		Measure: measureCanvas,
		Layout:  layoutCanvas,
		Paint:   paintCanvas,
	})
}

// Canvas 自定义绘制组件。允许用户通过回调函数直接访问 renderer.Context 进行绘制。
// 
// 用法：
//   H("canvas", Props{
//     Style: Style{Width: 800, Height: 600},
//     Attrs: map[string]any{
//       "onPaint": func(ctx renderer.Context, x, y, w, h float32) {
//         // 在这里进行自定义绘制
//         ctx.FillRect(x, y, w, h, renderer.RGB(255, 0, 0))
//       },
//     },
//   })
func Canvas(parts ...any) *runtime.VNode {
	return runtime.H("canvas", parts...)
}

// PaintCallback 是 canvas 组件的绘制回调函数类型。
// ctx: 渲染上下文
// x, y: canvas 在屏幕上的位置
// w, h: canvas 的宽高
type PaintCallback func(ctx renderer.Context, x, y, w, h float32)

func measureCanvas(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	w := n.Style.Width
	h := n.Style.Height
	
	if w <= 0 {
		w = maxW
	}
	if h <= 0 {
		h = maxH
	}
	if w <= 0 {
		w = 400 // 默认宽度
	}
	if h <= 0 {
		h = 300 // 默认高度
	}
	
	return w, h
}

func layoutCanvas(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
}

func paintCanvas(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	
	// 绘制背景
	renderer.PaintBackground(ctx, n)
	
	// 调用用户提供的绘制回调
	if n.Attrs != nil {
		if callback, ok := n.Attrs["onPaint"].(PaintCallback); ok {
			callback(ctx, n.X, n.Y, n.W, n.H)
		} else if callback, ok := n.Attrs["onPaint"].(func(renderer.Context, float32, float32, float32, float32)); ok {
			callback(ctx, n.X, n.Y, n.W, n.H)
		}
	}
}
