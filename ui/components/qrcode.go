package components

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("qrcode", renderer.Widget{
		Measure: measureQRCode,
		Paint:   paintQRCode,
	})
}

// QRCode 把一段文本（通常是 URL）画成二维码。扫码即可打开，常用于分享链接。
//
//	ui.QRCode(ui.Props{Style: ui.Style{Width: 220, Height: 220}}, "https://d.cn")
//	// 纠错等级 / 中央标签走 attrs（Props 没有对应字段）：
//	n := ui.QRCode(ui.Props{Badge: "d.cn", Style: ui.Style{Width: 220, Height: 220}}, "https://d.cn")
//	n.Attrs["level"] = "H"
//	n.Attrs["quiet"] = 2
//
// 属性（Attrs）：
//
//	text       要编码的内容；也可以把内容作为 children 传进来
//	level      纠错等级 "L"/"M"/"Q"/"H"（默认 M）
//	quiet      静区宽度，单位模块（默认 4，标准要求 ≥4）
//	badge      中央圆角标签文案；字号随码面缩放，边长 <96 DIP 时不画
//
// 样式相关：Style.Width / Height 决定二维码正方形边长（取较小值），
// Style.Background 是浅色底，Style.Color 是深色模块色。
//
// 二维码总是正方形，居中画在节点的盒子里；不会拉伸失真。编码失败
// （内容过长等）时只画底色，不 panic，方便界面继续跑。
func QRCode(parts ...any) *runtime.VNode {
	n := runtime.H("qrcode", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	if renderer.AttrString(n, "text") == "" {
		for _, c := range n.Children {
			if c != nil && c.Kind == runtime.KindText && c.Text != "" {
				n.Attrs["text"] = c.Text
				break
			}
		}
	}
	n.Children = nil // 内容走 Attrs，避免子节点参与布局
	return n
}

// QRCodeWith 是 QRCode 的显式参数版本：内容与纠错等级直接给，其余走 Props
// （Props.Badge 画中央标签；静区宽度仍可用 Attrs["quiet"] 覆盖）。
func QRCodeWith(text string, level Level, p runtime.Props) *runtime.VNode {
	n := QRCode(p)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["text"] = text
	n.Attrs["level"] = level.String()
	return n
}

// QRCodeInfo 返回内容编码后的版本与模块边长，供界面显示「版本 / 尺寸」。
func QRCodeInfo(text string, level Level) (version, size int, err error) {
	m, err := qrLookup(text, level)
	if err != nil {
		return 0, 0, err
	}
	return m.Version, m.Size, nil
}

// QRCodePNG 把二维码编码成 PNG（默认 4 个模块的静区、每模块 6 像素），
// 供保存 / 分享等场景复用同一套编码结果。
func QRCodePNG(text string, level Level, scale, quiet int) ([]byte, error) {
	m, err := qrLookup(text, level)
	if err != nil {
		return nil, err
	}
	return encodeQRPNG(m, scale, quiet)
}

func encodeQRPNG(m *QRMatrix, scale, quiet int) ([]byte, error) {
	if scale <= 0 {
		scale = 6
	}
	if quiet < 0 {
		quiet = 0
	}
	side := (m.Size + 2*quiet) * scale
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.NRGBA{R: 17, G: 24, B: 39, A: 255}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			img.SetNRGBA(x, y, white)
		}
	}
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; x++ {
			if !m.Dark(x, y) {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetNRGBA((x+quiet)*scale+dx, (y+quiet)*scale+dy, black)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func qrText(n *runtime.VNode) string {
	if text := renderer.AttrString(n, "text"); text != "" {
		return text
	}
	return renderer.AttrString(n, "label")
}

func qrLevel(n *runtime.VNode) Level {
	// 应用层可能塞 "H" 这样的字符串，也可能塞 Level 值，两种都认。
	switch v := renderer.Attr(n, "level").(type) {
	case Level:
		if v.valid() {
			return v
		}
	case string:
		return ParseLevel(v)
	}
	return LevelM
}

func qrQuiet(n *runtime.VNode) int {
	q := 4
	switch v := renderer.Attr(n, "quiet").(type) {
	case int:
		q = v
	case int32:
		q = int(v)
	case float32:
		q = int(v)
	case float64:
		q = int(v)
	}
	if q < 0 {
		q = 0
	}
	if q > 16 {
		q = 16
	}
	return q
}

func qrMatrixFor(n *runtime.VNode) *QRMatrix {
	m, err := qrLookup(qrText(n), qrLevel(n))
	if err != nil {
		return nil
	}
	return m
}

// qrBoxSide 取节点盒子的短边作为二维码边长，保证正方形居中。
func qrBoxSide(n *runtime.VNode) float32 {
	side := n.W
	if n.H < side {
		side = n.H
	}
	return side
}

func measureQRCode(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = maxH
	w, h := n.Style.Width, n.Style.Height
	// 只给一边时以那一边为准；都没给就跟可用宽度，最后兜底 160。
	if w <= 0 {
		w = h
	}
	if w <= 0 {
		w = maxW
	}
	if h <= 0 {
		h = w
	}
	if w <= 0 {
		w, h = 160, 160
	}
	if w > h {
		w = h
	} else if h > w {
		h = w
	}
	_ = style
	_ = th
	return w, h
}

func paintQRCode(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	bg := renderer.RGB(255, 255, 255)
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	} else if th.Dark {
		bg = renderer.RGB(245, 247, 250)
	}
	fg := renderer.RGB(17, 24, 39)
	if renderer.ColorSet(n.Style.Color) {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	r := n.Style.Radius
	if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, bg)
	}
	m := qrMatrixFor(n)
	if m == nil {
		return
	}
	quiet := qrQuiet(n)
	side := qrBoxSide(n)
	if side <= 0 {
		return
	}
	ctx.PushClip(n.X, n.Y, n.W, n.H)
	// 模块很多时逐个 FillRect 是 fine 的（21x21 到 177x177，最多约 3 万个小块）；
	// 这里按行合并连续深色模块，常见内容能省掉一半以上调用。
	modules := float32(m.Size + 2*quiet)
	cell := side / modules
	// 静区在盒子内部：第一格深色模块从 4 格（默认）之后开始。
	ox := n.X + (n.W-side)/2 + float32(quiet)*cell
	oy := n.Y + (n.H-side)/2 + float32(quiet)*cell
	for y := 0; y < m.Size; y++ {
		runStart := -1
		for x := 0; x <= m.Size; x++ {
			dark := x < m.Size && m.Dark(x, y)
			if dark && runStart < 0 {
				runStart = x
				continue
			}
			if !dark && runStart >= 0 {
				x0 := ox + float32(runStart)*cell
				x1 := ox + float32(x)*cell
				ctx.FillRect(x0, oy+float32(y)*cell, x1-x0, cell, fg)
				runStart = -1
			}
		}
	}
	ctx.PopClip()
	qrPaintBadge(ctx, n, m, style)
}

