package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("checkbox", renderer.Widget{
		Measure: measureCheckbox,
		Paint:   paintCheckbox,
	})
}

// 勾选框的默认几何：方框 14 DIP，方框与标签间距 8，整行最矮 20（好点）。
const (
	checkboxBox    = float32(14)
	checkboxGap    = float32(8)
	checkboxMinH   = float32(20)
	checkboxRadius = float32(3)
)

// Checkbox 复选框：方框 + 对勾 + 右侧标签，整块可点。
//
// Props 与 Switch 同一套约定：
//
//	Value    是否勾选（外部持有）
//	Label    右侧文本
//	Disabled 置灰且不吃点击
//	OnChange 点击后回调新值 func(bool)
func Checkbox(parts ...any) *runtime.VNode {
	n := runtime.H("checkbox", parts...)
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

func measureCheckbox(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = maxH
	w := n.Style.Width
	h := n.Style.Height
	if h <= 0 {
		h = checkboxMinH
	}
	// 宽度不给就按「方框 + 间距 + 标签宽」取自然尺寸；给了就尊重调用方
	//（表单里的勾选框常常要撑满一行）。
	if w <= 0 {
		w = checkboxBox + checkboxGap
		if label := renderer.AttrString(n, "label"); label != "" && ctx != nil {
			tw, thh := ctx.MeasureText(label, maxW-w, checkboxTextStyle(style, th))
			w += tw
			if thh > h {
				h = thh
			}
		}
	}
	return w, h
}

func paintCheckbox(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	on := renderer.AttrBool(n, "value")
	alpha := float32(1)
	if !renderer.Enabled(n) {
		alpha = 0.45
	}

	by := n.Y + (n.H-checkboxBox)/2
	if on {
		c := renderer.ColorFrom(th.Button)
		c.A *= alpha
		ctx.FillRoundedRect(n.X, by, checkboxBox, checkboxBox, checkboxRadius, c)
		// 对勾：两段线，白色，压得住色块。
		chk := renderer.RGB(255, 255, 255)
		chk.A *= alpha
		ctx.DrawLine(n.X+checkboxBox*0.24, by+checkboxBox*0.52, n.X+checkboxBox*0.42, by+checkboxBox*0.72, chk, 1.8)
		ctx.DrawLine(n.X+checkboxBox*0.42, by+checkboxBox*0.72, n.X+checkboxBox*0.78, by+checkboxBox*0.28, chk, 1.8)
	} else {
		c := renderer.ColorFrom(th.Background)
		c.A *= alpha
		ctx.FillRoundedRect(n.X, by, checkboxBox, checkboxBox, checkboxRadius, c)
		bc := renderer.ColorFrom(th.Separator)
		bc.A *= alpha
		ctx.DrawRect(n.X, by, checkboxBox, checkboxBox, bc, 1)
	}

	if label := renderer.AttrString(n, "label"); label != "" {
		ts := checkboxTextStyle(style, th)
		if !renderer.Enabled(n) {
			ts.Color.A *= alpha
		}
		ctx.DrawText(label, n.X+checkboxBox+checkboxGap, n.Y, n.W-checkboxBox-checkboxGap, n.H, ts)
	}
}

// checkboxTextStyle 勾选框的文字样式：默认跟随主题，调用方可用节点 Style 覆盖。
func checkboxTextStyle(style renderer.TextStyle, th theme.Theme) renderer.TextStyle {
	if style.FontFamily == "" {
		style.FontFamily = th.FontFamily
	}
	if style.FontSize <= 0 {
		style.FontSize = th.FontSize
	}
	if style.Color.A == 0 {
		style.Color = renderer.ColorFrom(th.Foreground)
	}
	style.NoWrap = true
	style.VAlign = renderer.AlignCenter
	return style
}
