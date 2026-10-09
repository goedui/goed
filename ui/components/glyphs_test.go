package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

var requiredGlyphs = []string{
	"edit", "link", "shield", "box", "user",
	"save", "trash", "eye", "refresh", "sparkle", "upload", "image",
}

func TestRequiredGlyphsRegistered(t *testing.T) {
	for _, name := range requiredGlyphs {
		if !HasGlyph(name) {
			t.Errorf("图标 %q 未登记", name)
		}
	}
}

func TestGlyphPaintersProduceDrawing(t *testing.T) {
	for _, name := range requiredGlyphs {
		ctx := &mockContext{w: 100, h: 100}
		if !PaintGlyph(ctx, name, 0, 0, 24, renderer.RGB(0, 0, 0)) {
			t.Errorf("PaintGlyph(%q) 返回 false", name)
			continue
		}
		if ctx.rects == 0 && ctx.lines == 0 {
			t.Errorf("图标 %q 没有产生任何笔画", name)
		}
	}
}

func TestPaintGlyphUnknownNameReturnsFalse(t *testing.T) {
	ctx := &mockContext{w: 100, h: 100}
	if PaintGlyph(ctx, "no-such-glyph", 0, 0, 24, renderer.RGB(0, 0, 0)) {
		t.Fatal("未登记的名字应返回 false 供调用方降级")
	}
}

func TestRegisterGlyphOverrides(t *testing.T) {
	called := false
	RegisterGlyph("edit", func(ctx renderer.Context, x, y, size float32, fg renderer.Color) {
		called = true
	})
	t.Cleanup(func() { RegisterGlyph("edit", paintGlyphEdit) })
	ctx := &mockContext{w: 100, h: 100}
	PaintGlyph(ctx, "edit", 0, 0, 24, renderer.RGB(0, 0, 0))
	if !called {
		t.Fatal("RegisterGlyph 同名覆盖未生效")
	}
}

func TestButtonMeasureIncludesIconWidth(t *testing.T) {
	ctx := &mockContext{w: 400, h: 100}
	style := renderer.TextStyle{FontSize: 14}
	plain := runtime.H("button", runtime.Props{Label: "生成"})
	withIcon := runtime.H("button", runtime.Props{Label: "生成", Icon: "sparkle"})

	w1, h1 := measureButton(ctx, plain, 0, 0, style, theme.Light)
	w2, h2 := measureButton(ctx, withIcon, 0, 0, style, theme.Light)
	if w2 <= w1 {
		t.Fatalf("带图标按钮应更宽：无图标 %.1f，带图标 %.1f", w1, w2)
	}
	if h2 != h1 {
		t.Fatalf("图标不应改变行高：%.1f vs %.1f", h1, h2)
	}
	// 图标宽（下限 14）+ 间距（7）= 21 的占位预算
	if d := w2 - w1; d < 21 || d > 22 {
		t.Fatalf("图标占位应为 21px 左右，实际 %.1f", d)
	}
}

func TestButtonPaintDrawsIconAndLabel(t *testing.T) {
	ctx := &mockContext{w: 400, h: 100}
	style := renderer.TextStyle{FontSize: 14}
	n := runtime.H("button", runtime.Props{Label: "保存", Icon: "save"})
	n.X, n.Y, n.W, n.H = 0, 0, 90, 34
	paintButton(ctx, n, style, theme.Light)
	if len(ctx.texts) != 1 || ctx.texts[0] != "保存" {
		t.Fatalf("文字应画一次，实际 %v", ctx.texts)
	}
	if ctx.lines == 0 && ctx.rects == 0 {
		t.Fatal("图标未产生绘制")
	}
	tw, _ := ctx.MeasureText("保存", 90, style)
	group := buttonIconSize(style) + buttonIconGap + tw
	want := (n.W-group)/2 + buttonIconSize(style) + buttonIconGap
	if got := ctx.draws[0].x; got < want-0.5 || got > want+0.5 {
		t.Fatalf("图标+文字应成组居中，文字起点 %.1f，期望 %.1f", got, want)
	}
}

func TestButtonWithoutIconUnchanged(t *testing.T) {
	ctx := &mockContext{w: 400, h: 100}
	style := renderer.TextStyle{FontSize: 14}
	n := runtime.H("button", runtime.Props{Label: "确定"})
	n.X, n.Y, n.W, n.H = 0, 0, 90, 34
	before := ctx.rects
	paintButton(ctx, n, style, theme.Light)
	if len(ctx.texts) != 1 {
		t.Fatalf("无图标按钮文字应画一次，实际 %v", ctx.texts)
	}
	// 无图标时除按钮本体外不应多出图标笔画。
	if extra := ctx.rects - before; extra > 1 {
		t.Fatalf("无图标按钮不应画额外形状，本体之外多了 %d 笔", extra-1)
	}
}

func TestButtonGradientOverlayUsesPainter(t *testing.T) {
	ctx := &gradientMock{mockContext: mockContext{w: 400, h: 100}}
	style := renderer.TextStyle{FontSize: 14}
	n := runtime.H("button", runtime.Props{
		Label:    "生成图片",
		Variant:  "primary",
		Gradient: &runtime.GradientFill{From: runtime.RGB(79, 134, 247), To: runtime.RGB(46, 144, 250)},
	})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 40
	paintButton(ctx, n, style, theme.Light)
	if !ctx.gotGradient {
		t.Fatal("带 Gradient 的按钮应调用渐变填充")
	}
}

// gradientMock 在 mockContext 之上补 GradientPainter 能力。
type gradientMock struct {
	mockContext
	gotGradient bool
}

func (g *gradientMock) FillGradientRoundedRect(x, y, w, h, r float32, from, to renderer.Color, vertical bool) {
	g.gotGradient = true
}

func TestEmptyAccentTintsBadgeAndIcon(t *testing.T) {
	plain := &mockContext{w: 400, h: 400}
	accented := &mockContext{w: 400, h: 400}
	mk := func(p runtime.Props) *runtime.VNode {
		p.Style = runtime.Style{Width: 360, Height: 320}
		p.Title = "尚未生成图片"
		p.Hint = "提示"
		p.Dashed = true
		return Empty(p)
	}
	n1 := mk(runtime.Props{})
	n1.X, n1.Y, n1.W, n1.H = 0, 0, 360, 320
	paintEmpty(plain, n1, renderer.TextStyle{FontSize: 13}, theme.Light)

	n2 := mk(runtime.Props{Accent: true})
	n2.X, n2.Y, n2.W, n2.H = 0, 0, 360, 320
	paintEmpty(accented, n2, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(accented.rounds) <= len(plain.rounds) {
		t.Fatalf("Accent 徽章应多画星点装饰：普通 %d 圆角 / 强调 %d 圆角",
			len(plain.rounds), len(accented.rounds))
	}
}
