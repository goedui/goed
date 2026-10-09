// lightBackground 与「自定义底色挑字色」的常驻回归。
//
// 白底胶囊标签（次要操作、标签按钮）如果沿用 primary 的白字，白字压白底等于
// 没有文字。paintButton 在「有自定义底色、没自定义字色、不是 selected」时用
// lightBackground 按亮度把字色换成前景色。
//
// 阈值 0.72 两侧都要有人守：调高会让浅灰底继续用白字（看不清），调低会把中灰底
// 也判成浅底（白字变黑字，反而更差）。所以除了纯白 / 纯黑，还钉住阈值上下各一格。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestLightBackgroundThreshold(t *testing.T) {
	cases := []struct {
		name string
		c    runtime.Color
		want bool
	}{
		{"纯白", runtime.RGB(255, 255, 255), true},
		{"纯黑", runtime.RGB(0, 0, 0), false},
		{"主题浅分隔色", theme.Light.Separator, true},
		{"阈值上方一格 184", runtime.RGB(184, 184, 184), true},
		{"阈值下方一格 183", runtime.RGB(183, 183, 183), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lightBackground(renderer.ColorFrom(tc.c)); got != tc.want {
				t.Fatalf("lightBackground(%+v) = %v，期望 %v", tc.c, got, tc.want)
			}
		})
	}
}

// paintedLabelColor 画一次按钮，取标签那一笔的字色。
func paintedLabelColor(t *testing.T, n *runtime.VNode, th theme.Theme) renderer.Color {
	t.Helper()
	// 指针停在按钮矩形之外，避开 hover / pressed 分支。
	renderer.SetPointer(0, 0, false)
	ctx := &mockContext{w: 400, h: 200}
	n.X, n.Y, n.W, n.H = 10, 10, 120, 36
	paintButton(ctx, n, renderer.TextStyle{FontSize: 13}, th)
	if len(ctx.draws) == 0 {
		t.Fatal("按钮没画标签")
	}
	return ctx.draws[0].style.Color
}

func TestButtonPicksForegroundOnLightBackground(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	th := theme.Light

	t.Run("白底换成前景色，不再是 primary 白字", func(t *testing.T) {
		n := Button(runtime.Props{
			Style: runtime.Style{Width: 120, Height: 36, Background: runtime.RGB(255, 255, 255)},
		}, "标签")
		got := paintedLabelColor(t, n, th)

		if got == renderer.RGB(255, 255, 255) {
			t.Fatal("白底仍用白字 —— 文字等于看不见")
		}
		want := renderer.ColorFrom(th.Foreground)
		if want.A == 0 {
			want = renderer.RGB(24, 30, 38)
		}
		if got != want {
			t.Fatalf("白底字色 = %+v，期望 %+v", got, want)
		}
	})

	t.Run("深底保持白字", func(t *testing.T) {
		n := Button(runtime.Props{
			Style: runtime.Style{Width: 120, Height: 36, Background: runtime.RGB(20, 30, 40)},
		}, "标签")
		if got := paintedLabelColor(t, n, th); got != renderer.RGB(255, 255, 255) {
			t.Fatalf("深底字色 = %+v，期望白色", got)
		}
	})

	t.Run("写了 Color 就按 Color，不按亮度猜", func(t *testing.T) {
		n := Button(runtime.Props{
			Style: runtime.Style{
				Width: 120, Height: 36,
				Background: runtime.RGB(255, 255, 255),
				Color:      runtime.RGB(9, 9, 9),
			},
		}, "标签")
		if got := paintedLabelColor(t, n, th); got != renderer.RGB(9, 9, 9) {
			t.Fatalf("字色 = %+v，期望 Style.Color", got)
		}
	})

	t.Run("selected 的白字不被覆盖", func(t *testing.T) {
		n := Button(runtime.Props{
			Style:    runtime.Style{Width: 120, Height: 36, Background: runtime.RGB(255, 255, 255)},
			Selected: true,
		}, "标签")
		if got := paintedLabelColor(t, n, th); got != renderer.RGB(255, 255, 255) {
			t.Fatalf("selected 字色 = %+v，期望保持 selected 的白字", got)
		}
	})
}

func TestBorderButtonHoverFollowsBackground(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	dark := runtime.RGB(32, 36, 44)
	n := Button(runtime.Props{
		Style: runtime.Style{
			Width: 112, Height: 32, Radius: 4,
			Background: dark, Border: runtime.RGB(52, 58, 68),
			Color: runtime.RGB(255, 255, 255),
		},
	}, "打开文件")
	n.X, n.Y, n.W, n.H = 10, 10, 112, 32
	renderer.SetPointer(20, 20, false)

	ctx := &mockContext{w: 400, h: 200}
	paintButton(ctx, n, renderer.TextStyle{FontSize: 12}, theme.Dark)
	if len(ctx.rounds) < 2 {
		t.Fatal("描边按钮没有画出内层底色")
	}
	got := ctx.rounds[len(ctx.rounds)-1].c
	if lightBackground(got) {
		t.Fatalf("暗色描边按钮 hover 变成浅色 %+v", got)
	}
	base := renderer.ColorFrom(dark)
	if got.R <= base.R || got.G <= base.G || got.B <= base.B {
		t.Fatalf("暗色 hover 没有提亮：got %+v base %+v", got, base)
	}
}
