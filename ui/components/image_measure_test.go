// image 度量与自然尺寸的常驻回归。
//
// image 只注册了 Measure + Paint（**没有 Layout**）：它是个叶子，矩形由父级
// flow 按 measure 的结果给。所以 measure 的兜底阶梯就是它的全部尺寸语义：
//
//	Style.Width/Height → maxW（宽）/ Flex>0 ? maxH : 56（高）→ 56
//
// 注意它**完全不看 ctx**（和 measureLayer / measureTitleBar 不一样）：没有窗口
// 尺寸可退，最后一律落到 56。把这条写进用例，是因为「随手加一句 ctx.Width()
// 兜底」看起来很像改进，实际会让没有宽度的图片突然变成整窗宽。
//
// imageNaturalSize 读 PNG/JPEG 头。坏字节 / 截断必须回 (0,0) —— 上层据此决定
// 「不做 letterbox，直接铺满盒子」，返回一个假尺寸会把图画歪。
package components

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestImageRegisteredUnderTag(t *testing.T) {
	w, ok := renderer.Lookup("image")
	if !ok {
		t.Fatal("image 未注册：节点会静默退回普通盒子，图片永远画不出来")
	}
	if w.Measure == nil || w.Paint == nil {
		t.Fatalf("image 注册表缺函数：measure=%v paint=%v", w.Measure != nil, w.Paint != nil)
	}
}

func TestMeasureImageFallbackLadder(t *testing.T) {
	// ctx 故意给一个和任何期望值都不同的尺寸：measureImage 不读它。
	ctx := &mockContext{w: 1000, h: 800}
	th := theme.Light
	st := renderer.TextStyle{FontSize: 13}

	cases := []struct {
		name         string
		style        runtime.Style
		maxW, maxH   float32
		wantW, wantH float32
		why          string
	}{
		{"显式宽高优先", runtime.Style{Width: 200, Height: 120}, 1000, 800, 200, 120, "Style 写了就用它"},
		{"只给宽，高退 56", runtime.Style{Width: 200}, 1000, 800, 200, 56, "没有 Flex 时高是缩略图默认"},
		{"都不给 → 宽吃满可用空间", runtime.Style{}, 300, 200, 300, 56, "宽跟容器，高仍 56"},
		{"Flex>0 → 高吃满 maxH", runtime.Style{Flex: 1}, 300, 200, 300, 200, "flex 图片当背景铺满"},
		{"Flex>0 但 maxH=0 → 仍退 56", runtime.Style{Flex: 1}, 300, 0, 300, 56, "maxH=0 不等于「高度 0」"},
		{"可用空间也没有 → 硬默认 56", runtime.Style{}, 0, 0, 56, 56, "两层兜底都失效；ctx 不参与"},
		{"只给高，宽退 56", runtime.Style{Height: 90}, 0, 400, 56, 90, "宽没有来源 → 硬默认"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := Image(runtime.Props{Style: tc.style})
			w, h := measureImage(ctx, n, tc.maxW, tc.maxH, st, th)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("measureImage = %gx%g，期望 %gx%g（%s）", w, h, tc.wantW, tc.wantH, tc.why)
			}
		})
	}
}

// pngBytes 造一张 w×h 的真 PNG。
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("构造测试 PNG 失败: %v", err)
	}
	return buf.Bytes()
}

func TestImageNaturalSizeReadsPNGHeader(t *testing.T) {
	w, h := imageNaturalSize(pngBytes(t, 3, 2))
	if w != 3 || h != 2 {
		t.Fatalf("imageNaturalSize = %gx%g，期望 3x2", w, h)
	}
}

// 只需要头部：DecodeConfig 解析到 IHDR 就返回，不必解像素。
// PNG 的头 33 字节 = 8 签名 + 4 长度 + 4 "IHDR" + 13 数据 + 4 CRC。
func TestImageNaturalSizeNeedsOnlyHeader(t *testing.T) {
	good := pngBytes(t, 7, 5)
	if w, h := imageNaturalSize(good[:33]); w != 7 || h != 5 {
		t.Fatalf("只有头部时 = %gx%g，期望 7x5", w, h)
	}
}

