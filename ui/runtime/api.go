package runtime

// Component 对应 Vue 3 的 <script setup>：setup 只在挂载时执行一次。
// ctx 用来读 props / slot；不需要时写 _。
func Component(setup func(ctx *SetupContext) func() *VNode) *ComponentDef {
	return DefineComponent("", func(ctx *SetupContext) RenderFunc {
		if setup == nil {
			return func() []*VNode { return nil }
		}
		render := setup(ctx)
		return func() []*VNode {
			if render == nil {
				return nil
			}
			n := render()
			if n == nil {
				return nil
			}
			return []*VNode{n}
		}
	})
}
