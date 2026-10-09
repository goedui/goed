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

// drawEditorText 按固定行距逐行画。一次把多行交给后端时，它用自己的行距，
// 和光标的行距差零点几像素，换行越多差得越多。
func drawEditorText(ctx renderer.Context, text string, x, y, w, lineH float32, style renderer.TextStyle) {
	if text == "" || lineH <= 0 {
		return
	}
	style.LineSpacing = lineH
	style.NoWrap = true
	// 一次画完这一段。逐行画会把视口外的行也画上，而且每行一个 DrawText。
	// 行距已经由 LineSpacing 定死，和光标是同一格。
	ctx.DrawText(text, x, y, w+8192, lineH*float32(strings.Count(text, "\n")+4), style)
}

// clampScrollY 把多行滚动偏移钳进 [0, 内容高度-可视高度]。
//
// 滚轮与绘制必须共用这一个边界函数：此前滚轮自由设值、只有绘制钳制，于是滚轮设了
// 48、下一帧被收成 47，偏移在两帧之间抖动。内容高度取自最近一次布局，还没布局过时
// 只保证非负，等下一帧再收进边界。

// revealCaretX 让横向视口跟着光标走，停在行末。
// 光标在视口里时不动；出了右边界就贴着右沿走，出了左边界就贴着左沿走。
// 行比视口短时回到 0。
func revealCaretX(st *editState, caretX, lineEnd, viewW float32) float32 {
	off := st.scrollX
	if viewW <= 0 {
		return 0
	}
	maxOff := lineEnd - viewW
	if maxOff < 0 {
		maxOff = 0
	}
	switch {
	case caretX-off < 0:
		off = caretX
	case caretX-off > viewW:
		off = caretX - viewW
	}
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	return off
}

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

// editPaint 是绘制开关，只描述「怎么画」。别往这里塞滚动偏移：权威副本只有
// editState.scrollY 一份 —— 这里曾经有个没人赋值的 scrollY 字段，每次绘制都把
// st.scrollY 覆盖回 0，滚轮一滚就被抹掉。
type editPaint struct {
	multiline     bool
	scrollToCaret bool
	showScroll    bool
}

