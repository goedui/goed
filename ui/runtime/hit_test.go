package runtime

import (
	"strings"
	"testing"
)

func TestHitTestTopmostOnClick(t *testing.T) {
	var hit string
	a := H("button", Props{OnClick: func() { hit = "a" }})
	a.X, a.Y, a.W, a.H = 0, 0, 100, 40
	b := H("button", Props{OnClick: func() { hit = "b" }})
	b.X, b.Y, b.W, b.H = 20, 10, 40, 20
	hits := CollectHits([]*VNode{a, b})
	fn := HitTest(hits, 30, 15)
	if fn == nil {
		t.Fatal("应命中上层按钮")
	}
	fn()
	if hit != "b" {
		t.Fatalf("期望 b，得到 %q", hit)
	}
	if HitTest(hits, 200, 200) != nil {
		t.Fatal("空白处不应命中")
	}
}

func TestHitNodePrefersTopmostPointerDown(t *testing.T) {
	a := H("input", Props{OnPointerDown: func() {}})
	a.X, a.Y, a.W, a.H = 0, 0, 100, 40
	b := H("input", Props{OnPointerDown: func() {}})
	b.X, b.Y, b.W, b.H = 20, 10, 40, 20
	if got := HitNode([]*VNode{a, b}, 30, 15); got != b {
		t.Fatal("应命中上层带 onPointerDown 的节点")
	}
	if HitNode([]*VNode{a, b}, 200, 200) != nil {
		t.Fatal("空白处不应命中")
	}
	btn := H("button", Props{OnClick: func() {}})
	btn.X, btn.Y, btn.W, btn.H = 20, 10, 40, 20
	if got := HitNode([]*VNode{a, btn}, 30, 15); got != btn {
		t.Fatal("上层按钮应盖住下层输入框")
	}
}

func TestDispatchHoverEnterLeaveOnce(t *testing.T) {
	var log []string
	hits := []Hit{{
		X: 10, Y: 10, W: 20, H: 20,
		OnEnter: func() { log = append(log, "enter") },
		OnLeave: func() { log = append(log, "leave") },
	}}
	DispatchHover(hits, 0, 0, 12, 12)
	DispatchHover(hits, 12, 12, 15, 15)
	DispatchHover(hits, 15, 15, 0, 0)
	if strings.Join(log, ",") != "enter,leave" {
		t.Fatalf("hover %v", log)
	}
}

func TestCollectHitsAbsoluteDoesNotRecurse(t *testing.T) {
	var hit string
	abs := H("button", Props{
		OnClick: func() { hit = "abs" },
		Style:   Style{Position: PositionAbsolute, Width: 16, Height: 16},
	})
	abs.X, abs.Y, abs.W, abs.H = 80, 4, 16, 16
	box := H("box", Props{Style: Style{Position: PositionRelative, Width: 100, Height: 40}}, abs)
	box.X, box.Y, box.W, box.H = 0, 0, 100, 40
	hits := CollectHits([]*VNode{box})
	fn := HitTest(hits, 84, 8)
	if fn == nil {
		t.Fatal("应命中绝对定位按钮")
	}
	fn()
	if hit != "abs" {
		t.Fatalf("期望 abs，得到 %q", hit)
	}
	if got := HitNode([]*VNode{box}, 84, 8); got != abs {
		t.Fatal("HitNode 应命中绝对定位按钮")
	}
}

func TestHoverChangedOnlyWhenTargetMoves(t *testing.T) {
	a := H("button", Props{OnClick: func() {}})
	a.X, a.Y, a.W, a.H = 0, 0, 40, 20
	b := H("button", Props{OnClick: func() {}})
	b.X, b.Y, b.W, b.H = 50, 0, 40, 20
	hits := CollectHits([]*VNode{a, b})
	if HoverChanged(hits, 10, 10, 20, 10) {
		t.Fatal("同一按钮内移动不应触发 hover 变化")
	}
	if !HoverChanged(hits, 10, 10, 60, 10) {
		t.Fatal("跨按钮移动应触发 hover 变化")
	}
	if !HoverChanged(hits, 10, 10, 200, 200) {
		t.Fatal("离开可点击区域应触发 hover 变化")
	}
	if HoverChanged(hits, 200, 200, 300, 300) {
		t.Fatal("空白处移动不应触发 hover 变化")
	}
}

// TestCollectHitsAbsoluteNoRecurse 防止书签栏崩溃回归：
// 绝对定位节点被延后处理后，不得再次进入延后逻辑，否则无限递归爆栈。
func TestCollectHitsAbsoluteNoRecurse(t *testing.T) {
	inner := H("button", Props{OnClick: func() {}})
	inner.X, inner.Y, inner.W, inner.H = 10, 10, 20, 20
	ov := H("div", Props{OnClick: func() {}})
	ov.Style.Position = PositionAbsolute
	ov.X, ov.Y, ov.W, ov.H = 0, 0, 100, 100
	ov.Children = []*VNode{inner}
	root := H("div")
	root.X, root.Y, root.W, root.H = 0, 0, 200, 200
	root.Children = []*VNode{ov}

	hits := CollectHits([]*VNode{root})
	if len(hits) != 2 {
		t.Fatalf("期望 2 个 hit（外层+内层），得到 %d", len(hits))
	}
	if HitTest(hits, 15, 15) == nil {
		t.Fatal("内层按钮应可命中")
	}
	if got := HitNode([]*VNode{root}, 15, 15); got != inner {
		t.Fatal("最内层节点应胜出")
	}
}
