package renderer

import "testing"

// plainContext 只实现最小 Context（模拟不支持可选接口的后端 / 测试 mock）。
type plainContext struct {
	fills    []string // 记录走了哪条填充路径
	lastC    Color
	lastR    float32
	callNum  int
}

func (c *plainContext) Width() float32  { return 100 }
func (c *plainContext) Height() float32 { return 100 }
func (c *plainContext) DPI() float32    { return 96 }
func (c *plainContext) Clear(Color)     {}

func (c *plainContext) FillRect(x, y, w, h float32, col Color) {
	c.fills = append(c.fills, "rect")
	c.lastC = col
}
func (c *plainContext) FillRoundedRect(x, y, w, h, r float32, col Color) {
	c.fills = append(c.fills, "round")
	c.lastC = col
	c.lastR = r
}
func (c *plainContext) DrawRect(x, y, w, h float32, col Color, width float32) {}
func (c *plainContext) DrawLine(x1, y1, x2, y2 float32, col Color, width float32) {
}
func (c *plainContext) DrawText(text string, x, y, maxW, maxH float32, style TextStyle) {}
func (c *plainContext) MeasureText(text string, maxW float32, style TextStyle) (float32, float32) {
	return float32(len(text)) * 8, 16
}
func (c *plainContext) DrawImage(data []byte, x, y, w, h float32)                     {}
func (c *plainContext) DrawPixels(pix []byte, pw, ph int, x, y, w, h float32)          {}
func (c *plainContext) PushClip(x, y, w, h float32)                                    {}
func (c *plainContext) PopClip()                                                       {}

// gradientContext 额外实现可选渐变接口，用于验证「支持则走原生」。
type gradientContext struct {
	plainContext
	gotGradient bool
	gotVertical bool
}

func (c *gradientContext) FillGradientRoundedRect(x, y, w, h, r float32, from, to Color, vertical bool) {
	c.gotGradient = true
	c.gotVertical = vertical
}

// shadowContext 额外实现可选阴影接口。
type shadowContext struct {
	plainContext
	gotShadow bool
	gotBlur   float32
}

func (c *shadowContext) FillShadowRoundedRect(x, y, w, h, r, blur, offsetY float32, c2, shadow Color) {
	c.gotShadow = true
	c.gotBlur = blur
	// 与接口契约一致：本体由该方法画。
	c.FillRoundedRect(x, y, w, h, r, c2)
}

func TestGradientFallsBackToSolidWithoutPainter(t *testing.T) {
	ctx := &plainContext{}
	ok := FillGradientRoundedRect(ctx, 0, 0, 10, 10, 4, RGB(1, 2, 3), RGB(4, 5, 6), true)
	if ok {
		t.Fatal("plainContext 不应报告支持渐变")
	}
	if len(ctx.fills) != 1 || ctx.fills[0] != "round" {
		t.Fatalf("降级应画一次纯色圆角矩形，实际 %v", ctx.fills)
	}
	if ctx.lastC.R == 0 {
		t.Fatal("降级应使用 from 端点色")
	}
}

func TestGradientUsesPainterWhenAvailable(t *testing.T) {
	ctx := &gradientContext{}
	ok := FillGradientRoundedRect(ctx, 0, 0, 10, 10, 4, RGB(1, 2, 3), RGB(4, 5, 6), true)
	if !ok || !ctx.gotGradient {
		t.Fatal("实现 GradientPainter 的后端应走原生渐变")
	}
	if !ctx.gotVertical {
		t.Fatal("方向参数未透传")
	}
	if len(ctx.fills) != 0 {
		t.Fatalf("原生渐变不应再画纯色回退，实际 %v", ctx.fills)
	}
}

func TestShadowPainterDrawsBodyOnce(t *testing.T) {
	ctx := &shadowContext{}
	FillShadowRoundedRect(ctx, 0, 0, 10, 10, 4, 8, 2, RGB(9, 9, 9), RGB(0, 0, 0))
	if !ctx.gotShadow || ctx.gotBlur != 8 {
		t.Fatal("实现 ShadowPainter 的后端应走原生阴影")
	}
	if len(ctx.fills) != 1 || ctx.fills[0] != "round" {
		t.Fatalf("本体只能被画一次，实际 %v", ctx.fills)
	}
}

func TestShadowFallbackLayersThenBody(t *testing.T) {
	ctx := &plainContext{}
	FillShadowRoundedRect(ctx, 0, 0, 10, 10, 4, 8, 2, RGB(9, 9, 9), RGBA(0, 0, 0, 128))
	// 四层近似 + 本体
	if len(ctx.fills) != 5 {
		t.Fatalf("近似阴影应为四层叠影 + 本体共 5 次填充，实际 %v", ctx.fills)
	}
	if ctx.lastC.A != 1 {
		t.Fatal("最后画的必须是不透明本体")
	}
}

func TestShadowNoColorSkipsLayers(t *testing.T) {
	ctx := &plainContext{}
	FillShadowRoundedRect(ctx, 0, 0, 10, 10, 4, 0, 0, RGB(9, 9, 9), Color{})
	if len(ctx.fills) != 1 {
		t.Fatalf("无阴影色/无模糊时只画本体，实际 %v", ctx.fills)
	}
}

func TestFillRoundedRoutesByRadius(t *testing.T) {
	ctx := &plainContext{}
	FillRounded(ctx, 0, 0, 5, 5, 0, RGB(1, 1, 1))
	FillRounded(ctx, 0, 0, 5, 5, 3, RGB(1, 1, 1))
	if len(ctx.fills) != 2 || ctx.fills[0] != "rect" || ctx.fills[1] != "round" {
		t.Fatalf("r<=0 走直角、r>0 走圆角，实际 %v", ctx.fills)
	}
}
