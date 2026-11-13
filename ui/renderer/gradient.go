package renderer

// RadialGradient 描述一次「径向渐变填充圆角矩形」。
//
// 与 GradientFill 的分工：线性渐变负责按钮 / 选中胶囊这类方向明确的面，
// 径向渐变负责背景光斑、插画光晕这类中心向外的辉光。两者都做成可选接口，
// 后端不实现时退回纯色 —— 能看见形状，只是没有过渡。
type RadialGradient struct {
	// X/Y/W/H/Corner 是目标矩形与圆角半径（DIP）。
	X, Y, W, H, Corner float32
	// CenterX/CenterY 是渐变圆心（DIP，绝对坐标）。
	CenterX, CenterY float32
	// Spread 是圆心到渐变终点的距离。<=0 时取目标矩形外接圆半径。
	Spread float32
	// From 是圆心色，To 是边缘色（通常带低 alpha）。
	From, To Color
}

// RadialGradientPainter 由支持径向渐变的后端实现：Windows 上是
// ID2D1RenderTarget::CreateRadialGradientBrush，Web 上是 canvas
// createRadialGradient。与 GradientPainter 同一套降级哲学。
type RadialGradientPainter interface {
	FillRadialGradient(g RadialGradient)
}

// FillRadialGradient 画一个径向渐变填充的圆角矩形，返回后端是否真画了渐变。
// 不支持的后端用 From 纯色填充（形状与位置仍然正确）。
func FillRadialGradient(ctx Context, g RadialGradient) bool {
	if ctx == nil || g.W <= 0 || g.H <= 0 {
		return false
	}
	if g.Spread <= 0 {
		// 外接圆半径：保证四个角也在渐变范围内，不会在角落突然断成纯色。
		g.Spread = 0.5 * sqrt32(g.W*g.W+g.H*g.H)
	}
	if g.Spread <= 0 {
		return false
	}
	if p, ok := ctx.(RadialGradientPainter); ok {
		p.FillRadialGradient(g)
		return true
	}
	FillRounded(ctx, g.X, g.Y, g.W, g.H, g.Corner, g.From)
	return false
}

func sqrt32(v float32) float32 {
	if v <= 0 {
		return 0
	}
	// 牛顿迭代，避免为一处引入 math 依赖到热路径上（math.Sqrt 本身也不慢，
	// 这里的意义是保持本文件只依赖 renderer 包内类型）。
	x := v
	for i := 0; i < 6; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