// qrPaintBadge 在二维码中央画一枚圆角标签（Attrs["badge"]），常见于品牌二维码。
// 标签会盖住部分模块，靠纠错等级兜底，所以只在等级 Q/H 时建议使用。
func qrPaintBadge(ctx renderer.Context, n *runtime.VNode, m *QRMatrix, style renderer.TextStyle) {
	text := renderer.AttrString(n, "badge")
	if text == "" || m == nil {
		return
	}
	side := qrBoxSide(n)
	if side < 96 {
		return
	}
	st := style
	st.FontSize = side / 14
	if st.FontSize < 10 {
		st.FontSize = 10
	}
	if st.FontSize > 20 {
		st.FontSize = 20
	}
	st.Weight = 700
	st.NoWrap = true
	st.Align = renderer.AlignCenter
	st.VAlign = renderer.AlignCenter
	tw, th2 := ctx.MeasureText(text, side, st)
	bw := tw + st.FontSize*1.4
	if bw > side*0.7 {
		bw = side * 0.7
	}
	bh := th2 + st.FontSize*0.8
	if bh > side*0.28 {
		bh = side * 0.28
	}
	bx := n.X + (n.W-bw)/2
	by := n.Y + (n.H-bh)/2
	radius := bh * 0.24
	pad := bh * 0.1
	ctx.FillRoundedRect(bx-pad, by-pad, bw+2*pad, bh+2*pad, radius+pad, renderer.RGB(255, 255, 255))
	ctx.FillRoundedRect(bx, by, bw, bh, radius, renderer.RGB(17, 24, 39))
	st.Color = renderer.RGB(255, 255, 255)
	ctx.DrawText(text, bx, by, bw, bh, st)
}
