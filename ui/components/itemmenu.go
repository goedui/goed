package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("itemmenu", renderer.Widget{
		Measure: measureItemMenu,
		Layout:  layoutItemMenu,
		Paint:   paintItemMenu,
	})
}

// MenuRow is one row of an ItemMenu: an icon, a name and a one-line gloss.
//
// It is deliberately not MenuItem: MenuItem is a flat command with a shortcut
// column, while a MenuRow is a two-line entry that describes a choice. Mixing
// them would make every menu carry a description column it never uses.
type MenuRow struct {
	ID     string
	Icon   string
	Label  string
	Desc   string
	Active bool
}

// Panel metrics. The reference panel is 316 wide with 44 tall rows; keeping
// them as constants makes the panel measurable before it is laid out, which is
// what lets it flip above the trigger instead of running off-screen.
const (
	itemMenuWidth    = float32(316)
	itemMenuHeaderH  = float32(54)
	itemMenuRowH     = float32(44)
	itemMenuFooterH  = float32(40)
	itemMenuPadX     = float32(12)
	itemMenuLabelSz  = float32(13)
	itemMenuDescSz   = float32(11)
	itemMenuHeaderSz = float32(11)
	itemMenuRadius   = float32(8)
)

// Selection blue from the reference. It is a flat colour rather than the accent
// at low alpha: blending the accent over the panel shifts the hue.
var (
	itemMenuSelBG = runtime.RGB(4, 57, 94)
	itemMenuSelFG = runtime.RGB(255, 255, 255)
	itemMenuInkC  = runtime.RGB(228, 228, 228)
	itemMenuMutC  = runtime.RGB(157, 157, 157)
	itemMenuDimC  = runtime.RGB(122, 122, 122)
	itemMenuRuleC = runtime.RGB(60, 60, 60)
	// itemMenuLinkC marks the footer label as a link. It is the accent, not the
	// dim body colour, because that row is the only clickable text in the panel.
	itemMenuLinkC = runtime.RGB(21, 121, 197)
)

