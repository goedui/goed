// dialog 浮层的常驻回归。
//
// Dialog 是导出 API，buddy 的「模型用量统计」和 editor 的删除确认框都在用
// （buddy/src/views/home/usage.go 的 usageDialog、editor/src/views/workspace/
// fileops.go 的 confirmDialog），但补这组用例之前它在包内的覆盖率是 **0%** ——
// 一条也没跑过。三个真实调用点各自踩中的正是下面这三条规则。
//
// 三条「注释里写着、代码里做着、谁也没验」的行为：
//
//  1. 键盘事件**始终返回 true**，避免漏到下层（docs/goed-ui说明文档v0.4.0.md）。
//     但 `!e.Down` 那一条早退是例外：**抬起事件返回 false**。所以「始终吞」
//     只对按下成立 —— 这也是「Dialog 里不要放输入框」这个结论的另一半依据：
//     文本输入走 WM_CHAR 不经过这里，而按键本身会被整层吞掉。
//  2. 子节点只有两种排法：**恰好一个 hstack** → 直接当正文；**其余情况**
//     （零个 / 多个 / 单个不是 hstack）→ 一律塞进一个 Justify:End 的按钮条里
//     右对齐。想让正文撑满卡片宽度，必须自己包一层 HStack{Style: Flex:1}。
//  3. `_body` 缓存：layoutDialog 结束时会把 n.Children 换成 [card]，所以**第二次
//     布局必须从缓存里取原始子节点**；没有这层缓存，第二帧就会把 card 当成正文
//     再包一层，弹框越点越深。
//
// 另外 `closeOnEnter` 在写了 `onEnter` 时**永远走不到** —— Enter 分支先看
// onEnter，命中就 return true，后面的 closeOnEnter 是死支。确认框两边都写
// （fileops.go 就是）时靠的是 onEnter 自己把状态清掉。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// dialogCtx 是 layoutDialog / paintDialog 的常用画布。
func dialogCtx() *mockContext { return &mockContext{w: 1000, h: 800} }

// TestDialogSwallowsEveryKeyDown 钉住「按键一律吞掉」这条规则，以及它的唯一例外。
func TestDialogSwallowsEveryKeyDown(t *testing.T) {
	cases := []struct {
		name         string
		key          int
		down         bool
		hasClose     bool
		hasEnter     bool
		closeOnEnter bool
		wantConsumed bool
		wantClose    int
		wantEnter    int
	}{
		{"Esc 调 onClose 并吞掉", runtime.KeyEscape, true, true, false, false, true, 1, 0},
		{"Esc 没写 onClose 也吞掉", runtime.KeyEscape, true, false, false, false, true, 0, 0},
		{"Enter 走 onEnter", runtime.KeyEnter, true, false, true, false, true, 0, 1},
		{"Enter 写了 onEnter 就不看 closeOnEnter", runtime.KeyEnter, true, true, true, true, true, 0, 1},
		{"Enter 只有 closeOnEnter → 关", runtime.KeyEnter, true, true, false, true, true, 1, 0},
		{"Enter 两样都没写 → 吞掉但不动作", runtime.KeyEnter, true, false, false, false, true, 0, 0},
		{"普通键也吞掉", runtime.KeyA, true, false, false, false, true, 0, 0},
		{"抬起事件不吞（唯一例外）", runtime.KeyEscape, false, true, false, false, false, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closes, enters := 0, 0
			var props runtime.Props
			props.CloseOnEnter = tc.closeOnEnter
			if tc.hasClose {
				props.OnClose = func() { closes++ }
			}
			if tc.hasEnter {
				props.OnEnter = func() { enters++ }
			}
			n := Dialog(props)

			got := runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: tc.key, Down: tc.down})
			if got != tc.wantConsumed {
				t.Fatalf("DispatchKey 返回 %v，期望 %v", got, tc.wantConsumed)
			}
			if closes != tc.wantClose {
				t.Fatalf("onClose 调了 %d 次，期望 %d 次", closes, tc.wantClose)
			}
			if enters != tc.wantEnter {
				t.Fatalf("onEnter 调了 %d 次，期望 %d 次", enters, tc.wantEnter)
			}
		})
	}
}

