// 编辑框的绘制：背景 / 边框 / 文本 / 选区 / 组合串下划线 / 光标 / 滚动条。
//
// 绘制是唯一把「滚动偏移」写回状态的地方（光标跟随会改它），因此钳制函数
// clampScrollY 也放在这里，滚轮（textarea.go）复用同一个边界。
package components

import (
	"strings"
	"unicode/utf8"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// clampScrollY 把多行滚动偏移钳进 [0, 内容高度-可视高度]。
//
// 滚轮改偏移、绘制也改偏移，边界必须出自同一个函数：此前滚轮自由设值、
// 只有绘制才钳制，于是滚轮设了 48、下一帧被收成 47，偏移在两帧之间抖动。
//
// 内容高度由最近一次布局推得（st.layout）。还没布局过（首次绘制前）
// 就没有上限可言，此时只保证非负，等下一帧绘制时再收进边界。
func clampScrollY(st *editState, viewH float32) float32 {
	off := st.scrollY
	if st.layout != nil {
		maxOff := float32(st.layout.maxLine+1)*st.layout.lineH + 2*inputTopPad - viewH
		if maxOff < 0 {
			maxOff = 0
		}
		if off > maxOff {
			off = maxOff
		}
	}
	if off < 0 {
		off = 0
	}
	return off
}

// editPaint 是绘制开关，只描述「怎么画」。
//
// 注意不要往这里塞滚动偏移：偏移是编辑状态（用户滚轮 / 光标跟随都会改它），
// 权威副本只有一份，就在 editState.scrollY。曾经这里留过一个 scrollY 字段，
// 但没有任何调用方赋值，于是每次绘制都把 st.scrollY 覆盖回 0 —— 滚轮一滚
// 就被抹掉。滚动偏移一律从 st 读。
type editPaint struct {
	multiline     bool
	scrollToCaret bool
	showScroll    bool
}

// paintEditable 画背景 / 边框 / 文本 / 选区 / 组合串下划线 / 光标，
// 并把光标矩形缓存起来（输入法定位候选窗用）。
func paintEditable(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme, st *editState, opt editPaint) {
	// 以节点上的 multiline attr 为准，和 handleEditKey 同一规则：
	// 组件类型只决定默认值。Input 加了 multiline 后必须顶对齐，
	// 否则光标在第一行、文字却垂直居中。
	if renderer.AttrBool(n, "multiline") {
		opt.multiline = true
	}
	bg := renderer.RGB(255, 255, 255)
	if renderer.ColorSet(n.Style.Background) {
		bg = renderer.ColorFrom(n.Style.Background)
	}
	border := renderer.ColorFrom(th.Separator)
	focused := runtime.IsFocused(n)
	if focused {
		border = renderer.ColorFrom(th.Button)
	}
	if !renderer.Enabled(n) {
		bg = renderer.RGB(232, 234, 238)
	}
	r := n.Style.Radius
	if r <= 0 {
		r = 8
	}
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	if !renderer.AttrBool(n, "borderless") {
		ctx.DrawRect(n.X, n.Y, n.W, n.H, border, 1)
	}

	shown := st.value
	if renderer.AttrBool(n, "password") {
		shown = strings.Repeat("•", utf8.RuneCountInString(st.value))
	}
	body := shown
	compStart := -1
	if st.comp != "" {
		at := clampCaret(shown, st.compAt)
		body = shown[:at] + st.comp + shown[at:]
		compStart = at
	}
	placeholder := renderer.AttrString(n, "placeholder")
	fg := style.Color
	empty := body == ""
	if empty {
		body = placeholder
		fg = renderer.RGB(120, 126, 140)
		if style.Color.A > 0 {
			fg = style.Color
			fg.A = 0.5
		}
	}
	// nowrap / 单行：不按宽度折行，改为横向滚动。
	nowrap := renderer.AttrBool(n, "nowrap") || !opt.multiline
	textStyle := style
	textStyle.Color = fg
	textStyle.NoWrap = nowrap
	textStyle.VAlign = renderer.AlignCenter
	if opt.multiline {
		textStyle.VAlign = renderer.AlignStart
	}
	wrapW := n.W - 2*inputPad

	layoutW := wrapW
	if nowrap {
		layoutW = 0 // 0 表示不折行，只在硬换行处断行
	}
	lay := layoutText(ctx, st, body, layoutW, textStyle)

	// 滚动：多行超出可视高度时按 scrollY 偏移，并把光标滚进可视范围。
	// 偏移取自编辑状态：这样滚轮浏览的位移能保留到下一帧。
	scroll := st.scrollY
	if opt.multiline {
		// 打字 / 移动光标时把光标滚进可视范围。
		if opt.scrollToCaret && st.scrollSync {
			caretLine := lay.pointAt(st.caretOffset() + compCursorBytes(st)).line
			caretTop := float32(caretLine) * lay.lineH
			viewH := n.H - 2*inputTopPad
			if viewH > 0 {
				if caretTop-scroll < 0 {
					scroll = caretTop
				}
				if caretTop+lay.lineH-scroll > viewH {
					scroll = caretTop + lay.lineH - viewH
				}
			}
			st.scrollSync = false
		}
		st.scrollY = scroll
		// 钳制交给公共函数：滚轮那边用的是同一个，边界一致。
		scroll = clampScrollY(st, n.H)
	}

	// 横向滚动：不换行时按 scrollX 偏移。打字 / 移动光标时把光标列滚进可视宽度，
	// 拖选浏览时不要每帧强行拉回（与多行 scrollSync 同一规则）。
	offsetX := float32(0)
	if nowrap {
		offsetX = st.scrollX
		if st.scrollSync {
			cp := lay.pointAt(st.caretOffset() + compCursorBytes(st))
			viewW := wrapW
			if viewW > 0 {
				switch {
				case cp.x-offsetX < 0:
					offsetX = cp.x
				case cp.x-offsetX > viewW:
					offsetX = cp.x - viewW
				}
			}
			if !opt.multiline {
				st.scrollSync = false
			}
		}
		if offsetX < 0 {
			offsetX = 0
		}
		st.scrollX = offsetX
	}

	ctx.PushClip(n.X+1, n.Y+1, n.W-2, n.H-2)
	textY := n.Y + inputTopPad - scroll
	if !opt.multiline {
		textY = n.Y
	}
	drawW := wrapW
	if nowrap {
		end := lay.pointAtLineEnd(0)
		if end.x+2 > drawW {
			drawW = end.x + 2
		}
	}
	ctx.DrawText(body, n.X+inputPad-offsetX, textY, drawW, n.H-2*inputTopPad+scroll, textStyle)

	// 选区高亮：逐可视行画矩形。
	if from, to, has := st.selRange(); has {
		sel := renderer.ColorFrom(th.Button)
		sel.A = 0.30
		line := -1
		var lineFrom float32
		for _, p := range lay.points {
			if p.off < from {
				continue
			}
			if p.off > to {
				break
			}
			if p.line != line {
				if line >= 0 {
					ctx.FillRect(n.X+inputPad+lineFrom-offsetX, n.Y+inputTopPad+float32(line)*lay.lineH-scroll,
						lay.pointAtLineEnd(line).x-lineFrom, lay.lineH, sel)
				}
				line = p.line
				lineFrom = p.x
			}
		}
		if line >= 0 {
			endX := lay.pointAt(clampCaret(body, to)).x
			if compStart >= 0 {
				endX = lay.pointAt(clampCaret(body, to)).x
			}
			ctx.FillRect(n.X+inputPad+lineFrom-offsetX, n.Y+inputTopPad+float32(line)*lay.lineH-scroll,
				endX-lineFrom, lay.lineH, sel)
		}
	}

	// 组合串下划线。
	if st.comp != "" {
		startP := lay.pointAt(st.caretOffset() - len(st.comp))
		endP := lay.pointAt(st.caretOffset())
		underline := renderer.ColorFrom(th.Button)
		y := n.Y + inputTopPad + float32(startP.line+1)*lay.lineH - 2 - scroll
		if startP.line == endP.line {
			ctx.FillRect(n.X+inputPad+startP.x, y, endP.x-startP.x, 1.5, underline)
		} else {
			ctx.FillRect(n.X+inputPad+startP.x, y, wrapW-startP.x, 1.5, underline)
			ctx.FillRect(n.X+inputPad, n.Y+inputTopPad+float32(endP.line+1)*lay.lineH-2-scroll, endP.x, 1.5, underline)
		}
	}

	// 光标。
	cp := lay.pointAt(st.caretOffset() + compCursorBytes(st))
	cx := n.X + inputPad + cp.x - offsetX
	cy := n.Y + inputTopPad + float32(cp.line)*lay.lineH + 1 - scroll
	caretH := lay.lineH - 2
	if caretH < minCaretH {
		caretH = minCaretH
	}
	ctx.PopClip()

	st.caretRect = [4]float32{cx, cy, caretWidth, caretH}
	st.caretOK = true

	if focused && renderer.Enabled(n) && !renderer.AttrBool(n, "readonly") && blinkVisible() {
		caretColor := style.Color
		if caretColor.A == 0 {
			caretColor = renderer.ColorFrom(th.Foreground)
		}
		if empty {
			caretColor = fg
		}
		ctx.FillRect(cx, cy, caretWidth, caretH, caretColor)
	}

	// 滚动条（多行且内容超高时）。
	if opt.showScroll {
		contentH := float32(lay.maxLine+1)*lay.lineH + 2*inputTopPad
		if contentH > n.H+1 {
			barH := n.H * n.H / contentH
			if barH < 24 {
				barH = 24
			}
			barY := n.Y + (n.H-barH)*(scroll/(contentH-n.H))
			ctx.FillRoundedRect(n.X+n.W-6, barY, 3.5, barH, 2, renderer.ColorFrom(th.Separator))
		}
	}
}
