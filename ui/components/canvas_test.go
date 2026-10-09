// Canvas 自定义绘制组件的常驻回归。
//
// Canvas 是 ui/components 里**唯一整个控件跨包 0%** 的：全仓（ui/ + buddy/ +
// editor/）的测试都没有渲染过它一次 —— measure / layout / paint 三个函数、连同
// 注册表里绑它们的那一行，全都一行没跑过。
//
// 这里重点守两件事：
//
//  1. measure 的兜底阶梯：Style 尺寸 → maxW/maxH → 硬默认 400/300。
//  2. onPaint 的**两种回调形状**都要认。文件头的用法注释教用户内联写
//     `func(ctx renderer.Context, x, y, w, h float32)`（**匿名**函数类型），
//     而代码第一个分支断言的是**具名类型** PaintCallback —— Go 对具名类型的
//     断言不成立，内联函数字面量只会命中第二条。两条分支各自独立，删掉任一条
//     都会让一种写法静默失效（回调再也不会被调用，界面上一片空白）。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// canvasStyle 是各用例共用的度量入参（canvas 不读字体，给个合法值即可）。
func canvasStyle() renderer.TextStyle { return renderer.TextStyle{FontSize: 13} }

// canvasProbe 记录 onPaint 收到的矩形与调用次数。
type canvasProbe struct {
	calls      int
	ctxNil     bool
	x, y, w, h float32
}

// canvasRecorder 返回探针与一个「匿名函数类型」的回调。
// 注意返回类型写全：它就是注释里教用户内联写的那种**未命名**函数类型，
// 存进 Props.OnPaint（any）之后动态类型仍是未命名类型。
func canvasRecorder() (*canvasProbe, func(renderer.Context, float32, float32, float32, float32)) {
	p := &canvasProbe{}
	return p, func(ctx renderer.Context, x, y, w, h float32) {
		p.calls++
		p.ctxNil = ctx == nil
		p.x, p.y, p.w, p.h = x, y, w, h
	}
}

// TestCanvasRegisteredUnderTag 控件靠 tag 注册表被发现：三个函数都要接上。
// 少一个就会静默退回「普通盒子」分支（measureFlow + 永不调用回调），
// 界面表现是「画不出来但也不报错」。
func TestCanvasRegisteredUnderTag(t *testing.T) {
	w, ok := renderer.Lookup("canvas")
	if !ok {
		t.Fatal(`renderer.Lookup("canvas") 未命中：init 里的 Register 没生效`)
	}
	if w.Measure == nil || w.Layout == nil || w.Paint == nil {
		t.Fatalf("canvas 的 Measure/Layout/Paint 必须都接上：%+v", w)
	}
}

// TestCanvasMeasureFallbackLadder 走 renderer.MeasureNode（内部 Lookup(tag).Measure），
// 顺带把注册表那条路径也覆盖掉。
//
// 注意**两道**尺寸逻辑叠加：measureCanvas 自己只管「Style 尺寸 → maxW/maxH → 硬默认
// 400/300」，之后 measureNode 还会跑一道 applySize（ui/renderer/layout.go），
// 把结果再钳到 maxW/maxH。所以「显式 Width=800 而可用宽度只有 500」时最终是 500 ——
// 写这条用例时正是这里第一版期望写错（以为 800），实测才看清。
func TestCanvasMeasureFallbackLadder(t *testing.T) {
	ctx := &mockContext{w: 500, h: 400}
	th := theme.Dark
	st := canvasStyle()

	cases := []struct {
		name       string
		style      runtime.Style
		maxW, maxH float32
		wantW      float32
		wantH      float32
	}{
		{"显式尺寸在可用空间内生效", runtime.Style{Width: 800, Height: 600}, 1000, 800, 800, 600},
		// overflow 默认 visible：显式尺寸可以大于父级给的可用空间，画出去由祖先裁。
		{"显式尺寸大于可用空间仍保留", runtime.Style{Width: 800, Height: 600}, 500, 400, 800, 600},
		{"hidden 才被可用空间钳住", runtime.Style{Width: 800, Height: 600, Overflow: runtime.OverflowHidden}, 500, 400, 500, 400},
		{"都不给 → 用可用空间", runtime.Style{}, 500, 400, 500, 400},
		{"可用空间也没有 → 硬默认", runtime.Style{}, 0, 0, 400, 300},
		{"只给宽，高用可用空间", runtime.Style{Width: 250}, 500, 400, 250, 400},
		{"只给高，宽用可用空间", runtime.Style{Height: 120}, 500, 400, 500, 120},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := Canvas(runtime.Props{Style: c.style})
			gotW, gotH := renderer.MeasureNode(ctx, n, c.maxW, c.maxH, st, th)
			if gotW != c.wantW || gotH != c.wantH {
				t.Fatalf("测得 (%g,%g)，期望 (%g,%g)", gotW, gotH, c.wantW, c.wantH)
			}
		})
	}
}

