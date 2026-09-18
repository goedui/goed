package components

import (
	"testing"
	"time"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// switch 组件：点击回调、颜色可覆盖、滑块位置随状态变化、动画朝目标推进。
func TestSwitchBasics(t *testing.T) {
	runtime.SetFocus("")
	ResetSwitchAnim("sw-basic")
	ctx := &mockContext{w: 200, h: 100}
	got := []bool{}
	n := Switch(runtime.Props{
		Style:    runtime.Style{Width: 40, Height: 22},
		ID:       "sw-basic",
		Value:    false,
		Label:    "测试开关",
		OnBG:     "#38D684",
		OffBG:    "#3A3A3C",
		Knob:     "#171718",
		OnChange: func(v bool) { got = append(got, v) },
	})
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if len(ctx.rounds) < 2 {
		t.Fatalf("开关应画轨道与滑块，实际 %d 个圆角矩形", len(ctx.rounds))
	}
	track := ctx.rounds[0]
	offKnob := ctx.rounds[len(ctx.rounds)-1]
	// 关闭态：轨道用 offbg，滑块贴左
	if diff(track.c, renderer.RGB(0x3A, 0x3A, 0x3C)) > 0.01 {
		t.Fatalf("关闭态轨道颜色不对：%+v", track.c)
	}
	if offKnob.x > n.X+6 {
		t.Fatalf("关闭态滑块应贴左：x=%.1f", offKnob.x)
	}

	// 点击 → 回调收到 true
	n.OnClick()
	if len(got) != 1 || got[0] != true {
		t.Fatalf("点击应回调 true：%+v", got)
	}

	// 打开态（同一 id）：轨道变 onbg，滑块滑到右边
	ResetSwitchAnim("sw-basic")
	ctx2 := &mockContext{w: 200, h: 100}
	on := Switch(runtime.Props{
		Style: runtime.Style{Width: 40, Height: 22},
		ID:    "sw-basic",
		Value: true,
		OnBG:  "#38D684",
		OffBG: "#3A3A3C",
		Knob:  "#171718",
	})
	renderer.PaintTree(ctx2, []*runtime.VNode{on}, theme.Dark)
	trackOn := ctx2.rounds[0]
	onKnob := ctx2.rounds[len(ctx2.rounds)-1]
	if diff(trackOn.c, renderer.RGB(0x38, 0xD6, 0x84)) > 0.01 {
		t.Fatalf("打开态轨道应使用 onbg：%+v", trackOn.c)
	}
	if onKnob.x < n.X+18 {
		t.Fatalf("打开态滑块应贴右：x=%.1f", onKnob.x)
	}
	if diff(onKnob.c, renderer.RGB(0x17, 0x17, 0x18)) > 0.02 {
		t.Fatalf("滑块颜色应可覆盖：%+v", onKnob.c)
	}
	ResetSwitchAnim("sw-basic")
}

// TestSwitchAnimates 状态翻转后滑块位置在若干帧里单调推进（不是一步到位）。
func TestSwitchAnimates(t *testing.T) {
	ResetSwitchAnim("sw-anim")
	p := switchProgress("sw-anim", 0, time.Now())
	if p != 0 {
		t.Fatalf("初始应为 0，实际 %v", p)
	}
	p2 := switchProgress("sw-anim", 1, time.Now().Add(40*time.Millisecond))
	if p2 <= 0 || p2 >= 1 {
		t.Fatalf("40ms 后应处于中间态，实际 %v", p2)
	}
	p3 := switchProgress("sw-anim", 1, time.Now().Add(400*time.Millisecond))
	if p3 != 1 {
		t.Fatalf("超过过渡时长后应到位，实际 %v", p3)
	}
	ResetSwitchAnim("sw-anim")
}

// TestParseHexColor 颜色解析与容错。
func TestParseHexColor(t *testing.T) {
	c, ok := ParseHexColor("#38d684")
	if !ok || diff(c, renderer.RGB(0x38, 0xD6, 0x84)) > 0.01 {
		t.Fatalf("解析 #38d684 失败：%+v %v", c, ok)
	}
	if _, ok := ParseHexColor("red"); ok {
		t.Fatal("非法颜色应返回 false")
	}
	if _, ok := ParseHexColor("#12345"); ok {
		t.Fatal("位数不对应返回 false")
	}
}

func diff(a, b renderer.Color) float32 {
	d := a.R - b.R
	if v := a.G - b.G; v > d {
		d = v
	} else if -v > d {
		d = -v
	}
	if v := a.B - b.B; v > d {
		d = v
	} else if -v > d {
		d = -v
	}
	if d < 0 {
		d = -d
	}
	return d
}
