package components

import (
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// ButtonHost 按钮宿主：状态 + 点击回调。
type ButtonHost struct {
	core.BaseComponent
	Label    string
	OnClick  func()
	Enabled  bool
	Radius   int32
	Variant  string
	Selected bool
	focused  bool

	state buttonState
}

func (b *ButtonHost) SetFocused(v bool) { b.focused = v }
func (b *ButtonHost) CanFocus() bool    { return b.Enabled && b.Visible }

type buttonState int

const (
	buttonNormal buttonState = iota
	buttonHover
	buttonPressed
)

// MeasureContent 实现 Measurable：按钮按文本 + 内边距测量。
func (b *ButtonHost) MeasureContent(ctx core.DrawingContext) {
	if b.Bounds.W > 0 || b.Bounds.H > 0 {
		return
	}
	th := theme.GetTheme()
	ctx.SetFont(th.UIFontFamily, th.UIFontSize)
	tw, thh := ctx.MeasureText(b.Label)
	if tw <= 0 {
		tw = int32(len([]rune(b.Label))) * 8
	}
	if thh <= 0 {
		thh = 20
	}
	b.Bounds.W, b.Bounds.H = tw+32, thh+16
}

// Render 绘制按钮。
func (b *ButtonHost) Render(ctx core.DrawingContext) {
	if !b.Visible {
		return
	}
	if b.Layout != nil {
		b.Layout.Arrange(b.Bounds)
	}
	th := theme.GetTheme()
	ctx.SetFont(th.UIFontFamily, th.UIFontSize)

	bg, fg := th.ButtonBg, th.ButtonFg
	if b.Variant == "secondary" {
		bg, fg = th.WindowBg, th.TitleBarFg
	}
	if b.Selected {
		bg = th.TitleBarBtnHover
	}
	if !b.Enabled {
		bg, fg = th.ButtonDisabledBg, th.ButtonDisabledFg
	} else {
		switch b.state {
		case buttonHover:
			bg = th.ButtonHoverBg
		case buttonPressed:
			bg = th.ButtonPressBg
		}
	}

	ctx.DrawRoundedRect(b.Bounds.X, b.Bounds.Y, b.Bounds.W, b.Bounds.H, b.Radius, bg, true)
	border := th.ButtonBorder
	if b.focused || b.Selected {
		border = th.ButtonBg
	}
	ctx.DrawRoundedRect(b.Bounds.X, b.Bounds.Y, b.Bounds.W, b.Bounds.H, b.Radius, border, false)

	label := fitLabel(ctx, b.Label, max(0, b.Bounds.W-16))
	tw, thh := ctx.MeasureText(label)
	ctx.DrawText(b.Bounds.X+(b.Bounds.W-tw)/2, b.Bounds.Y+(b.Bounds.H-thh)/2, label, fg)
}

// HandleEvent 处理鼠标事件，返回是否消费。
func (b *ButtonHost) HandleEvent(event core.Event) bool {
	if !b.Enabled || !b.Visible {
		return false
	}
	switch event.Type {
	case core.EventMouseLeave:
		changed := b.state != buttonNormal
		b.state = buttonNormal
		return changed
	case core.EventKeyDown:
		if b.focused && (event.Key == core.KeyEnter || event.Key == 32) {
			if b.OnClick != nil {
				b.OnClick()
			}
			return true
		}
	case core.EventMouseMove:
		hover := b.Contains(event.X, event.Y)
		if hover && b.state == buttonNormal {
			b.state = buttonHover
			return true
		}
		if !hover && b.state == buttonHover {
			b.state = buttonNormal
			return true
		}
	case core.EventMouseDown:
		if b.Contains(event.X, event.Y) && event.Button == 0 {
			b.state = buttonPressed
			return true
		}
	case core.EventMouseUp:
		if b.state == buttonPressed {
			b.state = buttonNormal
			if b.Contains(event.X, event.Y) && b.OnClick != nil {
				b.OnClick()
			}
			return true
		}
	}
	return false
}

// buttonElement "button" 元素实现。
type buttonElement struct{}

func (buttonElement) Tag() string { return "button" }

func (buttonElement) Create(props runtime.Props) core.Component {
	b := &ButtonHost{
		BaseComponent: *core.NewBaseComponent(),
		Label:         props.Str("label", ""),
		OnClick:       props.Fn("onClick"),
		Enabled:       props.Bool("enabled", true),
	}
	b.Visible = props.Bool("visible", true)
	b.Radius = props.Int("radius", 0)
	b.Variant = props.Str("variant", "")
	b.Selected = props.Bool("selected", false)
	return b
}

func (buttonElement) Update(host core.Component, old, new runtime.Props) {
	b := host.(*ButtonHost)
	b.Radius = new.Int("radius", 0)
	b.Variant = new.Str("variant", "")
	b.Selected = new.Bool("selected", false)
	b.OnClick = new.Fn("onClick")
	if l := new.Str("label", ""); l != old.Str("label", "") {
		b.Label = l
	}
	if fn := new.Fn("onClick"); fn != nil {
		b.OnClick = fn
	}
	if e := new.Bool("enabled", true); e != old.Bool("enabled", true) {
		b.Enabled = e
	}
	if v := new.Bool("visible", true); v != old.Bool("visible", true) {
		b.Visible = v
	}
}
