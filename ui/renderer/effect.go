package renderer

// 本文件是「渐变 / 阴影」的可选能力层，与 TransformPainter 同一个降级哲学：
// Context 是公开接口，给它加方法会打断所有外部实现和测试假后端，所以做成
// 可选接口由后端按条件实现，调用方通过包装函数探测；探测不到就退回纯色画法
// —— 能看见，只是不渐变 / 不投影。

// GradientPainter 由支持「线性渐变填充圆角矩形」的后端实现：Windows 上是
// ID2D1RenderTarget::CreateLinearGradientBrush + CreateGradientStopCollection，
// Web 上是 canvas createLinearGradient。
type GradientPainter interface {
	// FillGradientRoundedRect 画 from→to 的线性渐变圆角矩形。vertical=true 从上到下，
	// false 从左到右；r<=0 按直角矩形。
	FillGradientRoundedRect(x, y, w, h, r float32, from, to Color, vertical bool)
}

// ShadowPainter 由支持「带投影的圆角矩形」的后端实现：Windows 上是
// CreateLayer + PushLayer（DrawImageRounded 已在用同一张槽位表）。
type ShadowPainter interface {
	// FillShadowRoundedRect 画投影并填充圆角矩形本体，一次调用完成（阴影必须先于
	// 本体画，后端才能把它合成在同一图层里）。blur 是模糊半径，offsetY 是向下的偏移。
	FillShadowRoundedRect(x, y, w, h, r, blur, offsetY float32, c, shadow Color)
}

// FillGradientRoundedRect 画线性渐变圆角矩形，返回后端是否真画了渐变。
// 不支持的后端回退为 from 纯色填充。
func FillGradientRoundedRect(ctx Context, x, y, w, h, r float32, from, to Color, vertical bool) bool {
	if ctx == nil || w <= 0 || h <= 0 {
		return false
	}
	if g, ok := ctx.(GradientPainter); ok {
		g.FillGradientRoundedRect(x, y, w, h, r, from, to, vertical)
		return true
	}
	FillRounded(ctx, x, y, w, h, r, from)
	return false
}

// FillShadowRoundedRect 画投影 + 圆角矩形本体。后端支持 ShadowPainter 时走原生
// 模糊阴影；否则用两层低透明度的外扩偏移圆角矩形近似 —— 真实阴影本身就是模糊的，
// 低 alpha 大圆角的叠层在 1~6px 投影范围内肉眼难分。本体始终由本函数画完，
// 调用方不必再补一次填充。
func FillShadowRoundedRect(ctx Context, x, y, w, h, r, blur, offsetY float32, c, shadow Color) {
	if ctx == nil || w <= 0 || h <= 0 {
		return
	}
	if s, ok := ctx.(ShadowPainter); ok {
		s.FillShadowRoundedRect(x, y, w, h, r, blur, offsetY, c, shadow)
		return
	}
	if shadow.A > 0 && blur > 0 {
		// 四层由外到内收紧。没有高斯模糊时，层数决定投影是否看得见。
		for i, scale := range []float32{1, 0.62, 0.34, 0.14} {
			pad := blur * scale
			alpha := shadow.A * (0.10 + float32(i)*0.05)
			dy := offsetY * (0.35 + scale*0.7)
			FillRounded(ctx, x-pad, y-pad*0.45+dy, w+pad*2, h+pad*2, r+pad, withAlpha(shadow, alpha))
		}
	}
	FillRounded(ctx, x, y, w, h, r, c)
}

// FillRounded 按半径填充圆角矩形；r<=0 走直角分支。给可选接口的降级路径复用，
// 避免每个调用点重复 if r > 0。
func FillRounded(ctx Context, x, y, w, h, r float32, c Color) {
	if r > 0 {
		ctx.FillRoundedRect(x, y, w, h, r, c)
		return
	}
	ctx.FillRect(x, y, w, h, c)
}

// withAlpha 返回同色不同透明度的新颜色（只改 A 通道）。
func withAlpha(c Color, a float32) Color {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	c.A = a
	return c
}
