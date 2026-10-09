package runtime

import "testing"

// TestOnMountedRunsAfterFirstRender：onMounted 的时机是「首次渲染之后」，
// setup 阶段不能触发，第二次渲染也不能再触发一次。
func TestOnMountedRunsAfterFirstRender(t *testing.T) {
	mounted := 0
	rendered := 0
	c := DefineComponent("c", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { mounted++ })
		return func() []*VNode {
			rendered++
			return []*VNode{Text("x")}
		}
	})
	inst := Mount(c, nil)
	if mounted != 0 {
		t.Fatalf("setup 阶段不该触发 onMounted，得到 %d", mounted)
	}
	inst.Render()
	if rendered != 1 || mounted != 1 {
		t.Fatalf("首次渲染后应触发 1 次，rendered=%d mounted=%d", rendered, mounted)
	}
	inst.Render()
	if mounted != 1 {
		t.Fatalf("onMounted 只该触发一次，得到 %d", mounted)
	}
}

// TestOnMountedChildBeforeParent：子组件先挂载，顺序与 Vue 一致。
func TestOnMountedChildBeforeParent(t *testing.T) {
	var order []string
	child := DefineComponent("child", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { order = append(order, "child") })
		return func() []*VNode { return []*VNode{Text("c")} }
	})
	parent := DefineComponent("parent", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { order = append(order, "parent") })
		return func() []*VNode { return []*VNode{H(child)} }
	})
	Mount(parent, nil).Render()
	if len(order) != 2 || order[0] != "child" || order[1] != "parent" {
		t.Fatalf("期望子先父后，得到 %v", order)
	}
}

// TestOnUnmountedWhenChildDisappears：条件渲染消失的子实例触发 onUnmounted，
// 重新出现时是新实例，onMounted 会再跑一次。
func TestOnUnmountedWhenChildDisappears(t *testing.T) {
	var mounted, unmounted int
	show := NewRef(true)
	child := DefineComponent("child", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { mounted++ })
		ctx.OnUnmounted(func() { unmounted++ })
		return func() []*VNode { return []*VNode{Text("c")} }
	})
	parent := DefineComponent("parent", func(_ *SetupContext) RenderFunc {
		return func() []*VNode {
			if !show.Get() {
				return []*VNode{Text("empty")}
			}
			return []*VNode{H(child)}
		}
	})
	inst := Mount(parent, nil)
	inst.Render()
	if mounted != 1 {
		t.Fatalf("子组件挂载后应跑一次 onMounted，得到 %d", mounted)
	}
	show.Set(false)
	inst.Render()
	if unmounted != 1 {
		t.Fatalf("子组件消失后应跑一次 onUnmounted，得到 %d", unmounted)
	}
	show.Set(true)
	inst.Render()
	if mounted != 2 {
		t.Fatalf("重新出现应重新挂载（onMounted 二次），得到 %d", mounted)
	}
}

// TestUnmountBeforeFirstRenderSkipsOnMounted：卸载过的实例不该再跑 onMounted。
func TestUnmountBeforeFirstRenderSkipsOnMounted(t *testing.T) {
	mounted, unmounted := 0, 0
	c := DefineComponent("c", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { mounted++ })
		ctx.OnUnmounted(func() { unmounted++ })
		return func() []*VNode { return []*VNode{Text("x")} }
	})
	inst := Mount(c, nil)
	inst.Unmount()
	inst.Render()
	if mounted != 0 {
		t.Fatalf("卸载后不该触发 onMounted，得到 %d", mounted)
	}
	if unmounted != 1 {
		t.Fatalf("onUnmounted 应触发 1 次，得到 %d", unmounted)
	}
}

// TestOnMountedRegisteredAfterMountRunsImmediately：挂载之后再注册立即执行，
// 不会堆到下一帧、也不会重复执行。
func TestOnMountedRegisteredAfterMountRunsImmediately(t *testing.T) {
	called := 0
	var ctxRef *SetupContext
	c := DefineComponent("c", func(ctx *SetupContext) RenderFunc {
		ctxRef = ctx
		return func() []*VNode { return []*VNode{Text("x")} }
	})
	inst := Mount(c, nil)
	inst.Render()
	ctxRef.OnMounted(func() { called++ })
	if called != 1 {
		t.Fatalf("挂载后注册应立即执行，得到 %d", called)
	}
	inst.Render()
	if called != 1 {
		t.Fatalf("不应重复执行，得到 %d", called)
	}
}

// TestUnmountRunsHooksOnce：重复 Unmount 不该重复跑回调（关窗路径会用到）。
func TestUnmountRunsHooksOnce(t *testing.T) {
	unmounted := 0
	c := DefineComponent("c", func(ctx *SetupContext) RenderFunc {
		ctx.OnUnmounted(func() { unmounted++ })
		return func() []*VNode { return []*VNode{Text("x")} }
	})
	inst := Mount(c, nil)
	inst.Render()
	inst.Unmount()
	inst.Unmount()
	if unmounted != 1 {
		t.Fatalf("onUnmounted 只该跑一次，得到 %d", unmounted)
	}
}
