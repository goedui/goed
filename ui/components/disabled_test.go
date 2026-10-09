package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// disabledPalette 是一个「故意和内置主题不同」的调色板。
//
// 关键在于它**不等于**任何硬编码值：以前 button / select / textedit 三处各抄了
// 一份 #E8EAEE / #5F6673。如果哪一处又改回字面量，它就画不出 (1,2,3) / (4,5,6)，
// 断言立刻红。拿内置主题来断言是抓不到这种回退的——值恰好一样。
func disabledPalette() theme.Theme {
	th := theme.Light
	th.Disabled = runtime.RGB(1, 2, 3)
	th.DisabledForeground = runtime.RGB(4, 5, 6)
	return th
}

// fillColors 收集 mock 记录到的全部填充色（FillRect + FillRoundedRect）。
func fillColors(ctx *mockContext) []renderer.Color {
	out := make([]renderer.Color, 0, len(ctx.fills)+len(ctx.rounds))
	for _, f := range ctx.fills {
		out = append(out, f.c)
	}
	for _, r := range ctx.rounds {
		out = append(out, r.c)
	}
	return out
}

func hasColor(cs []renderer.Color, want renderer.Color) bool {
	for _, c := range cs {
		if c == want {
			return true
		}
	}
	return false
}

// 三个控件都必须从主题取禁用色，一处都不能漏。
//
// 之前它们各自硬编码同一个灰：主题切到暗色时禁用控件仍是近白色块
// （#E8EAEE 叠在 #1E1E1E 上是 13.84:1），而三处谁都以为别处会跟主题走。
func TestDisabledControlsUseThemeColors(t *testing.T) {
	th := disabledPalette()
	bgWant := renderer.ColorFrom(th.Disabled)
	fgWant := renderer.ColorFrom(th.DisabledForeground)

	t.Run("button", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := Button(runtime.Props{
			ID: "dis-btn", Variant: "primary", Disabled: true,
			Style: runtime.Style{Border: runtime.RGB(200, 200, 200), Color: runtime.RGB(47, 107, 242)},
		}, "禁用")
		n.X, n.Y, n.W, n.H = 10, 10, 120, 36
		paintButton(ctx, n, renderer.TextStyle{FontSize: 13}, th)

		if !hasColor(fillColors(ctx), bgWant) {
			t.Fatalf("按钮禁用底色没走主题：fills=%v", fillColors(ctx))
		}
		if len(ctx.draws) == 0 {
			t.Fatal("按钮没画标签")
		}
		if ctx.draws[0].style.Color != fgWant {
			t.Fatalf("按钮禁用文字色没走主题：got %+v want %+v", ctx.draws[0].style.Color, fgWant)
		}
	})

	t.Run("select", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		n := Select(runtime.Props{ID: "dis-sel", Items: []string{"A", "B"}, Value: "A", Disabled: true})
		n.X, n.Y, n.W, n.H = 10, 10, 160, 36
		paintSelect(ctx, n, renderer.TextStyle{FontSize: 13}, th)

		if !hasColor(fillColors(ctx), bgWant) {
			t.Fatalf("下拉禁用底色没走主题：fills=%v", fillColors(ctx))
		}
		found := false
		for _, d := range ctx.draws {
			if d.style.Color == fgWant {
				found = true
			}
		}
		if !found {
			t.Fatalf("下拉禁用文字色没走主题：draws=%v", ctx.draws)
		}
	})

	t.Run("input", func(t *testing.T) {
		ctx := &mockContext{w: 400, h: 200}
		ResetInputState("dis-in")
		n := Input(runtime.Props{ID: "dis-in", Value: "hello", Disabled: true})
		n.X, n.Y, n.W, n.H = 10, 10, 200, 38
		paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, th)

		if !hasColor(fillColors(ctx), bgWant) {
			t.Fatalf("输入框禁用底色没走主题：fills=%v", fillColors(ctx))
		}
		// 文字色**刻意不换**：框里是用户自己的值，禁用期间仍要读得清。
		if hasColor(fillColors(ctx), fgWant) {
			t.Log("提示：输入框把禁用文字色当填充用了")
		}
	})
}

// 禁用输入框不该留焦点环：灰底 + 蓝框读起来像「聚焦着但不能用」。
func TestDisabledInputDropsFocusRing(t *testing.T) {
	th := theme.Light
	ring := renderer.ColorFrom(th.Button)

	// 先证明「可用 + 聚焦」确实会画焦点环。少了这一步，下面的断言可能是空转的
	// ——描边压根不画的话，禁用时当然也查不到蓝色。
	t.Run("可用时画", func(t *testing.T) {
		runtime.SetFocus("focus-ok")
		t.Cleanup(func() { runtime.SetFocus("") })
		ResetInputState("focus-ok")

		ctx := &mockContext{w: 400, h: 200}
		n := Input(runtime.Props{ID: "focus-ok", Value: "hello"})
		n.X, n.Y, n.W, n.H = 10, 20, 300, 40
		paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, th)

		if hasColor(fillColors(ctx), ring) {
			return
		}
		t.Fatalf("可用输入框聚焦时没画焦点环：fills=%v strokes=%v", fillColors(ctx), ctx.strokes)
	})

	t.Run("禁用时不画", func(t *testing.T) {
		runtime.SetFocus("focus-dis")
		t.Cleanup(func() { runtime.SetFocus("") })
		ResetInputState("focus-dis")

		ctx := &mockContext{w: 400, h: 200}
		n := Input(runtime.Props{ID: "focus-dis", Value: "hello", Disabled: true})
		n.X, n.Y, n.W, n.H = 10, 20, 300, 40
		paintInput(ctx, n, renderer.TextStyle{FontSize: 13}, th)

		if hasColor(fillColors(ctx), ring) {
			t.Fatalf("禁用输入框仍画了焦点环 %+v", ring)
		}
	})
}

// 禁用按钮不响应指针：悬停 / 按下都不该改底色。
func TestDisabledButtonIgnoresPointer(t *testing.T) {
	th := disabledPalette()
	defer renderer.SetPointer(0, 0, false)

	ctx := &mockContext{w: 400, h: 200}
	n := Button(runtime.Props{ID: "dis-hover", Variant: "primary", Disabled: true}, "禁用")
	n.X, n.Y, n.W, n.H = 10, 10, 120, 36

	// 把指针挪到按钮正中并按住。
	renderer.SetPointer(70, 28, true)
	paintButton(ctx, n, renderer.TextStyle{FontSize: 13}, th)

	for _, c := range []renderer.Color{
		renderer.ColorFrom(th.ButtonHover),
		renderer.ColorFrom(th.ButtonPressed),
	} {
		if hasColor(fillColors(ctx), c) {
			t.Fatalf("禁用按钮仍画了悬停/按下底色 %+v：fills=%v", c, fillColors(ctx))
		}
	}
	if !hasColor(fillColors(ctx), renderer.ColorFrom(th.Disabled)) {
		t.Fatalf("禁用按钮底色应保持禁用色：fills=%v", fillColors(ctx))
	}
}
