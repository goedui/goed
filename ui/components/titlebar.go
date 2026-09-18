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
var TitleBar = runtime.DefineComponent("TitleBar", func(_ *runtime.SetupContext) runtime.RenderFunc {
	return func() []*runtime.VNode {
		st := platform.Frame()
		return []*runtime.VNode{
			runtime.Tag("titlebar", runtime.Attrs{
				"title":     st.Title,
				"maximized": st.Maximized,
				"active":    st.Active,
				"hover":     int(st.Hover),
				"pressed":   int(st.Pressed),
				"icon":      App.Paint,
			}),
		}
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
