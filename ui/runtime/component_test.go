package runtime

import "testing"

func TestDefineComponentRendersText(t *testing.T) {
	def := DefineComponent("HelloWorld", func(_ *SetupContext) RenderFunc {
		return func() []*VNode {
			return []*VNode{Text("HelloWorld")}
		}
	})
	inst := Mount(def, nil)
	nodes := inst.Render()
	if len(nodes) != 1 {
		t.Fatalf("期望 1 个节点，得到 %d", len(nodes))
	}
	if nodes[0].Kind != KindText || nodes[0].Text != "HelloWorld" {
		t.Fatalf("节点不符合预期: %+v", nodes[0])
	}
}
