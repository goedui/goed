package components

import (
	"sync"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("select", renderer.Widget{
		Measure: measureSelect,
		Layout:  layoutSelect,
		Paint:   paintSelect,
	})
}

var (
	selectMu   sync.Mutex
	selectOpen = map[string]bool{}
)

// SetSelectOpen 打开 / 关闭指定 id 的下拉（刷新模型后自动弹出等场景）。
func SetSelectOpen(id string, open bool) {
	if id == "" {
		return
	}
	selectMu.Lock()
	selectOpen[id] = open
	selectMu.Unlock()
}

func selectIsOpen(id string) bool {
	selectMu.Lock()
	defer selectMu.Unlock()
	return selectOpen[id]
}

// Select 下拉选择。Attrs: id, value, items []string, placeholder, onChange, disabled, compact。
// compact 只画箭头，给「输入框 + ▼」这种拆分布局用。
func Select(parts ...any) *runtime.VNode {
	n := runtime.H("select", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	id := n.ID()
	items := stringList(n.Attrs["items"])
	open := selectIsOpen(id)
	if v, ok := n.Attrs["open"].(bool); ok {
		open = v
	}
	n.Attrs["open"] = open

	if n.OnClick == nil {
		n.OnClick = func() {
			if !renderer.Enabled(n) {
				return
			}
			if len(items) == 0 {
				if fn := onVoid(n, "onEmpty"); fn != nil {
					fn()
				}
				return
			}
			next := !selectIsOpen(id)
			SetSelectOpen(id, next)
			if fn := onBool(n, "onOpen"); fn != nil {
				fn(next)
			}
		}
	}

	n.Children = n.Children[:0]
	if open {
		n.Children = append(n.Children, Popover(runtime.Props{
			Anchor: id, Items: items,
			OnClose: func() {
				SetSelectOpen(id, false)
				if fn := onBool(n, "onOpen"); fn != nil {
					fn(false)
				}
			},
			OnSelect: func(v string) {
				if isPopoverSep(v) {
					return
				}
				SetSelectOpen(id, false)
				if fn := onString(n, "onChange"); fn != nil {
					fn(v)
				}
				if fn := onBool(n, "onOpen"); fn != nil {
					fn(false)
				}
			},
		}))
	}
	return n
}

func selectLabel(n *runtime.VNode) string {
	if s := renderer.AttrString(n, "value"); s != "" {
		return s
	}
	if s := renderer.AttrString(n, "placeholder"); s != "" {
		return s
	}
	if s := renderer.AttrString(n, "label"); s != "" {
		return s
	}
	return ""
}

func measureSelect(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	_ = th
	if renderer.AttrBool(n, "compact") {
		w := n.Style.Width
		if w <= 0 {
			w = 44
		}
		h := n.Style.Height
		if h <= 0 {
			h = 32
		}
		_ = ctx
		_ = maxW
		_ = maxH
		_ = style
		return w, h
	}
	label := selectLabel(n) + "  ▼"
	padX, padY := float32(12), float32(8)
	if n.Style.Padding > 0 {
		padX, padY = n.Style.Padding, n.Style.Padding
	}
	limit := maxW
	if limit > 2*padX {
		limit -= 2 * padX
	}
	tw, textH := ctx.MeasureText(label, limit, style)
	if textH < style.FontSize {
		textH = style.FontSize
	}
	w := tw + 2*padX
	h := textH + 2*padY
	if n.Style.Height > 0 {
		h = n.Style.Height
	} else if h < 32 {
		h = 32
	}
	if n.Style.Width > 0 {
		w = n.Style.Width
	} else if n.Style.Flex > 0 && maxW > 0 {
		w = maxW
	} else if maxW > 0 && w > maxW {
		w = maxW
	}
	_ = maxH
	return w, h
}

func layoutSelect(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	n.X, n.Y, n.W, n.H = x, y, w, h
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		if c.Tag == "popover" {
			cw, ch := renderer.MeasureNode(ctx, c, w, h, style, th)
			renderer.LayoutNode(ctx, c, x, y, cw, ch, style, th)
			continue
		}
		renderer.LayoutNode(ctx, c, x, y, w, h, style, th)
	}
}

func paintSelect(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	enabled := renderer.Enabled(n)
	bg := renderer.ColorFrom(th.Background)
	if bg.A == 0 {
		bg = renderer.RGB(255, 255, 255)
	}
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	}
	fg := renderer.ColorFrom(th.Foreground)
	if renderer.ColorSet(n.Style.Color) {
		fg = renderer.ColorFrom(n.Style.Color)
	}
	border := renderer.ColorFrom(th.Separator)
	if renderer.ColorSet(n.Style.Border) {
		border = renderer.ColorFrom(n.Style.Border)
	}
	if !enabled {
		// 与 button 同一对值，走主题（见 theme.DisabledColors）。
		dbg, dfg := th.DisabledColors()
		bg = renderer.ColorFrom(dbg)
		fg = renderer.ColorFrom(dfg)
	}
	r := n.Style.Radius
	if r <= 0 {
		r = 8
	}
	if enabled && border.A > 0 && n.W > 2 && n.H > 2 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, border)
		inner := r - 1
		if inner < 0 {
			inner = 0
		}
		ctx.FillRoundedRect(n.X+1, n.Y+1, n.W-2, n.H-2, inner, bg)
	} else if r > 0 {
		ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	} else {
		ctx.FillRect(n.X, n.Y, n.W, n.H, bg)
	}

	pad := n.Style.Padding
	if pad <= 0 {
		pad = 12
	}
	if !renderer.AttrBool(n, "compact") {
		ctx.DrawText(selectLabel(n), n.X+pad, n.Y, n.W-2*pad-18, n.H, renderer.TextStyle{
			FontFamily: style.FontFamily,
			FontSize:   style.FontSize,
			Color:      fg,
			Weight:     style.Weight,
			Align:      renderer.AlignStart,
			VAlign:     renderer.AlignCenter,
			NoWrap:     true,
		})
	}
	arrow := fg
	arrow.A *= 0.7
	aw := float32(12)
	ax := n.X + n.W - pad - aw
	if renderer.AttrBool(n, "compact") {
		ax = n.X + (n.W-aw)/2
	}
	PaintGlyph(ctx, "chevron", ax, n.Y+(n.H-aw)/2, aw, arrow)
}
