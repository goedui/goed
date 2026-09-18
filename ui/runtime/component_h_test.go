package runtime

import "testing"

func TestHAcceptsComponent(t *testing.T) {
	child := Component(func(_ *SetupContext) func() *VNode {
		return func() *VNode { return Text("inner") }
	})
	n := H(child)
	if n.Kind != KindComponent || n.Component != child {
		t.Fatalf("H(组件) 应产出 KindComponent: %+v", n)
	}
	el := H("vstack", Text("x"))
	if el.Kind != KindElement || el.Tag != "vstack" {
		t.Fatalf("H(标签) 应产出 KindElement: %+v", el)
	}
}

func TestHCustomComponentReusesInstanceAndProps(t *testing.T) {
	setups := 0
	var label *Ref[string]
	child := Component(func(ctx *SetupContext) func() *VNode {
		setups++
		return func() *VNode {
			return Text(ctx.String("label"))
		}
	})
	parent := DefineComponent("parent", func(_ *SetupContext) RenderFunc {
		label = NewRef("a")
		return func() []*VNode {
			return []*VNode{H(child, Props{Label: label.Get()})}
		}
	})
	inst := Mount(parent, nil)
	got := inst.Render()
	if setups != 1 {
		t.Fatalf("setup 应只跑 1 次，得到 %d", setups)
	}
	if len(got) != 1 || got[0].Kind != KindComponent {
		t.Fatalf("期望组件节点: %+v", got)
	}
	if len(got[0].Children) != 1 || got[0].Children[0].Text != "a" {
		t.Fatalf("props 未传入: %+v", got[0].Children)
	}
	label.Set("b")
	got = inst.Render()
	if setups != 1 {
		t.Fatalf("第二次渲染不应再跑 setup，得到 %d", setups)
	}
	if got[0].Children[0].Text != "b" {
		t.Fatalf("更新后的 props 未生效: %q", got[0].Children[0].Text)
	}
}

func TestHCustomComponentSlot(t *testing.T) {
	box := Component(func(ctx *SetupContext) func() *VNode {
		return func() *VNode {
			return H("vstack", ctx.Children())
		}
	})
	parent := Component(func(_ *SetupContext) func() *VNode {
		return func() *VNode {
			return H(box, Text("slot"))
		}
	})
	got := Mount(parent, nil).Render()
	if len(got) != 1 || got[0].Kind != KindComponent {
		t.Fatalf("期望组件节点: %+v", got)
	}
	inner := got[0].Children
	if len(inner) != 1 || inner[0].Tag != "vstack" {
		t.Fatalf("slot 未包进子树: %+v", inner)
	}
	if len(inner[0].Children) != 1 || inner[0].Children[0].Text != "slot" {
		t.Fatalf("默认 slot 未传入: %+v", inner[0].Children)
	}
}

func TestHCustomComponentFallthroughStyle(t *testing.T) {
	inner := Component(func(_ *SetupContext) func() *VNode {
		return func() *VNode {
			return H("vstack", Props{Style: Style{Gap: 8}}, Text("x"))
		}
	})
	parent := Component(func(_ *SetupContext) func() *VNode {
		return func() *VNode {
			return H(inner, Props{Style: Style{Flex: 1, Padding: 32}})
		}
	})
	got := Mount(parent, nil).Render()
	root := got[0].Children[0]
	if root.Style.Flex != 1 || root.Style.Padding != 32 {
		t.Fatalf("父级 style 应落到单根: %+v", root.Style)
	}
	if root.Style.Gap != 8 {
		t.Fatalf("子组件自己的 style 不应被盖掉: %+v", root.Style)
	}
}