// TestDialogEnterPrefersOnEnter 单独把「closeOnEnter 是死支」钉死：
// 两边都写的时候，Enter 只该走 onEnter，绝不能再顺手调一次 onClose。
func TestDialogEnterPrefersOnEnter(t *testing.T) {
	var order []string
	n := Dialog(runtime.Props{
		OnEnter:      func() { order = append(order, "enter") },
		CloseOnEnter: true,
		OnClose:      func() { order = append(order, "close") },
	})
	if !runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyEnter, Down: true}) {
		t.Fatal("Enter 应该被吞掉")
	}
	if len(order) != 1 || order[0] != "enter" {
		t.Fatalf("调用序列 = %v，期望只有 enter（closeOnEnter 在写了 onEnter 时够不着）", order)
	}
}

// TestDialogMaskClickFollowsDismiss 钉住遮罩点击的三种挂法。
func TestDialogMaskClickFollowsDismiss(t *testing.T) {
	t.Run("默认点遮罩关闭", func(t *testing.T) {
		closes := 0
		n := Dialog(runtime.Props{OnClose: func() { closes++ }})
		if n.OnClick == nil {
			t.Fatal("默认要给遮罩挂 OnClick，否则点遮罩没反应")
		}
		n.OnClick()
		if closes != 1 {
			t.Fatalf("点遮罩调了 %d 次 onClose，期望 1 次", closes)
		}
	})

	t.Run("Modal 时点遮罩不关", func(t *testing.T) {
		closes := 0
		n := Dialog(runtime.Props{OnClose: func() { closes++ }, Modal: true})
		if n.OnClick == nil {
			t.Fatal("Modal 也要挂一个 OnClick 把点击吃掉")
		}
		n.OnClick()
		if closes != 0 {
			t.Fatalf("Modal 点遮罩关了 %d 次，期望 0 次", closes)
		}
	})

	t.Run("自己写了 OnClick 就不覆盖", func(t *testing.T) {
		closes, mine := 0, 0
		n := Dialog(runtime.Props{
			OnClick: func() { mine++ },
			OnClose: func() { closes++ },
		})
		n.OnClick()
		if mine != 1 || closes != 0 {
			t.Fatalf("自有 OnClick 调了 %d 次、onClose %d 次，期望 1 / 0", mine, closes)
		}
	})
}

// TestDialogDismissDefaultsToTrue 是 dialogDismiss 的直测：只有显式 bool 才算数。
func TestDialogDismissDefaultsToTrue(t *testing.T) {
	cases := []struct {
		name string
		n    *runtime.VNode
		want bool
	}{
		{"nil 节点 → 关", nil, true},
		{"Attrs 还没建 → 关", &runtime.VNode{}, true},
		{"没写 dismiss → 关", &runtime.VNode{Attrs: runtime.Attrs{"title": "t"}}, true},
		{"dismiss:true → 关", &runtime.VNode{Attrs: runtime.Attrs{"dismiss": true}}, true},
		{"dismiss:false → 不关", &runtime.VNode{Attrs: runtime.Attrs{"dismiss": false}}, false},
		{"dismiss 类型不对 → 当没写", &runtime.VNode{Attrs: runtime.Attrs{"dismiss": "no"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dialogDismiss(tc.n); got != tc.want {
				t.Fatalf("dialogDismiss = %v，期望 %v", got, tc.want)
			}
		})
	}
}

// TestDialogFillsParentByDefault 守住 Flex 兜底与 Attrs 兜底。
func TestDialogFillsParentByDefault(t *testing.T) {
	if got := Dialog().Style.Flex; got != 1 {
		t.Fatalf("Dialog 默认 Flex = %g，期望 1（不铺满父级遮罩就盖不住整窗）", got)
	}
	if got := Dialog(runtime.Props{Style: runtime.Style{Flex: 2}}).Style.Flex; got != 2 {
		t.Fatalf("显式 Flex 被改成了 %g", got)
	}
	if n := Dialog(); n.Attrs == nil {
		t.Fatal("Dialog 必须自己建 Attrs，否则 onKeyDown 挂不上去、Esc 就没人接")
	}
}

// TestDialogCardWidthClamps 是卡片宽度的夹取阶梯。
func TestDialogCardWidthClamps(t *testing.T) {
	cases := []struct {
		name   string
		width  any
		maxW   float32
		want   float32
		reason string
	}{
		{"没写 width → 默认 400", nil, 1000, 400, "dialogDefaultW"},
		{"float32 正数生效", float32(600), 1000, 600, "Props.ContentWidth 走这条"},
		{"int 正数生效", int(500), 1000, 500, "直接写 Attrs 的路径"},
		{"float32(0) 被忽略", float32(0), 1000, 400, "只有 >0 才算「写了」"},
		{"负数被忽略", float32(-5), 1000, 400, "同上"},
		{"字符串被忽略", "600", 1000, 400, "类型不对就当没写"},
		{"画布 300 → 两边各留 16", nil, 300, 268, "300-32 = 268，仍 >= 240"},
		{"画布 240 → 夹到 208 又抬回 240", nil, 240, 240, "208 < 240，但 maxW 不 < 240，所以取 240"},
		{"画布 200 < 最小宽 → 直接用画布宽", nil, 200, 200, "宁可窄也不能超出画布"},
		{"画布 33 → 夹到 1 又抬到 33", nil, 33, 33, "极窄也照办，不越界"},
		{"画布 32 → 不夹（边界）", nil, 32, 400, "maxW > 32 是严格大于，32 不进夹取"},
		{"画布 0 → 不夹", nil, 0, 400, "同上"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := Dialog()
			if tc.width != nil {
				n.Attrs["width"] = tc.width
			}
			got := dialogCardWidth(n, tc.maxW)
			if got != tc.want {
				t.Fatalf("dialogCardWidth(maxW=%g) = %g，期望 %g（%s）", tc.maxW, got, tc.want, tc.reason)
			}
		})
	}
}

