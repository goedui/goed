package components

import (
	"math"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// 锚点常在工作区里 absolute 定位。tooltip 若按普通流先排，悬停首帧
// 读到的锚点坐标还是零值，提示会跑到窗口左上角。
func TestTooltipAnchorsAfterAbsoluteContent(t *testing.T) {
	const ax, ay, aw, ah = float32(600), float32(300), float32(44), float32(36)
	anchor := runtime.H("box", runtime.Props{
		ID: "anchor",
		Style: runtime.Style{
			Position: runtime.PositionAbsolute,
			Left:     ax, Top: ay, Width: aw, Height: ah,
		},
	})
	tip := Tooltip(runtime.Props{ID: "tip", Anchor: "anchor", Label: "刷新模型列表"})
	root := runtime.H("layer", runtime.Props{Style: runtime.Style{Flex: 1}}, anchor, tip)

	renderer.PaintTree(&mockContext{w: 1000, h: 800}, []*runtime.VNode{root}, theme.Light)
	if anchor.X != ax || anchor.Y != ay {
		t.Fatalf("锚点布局 = (%g,%g)，期望 (%g,%g)", anchor.X, anchor.Y, ax, ay)
	}
	if tip.W <= 0 || tip.H <= 0 {
		t.Fatalf("提示未完成布局: %+v", tip)
	}
	if d := math.Abs(float64(tip.X + tip.W/2 - (ax + aw/2))); d > 0.5 {
		t.Fatalf("提示应水平居中于按钮: tip=%+v anchor=(%g,%g,%g,%g)", tip, ax, ay, aw, ah)
	}
	if tip.Y+tip.H > ay-4 {
		t.Fatalf("提示应浮在按钮上方: tip=%+v anchorY=%g", tip, ay)
	}
}
