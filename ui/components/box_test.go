// box 容器的度量与布局常驻回归。
//
// measureBox 的两条兜底**故意不对称**：
//
//	宽：Style.Width → maxW → 没有硬兜底（父级给 0 就量 0 宽）
//	高：Style.Height → (Flex>0 ? maxH) → 1
//
// 高那条 1 是 spacer 语义 —— buddy 里大量 `ui.Box{Height: 1}` 当分隔条 / 间距用
// （buddy/src/components/kit.go、rich.go），量成 0 就整条看不见。宽不需要这条：
// 盒子宽度总由父级 flow 铺满。
//
// 和 measureImage 一样不看 ctx；和 measureLayer 一样在 Style 没写时退到父级给的
// maxW/maxH。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestMeasureBoxFallbackLadder(t *testing.T) {
	// ctx 给一个和期望值都不同的尺寸：measureBox 不读它。
	ctx := &mockContext{w: 1000, h: 800}
	th := theme.Light
	st := renderer.TextStyle{FontSize: 13}

	cases := []struct {
		name         string
		style        runtime.Style
		maxW, maxH   float32
		wantW, wantH float32
		why          string
	}{
		{"显式宽高优先", runtime.Style{Width: 200, Height: 80}, 1000, 800, 200, 80, "Style 写了就用它"},
		{"都不给 → 宽吃 maxW、高退 1", runtime.Style{}, 300, 200, 300, 1, "1 是 spacer 语义，不是 0"},
		{"Flex>0 → 高吃 maxH", runtime.Style{Flex: 1}, 300, 200, 300, 200, "弹性盒铺满"},
		{"Flex>0 但 maxH=0 → 仍退 1", runtime.Style{Flex: 1}, 300, 0, 300, 1, "maxH=0 不等于「高度 0」"},
		{"宽没有来源 → 0（宽不兜底）", runtime.Style{}, 0, 0, 0, 1, "宽**故意**没有硬兜底"},
		{"只给高", runtime.Style{Height: 80}, 0, 400, 0, 80, "高写了就不动，宽仍是 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := Box(runtime.Props{Style: tc.style})
			w, h := measureBox(ctx, n, tc.maxW, tc.maxH, st, th)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("measureBox = %gx%g，期望 %gx%g（%s）", w, h, tc.wantW, tc.wantH, tc.why)
			}
		})
	}
}

// layoutBox 把子节点摆在内边距里；nil 子节点跳过（不能让 nil 走进 LayoutNode）。
func TestLayoutBoxPlacesChildrenInsidePadding(t *testing.T) {
	ctx := &mockContext{w: 400, h: 300}
	st := renderer.TextStyle{FontSize: 13}

	n := Box(runtime.Props{Style: runtime.Style{Padding: 8}}, runtime.Text("a"))
	n.Children = append(n.Children, nil) // 顺带守住 nil 跳过
	layoutBox(ctx, n, 10, 20, 100, 50, st, theme.Light)

	if n.X != 10 || n.Y != 20 || n.W != 100 || n.H != 50 {
		t.Fatalf("box 矩形 = (%g,%g,%g,%g)，期望 (10,20,100,50)", n.X, n.Y, n.W, n.H)
	}
	c := n.Children[0]
	if c.X != 18 || c.Y != 28 || c.W != 84 || c.H != 34 {
		t.Fatalf("子节点矩形 = (%g,%g,%g,%g)，期望 (18,28,84,34)", c.X, c.Y, c.W, c.H)
	}
}
