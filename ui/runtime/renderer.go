package runtime

import (
	"fmt"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/reactive"
)

// Renderer 声明式渲染器：把 VNode 树落到宿主组件树。
type Renderer struct {
	// Root 宿主根组件（由调用方提供，通常是 *core.BaseComponent）。
	Root Container
	// OnInvalidate 在每次 patch 后被调用，宿主应触发重绘。
	OnInvalidate func()
	// OnHostMounted configures platform services for newly created elements.
	OnHostMounted func(core.Component)

	rootInst *Instance
	rootEff  *reactive.Effect
}

// NewRenderer 创建渲染器。
func NewRenderer(root Container) *Renderer {
	return &Renderer{Root: root}
}

// Mount 挂载 render 函数产生的树。
// 节点直接挂到 Root 上；内部创建一个 Effect：render 读取的任何
// 响应式状态变化都会自动重新 render + patch（Vue3 的响应式更新闭环）。
func (r *Renderer) Mount(render RenderFunc) {
	r.Unmount()

	inst := &Instance{provides: map[string]any{}}
	r.rootInst = inst

	r.rootEff = reactive.NewEffect(func() {
		nodes := render()
		r.patchChildren(inst, inst.children, nodes, r.Root)
		inst.children = nodes
		if r.OnInvalidate != nil {
			r.OnInvalidate()
		}
	})
	r.rootEff.Run()
}

// Unmount 卸载整棵树并停止响应式更新。
func (r *Renderer) Unmount() {
	clearFocus(r.Root)
	if r.rootEff != nil {
		r.rootEff.Stop()
		r.rootEff = nil
	}
	if r.rootInst != nil {
		r.unmountChildren(r.rootInst.children, r.Root)
		r.rootInst.children = nil
		r.rootInst = nil
	}
}

// patchChildren 对比旧子节点列表与新子节点列表，做最小宿主操作。
// parent 是宿主容器；inst 是所属组件实例（可为 nil，用于根）。
//
// 复用规则（Vue3 简化版）：
//  1. 新节点带 key：按 key 在旧列表中找未复用的同 key 节点；
//  2. 无 key：按位置就近找第一个未复用且同类的旧节点；
//  3. 找不到可复用的旧节点则挂载新宿主；
//  4. 未被复用的旧节点全部卸载。
//
// Containers supporting ReorderChildren retain host identity on keyed moves.
func (r *Renderer) patchChildren(inst *Instance, oldNodes, newNodes []*VNode, parent Container) {
	reused := make([]bool, len(oldNodes))
	oldByKey := make(map[string]int, len(oldNodes))
	for i, o := range oldNodes {
		if o != nil && o.Key != "" {
			oldByKey[o.Key] = i
		}
	}

	for _, n := range newNodes {
		if n == nil {
			continue
		}
		var matched *VNode
		if n.Key != "" {
			if oi, ok := oldByKey[n.Key]; ok && !reused[oi] && sameKind(oldNodes[oi], n) {
				matched = oldNodes[oi]
				reused[oi] = true
			}
		} else {
			for j := 0; j < len(oldNodes); j++ {
				if oldNodes[j] != nil && !reused[j] && oldNodes[j].Key == "" && sameKind(oldNodes[j], n) {
					matched = oldNodes[j]
					reused[j] = true
					break
				}
			}
		}
		if matched != nil {
			r.patchVNode(inst, matched, n, parent)
		} else {
			r.mountVNode(inst, n, parent)
		}
	}

	// 卸载未被复用的旧节点。
	for i, o := range oldNodes {
		if o == nil {
			continue
		}
		if !reused[i] {
			r.unmountVNode(o, parent)
		}
	}
	if ordered, ok := parent.(interface{ ReorderChildren([]core.Component) }); ok {
		children := make([]core.Component, 0, len(newNodes))
		for _, n := range newNodes {
			if n != nil && n.host != nil {
				children = append(children, n.host)
			}
		}
		ordered.ReorderChildren(children)
	}
}

// sameKind 判断两个节点是否可原地复用。
func sameKind(a, b *VNode) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KindElement:
		return a.Tag == b.Tag
	case KindComponent:
		return a.Def == b.Def
	default:
		return true
	}
}

// mountVNode 挂载单个节点到 parent。
func (r *Renderer) mountVNode(inst *Instance, n *VNode, parent Container) {
	if n == nil || n.Kind == KindSlot {
		return
	}
	switch n.Kind {
	case KindText:
		host := &TextHost{Text: n.Text, Color: n.Props.Color("color", 0xFFFFFF)}
		host.Visible = n.Props.Bool("visible", true)
		n.host = host
		parent.AddChild(host)

	case KindElement:
		el, ok := hostElements[n.Tag]
		if !ok {
			panic(fmt.Sprintf("runtime: 未注册的元素 %q", n.Tag))
		}
		host := el.Create(n.Props)
		applyHostProps(host, n.Props)
		n.host = host
		parent.AddChild(host)
		if container, ok := host.(Container); ok {
			r.patchChildren(inst, nil, n.Children, container)
		}
		hostRef(n.Props, host)
		if r.OnHostMounted != nil {
			r.OnHostMounted(host)
		}

	case KindComponent:
		slots, scopedSlots := collectSlots(n.Children)
		child := &Instance{
			Def:         n.Def,
			Props:       n.Props,
			rawProps:    n.rawProps,
			parent:      inst,
			provides:    map[string]any{},
			slots:       slots,
			scopedSlots: scopedSlots,
		}
		n.instance = child
		// 组件宿主容器：把布局相关 props（flex）提升到容器上，
		// 让父容器的 syncLayout 能探测到。
		container := &ComponentHost{BaseComponent: *core.NewBaseComponent(), flex: n.Props.Int("flex", 0)}
		container.Visible = true
		var c Container = container
		n.host = c
		parent.AddChild(c)

		ctx := &SetupContext{props: n.Props, instance: child}
		child.setupCtx = ctx
		child.render = n.Def.Setup(ctx)
		child.effect = reactive.NewEffect(func() { r.renderComponent(child, container) })
		child.effect.Run()
	}
}

