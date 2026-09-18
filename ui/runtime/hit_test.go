package runtime

import "testing"

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
