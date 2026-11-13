package components

import (
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("caption", renderer.Widget{
		Measure: measureCaption,
		Layout:  layoutCaption,
		Paint:   paintCaption,
	})
}

// WindowCaption 是窗口右上角的最小化 / 最大化 / 关闭。
//
// 点击不走 OnClick。宿主把客户区最右三格判成 HTMINBUTTON / HTMAXBUTTON /
// HTCLOSE，这里只负责把图形画在那三格上。挂了 OnClick 会被 ClientHitTest
// 当成应用控件，系统按钮就失灵了。
//
// 悬停和最大化图标在绘制时读 platform.Frame。应用树只在 dirty 时重建，
// 而系统按钮的 hover 不经过 dirty，写进 vnode 的话红色悬停出不来。
//
// 宿主没有系统按钮（浏览器）时返回 nil，调用方不必再判断。
func WindowCaption() *runtime.VNode {
	if !platform.CaptionButtons {
		return nil
	}
	return runtime.H("caption", runtime.Props{
		Style: runtime.Style{
			Width:  float32(3 * renderer.CaptionButtonWidth),
			Height: float32(renderer.TitleBarHeight),
		},
	})
}

func measureCaption(_ renderer.Context, n *runtime.VNode, _, _ float32, _ renderer.TextStyle, _ theme.Theme) (float32, float32) {
	w := float32(3 * renderer.CaptionButtonWidth)
	h := float32(renderer.TitleBarHeight)
	if n != nil && n.Style.Width > 0 {
		w = n.Style.Width
	}
	if n != nil && n.Style.Height > 0 {
		h = n.Style.Height
	}
	return w, h
}

func layoutCaption(ctx renderer.Context, n *runtime.VNode, _, _, w, h float32, _ renderer.TextStyle, _ theme.Theme) {
	if n == nil {
		return
	}
	// 命中区是客户区最右三格，从顶边起。父级 Padding 会把传入的 x 往里推，
	// 这里改钉到窗口坐标，图形和点击才是同一块。
	x := float32(0)
	if ctx != nil {
		x = ctx.Width() - w
	}
	if x < 0 {
		x = 0
	}
	n.X, n.Y, n.W, n.H = x, 0, w, h
}

// paintCaption 按窗口右缘画三按钮，不跟节点的布局坐标走。
//
// 宿主命中区是客户区最右三格、顶边起 32 DIP。菜单栏有 Padding 时，绝对定位的
// 包含块会往里缩，节点盒子就和命中区错开。图形必须钉在窗口坐标上。
func paintCaption(ctx renderer.Context, n *runtime.VNode, _ renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil || !platform.CaptionButtons {
		return
	}
	w := ctx.Width()
	strip := float32(3 * renderer.CaptionButtonWidth)
	h := float32(renderer.TitleBarHeight)
	if w < strip {
		return
	}
	st := platform.Frame()
	fg := renderer.Color{}
	if !st.Active {
		fg = renderer.ColorFrom(th.TitlebarForeground)
		if fg.A == 0 {
			fg = renderer.ColorFrom(th.Foreground)
		}
		fg.A *= 0.6
	}
	paintCaptionStrip(ctx, w-strip, 0, strip, h, int(st.Hover), int(st.Pressed), st.Maximized, th, renderer.Color{}, fg)
}

// paintCaptionStrip 在 (x,y,w,h) 里从右往左画关闭、最大化、最小化。
// barBg 的 A 为 0 时，悬停底色直接铺在调用方已经画好的条上。
// fg 的 A 为 0 时走主题标题栏前景色。
func paintCaptionStrip(ctx renderer.Context, x, y, w, h float32, hover, pressed int, maximized bool, th theme.Theme, barBg, fg renderer.Color) {
	if ctx == nil || !platform.CaptionButtons || w <= 0 || h <= 0 {
		return
	}
	btnW := float32(renderer.CaptionButtonWidth)
	if fg.A == 0 {
		fg = renderer.ColorFrom(th.TitlebarForeground)
		if fg.A == 0 {
			fg = renderer.ColorFrom(th.Foreground)
		}
	}
	paintCaptionButton(ctx, x+w-3*btnW, y, btnW, h, hover == 1, pressed == 1, false, th, barBg, fg, captionMin)
	paintCaptionButton(ctx, x+w-2*btnW, y, btnW, h, hover == 2, pressed == 2, false, th, barBg, fg, captionMaxRestore(maximized))
	paintCaptionButton(ctx, x+w-btnW, y, btnW, h, hover == 3, pressed == 3, true, th, barBg, fg, captionClose)
}