func menuRowsFrom(v any) []MenuRow {
	switch t := v.(type) {
	case []MenuRow:
		return t
	case []any:
		out := make([]MenuRow, 0, len(t))
		for _, x := range t {
			if r, ok := x.(MenuRow); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// ItemMenu is the rich dropdown panel: an optional header (description plus a
// keyboard hint), one row per choice and an optional footer.
//
// The root is a full-canvas mask, like ContextMenu: clicking anywhere outside
// the card closes the panel. The card itself is laid out from the anchor and
// flips above it when the bottom of the canvas is too close, which is what
// makes it usable on the composer's mode bar.
//
// Attrs:
//
//	anchor             触发按钮的 id
//	items              []MenuRow
//	title / hint       头部说明与键盘提示（都可省）
//	footer             底部右侧文案（省则不画底栏）
//	onSelect           func(string)，收到 MenuRow.ID
//	onClose            关闭回调：Esc / 点面板外都会走它
//	selectedBackground 选中行底色（缺省 #04395E）
//	selectedColor      选中行文字色（缺省白）
func ItemMenu(parts ...any) *runtime.VNode {
	n := runtime.H("itemmenu", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	rows := menuRowsFrom(n.Attrs["items"])
	if len(rows) == 0 {
		return nil
	}
	// 遮罩铺满画布，点哪儿都算「点在面板外面」。行自己的 OnClick 在子树里，
	// 命中时后遍历的子节点会覆盖它，所以点行不会误关面板。
	if n.OnClick == nil {
		n.OnClick = func() {
			if fn := onVoid(n, "onClose"); fn != nil {
				fn()
			}
		}
	}
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		switch e.Key {
		case runtime.KeyEscape:
			if fn := onVoid(n, "onClose"); fn != nil {
				fn()
			}
			return true
		case runtime.KeyUp, runtime.KeyDown:
			// 上下键在选项之间移动，而不是「选中第一项」：把当前项的相邻项
			// 交给 onSelect，键盘习惯与列表导航一致。
			idx := -1
			for i, r := range rows {
				if r.Active {
					idx = i
					break
				}
			}
			step := 1
			if e.Key == runtime.KeyUp {
				step = -1
			}
			if idx < 0 {
				idx = 0
			} else {
				idx = (idx + step + len(rows)) % len(rows)
			}
			if fn := onString(n, "onSelect"); fn != nil {
				fn(rows[idx].ID)
			}
			return true
		}
		return false
	})
	n.Children = buildItemMenu(n, rows)
	return n
}

// buildItemMenu materialises the card contents as real nodes so the engine
// paints them and collects their hit areas. Fixed row heights make this
// possible before layout runs.
func buildItemMenu(n *runtime.VNode, rows []MenuRow) []*runtime.VNode {
	selBG, selFG := itemMenuSelBG, itemMenuSelFG
	if c, ok := renderer.Attr(n, "selectedBackground").(runtime.Color); ok && renderer.ColorSet(c) {
		selBG = c
	}
	if c, ok := renderer.Attr(n, "selectedColor").(runtime.Color); ok && renderer.ColorSet(c) {
		selFG = c
	}

	var kids []*runtime.VNode

	title := renderer.AttrString(n, "title")
	hint := renderer.AttrString(n, "hint")
	if title != "" || hint != "" {
		var head []*runtime.VNode
		if title != "" {
			head = append(head, runtime.Text(runtime.Props{
				Style: runtime.Style{FontSize: itemMenuHeaderSz, Color: itemMenuMutC},
			}, title))
		}
		if hint != "" {
			head = append(head, runtime.Text(runtime.Props{
				Style: runtime.Style{FontSize: itemMenuHeaderSz, Color: itemMenuDimC},
			}, hint))
		}
		kids = append(kids, VStack(runtime.Props{
			Style: runtime.Style{
				Height: itemMenuHeaderH, Padding: itemMenuPadX,
				Gap: 4, Align: runtime.Stretch, Justify: runtime.Center,
			},
		}, head))
		kids = append(kids, Divider(runtime.Props{
			Style: runtime.Style{Background: itemMenuRuleC},
		}))
	}

	for _, r := range rows {
		r := r
		fg, descFG := itemMenuInkC, itemMenuMutC
		var rowBG runtime.Color
		if r.Active {
			fg, descFG = selFG, selFG
			rowBG = selBG
		}
		var lead *runtime.VNode
		if r.Icon != "" {
			lead = Glyph(runtime.Props{
				Icon:  r.Icon,
				Style: runtime.Style{Width: 15, Height: 15, Color: fg},
			})
		}
		col := VStack(runtime.Props{
			Style: runtime.Style{Flex: 1, Gap: 2, Align: runtime.Stretch},
		},
			runtime.Text(runtime.Props{
				Style:    runtime.Style{FontSize: itemMenuLabelSz, Color: fg},
				Ellipsis: true,
				Nowrap:   true,
			}, r.Label),
			runtime.Text(runtime.Props{
				Style:    runtime.Style{FontSize: itemMenuDescSz, Color: descFG},
				Ellipsis: true,
				Nowrap:   true,
			}, r.Desc),
		)
		kids = append(kids, HStack(runtime.Props{
			Style: runtime.Style{
				Height: itemMenuRowH, Padding: 8, Gap: 10,
				Align: runtime.Center, Background: rowBG,
			},
			OnClick: func() {
				if fn := onString(n, "onSelect"); fn != nil {
					fn(r.ID)
				}
			},
		}, lead, col))
	}

	if footer := renderer.AttrString(n, "footer"); footer != "" {
		kids = append(kids, Divider(runtime.Props{
			Style: runtime.Style{Background: itemMenuRuleC},
		}))
		var lead *runtime.VNode
		// Props.Icon doubles as the footer glyph here: the panel has no other
		// icon slot, and reusing it avoids a second "which icon" field.
		if icon := renderer.AttrString(n, "icon"); icon != "" {
			lead = Glyph(runtime.Props{
				Icon:  icon,
				Style: runtime.Style{Width: 13, Height: 13, Color: itemMenuDimC},
			})
		}
		kids = append(kids, HStack(runtime.Props{
			Style: runtime.Style{
				Height: itemMenuFooterH, Padding: 10, Align: runtime.Center, Gap: 8,
			},
		},
			lead,
			runtime.Text(runtime.Props{
				Style: runtime.Style{Flex: 1, FontSize: itemMenuDescSz, Color: itemMenuDimC},
			}, ""),
			// The footer label is the panel's "go configure this" link, so it is a
			// button: a plain Text would look clickable and do nothing.
			Button(runtime.Props{
				Style: runtime.Style{
					Height: 22, Radius: 5, Padding: 0,
					Background: runtime.Color{}, Border: runtime.Color{},
					Color: itemMenuLinkC, FontSize: itemMenuDescSz,
				},
				Variant: "ghost",
				OnClick: func() {
					if fn := onVoid(n, "onFooter"); fn != nil {
						fn()
					}
				},
			}, footer),
		))
	}
	return kids
}

// itemMenuCardHeight is the card height for a given row count.
func itemMenuCardHeight(n *runtime.VNode, rows int) float32 {
	var h float32
	if renderer.AttrString(n, "title") != "" || renderer.AttrString(n, "hint") != "" {
		h += itemMenuHeaderH + 1
	}
	h += float32(rows) * itemMenuRowH
	if renderer.AttrString(n, "footer") != "" {
		h += 1 + itemMenuFooterH
	}
	return h
}

func itemMenuCardWidth(n *runtime.VNode) float32 {
	if n != nil && n.Style.Width > 0 {
		return n.Style.Width
	}
	return itemMenuWidth
}

// cardRect computes where the card goes: below the anchor by default, above it
// when the canvas bottom is too close, clamped to the canvas.
func itemMenuCardRect(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32) (float32, float32, float32, float32) {
	rows := len(menuRowsFrom(renderer.Attr(n, "items")))
	cw := itemMenuCardWidth(n)
	ch := itemMenuCardHeight(n, rows)

	ax, ay, aw, ah := x, y, cw, float32(0)
	if id := renderer.AttrString(n, "anchor"); id != "" {
		if a := runtime.FindByID(renderer.LayoutRoot(), id); a != nil {
			ax, ay, aw, ah = a.X, a.Y, a.W, a.H
		}
	}
	_ = aw
	px := ax
	py := ay + ah + 4
	if ctx != nil {
		if px+cw > ctx.Width() {
			px = ctx.Width() - cw
		}
		if py+ch > ctx.Height() {
			py = ay - ch - 4
		}
	}
	if px < 0 {
		px = 0
	}
	if py < 0 {
		py = 0
	}
	if cw > w && w > 0 {
		cw = w
	}
	if ch > h && h > 0 {
		ch = h
	}
	return px, py, cw, ch
}

func measureItemMenu(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	// 遮罩铺满父级：给多大就多大。父级没给尺寸时退到画布，免得量成 0 ——
	// 0 尺寸节点在命中测试里会被直接跳过，遮罩就白铺了。
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

func layoutItemMenu(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	// 和 layoutLayer / layoutContextMenu 同一套：宽按画布、高到画布底边。
	if ctx != nil {
		if w <= 0 {
			w = ctx.Width()
		}
		if bottom := ctx.Height() - y; bottom > 0 {
			h = bottom
		}
	}
	n.X, n.Y, n.W, n.H = x, y, w, h

	px, py, pw, ph := itemMenuCardRect(ctx, n, x, y, w, h)
	// 卡片矩形要在绘制阶段复用；布局先于绘制，写回 Attrs 是最短的一条路。
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs["cardX"], n.Attrs["cardY"] = px, py
	n.Attrs["cardW"], n.Attrs["cardH"] = pw, ph

	cy := py
	for _, c := range n.Children {
		if c == nil {
			continue
		}
		_, ch := renderer.MeasureNode(ctx, c, pw, ph, style, th)
		renderer.LayoutNode(ctx, c, px, cy, pw, ch, style, th)
		cy += ch
	}
}

func paintItemMenu(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	if ctx == nil || n == nil {
		return
	}
	px, py := attrFloat(n, "cardX"), attrFloat(n, "cardY")
	pw, ph := attrFloat(n, "cardW"), attrFloat(n, "cardH")
	if pw <= 0 || ph <= 0 {
		return
	}
	bg := renderer.ColorFrom(th.Titlebar)
	if bg.A == 0 {
		bg = renderer.RGB(31, 31, 31)
	}
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	}
	ctx.FillRoundedRect(px, py, pw, ph, itemMenuRadius, bg)
	// 顶部 1px 强调线：参考面板靠它把浮层与背后的输入区分开。
	accent := renderer.ColorFrom(th.Button)
	if accent.A == 0 {
		accent = renderer.RGB(3, 111, 194)
	}
	ctx.FillRect(px+itemMenuRadius, py, pw-2*itemMenuRadius, 1, accent)
	renderer.PaintChildren(ctx, n, style, th)
}