func TestImageNaturalSizeRejectsBadBytes(t *testing.T) {
	good := pngBytes(t, 4, 4)
	cases := []struct {
		name string
		data []byte
	}{
		{"空字节", nil},
		{"垃圾字节", []byte("not an image at all")},
		{"只有 PNG 签名", good[:8]},
		{"IHDR 被截断", good[:20]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w, h := imageNaturalSize(tc.data); w != 0 || h != 0 {
				t.Fatalf("imageNaturalSize(%s) = %gx%g，期望 0x0", tc.name, w, h)
			}
		})
	}
}

// imageDraw 记录一次 DrawImage 的矩形，用来验证 letterbox。
type imageDraw struct {
	data       []byte
	x, y, w, h float32
}

// imageProbeContext 在 mockContext 之上补记 DrawImage —— 基类把参数全丢了，
// 「画在哪、画多大」正是 letterbox 的全部内容。
type imageProbeContext struct {
	mockContext
	images []imageDraw
}

func (m *imageProbeContext) DrawImage(data []byte, x, y, w, h float32) {
	m.images = append(m.images, imageDraw{data: data, x: x, y: y, w: w, h: h})
}

// 2:1 的图放进 200x200 的盒子 → 缩成 200x100 上下居中。
// 这条把 imageNaturalSize 和 fitContain 串起来：自然尺寸读错，letterbox 就错。
func TestPaintImageLetterboxesNaturalSize(t *testing.T) {
	ctx := &imageProbeContext{mockContext: mockContext{w: 400, h: 300}}
	n := Image(runtime.Props{
		Style: runtime.Style{Width: 200, Height: 200},
		Data:  pngBytes(t, 2, 1),
	})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 200
	paintImage(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(ctx.images) != 1 {
		t.Fatalf("应画一次图片，得到 %d 次", len(ctx.images))
	}
	d := ctx.images[0]
	if d.w != 200 || d.h != 100 {
		t.Fatalf("letterbox 尺寸 = %gx%g，期望 200x100", d.w, d.h)
	}
	if d.x != 0 || d.y != 50 {
		t.Fatalf("letterbox 位置 = (%g,%g)，期望 (0,50)", d.x, d.y)
	}
}

// fitContain 的两条退化分支：盒子没尺寸、自然尺寸不可用。
// 前者返回负的居中偏移会把内容画到盒子外；后者必须铺满盒子，否则「字节坏了」
// 的图片会缩成 0x0 —— 界面上一片空白，连占位底都看不出有东西。
func TestFitContainDegenerateInputs(t *testing.T) {
	if x, y, w, h := fitContain(0, 100, 10, 10); x != 0 || y != 0 || w != 0 || h != 0 {
		t.Fatalf("零宽盒子 = (%g,%g,%g,%g)，期望全 0", x, y, w, h)
	}
	if x, y, w, h := fitContain(100, 0, 10, 10); x != 0 || y != 0 || w != 0 || h != 0 {
		t.Fatalf("零高盒子 = (%g,%g,%g,%g)，期望全 0", x, y, w, h)
	}
	if x, y, w, h := fitContain(100, 50, 0, 0); x != 0 || y != 0 || w != 100 || h != 50 {
		t.Fatalf("自然尺寸不可用 = (%g,%g,%g,%g)，期望铺满 100x50", x, y, w, h)
	}
}

// 坏字节走完整条 paintImage：自然尺寸读不出来 → 铺满盒子，而不是画一个 0x0。
func TestPaintImageFillsBoxWhenDataUnreadable(t *testing.T) {
	ctx := &imageProbeContext{mockContext: mockContext{w: 400, h: 300}}
	n := Image(runtime.Props{
		Style: runtime.Style{Width: 200, Height: 120},
		Data:  []byte("definitely not a png"),
	})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 120
	paintImage(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(ctx.images) != 1 {
		t.Fatalf("坏字节也应画一次（铺满）：得到 %d 次", len(ctx.images))
	}
	d := ctx.images[0]
	if d.x != 0 || d.y != 0 || d.w != 200 || d.h != 120 {
		t.Fatalf("坏字节 letterbox = (%g,%g,%g,%g)，期望铺满 (0,0,200,120)", d.x, d.y, d.w, d.h)
	}
}

// 没有 data 时不能去问自然尺寸，也不能画图（画一块空的灰底即可）。
func TestPaintImageWithoutDataSkipsDraw(t *testing.T) {
	ctx := &imageProbeContext{mockContext: mockContext{w: 400, h: 300}}
	n := Image(runtime.Props{Style: runtime.Style{Width: 200, Height: 120}})
	n.X, n.Y, n.W, n.H = 0, 0, 200, 120
	paintImage(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)

	if len(ctx.images) != 0 {
		t.Fatalf("无 data 仍画了 %d 次图片", len(ctx.images))
	}
	if len(ctx.fills) != 1 {
		t.Fatalf("无 data 也应画一块占位底：fills=%+v", ctx.fills)
	}
}

// paintImageBadge 画右上角多图角标（Attrs["badge"]），不参与布局。
// 补用例之前它 15.8%（只剩开头那条早退守卫被走到），整块胶囊 + 文字绘制逻辑
// 全仓没验过：位置算错会跑到图外，宽度不夹会顶出右缘。
func TestPaintImageBadgeDrawsPillAndLabel(t *testing.T) {
	ctx := &mockContext{w: 400, h: 300}
	n := Image(runtime.Props{
		Style: runtime.Style{Width: 120, Height: 90},
		Badge: "3",
	})
	n.X, n.Y, n.W, n.H = 0, 0, 120, 90
	paintImageBadge(ctx, n, renderer.TextStyle{FontSize: 13})

	if len(ctx.rounds) != 1 {
		t.Fatalf("应画一个圆角胶囊，得到 %d 个：%+v", len(ctx.rounds), ctx.rounds)
	}
	pill := ctx.rounds[0]
	if pill.h != 18 || pill.r != 9 {
		t.Fatalf("胶囊高/圆角 = %g/%g，期望 18/9（半径 = 高的一半）", pill.h, pill.r)
	}
	// tw = 1 字符 * 11 * 0.5 = 5.5 → bw = 17.5；bx = 0 + 120 - 17.5 - 4
	if pill.w != 17.5 || pill.x != 98.5 || pill.y != 4 {
		t.Fatalf("胶囊矩形 = (%g,%g,%g,%g)，期望 (98.5,4,17.5,18)", pill.x, pill.y, pill.w, pill.h)
	}
	if len(ctx.draws) != 1 || ctx.draws[0].text != "3" {
		t.Fatalf("角标文字 = %+v，期望一次 \"3\"", ctx.draws)
	}
	st := ctx.draws[0].style
	if st.FontSize != 11 || st.Weight != 600 || !st.NoWrap {
		t.Fatalf("角标字重/字号 = %g/%v，期望 11/600 且不换行", st.FontSize, st.Weight)
	}
	if st.Color != renderer.RGB(255, 255, 255) || st.Align != renderer.AlignCenter || st.VAlign != renderer.AlignCenter {
		t.Fatalf("角标文字样式 = %+v，期望白色居中", st)
	}
}

// 角标宽度要夹进盒子：长文字不能顶出右缘。
func TestPaintImageBadgeClampsToBoxWidth(t *testing.T) {
	ctx := &mockContext{w: 400, h: 300}
	n := Image(runtime.Props{
		Style: runtime.Style{Width: 50, Height: 40},
		Badge: "很长的角标文字",
	})
	n.X, n.Y, n.W, n.H = 0, 0, 50, 40
	paintImageBadge(ctx, n, renderer.TextStyle{FontSize: 13})

	if len(ctx.rounds) != 1 {
		t.Fatalf("应画一个胶囊，得到 %d 个", len(ctx.rounds))
	}
	if got := ctx.rounds[0].w; got != 42 {
		t.Fatalf("胶囊宽 = %g，期望 42（= 盒宽 50 - 8 的留白）", got)
	}
}

// 没地方画就一个像素都不画：空角标、盒太窄、盒太矮。
func TestPaintImageBadgeSkipsWhenNoRoom(t *testing.T) {
	cases := []struct {
		name  string
		badge string
		w, h  float32
	}{
		{"没有角标文字", "", 120, 90},
		{"盒太窄", "3", 39, 90},
		{"盒太矮", "3", 120, 23},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &mockContext{w: 400, h: 300}
			n := Image(runtime.Props{
				Style: runtime.Style{Width: tc.w, Height: tc.h},
				Badge: tc.badge,
			})
			n.X, n.Y, n.W, n.H = 0, 0, tc.w, tc.h
			paintImageBadge(ctx, n, renderer.TextStyle{FontSize: 13})

			if len(ctx.rounds) != 0 || len(ctx.draws) != 0 {
				t.Fatalf("不该画角标：rounds=%+v draws=%+v", ctx.rounds, ctx.draws)
			}
		})
	}
}