// TestCanvasLayoutPlacesNodeAtGivenRect layout 只做一件事：把矩形写进节点。
func TestCanvasLayoutPlacesNodeAtGivenRect(t *testing.T) {
	w, _ := renderer.Lookup("canvas")
	n := Canvas(runtime.Props{Style: runtime.Style{Width: 800, Height: 600}})
	w.Layout(&mockContext{w: 500, h: 400}, n, 10, 20, 300, 200, canvasStyle(), theme.Dark)
	if n.X != 10 || n.Y != 20 || n.W != 300 || n.H != 200 {
		t.Fatalf("布局后矩形 (%g,%g,%g,%g)，期望 (10,20,300,200)", n.X, n.Y, n.W, n.H)
	}
}

// TestCanvasPaintAcceptsDocumentedAnonymousCallback 文件头注释教的写法就是内联
// 匿名函数，必须认（这是绝大多数调用方的写法）。
func TestCanvasPaintAcceptsDocumentedAnonymousCallback(t *testing.T) {
	probe, fn := canvasRecorder()
	w, _ := renderer.Lookup("canvas")
	n := Canvas(runtime.Props{Style: runtime.Style{Width: 800, Height: 600}, OnPaint: fn})
	w.Layout(nil, n, 7, 9, 800, 600, canvasStyle(), theme.Dark)
	w.Paint(&mockContext{w: 500, h: 400}, n, canvasStyle(), theme.Dark)

	if probe.calls != 1 {
		t.Fatalf("匿名函数写法的 onPaint 被调用 %d 次，期望 1 次（具名类型断言把它挡了）", probe.calls)
	}
	if probe.x != 7 || probe.y != 9 || probe.w != 800 || probe.h != 600 {
		t.Fatalf("回调收到 (%g,%g,%g,%g)，期望节点矩形 (7,9,800,600)",
			probe.x, probe.y, probe.w, probe.h)
	}
}

// TestCanvasPaintAcceptsNamedPaintCallbackType 显式转成 PaintCallback（或声明成
// 该类型的变量）走第一条分支。与上一条是两条独立路径。
func TestCanvasPaintAcceptsNamedPaintCallbackType(t *testing.T) {
	probe, fn := canvasRecorder()
	w, _ := renderer.Lookup("canvas")
	n := Canvas(runtime.Props{
		Style:   runtime.Style{Width: 800, Height: 600},
		OnPaint: PaintCallback(fn),
	})
	w.Layout(nil, n, 3, 4, 800, 600, canvasStyle(), theme.Dark)
	w.Paint(&mockContext{w: 500, h: 400}, n, canvasStyle(), theme.Dark)

	if probe.calls != 1 {
		t.Fatalf("PaintCallback 具名类型的 onPaint 被调用 %d 次，期望 1 次", probe.calls)
	}
	if probe.x != 3 || probe.y != 4 || probe.w != 800 || probe.h != 600 {
		t.Fatalf("回调收到 (%g,%g,%g,%g)，期望节点矩形 (3,4,800,600)",
			probe.x, probe.y, probe.w, probe.h)
	}
}

