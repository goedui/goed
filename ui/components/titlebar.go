package components

import (
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("titlebar", renderer.Widget{
		Measure: measureTitleBar,
		Paint:   paintTitleBar,
	})
}

// TitleBar 是纯 Go 的自定义窗口标题栏。
// 系统非客户区已去掉；最小化 / 最大化 / 关闭由 WM_NCHITTEST
// 映射到 HTMINBUTTON / HTMAXBUTTON / HTCLOSE，交给 Windows 执行。
// 左侧绘制默认应用图标 App。
//
// Style.Background / Color 覆盖主题标题栏底色和文字色；未设置时仍走主题。
// platform.AppTitleBar 为 true 时（应用自绘标题栏）只把 apptitle 传下去：
// 框架不画这一条，也不占它的高度，见 renderer.paintFrame / paintTitleBar。
var TitleBar = runtime.DefineComponent("TitleBar", func(ctx *runtime.SetupContext) runtime.RenderFunc {
	return func() []*runtime.VNode {
		st := platform.Frame()
		style := runtime.Style{}
		if ctx != nil {
			style = ctx.Props().Style
		}
		n := runtime.Tag("titlebar", runtime.Attrs{
			"title":     st.Title,
			"maximized": st.Maximized,
			"active":    st.Active,
			"hover":     int(st.Hover),
			"pressed":   int(st.Pressed),
			"icon":      App.Paint,
			"apptitle":  platform.AppTitleBar,
		})
		n.Style = style
		return []*runtime.VNode{n}
	}
})

func measureTitleBar(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = n
	_ = maxH
	_ = style
	_ = th
	w := maxW
	if w <= 0 && ctx != nil {
		w = ctx.Width()
	}
	return w, renderer.TitleBarHeight
}
