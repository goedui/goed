package components

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("image", renderer.Widget{
		Measure: measureImage,
		Paint:   paintImage,
	})
}

// Image 显示 PNG/JPEG 字节。Attrs: data []byte, name, badge。
func Image(parts ...any) *runtime.VNode {
	return runtime.H("image", parts...)
}

func measureImage(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = style
	_ = th
	w, h := n.Style.Width, n.Style.Height
	if w <= 0 {
		w = maxW
	}
	if h <= 0 {
		if n.Style.Flex > 0 {
			h = maxH
		} else {
			h = 56
		}
	}
	if w <= 0 {
		w = 56
	}
	if h <= 0 {
		h = 56
	}
	return w, h
}

func paintImage(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	bg := renderer.ColorFrom(th.Separator)
	if bg.A == 0 {
		bg = renderer.RGB(240, 240, 240)
	}
	r := n.Style.Radius
	if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, bg)
	}
	ctx.PushClip(n.X, n.Y, n.W, n.H)
	data := renderer.AttrBytes(n, "data")
	if len(data) > 0 {
		iw, ih := imageNaturalSize(data)
		dx, dy, dw, dh := fitContain(n.W, n.H, iw, ih)
		ctx.DrawImage(data, n.X+dx, n.Y+dy, dw, dh)
	}
	paintImageBadge(ctx, n, style)
	ctx.PopClip()
}

// paintImageBadge 在右上角画多图角标（Attrs["badge"]），不参与布局。
func paintImageBadge(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle) {
	text := renderer.AttrString(n, "badge")
	if text == "" || n.W < 40 || n.H < 24 {
		return
	}
	st := style
	st.FontSize = 11
	st.Weight = 600
	st.NoWrap = true
	tw, _ := ctx.MeasureText(text, 0, st)
	bw := tw + 12
	if bw > n.W-8 {
		bw = n.W - 8
	}
	bh := float32(18)
	bx := n.X + n.W - bw - 4
	by := n.Y + 4
	ctx.FillRoundedRect(bx, by, bw, bh, bh/2, renderer.RGBA(18, 24, 30, 190))
	st.Color = renderer.RGB(255, 255, 255)
	st.Align = renderer.AlignCenter
	st.VAlign = renderer.AlignCenter
	ctx.DrawText(text, bx, by, bw, bh, st)
}

func imageNaturalSize(data []byte) (float32, float32) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return 0, 0
	}
	return float32(cfg.Width), float32(cfg.Height)
}

// fitContain 在 box 内按原图比例缩放并居中（letterbox）。
func fitContain(boxW, boxH, imgW, imgH float32) (x, y, w, h float32) {
	if boxW <= 0 || boxH <= 0 {
		return 0, 0, 0, 0
	}
	if imgW <= 0 || imgH <= 0 {
		return 0, 0, boxW, boxH
	}
	scale := boxW / imgW
	if boxH/imgH < scale {
		scale = boxH / imgH
	}
	w = imgW * scale
	h = imgH * scale
	return (boxW - w) / 2, (boxH - h) / 2, w, h
}
