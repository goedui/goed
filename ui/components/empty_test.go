package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestEmptyPaintsTitleHintAndMutedColor(t *testing.T) {
	ctx := &mockContext{w: 400, h: 400}
	n := Empty(runtime.Props{
		Style:  runtime.Style{Width: 360, Height: 320, Color: theme.Light.Foreground},
		Title:  "尚未生成图片",
		Hint:   "在左侧填写提示词",
		Dashed: true,
		Icon:   "image",
	})
	n.X, n.Y, n.W, n.H = 10, 20, 360, 320
	paintEmpty(ctx, n, renderer.TextStyle{FontSize: 13, Color: renderer.ColorFrom(theme.Light.Foreground)}, theme.Light)
	if len(ctx.texts) < 2 {
		t.Fatalf("应绘制标题和提示: %v", ctx.texts)
	}
	if ctx.texts[0] != "尚未生成图片" || ctx.texts[1] != "在左侧填写提示词" {
		t.Fatalf("文案不对: %v", ctx.texts)
	}
	hint := ctx.draws[1]
	want := renderer.RGB(122, 130, 142)
	if hint.style.Color != want {
		t.Fatalf("提示色应可读，got %+v want %+v", hint.style.Color, want)
	}
	if hint.style.Color == renderer.ColorFrom(theme.Light.TitlebarInactive) {
		t.Fatal("提示色不能再用 TitlebarInactive")
	}
}

func TestEmptyLaysOutChildBelowHint(t *testing.T) {
	ctx := &mockContext{w: 400, h: 400}
	btn := Button(runtime.Props{
		Style: runtime.Style{Width: 120, Height: 34},
		ID:    "cta",
		Label: "去生成",
	}, "去生成")
	n := Empty(runtime.Props{
		Style: runtime.Style{Width: 360, Height: 320},
		ID:    "empty",
		Title: "暂无历史记录",
		Hint:  "生成后会出现在这里",
		Icon:  "history",
	}, btn)
	layoutEmpty(ctx, n, 0, 0, 360, 320, renderer.TextStyle{FontSize: 13}, theme.Light)
	if btn.Y <= n.Y+emptyBadge {
		t.Fatalf("CTA 应在图标下方: btn=%+v empty=%+v", btn, n)
	}
	if btn.X < n.X || btn.X+btn.W > n.X+n.W {
		t.Fatalf("CTA 应水平居中落在框内: btn=%+v empty=%+v", btn, n)
	}
	mid := n.X + n.W/2
	btnMid := btn.X + btn.W/2
	if btnMid < mid-8 || btnMid > mid+8 {
		t.Fatalf("CTA 未水平居中: btnMid=%.1f mid=%.1f", btnMid, mid)
	}
}

func TestEmptyIconKindsPaint(t *testing.T) {
	for _, kind := range []string{"image", "history", "spinner"} {
		ctx := &mockContext{w: 200, h: 200}
		n := Empty(runtime.Props{
			Style:  runtime.Style{Width: 180, Height: 180},
			Title:  kind,
			Icon:   kind,
			Dashed: true,
		})
		n.X, n.Y, n.W, n.H = 0, 0, 180, 180
		paintEmpty(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
		if ctx.rects == 0 {
			t.Fatalf("%s 图标没有画出任何几何", kind)
		}
		if len(ctx.texts) == 0 || ctx.texts[0] != kind {
			t.Fatalf("%s 标题未绘制: %v", kind, ctx.texts)
		}
	}
}