func (r *Renderer) renderComponent(child *Instance, container Container) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err := panicAsError(recovered)
			if !r.captureError(child, err) {
				panic(recovered)
			}
		}
	}()
	initial := !child.isMounted
	if initial {
		for _, fn := range child.beforeMount {
			fn()
		}
	} else {
		for _, fn := range child.beforeUpdate {
			fn()
		}
	}
	nodes := child.render()
	r.patchChildren(child, child.children, nodes, container)
	child.children = nodes
	if initial {
		child.isMounted = true
		for _, fn := range child.mounted {
			fn()
		}
	} else {
		for _, fn := range child.updated {
			fn()
		}
	}
	if r.OnInvalidate != nil {
		r.OnInvalidate()
	}
}

func panicAsError(value any) error {
	if err, ok := value.(error); ok {
		return err
	}
	return fmt.Errorf("%v", value)
}

func (r *Renderer) captureError(instance *Instance, err error) bool {
	for current := instance; current != nil; current = current.parent {
		for _, handler := range current.errorCaptured {
			if handler(err) {
				return true
			}
		}
	}
	return false
}

// patchVNode 复用旧节点宿主，更新属性与子树。
func (r *Renderer) patchVNode(inst *Instance, old, new *VNode, parent Container) {
	switch new.Kind {
	case KindText:
		host := old.host.(*TextHost)
		if host.Text != new.Text {
			host.Text = new.Text
			// The host is reused by the keyed/unkeyed diff. Clear its intrinsic
			// measurement so the parent stack recomputes size for the new text.
			host.measuredText = ""
			host.measuredW, host.measuredH = 0, 0
			host.Bounds.W, host.Bounds.H = 0, 0
			r.invalidate()
		}
		visible := new.Props.Bool("visible", true)
		if host.Visible != visible {
			host.Visible = visible
			r.invalidate()
		}
		new.host = host

	case KindElement:
		el := hostElements[new.Tag]
		el.Update(old.host, old.Props, new.Props)
		applyHostProps(old.host, new.Props)
		new.host = old.host
		if container, ok := new.host.(Container); ok {
			r.patchChildren(inst, old.Children, new.Children, container)
		}

	case KindComponent:
		child := old.instance
		new.instance = child
		new.host = old.host
		// props 不可变：直接替换（Vue3 同样不深更新 props）。
		child.Props = new.Props
		child.rawProps = new.rawProps
		if child.setupCtx != nil {
			child.setupCtx.props = new.Props
		}
		child.slots, child.scopedSlots = collectSlots(new.Children)
		if ch, ok := new.host.(*ComponentHost); ok {
			ch.flex = new.Props.Int("flex", 0)
			ch.Visible = new.Props.Bool("visible", true)
		}
		container := new.host.(Container)
		// 组件的响应式 effect 会在依赖变化时自动重渲染；
		// 这里同步强制一次，保证非响应式 props 变化也生效。
		r.renderComponent(child, container)
	}
}

// unmountVNode 从 parent 卸载节点并清理实例。
func (r *Renderer) unmountVNode(n *VNode, parent Container) {
	if n == nil || n.Kind == KindSlot {
		return
	}
	if n.host == focusedComponent {
		clearFocus(r.Root)
	}
	switch n.Kind {
	case KindComponent:
		child := n.instance
		if child != nil {
			for _, fn := range child.beforeUnmount {
				fn()
			}
			if child.effect != nil {
				child.effect.Stop()
			}
			for _, stop := range child.watchStops {
				stop()
			}
			container, _ := n.host.(Container)
			r.unmountChildren(child.children, container)
			for _, fn := range child.unmounted {
				fn()
			}
			child.isMounted = false
		}
	case KindElement:
		hostRef(n.Props, nil)
		if container, ok := n.host.(Container); ok {
			r.unmountChildren(n.Children, container)
		}
	}
	if parent != nil && n.host != nil {
		parent.RemoveChild(n.host)
	}
}

func (r *Renderer) unmountChildren(nodes []*VNode, parent Container) {
	for _, n := range nodes {
		r.unmountVNode(n, parent)
	}
}

func (r *Renderer) invalidate() {
	if r.OnInvalidate != nil {
		r.OnInvalidate()
	}
}