// TestMeasureDialogFallsBackToCanvas 是遮罩尺寸的兜底：父级给了就用，没给就吃画布。
func TestMeasureDialogFallsBackToCanvas(t *testing.T) {
	st := renderer.TextStyle{FontSize: 13}
	th := theme.Light
	ctx := dialogCtx()

	cases := []struct {
		name         string
		maxW, maxH   float32
		ctx          renderer.Context
		wantW, wantH float32
	}{
		{"父级给了尺寸 → 原样返回", 300, 200, ctx, 300, 200},
		{"只给宽 → 高退画布", 300, 0, ctx, 300, 800},
		{"都没给 → 全吃画布", 0, 0, ctx, 1000, 800},
		{"ctx 为 nil → 就是父级给的（0 也认）", 0, 0, nil, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := Dialog(runtime.Props{})
			w, h := measureDialog(tc.ctx, n, tc.maxW, tc.maxH, st, th)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("measureDialog = %gx%g，期望 %gx%g", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

// dialogLayout 跑一次布局并返回那张卡片。
func dialogLayout(t *testing.T, ctx renderer.Context, n *runtime.VNode, x, y, w, h float32) *runtime.VNode {
	t.Helper()
	layoutDialog(ctx, n, x, y, w, h, renderer.TextStyle{FontSize: 13}, theme.Light)
	if len(n.Children) != 1 {
		t.Fatalf("dialog 布局后应该只剩一张卡片，实际 %d 个子节点", len(n.Children))
	}
	card := n.Children[0]
	if card == nil {
		t.Fatal("dialog 的子节点是 nil —— 卡片没挂上（布局收尾那句被删了？）")
	}
	if card.Tag != "vstack" {
		t.Fatalf("dialog 的子节点应该是卡片（vstack），实际 %q", card.Tag)
	}
	return card
}

// TestLayoutDialogChildrenRules 钉住「子节点只有两种排法」这条容易记错的规则。
func TestLayoutDialogChildrenRules(t *testing.T) {
	t.Run("标题与正文各自成行", func(t *testing.T) {
		n := Dialog(runtime.Props{Title: "标题", Message: "正文"})
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if card.Tag != "vstack" {
			t.Fatalf("卡片是 %q，期望 vstack", card.Tag)
		}
		if len(card.Children) != 2 {
			t.Fatalf("卡片有 %d 行，期望 2 行（标题 + 正文）", len(card.Children))
		}
		if got := card.Children[0]; got.Text != "标题" || got.Style.FontSize != 16 || got.Style.Weight != 600 {
			t.Fatalf("标题行 = %q/%g/%d，期望 标题/16/600", got.Text, got.Style.FontSize, got.Style.Weight)
		}
		if got := card.Children[1]; got.Text != "正文" || got.Style.FontSize != 13 {
			t.Fatalf("正文行 = %q/%g，期望 正文/13", got.Text, got.Style.FontSize)
		}
	})

	t.Run("单个 hstack → 原样当正文，不包按钮条", func(t *testing.T) {
		body := HStack(runtime.Props{Style: runtime.Style{Flex: 1, Align: runtime.Stretch}})
		n := Dialog(runtime.Props{}, body)
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 1 {
			t.Fatalf("卡片有 %d 行，期望 1 行", len(card.Children))
		}
		if card.Children[0] != body {
			t.Fatal("恰好一个 hstack 时必须原样放进去（同一个指针），再包一层就吃掉正文的 Flex")
		}
	})

	t.Run("多个子节点 → 包进右对齐按钮条", func(t *testing.T) {
		cancel := Button(runtime.Props{}, "取消")
		del := Button(runtime.Props{}, "删除")
		n := Dialog(runtime.Props{}, cancel, del)
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 1 {
			t.Fatalf("卡片有 %d 行，期望 1 行（合成一条按钮条）", len(card.Children))
		}
		bar := card.Children[0]
		if bar.Tag != "hstack" {
			t.Fatalf("button bar is %q, want hstack", bar.Tag)
		}
		if bar.Style.Justify != runtime.End || bar.Style.Align != runtime.Center || bar.Style.Gap != 8 {
			t.Fatalf("按钮条样式 justify=%q align=%q gap=%g，期望 end/center/8",
				bar.Style.Justify, bar.Style.Align, bar.Style.Gap)
		}
		if len(bar.Children) != 2 || bar.Children[0] != cancel || bar.Children[1] != del {
			t.Fatal("按钮条里应该是原来那两个子节点，顺序不变")
		}
	})

	t.Run("单个非按钮留在正文", func(t *testing.T) {
		body := Box(runtime.Props{})
		n := Dialog(runtime.Props{}, body)
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 1 || card.Children[0] != body {
			t.Fatal("非按钮应留在正文，不包进按钮条")
		}
	})

	t.Run("零个子节点 → 没有按钮条", func(t *testing.T) {
		n := Dialog(runtime.Props{Title: "只有标题"})
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 1 {
			t.Fatalf("卡片有 %d 行，期望 1 行（不能凭空造一条空按钮条）", len(card.Children))
		}
	})

	t.Run("nil 子节点被跳过", func(t *testing.T) {
		// 两个坑：
		//
		//  1. 不能靠 `Dialog(Props{}, nil, body)` 造 nil 子节点 —— runtime.H 的
		//     splitProps 里 `case nil:` 是直接丢掉的，那个 nil 根本进不了 n.Children。
		//     得往 n.Children 里直接塞。
		//  2. 塞成 `[nil, body]` **测不出这道跳过**：不跳过时 user 有两个元素，
		//     `len(user) == 1` 先短路，而后面 `HStack(..., user)` 又会把 nil 丢掉，
		//     结果跟跳过了一样 —— 假缺口形态 (j) 的变体。这道跳过唯一承重的输入是
		//     **全 nil**：此时不跳过就变成 `len(user) == 1 && user[0].Tag`，
		//     当场解引用崩。
		n := Dialog(runtime.Props{})
		n.Children = []*runtime.VNode{nil}
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 0 {
			t.Fatalf("全 nil 的子节点列表应该滤成空（没有正文行），实际 %d 行", len(card.Children))
		}
	})

	t.Run("nil 混在正文里不影响按钮条", func(t *testing.T) {
		// 这条**故意是弱断言**：如上所述，混进 nil 时下游 HStack 也会丢，
		// 所以它抓不住「跳过被删」。留着是为了钉住「混着写也不会炸」这个契约。
		body := Box(runtime.Props{})
		n := Dialog(runtime.Props{}, body)
		n.Children = append([]*runtime.VNode{nil}, n.Children...)
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if len(card.Children) != 1 || card.Children[0] != body {
			t.Fatalf("nil filtered, got %d rows", len(card.Children))
		}
	})
}

// TestLayoutDialogCachesBodyAcrossFrames 是 `_body` 缓存的守门人。
//
// layoutDialog 收尾会把 n.Children 换成 [card]，而它开头是从 n.Children 取正文的。
// 两件事撞在一起，第二帧如果没有缓存就会把上一帧的 card 当成正文再包一层 ——
// 每重建一次帧就深一层。这个 bug 在真实应用里表现为「弹框点着点着变小/变乱」。
func TestLayoutDialogCachesBodyAcrossFrames(t *testing.T) {
	cancel := Button(runtime.Props{}, "取消")
	del := Button(runtime.Props{}, "删除")
	n := Dialog(runtime.Props{Title: "标题", Message: "正文"}, cancel, del)
	ctx := dialogCtx()

	first := dialogLayout(t, ctx, n, 0, 0, 1000, 800)
	rows := len(first.Children)

	// 第二帧：进 layoutDialog 之前 n.Children 已经是 [card] 了。
	second := dialogLayout(t, ctx, n, 0, 0, 1000, 800)
	if second.Tag != "vstack" {
		t.Fatalf("第二帧卡片是 %q，期望 vstack —— 说明上一帧的卡片被当成了正文", second.Tag)
	}
	if len(second.Children) != rows {
		t.Fatalf("第二帧 %d 行、第一帧 %d 行 —— _body 缓存没起作用，卡片套卡片了", len(second.Children), rows)
	}
	if second.Children[0].Text != "标题" {
		t.Fatalf("第二帧第一行是 %q，期望 标题", second.Children[0].Text)
	}
	bar := second.Children[len(second.Children)-1]
	if bar.Tag != "hstack" || len(bar.Children) != 2 {
		t.Fatalf("第二帧按钮条 = %q/%d 个按钮，期望 hstack/2", bar.Tag, len(bar.Children))
	}
	if bar.Children[0] != cancel || bar.Children[1] != del {
		t.Fatal("第二帧按钮条里应该还是原来那两个按钮")
	}
}

// TestLayoutDialogCentersCard 钉住居中与「夹回左上角」两条。
func TestLayoutDialogCentersCard(t *testing.T) {
	st := renderer.TextStyle{FontSize: 13}
	th := theme.Light

	t.Run("画布够大 → 正中", func(t *testing.T) {
		n := Dialog(runtime.Props{Title: "标题"})
		card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
		if card.W != 400 {
			t.Fatalf("卡片宽 = %g，期望 400", card.W)
		}
		if want := (1000 - card.W) / 2; card.X != want {
			t.Fatalf("卡片 X = %g，期望 %g", card.X, want)
		}
		if want := (800 - card.H) / 2; card.Y != want {
			t.Fatalf("卡片 Y = %g，期望 %g", card.Y, want)
		}
	})

	t.Run("画布比卡片还小 → 夹回左上角", func(t *testing.T) {
		small := &mockContext{w: 32, h: 20}
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(small, n, 0, 0, 32, 20, st, th)
		if len(n.Children) != 1 {
			t.Fatalf("卡片数 = %d，期望 1", len(n.Children))
		}
		card := n.Children[0]
		if card.X != 0 || card.Y != 0 {
			t.Fatalf("居中算出负数时要夹回 (0,0)，实际 (%g,%g)", card.X, card.Y)
		}
		if card.W != 400 {
			t.Fatalf("卡片宽 = %g，期望 400（画布 32 不进 maxW-32 夹取）", card.W)
		}
	})

	t.Run("y 不为 0 → 高度收成 y 以下", func(t *testing.T) {
		// 标题栏占掉上面 32 DIP 时，遮罩只能从 32 往下铺，否则底栏会被顶出窗口。
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(dialogCtx(), n, 0, 32, 1000, 800, st, th)
		if n.H != 768 {
			t.Fatalf("dialog 高 = %g，期望 768（800-32）", n.H)
		}
	})

	t.Run("y 触底 → 高度保持父级给的", func(t *testing.T) {
		// bottom == 0 时那道 `bottom > 0` 是**必须**的：松成 >= 0 会把高度写成 0，
		// 整块遮罩连卡片一起消失。
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(dialogCtx(), n, 0, 800, 1000, 800, st, th)
		if n.H != 800 {
			t.Fatalf("dialog 高 = %g，期望 800（bottom=0 不该改写高度）", n.H)
		}
	})

	t.Run("ctx 为 nil → 用父级给的矩形", func(t *testing.T) {
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(nil, n, 5, 6, 300, 200, st, th)
		if n.X != 5 || n.Y != 6 || n.W != 300 || n.H != 200 {
			t.Fatalf("dialog 矩形 = (%g,%g,%g,%g)，期望 (5,6,300,200)", n.X, n.Y, n.W, n.H)
		}
		if len(n.Children) != 1 {
			t.Fatalf("卡片数 = %d，期望 1", len(n.Children))
		}
		card := n.Children[0]
		if card.W != 268 {
			t.Fatalf("卡片宽 = %g，期望 268（300-32）", card.W)
		}
		if card.X != 5+16 {
			t.Fatalf("卡片 X = %g，期望 21（父级矩形居中）", card.X)
		}
	})
}

// TestLayoutDialogDegenerateInputs 守住两条兜底分支。
//
// 注意 `if cw < cardW { cw = cardW }`（dialog.go:195）**是不可达的死支**，故意
// 不追：`MeasureNode` 走 `applySize`，而 `card.Style.Width` 就是刚算出来的
// `cardW`，`applySize` 里 `if n.Style.Width > 0 { w = n.Style.Width }` 会把它
// 钉成 cardW，所以 `cw` 恒等于 `cardW`。留着它只在「将来有人不再用 Style.Width
// 表达卡片宽度」时才有意义。
func TestLayoutDialogDegenerateInputs(t *testing.T) {
	st := renderer.TextStyle{FontSize: 13}
	th := theme.Light

	t.Run("父级没给宽 → 吃画布宽", func(t *testing.T) {
		ctx := dialogCtx()
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(ctx, n, 0, 0, 0, 800, st, th)
		if n.W != 1000 {
			t.Fatalf("dialog 宽 = %g，期望 1000（w<=0 时退画布宽）", n.W)
		}
		if len(n.Children) != 1 {
			t.Fatalf("卡片数 = %d，期望 1", len(n.Children))
		}
	})

	t.Run("手搓的节点没有 Attrs 也能布局", func(t *testing.T) {
		n := &runtime.VNode{Kind: runtime.KindElement, Tag: "dialog"}
		if n.Attrs != nil {
			t.Fatal("这个用例的前提就是 Attrs 为 nil")
		}
		layoutDialog(dialogCtx(), n, 0, 0, 1000, 800, st, th)
		if n.Attrs == nil {
			t.Fatal("layoutDialog 必须自己补 Attrs，否则下一行写 _body 就 panic")
		}
		if _, ok := n.Attrs["_body"]; !ok {
			t.Fatal("_body 没建起来，下一帧会把卡片当成正文")
		}
		if len(n.Children) != 1 {
			t.Fatalf("卡片数 = %d，期望 1", len(n.Children))
		}
	})
}

// TestLayoutDialogCardStyle 守住卡片的几何与配色来源。
func TestLayoutDialogCardStyle(t *testing.T) {
	th := theme.Light
	n := Dialog(runtime.Props{ID: "confirm", ContentWidth: 320})
	card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)

	if card.Style.Width != 320 {
		t.Fatalf("卡片宽 = %g，期望 320（ContentWidth 生效）", card.Style.Width)
	}
	if card.Style.Padding != dialogPad || card.Style.Gap != dialogGap || card.Style.Radius != dialogRadius {
		t.Fatalf("卡片内边距/间距/圆角 = %g/%g/%g，期望 %g/%g/%g",
			card.Style.Padding, card.Style.Gap, card.Style.Radius, dialogPad, dialogGap, dialogRadius)
	}
	if card.Style.Background != dialogCardBG(th) {
		t.Fatalf("卡片底色 = %+v，期望 %+v", card.Style.Background, dialogCardBG(th))
	}
	if card.Style.Border != dialogCardBorder(th) {
		t.Fatalf("卡片描边 = %+v，期望 %+v", card.Style.Border, dialogCardBorder(th))
	}
	if card.ID() != "confirm-card" {
		t.Fatalf("卡片 id = %q，期望 confirm-card", card.ID())
	}
	if card.OnClick == nil {
		t.Fatal("卡片必须自己挂一个 OnClick，否则点卡片会穿透到遮罩把弹框关掉")
	}

	// 没写 id 时卡片也不该凭空有 id（否则同屏两个弹框会撞 id）。
	plain := Dialog(runtime.Props{})
	if got := dialogLayout(t, dialogCtx(), plain, 0, 0, 1000, 800).ID(); got != "" {
		t.Fatalf("无 id 的弹框，卡片 id = %q，期望空", got)
	}
}

// TestDialogCardClickDoesNotClose 是上一条「卡片吃掉点击」的行为面：
// 调卡片的 OnClick 不能把弹框关掉。
func TestDialogCardClickDoesNotClose(t *testing.T) {
	closes := 0
	n := Dialog(runtime.Props{OnClose: func() { closes++ }})
	card := dialogLayout(t, dialogCtx(), n, 0, 0, 1000, 800)
	if card.OnClick == nil {
		t.Fatal("卡片没挂 OnClick，点卡片会穿透到遮罩")
	}
	card.OnClick()
	if closes != 0 {
		t.Fatalf("点卡片关了 %d 次，期望 0 次", closes)
	}
}

// TestPaintDialogMaskAndChildren 钉住遮罩的颜色与覆盖范围。
func TestPaintDialogMaskAndChildren(t *testing.T) {
	st := renderer.TextStyle{FontSize: 13}
	th := theme.Light

	t.Run("默认遮罩是半透明黑且铺满节点", func(t *testing.T) {
		ctx := dialogCtx()
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(ctx, n, 0, 0, 1000, 800, st, th)
		paintDialog(ctx, n, st, th)

		if len(ctx.fills) == 0 {
			t.Fatal("遮罩没画")
		}
		got := ctx.fills[0]
		if got.x != 0 || got.y != 0 || got.w != 1000 || got.h != 800 {
			t.Fatalf("遮罩矩形 = (%g,%g,%g,%g)，期望 (0,0,1000,800)", got.x, got.y, got.w, got.h)
		}
		if want := renderer.RGBA(0, 0, 0, 72); got.c != want {
			t.Fatalf("遮罩色 = %+v，期望半透明黑 %+v", got.c, want)
		}
	})

	t.Run("Style.Background 覆盖遮罩色", func(t *testing.T) {
		ctx := dialogCtx()
		tint := runtime.RGBA(10, 20, 30, 200)
		n := Dialog(runtime.Props{Style: runtime.Style{Background: tint}})
		layoutDialog(ctx, n, 0, 0, 1000, 800, st, th)
		paintDialog(ctx, n, st, th)
		if len(ctx.fills) == 0 {
			t.Fatal("遮罩没画")
		}
		if want := renderer.ColorFrom(tint); ctx.fills[0].c != want {
			t.Fatalf("遮罩色 = %+v，期望 %+v", ctx.fills[0].c, want)
		}
	})

	t.Run("遮罩之下会把卡片画出来", func(t *testing.T) {
		ctx := dialogCtx()
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(ctx, n, 0, 0, 1000, 800, st, th)
		paintDialog(ctx, n, st, th)

		if len(ctx.rounds) == 0 {
			t.Fatal("卡片的圆角底没画")
		}
		if ctx.rounds[0].r != dialogRadius {
			t.Fatalf("卡片圆角 = %g，期望 %g", ctx.rounds[0].r, dialogRadius)
		}
		if len(ctx.texts) == 0 || ctx.texts[0] != "标题" {
			t.Fatalf("画出来的文字 = %v，期望第一行是 标题", ctx.texts)
		}
	})

	t.Run("遮罩跟着节点矩形走（不是从 0,0 起）", func(t *testing.T) {
		ctx := dialogCtx()
		n := Dialog(runtime.Props{Title: "标题"})
		layoutDialog(ctx, n, 5, 7, 1000, 800, st, th)
		paintDialog(ctx, n, st, th)
		if len(ctx.fills) == 0 {
			t.Fatal("遮罩没画")
		}
		got := ctx.fills[0]
		if got.x != 5 || got.y != 7 || got.w != 1000 || got.h != 793 {
			t.Fatalf("遮罩矩形 = (%g,%g,%g,%g)，期望 (5,7,1000,793)（高收到画布底 800-7）", got.x, got.y, got.w, got.h)
		}
	})

	t.Run("ctx 或 n 为 nil 时什么都不画", func(t *testing.T) {
		ctx := &mockContext{w: 10, h: 10}
		paintDialog(nil, Dialog(runtime.Props{}), st, th)
		paintDialog(ctx, nil, st, th)
		if ctx.rects != 0 {
			t.Fatalf("nil 入参不该画东西，实际 FillRect/FillRoundedRect 共 %d 次", ctx.rects)
		}
	})
}

// TestDialogPaletteFallbacks 是四张调色函数的兜底阶梯。
//
// 明暗两套的分工不一样：标题色只看 Foreground、副标题只看 Dark 开关、
// 卡片底暗色走 Titlebar / 亮色走 Background、描边只看 Separator。写成
// 「都看 Dark」或「都看 Foreground」在真实主题上都还能出图，所以只能靠
// 零值主题（A=0）把「有兜底」和「没兜底」分开。
func TestDialogPaletteFallbacks(t *testing.T) {
	t.Run("标题色", func(t *testing.T) {
		if got := dialogTitleColor(theme.Light); got != theme.Light.Foreground {
			t.Fatalf("亮色标题 = %+v，期望 %+v", got, theme.Light.Foreground)
		}
		if got := dialogTitleColor(theme.Dark); got != theme.Dark.Foreground {
			t.Fatalf("暗色标题 = %+v，期望 %+v", got, theme.Dark.Foreground)
		}
		if got, want := dialogTitleColor(theme.Theme{}), runtime.RGB(20, 20, 20); got != want {
			t.Fatalf("零值主题标题 = %+v，期望 %+v", got, want)
		}
	})

	t.Run("副标题色", func(t *testing.T) {
		if got, want := dialogMuted(theme.Dark), runtime.RGB(180, 184, 190); got != want {
			t.Fatalf("暗色副标题 = %+v，期望 %+v", got, want)
		}
		if got, want := dialogMuted(theme.Light), runtime.RGB(96, 100, 108); got != want {
			t.Fatalf("亮色副标题 = %+v，期望 %+v", got, want)
		}
		if got, want := dialogMuted(theme.Theme{Dark: true}), runtime.RGB(180, 184, 190); got != want {
			t.Fatalf("只看 Dark 开关：零值主题 + Dark = %+v，期望 %+v", got, want)
		}
	})

	t.Run("卡片底色", func(t *testing.T) {
		if got := dialogCardBG(theme.Dark); got != theme.Dark.Titlebar {
			t.Fatalf("暗色卡片底 = %+v，期望 Titlebar %+v", got, theme.Dark.Titlebar)
		}
		if got := dialogCardBG(theme.Light); got != theme.Light.Background {
			t.Fatalf("亮色卡片底 = %+v，期望 Background %+v", got, theme.Light.Background)
		}
		// 上面那条 Light 断言**抓不住**「亮色分支不读主题 Background」：内置亮色主题的
		// Background 恰好就是兜底色 #FFFFFF，两条路出同一个值。要分开只能换一个
		// 「底色不是白」的主题。
		custom := theme.Theme{Background: runtime.RGB(200, 210, 220)}
		if got := dialogCardBG(custom); got != custom.Background {
			t.Fatalf("自定义亮色卡片底 = %+v，期望读主题 %+v（不是兜底的 255,255,255）", got, custom.Background)
		}
		if got, want := dialogCardBG(theme.Theme{Dark: true}), runtime.RGB(32, 32, 36); got != want {
			t.Fatalf("暗色兜底 = %+v，期望 %+v", got, want)
		}
		if got, want := dialogCardBG(theme.Theme{}), runtime.RGB(255, 255, 255); got != want {
			t.Fatalf("亮色兜底 = %+v，期望 %+v", got, want)
		}
	})

	t.Run("卡片描边", func(t *testing.T) {
		if got := dialogCardBorder(theme.Light); got != theme.Light.Separator {
			t.Fatalf("亮色描边 = %+v，期望 %+v", got, theme.Light.Separator)
		}
		if got := dialogCardBorder(theme.Dark); got != theme.Dark.Separator {
			t.Fatalf("暗色描边 = %+v，期望 %+v", got, theme.Dark.Separator)
		}
		if got, want := dialogCardBorder(theme.Theme{}), runtime.RGB(210, 210, 210); got != want {
			t.Fatalf("零值主题描边 = %+v，期望 %+v", got, want)
		}
	})
}

func TestDialogWheelDoesNotReachContentBelow(t *testing.T) {
	ResetScroll("under")
	t.Cleanup(func() { ResetScroll("under") })
	rows := make([]*runtime.VNode, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, runtime.Text(runtime.Props{Style: runtime.Style{Height: 24}}, "行"))
	}
	under := Scroll(runtime.Props{
		ID: "under", Style: runtime.Style{Width: 400, Height: 120},
	}, VStack(runtime.Props{Style: runtime.Style{Gap: 2}}, rows))
	dlg := Dialog(runtime.Props{ID: "dlg", Title: "快捷键"})
	root := runtime.H("layer", runtime.Props{Style: runtime.Style{Width: 400, Height: 300}}, under, dlg)
	ctx := dialogCtx()
	renderer.PaintTree(ctx, []*runtime.VNode{root}, theme.Light)
	HandleWheel([]*runtime.VNode{root}, 40, 40, -3)
	if got := ScrollOffset("under"); got != 0 {
		t.Fatalf("对话框打开时滚轮不应滚到下层：offset=%.1f", got)
	}
}
