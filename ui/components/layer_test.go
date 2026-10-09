// layer 度量与布局的常驻回归。
//
// layer 是覆盖整窗的浮层（模态、遮罩、状态栏）。它和 titlebar 一样**不看自己的
// Style.Width/Height**，走 maxW/maxH → ctx.Width()/Height() 两级兜底：父级常用
// maxH=0 重新度量，layer 必须自己补上「整窗高度」，否则量出 0 高、浮层整块消失。
//
// layoutLayer 还有一条和度量**故意不一致**的规则：高度改成「y 以下的画布」，
// 因为标题栏已经占了 32 DIP，沿用整窗高度会把底栏顶出窗口。这两条要一起守：
// 把布局改回沿用 h 会顶出底栏，把度量改成也减 32 会让没进标题栏的场景矮一截。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestMeasureLayerFallbackLadder(t *testing.T) {
	ctx := &mockContext{w: 1000, h: 800}
	th := theme.Light
	st := renderer.TextStyle{FontSize: 13}

	cases := []struct {
		name         string
		maxW, maxH   float32
		useCtx       bool
		wantW, wantH float32
		why          string
	}{
		{"可用空间优先", 600, 400, true, 600, 400, "maxW/maxH 给了就用它，不看 ctx"},
		{"可用空间为 0 → 退到窗口", 0, 0, true, 1000, 800, "父级常用 maxH=0 重量"},
		{"只缺一轴", 300, 0, true, 300, 800, "缺哪轴补哪轴"},
		{"ctx 为 nil 时保持 0", 0, 0, false, 0, 0, "没有窗口可问，不能 panic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var c renderer.Context
			if tc.useCtx {
				c = ctx
			}
			n := runtime.H("layer")
			w, h := measureLayer(c, n, tc.maxW, tc.maxH, st, th)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("measureLayer = %gx%g，期望 %gx%g（%s）", w, h, tc.wantW, tc.wantH, tc.why)
			}
		})
	}
}

// TestLayoutLayerShrinksHeightToCanvasBottom 守住「度量说整窗高、布局说 y 以下」
// 这条故意的不一致。标题栏 32 DIP 之下，父级把量出来的整窗高 800 递进来，
// 布局必须收成 768。
func TestLayoutLayerShrinksHeightToCanvasBottom(t *testing.T) {
	ctx := &mockContext{w: 1000, h: 800}
	st := renderer.TextStyle{FontSize: 13}

	n := runtime.H("layer")
	layoutLayer(ctx, n, 0, 32, 1000, 800, st, theme.Light)
	if n.Y != 32 || n.W != 1000 || n.H != 768 {
		t.Fatalf("layer 矩形 = (%g,%g,%g,%g)，期望 (0,32,1000,768)", n.X, n.Y, n.W, n.H)
	}

	// y 已经是画布底（bottom <= 0）时不动高度，避免出现负高。
	n2 := runtime.H("layer")
	layoutLayer(ctx, n2, 0, 800, 1000, 800, st, theme.Light)
	if n2.H != 800 {
		t.Fatalf("y 触底时高度 = %g，期望保持 800（不能取负）", n2.H)
	}

	// 父级给 0 宽时补成窗口宽 —— 和高度那条同一个病：不补就量出 0 宽，浮层横向消失。
	n4 := runtime.H("layer")
	n4.Children = append(n4.Children, nil) // nil 子节点要跳过，不能走进 LayoutNode
	layoutLayer(ctx, n4, 0, 32, 0, 800, st, theme.Light)
	if n4.W != 1000 {
		t.Fatalf("宽为 0 时 layer 宽 = %g，期望补成窗口宽 1000", n4.W)
	}

	// ctx 为 nil：没窗口可问，原样写入父级给的矩形，且不能 panic。
	n3 := runtime.H("layer")
	layoutLayer(nil, n3, 0, 32, 1000, 800, st, theme.Light)
	if n3.W != 1000 || n3.H != 800 {
		t.Fatalf("ctx 为 nil 时 layer 矩形被改动：%gx%g", n3.W, n3.H)
	}
}
