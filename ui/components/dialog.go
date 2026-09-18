package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("dialog", renderer.Widget{
		Measure: measureDialog,
		Layout:  layoutDialog,
		Paint:   paintDialog,
	})
}

const (
	dialogDefaultW = float32(400)
	dialogMinW     = float32(240)
	dialogPad      = float32(20)
	dialogGap      = float32(12)
	dialogRadius   = float32(8)
)

// Dialog 居中悬浮对话框：铺满父级的半透明遮罩 + 居中卡片。
//
// Attrs:
//
//	title        标题
//	message      正文（可含换行）
//	onClose      Esc / 点击遮罩时回调
//	onEnter      Enter 回调（确认框用来「保存」）
//	dismiss      点击遮罩是否关闭，默认 true；确认框可设 false
//	closeOnEnter Enter 是否关闭，默认 false
//	width        卡片宽度，默认 400
func Dialog(parts ...any) *runtime.VNode {
	n := runtime.H("dialog", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	if n.Style.Flex == 0 {
		n.Style.Flex = 1
	}
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		if !e.Down {
			return false
		}
		if e.Key == runtime.KeyEscape {
			if fn := onVoid(n, "onClose"); fn != nil {
				fn()
			}
			return true
		}
		if e.Key == runtime.KeyEnter {
			if fn := onVoid(n, "onEnter"); fn != nil {
				fn()
				return true
			}
			if renderer.AttrBool(n, "closeOnEnter") {
				if fn := onVoid(n, "onClose"); fn != nil {
					fn()
				}
				return true
			}
		}
		return true
	})
	if n.OnClick == nil {
		if dialogDismiss(n) {
			n.OnClick = func() {
				if fn := onVoid(n, "onClose"); fn != nil {
					fn()
				}
			}
		} else {
			n.OnClick = func() {}
		}
	}
	return n
}

func dialogDismiss(n *runtime.VNode) bool {
	if n == nil || n.Attrs == nil {
		return true
	}
	if v, ok := n.Attrs["dismiss"].(bool); ok {
		return v
	}
	return true
}

func dialogCardWidth(n *runtime.VNode, maxW float32) float32 {
	w := dialogDefaultW
	switch v := renderer.Attr(n, "width").(type) {
	case float32:
		if v > 0 {
			w = v
		}
	case int:
		if v > 0 {
			w = float32(v)
		}
	}
	if maxW > 32 && w > maxW-32 {
		w = maxW - 32
	}
	if w < dialogMinW {
		if maxW > 0 && maxW < dialogMinW {
			return maxW
		}
		w = dialogMinW
	}
	return w
}

func measureDialog(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	w, h := maxW, maxH
	if w <= 0 && ctx != nil {
		w = ctx.Width()
	}
	if h <= 0 && ctx != nil {
		h = ctx.Height()
	}
	_ = n
	_ = style
	_ = th
	return w, h
}

func layoutDialog(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	if ctx != nil {
		if w <= 0 {
			w = ctx.Width()
		}
		if bottom := ctx.Height() - y; bottom > 0 {
			h = bottom
		}
	}
	n.X, n.Y, n.W, n.H = x, y, w, h

	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	if _, ok := n.Attrs["_body"]; !ok {
		body := make([]*runtime.VNode, 0, len(n.Children))
		for _, c := range n.Children {
			if c != nil {
				body = append(body, c)
			}
		}
		n.Attrs["_body"] = body
	}
	user, _ := n.Attrs["_body"].([]*runtime.VNode)

	cardW := dialogCardWidth(n, w)
	kids := make([]*runtime.VNode, 0, 4)
	if title := renderer.AttrString(n, "title"); title != "" {
		kids = append(kids, runtime.Text(runtime.Props{
			Style: runtime.Style{FontSize: 16, Weight: 600, Color: dialogTitleColor(th)},
		}, title))
	}
	if msg := renderer.AttrString(n, "message"); msg != "" {
		kids = append(kids, runtime.Text(runtime.Props{
			Style: runtime.Style{FontSize: 13, Color: dialogMuted(th)},
		}, msg))
	}
	if len(user) > 0 {
		if len(user) == 1 && user[0].Tag == "hstack" {
			kids = append(kids, user[0])
		} else {
			kids = append(kids, HStack(runtime.Props{
				Style: runtime.Style{Gap: 8, Justify: runtime.End, Align: runtime.Center},
			}, user))
		}
	}

	cardID := n.ID()
	if cardID != "" {
		cardID += "-card"
	}
	card := VStack(runtime.Props{
		Style: runtime.Style{
			Width:      cardW,
			Padding:    dialogPad,
			Gap:        dialogGap,
			Radius:     dialogRadius,
			Background: dialogCardBG(th),
			Border:     dialogCardBorder(th),
		},
		OnClick: func() {},
		ID:      cardID,
	}, kids)

	cw, ch := renderer.MeasureNode(ctx, card, cardW, 0, style, th)
	if cw < cardW {
		cw = cardW
	}
	cx := x + (w-cw)/2
	cy := y + (h-ch)/2
	if cx < x {
		cx = x
	}
	if cy < y {
		cy = y
	}
	renderer.LayoutNode(ctx, card, cx, cy, cw, ch, style, th)
	n.Children = []*runtime.VNode{card}
}

func paintDialog(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	bg := renderer.RGBA(0, 0, 0, 72)
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	}
	ctx.FillRect(n.X, n.Y, n.W, n.H, bg)
	_ = th
	renderer.PaintChildren(ctx, n, style, th)
}

func dialogTitleColor(th theme.Theme) runtime.Color {
	if renderer.ColorSet(th.Foreground) {
		return th.Foreground
	}
	return runtime.RGB(20, 20, 20)
}

func dialogMuted(th theme.Theme) runtime.Color {
	if th.Dark {
		return runtime.RGB(180, 184, 190)
	}
	return runtime.RGB(96, 100, 108)
}

func dialogCardBG(th theme.Theme) runtime.Color {
	if th.Dark {
		if renderer.ColorSet(th.Titlebar) {
			return th.Titlebar
		}
		return runtime.RGB(32, 32, 36)
	}
	if renderer.ColorSet(th.Background) {
		return th.Background
	}
	return runtime.RGB(255, 255, 255)
}

func dialogCardBorder(th theme.Theme) runtime.Color {
	if renderer.ColorSet(th.Separator) {
		return th.Separator
	}
	return runtime.RGB(210, 210, 210)
}