// paintEditable 画背景 / 边框 / 文本 / 选区 / 组合串下划线 / 光标，
// 并把光标矩形缓存起来（输入法定位候选窗用）。
func paintEditable(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme, st *editState, opt editPaint) {
	if st != nil {
		st.viewH = n.H
	}
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
		// 只有底从主题取，文字色不换：框里是用户自己的配置值（API 地址这类），
		// 生成期间仍要读得清，「被禁用」由底色表达就够了。
		dbg, _ := th.DisabledColors()
		bg = renderer.ColorFrom(dbg)
		// 顺手收掉焦点环：灰底 + 蓝框读起来像「聚焦着但不能用」，
		// 而边框本来就不该在不可交互时宣称焦点。
		border = renderer.ColorFrom(th.Separator)
	}
	r := n.Style.Radius
	if r <= 0 {
		r = 8
	}
	ctx.FillRoundedRect(n.X, n.Y, n.W, n.H, r, bg)
	if !renderer.AttrBool(n, "borderless") {
		renderer.StrokeRounded(ctx, n.X, n.Y, n.W, n.H, r, 1, border)
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
	// 纵向位置一律由组件自己算好（见下面 textY 那段），后端只负责从给定的
	// 起点往下画，所以这里固定顶对齐，不依赖各后端对 VAlign 的实现。
	textStyle.VAlign = renderer.AlignStart
	wrapW := n.W - 2*inputPad

	layoutW := wrapW
	if nowrap {
		layoutW = 0 // 0 表示不折行，只在硬换行处断行
	}
	lay := layoutText(ctx, st, body, layoutW, textStyle)
	// 绘制行距跟光标同一格。web 按这个行高的 0.8 落基线。
	if lay != nil && lay.lineH > 0 {
		textStyle.LineSpacing = lay.lineH
	}
	if lay.sparse() {
		paintLargeEditable(ctx, n, style, th, st, lay, body, textStyle, focused, empty, fg, opt)
		return
	}

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
		if opt.scrollToCaret && st.scrollSync {
			cp := lay.pointAt(st.caretOffset() + compCursorBytes(st))
			end := lay.pointAtLineEnd(cp.line).x
			offsetX = revealCaretX(st, cp.x, end, wrapW)
		}
		if opt.scrollToCaret && st.scrollSync {
			st.scrollSync = false
		}
		if offsetX < 0 {
			offsetX = 0
		}
		st.scrollX = offsetX
	}

	ctx.PushClip(n.X+1, n.Y+1, n.W-2, n.H-2)

	// 文本块在框内的纵向起点：单行居中，多行顶对齐并跟随滚动偏移。
	//
	// 自己算，不把 VAlign=Center 丢给后端：光标、选区、组合串下划线都要用同一个
	// 起点，交给后端居中就等于「后端按它的行高居中一次 + 组件按 lay.lineH 摆光标
	// 一次」，两边行高差一点光标就和文字错开。
	//
	// 以前走的就是后者，而且盒子还传成 n.H-2*inputTopPad，文字整体偏高 5px（无头
	// Chrome 逐行量过：框 144..181，墨迹 151..164，中心差 5.0px）。
	textH := float32(lay.maxLine+1) * lay.lineH
	textY := n.Y + inputTopPad - scroll
	if !opt.multiline {
		textY = n.Y + (n.H-textH)/2 - scroll
		// 内容比框还高时（单行框里被粘进多行文本）别顶出上边框。
		if minY := n.Y + inputTopPad - scroll; textY < minY {
			textY = minY
		}
	}

	drawW := wrapW
	if nowrap {
		end := lay.pointAtLineEnd(0)
		if end.x+2 > drawW {
			drawW = end.x + 2
		}
	}
	drawBody := body
	drawY := textY
	drawH := n.H - 2*inputTopPad
	if opt.multiline && lay.lineH > 0 && body != "" {
		startLine := int(scroll / lay.lineH)
		if startLine < 0 {
			startLine = 0
		}
		if startLine > lay.maxLine {
			startLine = lay.maxLine
		}
		visible := int((n.H-2*inputTopPad)/lay.lineH) + 2
		if visible < 2 {
			visible = 2
		}
		endLine := startLine + visible
		if endLine > lay.maxLine+1 {
			endLine = lay.maxLine + 1
		}
		startOff := lay.lineByteStart(startLine)
		endOff := len(body)
		if endLine <= lay.maxLine {
			endOff = lay.lineByteStart(endLine)
		}
		if startOff < 0 {
			startOff = 0
		}
		if endOff < startOff {
			endOff = startOff
		}
		if endOff > len(body) {
			endOff = len(body)
		}
		drawBody = body[startOff:endOff]
		drawY = n.Y + inputTopPad + float32(startLine)*lay.lineH - scroll
		drawH = n.H + lay.lineH
	}
	if opt.multiline {
		drawEditorText(ctx, drawBody, n.X+inputPad-offsetX, drawY, drawW, lay.lineH, textStyle)
	} else {
		ctx.DrawText(drawBody, n.X+inputPad-offsetX, drawY, drawW, drawH, textStyle)
	}

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
					ctx.FillRect(n.X+inputPad+lineFrom-offsetX, textY+float32(line)*lay.lineH,
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
			ctx.FillRect(n.X+inputPad+lineFrom-offsetX, textY+float32(line)*lay.lineH,
				endX-lineFrom, lay.lineH, sel)
		}
	}

	// 组合串下划线。
	if st.comp != "" {
		startP := lay.pointAt(st.caretOffset() - len(st.comp))
		endP := lay.pointAt(st.caretOffset())
		underline := renderer.ColorFrom(th.Button)
		y := textY + float32(startP.line+1)*lay.lineH - 2
		if startP.line == endP.line {
			ctx.FillRect(n.X+inputPad+startP.x, y, endP.x-startP.x, 1.5, underline)
		} else {
			ctx.FillRect(n.X+inputPad+startP.x, y, wrapW-startP.x, 1.5, underline)
			ctx.FillRect(n.X+inputPad, textY+float32(endP.line+1)*lay.lineH-2, endP.x, 1.5, underline)
		}
	}

	// 光标按字形在行盒里的位置下移，不贴行顶、也不在行盒里居中。
	cp := lay.pointAt(st.caretOffset() + compCursorBytes(st))
	cx := n.X + inputPad + cp.x - offsetX
	cy, caretH := caretInLine(textY, cp.line, lay.lineH, textStyle.FontSize)
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

	if opt.showScroll || nowrap {
		paintEditorBars(ctx, n, th, st, lay, scroll, nowrap)
	}
	notifyScroll(n, st)
}

