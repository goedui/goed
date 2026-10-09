package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func init() {
	renderer.Register("ctxmenu", renderer.Widget{
		Measure: measureContextMenu,
		Layout:  layoutContextMenu,
		Paint:   paintContextMenu,
	})
}

// ContextMenu 右键面板：铺满父级的无色遮罩 + 一张按指针坐标摆的菜单卡片。
//
// 遮罩为什么要铺满：组件自己收不到「外面」的点击，不铺满就会穿到底下 —— 菜单还
// 开着，光标已经跳走了。铺满后它是浮层（runtime.OverlayTag），命中顺序是
// 菜单项 → 卡片 → 遮罩：分隔线那几像素的空档由卡片自己吃掉（否则会当成「点空白」
// 把菜单关掉），其余落空的地方由遮罩接住并关闭。代价是菜单开着时底下点不到，
// 这是上下文菜单的通行行为。
//
// 卡片复用 Popover：排版、「标签\t快捷键」的变宽空格对齐、靠边夹取与上下翻转都在
// 里面；这里只补遮罩，以及把卡片摆到指针处（右键没有 anchor，只有坐标）。
//
// Attrs:
//
//	atX / atY  指针位置（DIP），菜单从这儿长出来
//	items      []MenuItem；Sep 画分隔线，Action 是点击回调
//	onClose    关闭回调：点空白、选中菜单项、Esc 都会走它
//	onSelect   func(string)，在选中项的 Action **之后**调用（可省）。收到的是整条
//	           标签（含前缀与 "\t快捷键"）；要按语义分支就用 MenuItem.Action
//
// 约定：不吃子节点（菜单内容就是 items）；items 为空返回 nil（layer / 命中 / 绘制
// 都认「nil = 这一项不存在」）；同名菜单项共用最后一个回调（label 是唯一的键，
// 与 MenuBar 同一套）。
func ContextMenu(parts ...any) *runtime.VNode {
	n := runtime.H("ctxmenu", parts...)
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	items := menuItemsFrom(n.Attrs["items"])
	if len(items) == 0 {
		return nil
	}
	if n.Style.Flex == 0 {
		n.Style.Flex = 1
	}
	// 遮罩：整块画布都归它，点哪儿都算「点在面板外面」。
	if n.OnClick == nil {
		n.OnClick = func() { closeContextMenu(n) }
	}
	// Esc 关掉。面板是 OverlayTag，但 DispatchKey 只特判 dialog / popover，
	// 所以这里挂自己的 onKeyDown（runtime.DispatchKey 里有 ctxmenu 那一支）。
	n.Attrs["onKeyDown"] = runtime.KeyHandler(func(e runtime.KeyEvent) bool {
		if e.Key == runtime.KeyEscape {
			closeContextMenu(n)
			return true
		}
		return false
	})

	labels := make([]string, 0, len(items))
	actions := map[string]func(){}
	for _, it := range items {
		if it.Sep {
			labels = append(labels, "---")
			continue
		}
		l := menuItemLabel(it)
		labels = append(labels, l)
		if it.Action != nil {
			actions[l] = it.Action
		}
	}
	card := Popover(runtime.Props{
		Items:   labels,
		OnClose: func() { closeContextMenu(n) },
		OnSelect: func(v string) {
			// 先关再执行（和 MenuBar 一致）：菜单项的动作可能弹对话框、可能换
			// 焦点，先把面板收掉，界面不会出现「对话框压在菜单上」。
			closeContextMenu(n)
			if a := actions[v]; a != nil {
				a()
			}
			if fn := onString(n, "onSelect"); fn != nil {
				fn(v)
			}
		},
	})
	// 卡片自己吃掉落在它身上的点击：分隔线那几像素的空档不属于任何菜单项，
	// 不接住就会穿到遮罩上，变成「点分隔线把菜单关了」。
	card.OnClick = func() {}
	n.Children = []*runtime.VNode{card}
	return n
}

// closeContextMenu 走一遍面板的关闭回调。宿主没接 onClose 时是空操作。
func closeContextMenu(n *runtime.VNode) {
	if fn := onVoid(n, "onClose"); fn != nil {
		fn()
	}
}

func measureContextMenu(ctx renderer.Context, n *runtime.VNode, maxW, maxH float32, style renderer.TextStyle, th theme.Theme) (float32, float32) {
	// 遮罩铺满父级：给多大就多大。父级没给尺寸（maxW/maxH 为 0）时退到画布，
	// 免得量成 0 高 —— 0 尺寸的节点在命中测试里会被直接跳过，遮罩就白铺了。
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

func layoutContextMenu(ctx renderer.Context, n *runtime.VNode, x, y, w, h float32, style renderer.TextStyle, th theme.Theme) {
	// 和 layoutLayer 同一套算法：宽度按画布、高度到画布底边为止。
	if ctx != nil {
		if w <= 0 {
			w = ctx.Width()
		}
		if bottom := ctx.Height() - y; bottom > 0 {
			h = bottom
		}
	}
	n.X, n.Y, n.W, n.H = x, y, w, h

	var card *runtime.VNode
	for _, c := range n.Children {
		if c != nil && c.Tag == "popover" {
			card = c
			break
		}
	}
	if card == nil {
		return
	}
	// 卡片的上限是画布，不是面板自己那一块。
	//
	// 面板的盒子就是画布，这里看起来是同一件事，但规矩得写明白：layoutPopover
	// 靠 ctx.Width()/Height() 做靠边夹取与上下翻转，喂给它面板尺寸就等于把
	// 「画布」这个上限换成了一个可能已经被夹过的值 —— 菜单栏那边踩过一模一样的
	// 坑（见 layoutMenuBar 的注释）。
	maxW, maxH := w, h
	if ctx != nil {
		maxW, maxH = ctx.Width(), ctx.Height()
	}
	cw, ch := renderer.MeasureNode(ctx, card, maxW, maxH, style, th)
	// 摆到指针处。layoutPopover 在 anchor 为空时把父级给的盒当锚点，但它会在
	// 锚点下方留 4 DIP 的「下拉缝」（菜单栏的下拉要离菜单项几像素）—— 右键面板
	// 上这条缝正好让指针不压在第一个菜单项上，留着。
	renderer.LayoutNode(ctx, card, attrFloat(n, "atX"), attrFloat(n, "atY"), cw, ch, style, th)
}

func paintContextMenu(ctx renderer.Context, n *runtime.VNode, style renderer.TextStyle, th theme.Theme) {
	// 遮罩是无色的：什么都不画才对。
	//
	// 卡片（子节点里那个 popover）自己也是浮层，paintOverlays 会顺着树走到它、
	// 按正确顺序画一遍；这里再画就成了两遍。
	_ = ctx
	_ = n
	_ = style
	_ = th
}

// attrFloat 读一个浮点 Attrs。缺省 0 —— 面板坐标的默认值就是原点，和「没写」同义。
func attrFloat(n *runtime.VNode, key string) float32 {
	switch v := renderer.Attr(n, key).(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int:
		return float32(v)
	default:
		return 0
	}
}
