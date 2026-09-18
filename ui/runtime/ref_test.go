package runtime

import "testing"

func TestRefInvalidatesAndRenders(t *testing.T) {
	var n int
	var r *Ref[int]
	def := DefineComponent("counter", func(_ *SetupContext) RenderFunc {
		r = NewRef(0)
		return func() []*VNode {
			return []*VNode{Text("你点击了 ", r, " 次")}
		}
	})
	inst := Mount(def, func() { n++ })
	got := inst.Render()
	if len(got) != 1 || got[0].Text != "你点击了 0 次" {
		t.Fatalf("初次渲染: %+v", got)
	}
	r.Set(r.Get() + 1)
	if n != 1 {
		t.Fatalf("Set 应触发 Invalidate，得到 %d", n)
	}
	got = inst.Render()
	if got[0].Text != "你点击了 1 次" {
		t.Fatalf("更新后渲染: %q", got[0].Text)
	}
}

func TestComponentReturnsSingleRoot(t *testing.T) {
	def := Component(func(_ *SetupContext) func() *VNode {
		return func() *VNode {
			return H("vstack", Text("Hello"), H("button"))
		}
	})
	nodes := Mount(def, nil).Render()
	if len(nodes) != 1 || nodes[0].Tag != "vstack" {
		t.Fatalf("期望单根 vstack，得到 %+v", nodes)
	}
	if len(nodes[0].Children) != 2 {
		t.Fatalf("子节点: %+v", nodes[0].Children)
	}
	if nodes[0].Children[1].Tag != "button" {
		t.Fatal("第二个子节点应是 button")
	}
}