// TestCanvasPaintNilContextOrNodeIsSafe paint 开头那道守卫是**承重**的：
// renderer.PaintBackground 自己虽然 nil-safe，但紧接着的 `n.Attrs` 不是 ——
// 去掉守卫后 paintCanvas(ctx, nil, …) 会直接解引用空指针。
// ctx 为 nil 时也不该把 nil 上下文递给用户回调。
func TestCanvasPaintNilContextOrNodeIsSafe(t *testing.T) {
	probe, fn := canvasRecorder()
	w, _ := renderer.Lookup("canvas")
	n := Canvas(runtime.Props{Style: runtime.Style{Width: 800, Height: 600}, OnPaint: fn})
	w.Layout(nil, n, 0, 0, 800, 600, canvasStyle(), theme.Dark)

	w.Paint(nil, n, canvasStyle(), theme.Dark)
	if probe.calls != 0 {
		t.Fatal("ctx 为 nil 时不该把 nil 上下文递给用户回调")
	}

	// n 为 nil：这里若 panic，说明守卫被删了。
	w.Paint(&mockContext{w: 500, h: 400}, nil, canvasStyle(), theme.Dark)
}

// TestCanvasPaintsBackgroundThenCallback 有 background 时先铺底再交给回调。
func TestCanvasPaintsBackgroundThenCallback(t *testing.T) {
	probe, fn := canvasRecorder()
	w, _ := renderer.Lookup("canvas")
	n := Canvas(runtime.Props{
		Style:   runtime.Style{Width: 800, Height: 600, Background: runtime.RGB(10, 20, 30)},
		OnPaint: fn,
	})
	w.Layout(nil, n, 1, 2, 800, 600, canvasStyle(), theme.Dark)
	ctx := &mockContext{w: 500, h: 400}
	w.Paint(ctx, n, canvasStyle(), theme.Dark)

	if len(ctx.fills) != 1 {
		t.Fatalf("背景应铺 1 次，实际 %d 次", len(ctx.fills))
	}
	if f := ctx.fills[0]; f.x != 1 || f.y != 2 || f.w != 800 || f.h != 600 {
		t.Fatalf("背景矩形 (%g,%g,%g,%g)，期望节点矩形 (1,2,800,600)", f.x, f.y, f.w, f.h)
	}
	if probe.calls != 1 {
		t.Fatalf("回调应被调用 1 次，实际 %d 次", probe.calls)
	}
}

// TestCanvasThroughPaintTree 端到端：PaintTree 会查注册表 → measure → layout → paint。
// 断言写成**自洽不变量**（回调矩形 == 节点最终矩形），不写死绝对坐标 ——
// 这样换主题内边距 / 换窗口尺寸不会假红，而「画在哪儿报哪儿」被破坏时立刻红。
func TestCanvasThroughPaintTree(t *testing.T) {
	probe, fn := canvasRecorder()
	n := Canvas(runtime.Props{Style: runtime.Style{Width: 320, Height: 240}, OnPaint: fn})
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)

	if probe.calls != 1 {
		t.Fatalf("PaintTree 之后 onPaint 被调用 %d 次，期望 1 次", probe.calls)
	}
	if n.W <= 0 || n.H <= 0 {
		t.Fatalf("节点尺寸应当已落定，实际 (%g,%g)", n.W, n.H)
	}
	if probe.x != n.X || probe.y != n.Y || probe.w != n.W || probe.h != n.H {
		t.Fatalf("回调矩形 (%g,%g,%g,%g) 与节点矩形 (%g,%g,%g,%g) 不一致",
			probe.x, probe.y, probe.w, probe.h, n.X, n.Y, n.W, n.H)
	}
}
