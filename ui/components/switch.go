package components

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("switch", renderer.Widget{
		Measure: measureSwitch,
		Paint:   paintSwitch,
	})
}

// Switch 布尔开关。
//
// Attrs：
//
//	value       当前开关状态（外部持有）
//	onChange    点击后回调新值 func(bool)
//	onbg/offbg  轨道颜色（默认取主题；buddy 里用绿色表示开启）
//	knob        滑块颜色（默认白）
//	label       无障碍标签 / 截图工具定位用
//
// 滑块位置带 140ms 过渡：状态一变就朝目标滑，滑动过程中自己请求重绘。
func Switch(parts ...any) *runtime.VNode {
	n := runtime.H("switch", parts...)
	if n.OnClick == nil {
		n.OnClick = func() {
			if !renderer.Enabled(n) {
				return
			}
			on := renderer.AttrBool(n, "value")
			if fn := onBool(n, "onChange"); fn != nil {
				fn(!on)
			}
		}
	}
	return n
}

func measureSwitch(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = ctx
	_ = maxW
	_ = maxH
	_ = style
	_ = th
	// 尺寸可由 style 覆盖：表单里的开关往往要比默认小一档。
	w := n.Style.Width
	if w <= 0 {
		w = 40
	}
	h := n.Style.Height
	if h <= 0 {
		h = 22
	}
	return w, h
}

// switchAnim 记录每个开关的动画进度（0=关，1=开）。
var (
	switchMu    sync.Mutex
	switchState = map[string]float32{}
	switchTick  = map[string]time.Time{}
)

const switchDuration = 140 * time.Millisecond

// switchProgress 返回平滑推进后的进度，并驱动动画帧。
func switchProgress(id string, target float32, now time.Time) float32 {
	switchMu.Lock()
	defer switchMu.Unlock()
	cur, ok := switchState[id]
	if !ok {
		switchState[id] = target
		switchTick[id] = now
		return target
	}
	last, ok := switchTick[id]
	if !ok || last.IsZero() {
		last = now
	}
	dt := now.Sub(last)
	switchTick[id] = now
	if cur == target {
		return cur
	}
	step := float32(dt) / float32(switchDuration)
	if step < 0 {
		step = 0
	}
	if cur < target {
		cur += step
		if cur > target {
			cur = target
		}
	} else {
		cur -= step
		if cur < target {
			cur = target
		}
	}
	switchState[id] = cur
	if cur != target {
		platform.RequestFrame()
	}
	return cur
}

// ResetSwitchAnim 丢弃某个开关的动画状态（测试与重建界面时用）。
func ResetSwitchAnim(id string) {
	switchMu.Lock()
	delete(switchState, id)
	delete(switchTick, id)
	switchMu.Unlock()
}

func paintSwitch(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	_ = style
	on := renderer.AttrBool(n, "value")
	target := float32(0)
	if on {
		target = 1
	}
	prog := switchProgress(n.ID(), target, time.Now())

	off := renderer.ColorFrom(th.Separator)
	onColor := renderer.ColorFrom(th.Button)
	if c, ok := colorAttr(n, "offbg"); ok {
		off = c
	}
	if c, ok := colorAttr(n, "onbg"); ok {
		onColor = c
	}
	bg := mixColor(off, onColor, prog)
	if !renderer.Enabled(n) {
		bg.A *= 0.45
	}
	r := n.H / 2
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)

	knobSize := n.H - 6
	kx := n.X + 3 + prog*(n.W-6-knobSize)
	knob := renderer.RGB(255, 255, 255)
	if c, ok := colorAttr(n, "knob"); ok {
		knob = c
	}
	ctx.FillRoundedRect(kx, n.Y+3, knobSize, knobSize, knobSize/2, knob)
}

// colorAttr 读一个颜色 Attr（支持 ui.Color / renderer.Color / "#RRGGBB" 字符串）。
func colorAttr(n *runtime.VNode, key string) (renderer.Color, bool) {
	if n == nil || n.Attrs == nil {
		return renderer.Color{}, false
	}
	switch v := n.Attrs[key].(type) {
	case renderer.Color:
		return v, true
	case string:
		if c, ok := ParseHexColor(v); ok {
			return c, true
		}
	}
	return renderer.Color{}, false
}

// ParseHexColor 解析 "#RRGGBB" / "RRGGBB"；失败返回 false。
func ParseHexColor(s string) (renderer.Color, bool) {
	t := strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(t) != 6 {
		return renderer.Color{}, false
	}
	v, err := strconv.ParseUint(t, 16, 32)
	if err != nil {
		return renderer.Color{}, false
	}
	return renderer.RGB(uint8(v>>16&0xFF), uint8(v>>8&0xFF), uint8(v&0xFF)), true
}

// mixColor 线性插值（t=0 取 a，t=1 取 b）。
func mixColor(a, b renderer.Color, t float32) renderer.Color {
	t = float32(math.Max(0, math.Min(1, float64(t))))
	return renderer.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

// TintColor 把颜色按 alpha 稀释成底色（徽标色块用）。
func TintColor(c renderer.Color, alpha float32) renderer.Color {
	c.A = alpha
	return c
}
