package renderer

import (
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

const (
	// TitleBarHeight 是自定义标题栏高度，单位 DIP。与 Windows 11 标题栏接近。
	TitleBarHeight = 32
	// CaptionButtonWidth 是每个系统按钮的宽度，单位 DIP。
	CaptionButtonWidth = 46
)

func isTitleBarNode(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	if n.Kind == runtime.KindElement && n.Tag == "titlebar" {
		return true
	}
	if n.Kind == runtime.KindComponent && n.Component != nil && n.Component.Name == "TitleBar" {
		return true
	}
	return false
}

func unwrapTitleBar(n *runtime.VNode) *runtime.VNode {
	if n == nil {
		return nil
	}
	if n.Kind == runtime.KindElement && n.Tag == "titlebar" {
		return n
	}
	if n.Kind == runtime.KindComponent && n.Component != nil {
		runtime.ResolveComponent(n)
		for _, c := range n.Children {
			if t := unwrapTitleBar(c); t != nil {
				return t
			}
		}
	}
	return nil
}

func paintFrame(ctx Context, n *runtime.VNode, th theme.Theme) float32 {
	st := TextStyle{
		FontFamily: th.FontFamily,
		FontSize:   th.FontSize,
		Color:      ColorFrom(th.Foreground),
		Weight:     400,
	}
	if w, ok := Lookup("titlebar"); ok && w.Paint != nil {
		w.Paint(ctx, n, st, th)
	}
	return TitleBarHeight
}