// paintEditorBars 画竖、横滚动条，并记下内容宽供拖动换算。
func paintEditorBars(ctx renderer.Context, n *runtime.VNode, th theme.Theme, st *editState, lay *inputLayout, scroll float32, nowrap bool) {
	col := renderer.ColorFrom(th.Separator)
	contentH := float32(lay.maxLine+1)*lay.lineH + 2*inputTopPad
	if contentH > n.H+1 {
		barH := n.H * n.H / contentH
		if barH < 24 {
			barH = 24
		}
		track := n.H - barH
		barY := n.Y
		if contentH > n.H && track > 0 {
			barY = n.Y + track*(scroll/(contentH-n.H))
		}
		ctx.FillRoundedRect(n.X+n.W-6, barY, 3.5, barH, 2, col)
	}
	if !nowrap {
		return
	}
	contentW := st.contentW
	if st.contentWFor != lay.text {
		contentW = 0
		for line := 0; line <= lay.maxLine; line++ {
			if end := lay.pointAtLineEnd(line).x; end > contentW {
				contentW = end
			}
		}
		st.contentW = contentW
		st.contentWFor = lay.text
	}
	viewW := n.W - 2*inputPad
	if contentW <= viewW+1 || viewW <= 0 {
		return
	}
	barW := viewW * viewW / contentW
	if barW < 24 {
		barW = 24
	}
	track := viewW - barW
	barX := n.X + inputPad
	if track > 0 {
		barX += track * (st.scrollX / (contentW - viewW))
	}
	ctx.FillRoundedRect(barX, n.Y+n.H-6, barW, 3.5, 2, col)
}

// caretInLine 把光标画在字形墨迹上：上沿对齐字顶，下沿停在基线。
//
// 基线按行高的 0.8，和 DirectWrite UNIFORM、web canvas 的落笔一致。
// 字顶大约在行顶下 (lineH-fontSize)*0.7。光标若取整段字号，底会落到基线下面。
func caretInLine(textY float32, line int, lineH, fontSize float32) (cy, caretH float32) {
	top := textY + float32(line)*lineH
	if lineH <= 0 {
		h := fontSize
		if h < minCaretH {
			h = minCaretH
		}
		return top, h
	}
	inset := float32(0)
	if lineH > fontSize && fontSize > 0 {
		inset = (lineH - fontSize) * 0.7
	}
	baseline := lineH * 0.8
	if baseline <= inset {
		baseline = lineH
	}
	caretH = baseline - inset
	// 只在光标短到看不清时才抬到最小高度，并且仍停在基线。
	// 14px 编辑区算出来约 11.6，不能再抬，否则顶会离开字顶。
	if caretH < minCaretH && caretH < fontSize*0.7 {
		if baseline < minCaretH {
			caretH = baseline
			inset = 0
		} else {
			caretH = minCaretH
			inset = baseline - caretH
		}
	}
	return top + inset, caretH
}

// paintSparseSelection 给大文档的可视行画选区。普通布局按 points 逐行画，
// 稀疏布局没有 points，全选会选中但屏幕上没有高亮。
func paintSparseSelection(ctx renderer.Context, n *runtime.VNode, th theme.Theme, st *editState, lay *inputLayout, body string, textStyle renderer.TextStyle, startLine, endLine int, offsetX, scroll float32) {
	from, to, has := st.selRange()
	if !has || lay == nil || lay.lineH <= 0 {
		return
	}
	sel := renderer.ColorFrom(th.Button)
	sel.A = 0.30
	lineH := lay.lineH
	for line := startLine; line < endLine; line++ {
		lineFrom := lay.lineStart(line)
		lineTo := lay.lineEnd(line)
		if lineTo > lineFrom && lineTo <= len(body) && body[lineTo-1] == '\n' {
			lineTo--
		}
		if lineTo < from || lineFrom > to {
			continue
		}
		segFrom := lineFrom
		if from > segFrom {
			segFrom = from
		}
		segTo := lineTo
		if to < segTo {
			segTo = to
		}
		if segFrom >= segTo && lineTo < to {
			continue
		}
		x0, _ := ctx.MeasureText(body[lineFrom:segFrom], 0, textStyle)
		x1, _ := ctx.MeasureText(body[lineFrom:segTo], 0, textStyle)
		if x1 < x0 {
			x1 = x0
		}
		w := x1 - x0
		if w < 2 && lineTo <= to {
			w = 8
		}
		y := n.Y + inputTopPad + float32(line)*lineH - scroll
		ctx.FillRect(n.X+inputPad+x0-offsetX, y, w, lineH, sel)
	}
}

