package runtime

import "strconv"

// RenderFunc 每次重绘时调用，返回最新的虚拟子树。
type RenderFunc func() []*VNode

// SetupFunc 对应 Vue 3 的 setup()：组件创建时执行一次，返回渲染函数。
type SetupFunc func(ctx *SetupContext) RenderFunc

// ComponentDef 是 DefineComponent 的产物，可被 CreateApp 或 H() 使用。
type ComponentDef struct {
	Name  string
	Setup SetupFunc
}

// DefineComponent 注册一个组件。name 仅用于调试。
func DefineComponent(name string, setup SetupFunc) *ComponentDef {
	if setup == nil {
		setup = func(_ *SetupContext) RenderFunc {
			return func() []*VNode { return nil }
		}
	}
	return &ComponentDef{Name: name, Setup: setup}
}

// SetupContext 是 setup 阶段的上下文。对应 Vue 3 setup(props, ctx)。
type SetupContext struct {
	instance   *Instance
	invalidate func()
	props      Props
	slots      []*VNode
	unmount    []func()
}

// OnUnmounted 注册卸载回调，对应 Vue 的 onUnmounted。
func (c *SetupContext) OnUnmounted(fn func()) {
	if c == nil || fn == nil {
		return
	}
	c.unmount = append(c.unmount, fn)
}

// Invalidate 请求下一帧重绘。静态界面可以不调用。
func (c *SetupContext) Invalidate() {
	if c != nil && c.invalidate != nil {
		c.invalidate()
	}
}

// Props 返回父组件通过 h(Comp, props) 传入的属性。每帧 render 时读取最新值。
func (c *SetupContext) Props() Props {
	if c == nil {
		return Props{}
	}
	return c.props
}

// Attr 读取 h(Comp, props) 里仍按名字注入的字段（测试里的 state 等）。
func (c *SetupContext) Attr(key string) any {
	if c == nil {
		return nil
	}
	p := c.props
	switch key {
	case "state":
		return p.State
	case "dispatcher":
		return p.Dispatcher
	case "focus":
		if p.Focus {
			return true
		}
		return nil
	case "path":
		if p.Path == "" {
			return nil
		}
		return p.Path
	case "label":
		if p.Label == "" {
			return nil
		}
		return p.Label
	default:
		return nil
	}
}

// String 读取字符串字段。
func (c *SetupContext) String(key string) string {
	s, _ := c.Attr(key).(string)
	return s
}

// Children 返回默认 slot，即 h(Comp, props, children...) 的 children。
func (c *SetupContext) Children() []*VNode {
	if c == nil {
		return nil
	}
	return c.slots
}

// ResolveComponent 展开尚未挂载的组件节点。已由父实例 patch 过的节点不会重跑 setup。
func ResolveComponent(n *VNode) {
	if n == nil || n.Kind != KindComponent || n.Component == nil || n.resolved {
		return
	}
	inst := Mount(n.Component, nil)
	n.Children = inst.Render()
	n.resolved = true
}

// Instance 是组件的一次挂载。子组件实例按 vnode 路径缓存，跨帧复用。
type Instance struct {
	Def    *ComponentDef
	Render RenderFunc
	ctx    *SetupContext
	kids   map[string]*Instance
}

func newInstance(def *ComponentDef, invalidate func()) *Instance {
	inst := &Instance{Def: def}
	inst.ctx = &SetupContext{instance: inst, invalidate: invalidate}
	return inst
}

// Mount 执行 setup 并得到可反复调用的渲染函数。
func Mount(def *ComponentDef, invalidate func()) *Instance {
	return MountWith(def, invalidate, Props{})
}

// MountWith 在 setup 前写入 props，供测试注入状态。
func MountWith(def *ComponentDef, invalidate func(), props Props) *Instance {
	if def == nil {
		return &Instance{
			Render: func() []*VNode { return nil },
		}
	}
	inst := newInstance(def, invalidate)
	inst.ctx.props = props
	inst.setup()
	return inst
}

func (inst *Instance) setup() {
	if inst == nil || inst.Render != nil {
		return
	}
	if inst.Def == nil {
		inst.Render = func() []*VNode { return nil }
		return
	}
	prev := current
	current = inst.ctx
	user := inst.Def.Setup(inst.ctx)
	current = prev
	if user == nil {
		user = func() []*VNode { return nil }
	}
	inst.Render = func() []*VNode {
		nodes := user()
		inst.patch(nodes, "r")
		return nodes
	}
}

func (inst *Instance) patch(nodes []*VNode, path string) {
	if inst == nil {
		return
	}
	seen := make(map[string]struct{})
	inst.walk(nodes, path, seen)
	for k, child := range inst.kids {
		if _, ok := seen[k]; !ok {
			child.Unmount()
			delete(inst.kids, k)
		}
	}
}

// Unmount 运行 onUnmounted 并卸载子实例。
func (inst *Instance) Unmount() {
	if inst == nil {
		return
	}
	for _, child := range inst.kids {
		child.Unmount()
	}
	inst.kids = nil
	if inst.ctx == nil {
		return
	}
	fns := inst.ctx.unmount
	inst.ctx.unmount = nil
	for _, fn := range fns {
		fn()
	}
}

func (inst *Instance) walk(nodes []*VNode, path string, seen map[string]struct{}) {
	for i, n := range nodes {
		if n == nil {
			continue
		}
		p := path + "/" + strconv.Itoa(i)
		if n.Kind == KindComponent && n.Component != nil {
			key := p
			if n.Props.Key != "" {
				key = path + "/#" + n.Props.Key
			}
			seen[key] = struct{}{}
			child := inst.child(key, n)
			child.ctx.props = n.Props
			child.ctx.slots = n.Children
			child.setup()
			n.Children = child.Render()
			fallthroughProps(n, n.Children)
			n.resolved = true
			continue
		}
		if len(n.Children) > 0 {
			inst.walk(n.Children, p, seen)
		}
	}
}

func (inst *Instance) child(key string, n *VNode) *Instance {
	if inst.kids == nil {
		inst.kids = make(map[string]*Instance)
	}
	child := inst.kids[key]
	if child == nil || child.Def != n.Component {
		inv := (func())(nil)
		if inst.ctx != nil {
			inv = inst.ctx.invalidate
		}
		child = newInstance(n.Component, inv)
		inst.kids[key] = child
	}
	return child
}
