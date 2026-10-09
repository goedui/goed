// HStack 悬停高亮的常驻回归。
//
// paintHStack 里那条 if 是四个条件的合取，缺一条就不该压暗：
//
//	n.OnClick != nil                        —— 只有「整行可点」才需要给反馈
//	renderer.Enabled(n)                     —— 禁用行不给反馈
//	renderer.PointerIn(n.X, n.Y, n.W, n.H)  —— 指针不在行内不压暗
//	renderer.ColorSet(n.Style.Background)   —— 没有底色就没东西可压
//
// 四个条件各自都要有人守：漏掉 Enabled 会让禁用行亮起来；漏掉 ColorSet 会拿
// 零值（全黑）去乘系数，画出一块不该存在的黑条。
//
// paintHStackHover 自身还有两条几何分支（圆角 / 直角）和一条 Border 分支。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// hoverTint 是 paintHStackHover 压暗系数的镜像，只用来算期望值。
// 主判断是「和原底色不同」——那一条不依赖这份镜像。
func hoverTint(c renderer.Color) renderer.Color {
	c.R *= 0.97
	c.G *= 0.975
	c.B *= 0.98
	return c
}

// hoverRow 造一个已经摆好位置的行节点（paint 只读 X/Y/W/H）。
func hoverRow(p runtime.Props) *runtime.VNode {
	n := HStack(p)
	n.X, n.Y, n.W, n.H = 0, 0, 200, 40
	return n
}

func TestHStackHoverDimsTintedClickableRow(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	renderer.SetPointer(100, 20, false) // 落在行内

	bg := runtime.RGB(200, 100, 50)
	ctx := &mockContext{w: 400, h: 200}
	n := hoverRow(runtime.Props{
		Style:   runtime.Style{Width: 200, Height: 40, Background: bg},
		OnClick: func() {},
	})
	paintHStack(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(ctx.fills) != 1 {
		t.Fatalf("悬停应恰好填充一次，得到 %d 次：%+v", len(ctx.fills), ctx.fills)
	}
	got := ctx.fills[0].c
	if got == renderer.ColorFrom(bg) {
		t.Fatalf("悬停底色和原底色一样 %+v —— 压暗这一步没生效", got)
	}
	if want := hoverTint(renderer.ColorFrom(bg)); got != want {
		t.Fatalf("悬停底色 = %+v，期望 %+v", got, want)
	}
}

func TestHStackHoverTriggerNeedsAllFourConditions(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	bg := runtime.RGB(200, 100, 50)
	plain := renderer.ColorFrom(bg)

	cases := []struct {
		name   string
		click  func()
		attrs  runtime.Attrs
		ptrX   float32
		reason string
	}{
		{"指针在行外", func() {}, nil, 260, "指针在矩形右缘之外"},
		{"没有 OnClick", nil, nil, 100, "不可点的行不该给反馈"},
		{"禁用", func() {}, runtime.Attrs{"disabled": true}, 100, "禁用行不该亮"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			renderer.SetPointer(tc.ptrX, 20, false)
			ctx := &mockContext{w: 400, h: 200}
			n := hoverRow(runtime.Props{
				Style:   runtime.Style{Width: 200, Height: 40, Background: bg},
				OnClick: tc.click,
			})
			if tc.attrs != nil {
				n.Attrs = tc.attrs
			}
			paintHStack(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

			if len(ctx.fills) != 1 {
				t.Fatalf("应恰好填充一次，得到 %d 次：%+v", len(ctx.fills), ctx.fills)
			}
			if ctx.fills[0].c != plain {
				t.Fatalf("底色被压暗成 %+v（%s），期望保持原色 %+v", ctx.fills[0].c, tc.reason, plain)
			}
		})
	}
}

// 没有底色时一个像素都不该填：压暗零值会画出全黑条。
func TestHStackHoverNeedsTint(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	renderer.SetPointer(100, 20, false)

	ctx := &mockContext{w: 400, h: 200}
	n := hoverRow(runtime.Props{
		Style:   runtime.Style{Width: 200, Height: 40},
		OnClick: func() {},
	})
	paintHStack(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(ctx.fills)+len(ctx.rounds) != 0 {
		t.Fatalf("没有底色时不该填任何东西：fills=%+v rounds=%+v", ctx.fills, ctx.rounds)
	}
}

func TestHStackHoverGeometryAndBorder(t *testing.T) {
	defer renderer.SetPointer(0, 0, false)
	renderer.SetPointer(100, 20, false)

	bg := runtime.RGB(200, 100, 50)
	border := runtime.RGB(10, 20, 30)
	st := renderer.TextStyle{FontSize: 13}

	t.Run("圆角行走 FillRoundedRect", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := hoverRow(runtime.Props{
			Style:   runtime.Style{Width: 200, Height: 40, Radius: 8, Background: bg},
			OnClick: func() {},
		})
		paintHStack(ctx, n, st, theme.Light)

		if len(ctx.rounds) != 1 || ctx.rounds[0].r != 8 {
			t.Fatalf("圆角行应走 FillRoundedRect(r=8)：rounds=%+v", ctx.rounds)
		}
		if len(ctx.fills) != 0 {
			t.Fatalf("圆角行不该再走 FillRect：fills=%+v", ctx.fills)
		}
	})

	t.Run("直角行走 FillRect", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := hoverRow(runtime.Props{
			Style:   runtime.Style{Width: 200, Height: 40, Background: bg},
			OnClick: func() {},
		})
		paintHStack(ctx, n, st, theme.Light)

		if len(ctx.fills) != 1 {
			t.Fatalf("直角行应走 FillRect：fills=%+v rounds=%+v", ctx.fills, ctx.rounds)
		}
		if len(ctx.rounds) != 0 {
			t.Fatalf("直角行不该走 FillRoundedRect：rounds=%+v", ctx.rounds)
		}
	})

	t.Run("有 Border 时补一圈描边", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := hoverRow(runtime.Props{
			Style:   runtime.Style{Width: 200, Height: 40, Background: bg, Border: border},
			OnClick: func() {},
		})
		paintHStack(ctx, n, st, theme.Light)

		if len(ctx.strokes) != 1 {
			t.Fatalf("应描一圈边，得到 %d 圈：%+v", len(ctx.strokes), ctx.strokes)
		}
		s := ctx.strokes[0]
		if s.c != renderer.ColorFrom(border) || s.width != 1 {
			t.Fatalf("描边 = %+v，期望 %+v 宽 1", s, renderer.ColorFrom(border))
		}
	})

	t.Run("没有 Border 时不描边", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := hoverRow(runtime.Props{
			Style:   runtime.Style{Width: 200, Height: 40, Background: bg},
			OnClick: func() {},
		})
		paintHStack(ctx, n, st, theme.Light)

		if len(ctx.strokes) != 0 {
			t.Fatalf("没写 Border 却描了边：%+v", ctx.strokes)
		}
	})
}