// notifyScroll 滚动偏移真正变了才回调 onScroll(y)。
// 绘制是滚动偏移的必经之路——滚轮、光标跟随、SetTextareaScrollY 最后都要画
// 一帧——所以挂在绘制出口就能覆盖全部滚动来源，调用方不必各自记得通知。
func notifyScroll(n *runtime.VNode, st *editState) {
	if st == nil || st.scrollY == st.notifiedScrollY {
		return
	}
	st.notifiedScrollY = st.scrollY
	if fn := onFloat(n, "onScroll"); fn != nil {
		fn(st.scrollY)
	}
}

func paintLargeEditable(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme, st *editState, lay *inputLayout, body string, textStyle renderer.TextStyle, focused, empty bool, fg renderer.Color, opt editPaint) {
	// 打字 / 移动光标 / 翻页时把光标滚进可视范围——与 paintEditable 同一规则。
	// 大文档走 sparse 落到这里，此前这半边没人做跟随：PageUp/PageDown 光标
	// 翻页了，视图却纹丝不动，看起来就像「编辑区翻页没生效」。
	scroll := st.scrollY
	if opt.multiline && opt.scrollToCaret && st.scrollSync {
		caret := clampCaret(body, st.caretOffset()+compCursorBytes(st))
		caretTop := float32(lay.lineForOffset(caret)) * lay.lineH
		viewH := n.H - 2*inputTopPad
		if viewH > 0 {
			if caretTop-scroll < 0 {
				scroll = caretTop
			}
			if caretTop+lay.lineH-scroll > viewH {
				scroll = caretTop + lay.lineH - viewH
			}
		}
	}
	st.scrollY = scroll
	scroll = clampScrollY(st, n.H)
	lineH := lay.lineH
	if lineH <= 0 {
		lineH = 18
	}
	startLine := int(scroll / lineH)
	if startLine < 0 {
		startLine = 0
	}
	if startLine > lay.maxLine {
		startLine = lay.maxLine
	}
	visibleLines := int((n.H-2*inputTopPad)/lineH) + 2
	if visibleLines < 2 {
		visibleLines = 2
	}
	endLine := startLine + visibleLines
	if endLine > lay.maxLine+1 {
		endLine = lay.maxLine + 1
	}
	start := lay.lineStart(startLine)
	end := len(body)
	if endLine <= lay.maxLine {
		end = lay.lineStart(endLine)
	}
	if start > len(body) {
		start = len(body)
	}
	if end < start || end > len(body) {
		end = len(body)
	}

	offsetX := st.scrollX
	if opt.scrollToCaret && st.scrollSync {
		caret := clampCaret(body, st.caretOffset()+compCursorBytes(st))
		line := lay.lineForOffset(caret)
		startOff := lay.lineStart(line)
		if startOff > caret {
			startOff = caret
		}
		caretX, _ := ctx.MeasureText(body[startOff:caret], 0, textStyle)
		endX, _ := ctx.MeasureText(body[startOff:lay.lineEnd(line)], 0, textStyle)
		offsetX = revealCaretX(st, caretX, endX, n.W-2*inputPad)
		st.scrollX = offsetX
		st.scrollSync = false
	}
	textY := n.Y + inputTopPad + float32(startLine)*lineH - scroll
	ctx.PushClip(n.X+1, n.Y+1, n.W-2, n.H-2)
	drawEditorText(ctx, body[start:end], n.X+inputPad-offsetX, textY, n.W-2*inputPad, lineH, textStyle)
	paintSparseSelection(ctx, n, th, st, lay, body, textStyle, startLine, endLine, offsetX, scroll)

	caret := clampCaret(body, st.caretOffset())
	caretLine := lay.lineForOffset(caret)
	caretStart := lay.lineStart(caretLine)
	if caretStart > caret {
		caretStart = caret
	}
	caretX, _ := ctx.MeasureText(body[caretStart:caret], 0, textStyle)
	cx := n.X + inputPad + caretX - offsetX
	cy, caretH := caretInLine(n.Y+inputTopPad-scroll, caretLine, lineH, textStyle.FontSize)
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
	ctx.PopClip()

	paintEditorBars(ctx, n, th, st, lay, scroll, true)
	notifyScroll(n, st)
}
