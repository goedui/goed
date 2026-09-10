package runtime

import (
	"fmt"
	"github.com/goedui/goed/ui/reactive"
	"reflect"
	"strings"
)

// Def 组件定义。
type Def struct {
	Name string
	// Setup 在组件实例创建时调用一次，返回渲染函数。
	// 与 Vue 的 <script setup> 等价：在这里声明响应式状态、
	// computed、生命周期钩子，闭包持有状态。
	Setup func(ctx *SetupContext) RenderFunc
}

// RenderFunc 渲染函数：纯函数，只读状态并构造 VNode 树。
type RenderFunc func() []*VNode

// DefineComponent 定义组件（对应 Vue 的 defineComponent）。
func DefineComponent(name string, setup func(ctx *SetupContext) RenderFunc) *Def {
	return &Def{Name: name, Setup: setup}
}

// DefineComponentWithProps is the typed counterpart to DefineComponent. The
// component receives a concrete Props value during setup while the renderer
// keeps the normalized map for host/layout attributes.
func DefineComponentWithProps[T any](name string, setup func(ctx *SetupContext, props T) RenderFunc) *Def {
	return DefineComponent(name, func(ctx *SetupContext) RenderFunc {
		return setup(ctx, MustProps[T](ctx))
	})
}

// SetupContext 提供给 setup 的上下文：props 读取、事件发射、依赖注入。
type SetupContext struct {
	props    Props
	instance *Instance
}

// Props 返回当前 props（快照，组件 props 不可变）。
func (c *SetupContext) Props() Props { return c.props }

// PropsAs returns the concrete value supplied to C. The bool is false when
// the component was created with a different props type or without props.
func PropsAs[T any](c *SetupContext) (T, bool) {
	var zero T
	if c == nil || c.instance == nil {
		return zero, false
	}
	if value, ok := c.instance.rawProps.(T); ok {
		return value, true
	}
	if value, ok := c.instance.rawProps.(*T); ok && value != nil {
		return *value, true
	}
	return zero, false
}

// MustProps is convenient for typed components whose props contract is
// required. It panics with a useful message instead of a cryptic assertion.
func MustProps[T any](c *SetupContext) T {
	if value, ok := PropsAs[T](c); ok {
		return value
	}
	var zero T
	name := "unknown"
	if c != nil && c.instance != nil && c.instance.Def != nil {
		name = c.instance.Def.Name
	}
	panic(fmt.Sprintf("runtime: component %q expects props %T", name, zero))
}

// Emit 触发组件事件：调用 props 中名为 "on"+Name 的回调。
// 例如 Emit("click") 调用 props["onClick"]。
func (c *SetupContext) Emit(name string, args ...any) {
	if fn, ok := c.props["on"+name].(func(args ...any)); ok {
		fn(args...)
	}
}

// Provide 向后代提供值。
func (c *SetupContext) Provide(key string, value any) {
	c.instance.provides[key] = value
}

// Inject 读取祖先提供的值，找不到返回 def。
func (c *SetupContext) Inject(key string, def any) any {
	return c.instance.inject(key, def)
}

// OnMounted 注册挂载后回调（首次渲染完成后触发）。
func (c *SetupContext) OnMounted(fn func()) {
	c.instance.mounted = append(c.instance.mounted, fn)
}

// OnUnmounted 注册卸载回调。
func (c *SetupContext) OnUnmounted(fn func()) {
	c.instance.unmounted = append(c.instance.unmounted, fn)
}

// OnBeforeMount runs immediately before the component's first render.
func (c *SetupContext) OnBeforeMount(fn func()) {
	c.instance.beforeMount = append(c.instance.beforeMount, fn)
}

// OnBeforeUpdate runs before a mounted component re-renders.
func (c *SetupContext) OnBeforeUpdate(fn func()) {
	c.instance.beforeUpdate = append(c.instance.beforeUpdate, fn)
}

// OnUpdated runs after a mounted component's subtree has been patched.
func (c *SetupContext) OnUpdated(fn func()) { c.instance.updated = append(c.instance.updated, fn) }

// OnBeforeUnmount runs before a component and its subtree are removed.
func (c *SetupContext) OnBeforeUnmount(fn func()) {
	c.instance.beforeUnmount = append(c.instance.beforeUnmount, fn)
}

// OnErrorCaptured registers an error boundary. Returning true stops the error
// from propagating to an ancestor; returning false lets the ancestor inspect it.
func (c *SetupContext) OnErrorCaptured(fn func(error) bool) {
	c.instance.errorCaptured = append(c.instance.errorCaptured, fn)
}

// RenderSlot returns VNodes exposed through a named slot.
func (c *SetupContext) RenderSlot(name string) []*VNode {
	if c == nil || c.instance == nil || c.instance.slots == nil {
		return nil
	}
	return c.instance.slots[name]
}

// RenderSlotScoped evaluates a scoped slot with the supplied scope.
func (c *SetupContext) RenderSlotScoped(name string, scope Props) []*VNode {
	if c == nil || c.instance == nil || c.instance.scopedSlots == nil {
		return nil
	}
	if render := c.instance.scopedSlots[name]; render != nil {
		if node := render(scope); node != nil {
			return []*VNode{node}
		}
	}
	return nil
}

// WatchEffect runs a reactive side effect immediately and stops it when the
// component unmounts.
func (c *SetupContext) WatchEffect(fn func()) func() {
	stop := reactive.Watch(fn)
	c.instance.watchStops = append(c.instance.watchStops, stop)
	return stop
}

// Watch is the ergonomic, non-generic form for setup code. It accepts any
// reactive Ref and a typed two-argument callback; reflection is confined to
// this adapter while the reactive core remains fully generic and type-safe.
func (c *SetupContext) Watch(source any, callback any) func() {
	get := reflect.ValueOf(source).MethodByName("Get")
	cb := reflect.ValueOf(callback)
	if !get.IsValid() || !cb.IsValid() || cb.Kind() != reflect.Func || cb.Type().NumIn() != 2 {
		panic("runtime: Watch expects a reactive Ref and func(newValue, oldValue)")
	}
	oldValue := get.Call(nil)[0].Interface()
	first := true
	stop := reactive.Watch(func() {
		newValue := get.Call(nil)[0].Interface()
		if first {
			first = false
			return
		}
		if reflect.DeepEqual(newValue, oldValue) {
			return
		}
		previous := oldValue
		oldValue = newValue
		args := []reflect.Value{reflect.ValueOf(newValue), reflect.ValueOf(previous)}
		if args[0].IsValid() && args[1].IsValid() && args[0].Type().AssignableTo(cb.Type().In(0)) && args[1].Type().AssignableTo(cb.Type().In(1)) {
			cb.Call(args)
		}
	})
	c.instance.watchStops = append(c.instance.watchStops, stop)
	return stop
}

// WatchRef observes one Ref without firing during setup. It returns a stop
// function and is automatically stopped with the component.
func WatchRef[T any](c *SetupContext, source *reactive.RefValue[T], fn func(newValue, oldValue T)) func() {
	oldValue := source.Peek()
	first := true
	stop := reactive.Watch(func() {
		newValue := source.Get()
		if first {
			first = false
			return
		}
		if !reflect.DeepEqual(newValue, oldValue) {
			previous := oldValue
			oldValue = newValue
			fn(newValue, previous)
		}
	})
	c.instance.watchStops = append(c.instance.watchStops, stop)
	return stop
}

// Instance 组件实例：持有 setup 产生的渲染函数与已挂载子树。
type Instance struct {
	Def           *Def
	Props         Props
	render        RenderFunc
	children      []*VNode
	parent        *Instance
	provides      map[string]any
	rawProps      any
	slots         map[string][]*VNode
	scopedSlots   map[string]func(Props) *VNode
	beforeMount   []func()
	mounted       []func()
	beforeUpdate  []func()
	updated       []func()
	beforeUnmount []func()
	unmounted     []func()
	errorCaptured []func(error) bool
	isMounted     bool
	watchStops    []func()
	setupCtx      *SetupContext
	effect        *reactive.Effect
}

func (i *Instance) inject(key string, def any) any {
	for cur := i; cur != nil; cur = cur.parent {
		if v, ok := cur.provides[key]; ok {
			return v
		}
	}
	return def
}

// normalizeProps preserves the original typed value and exposes exported
// struct fields as the normalized Props map used by host elements.
func normalizeProps(value any) (Props, any) {
	if value == nil {
		return Props{}, nil
	}
	if props, ok := value.(Props); ok {
		return props, value
	}
	if props, ok := value.(map[string]any); ok {
		return Props(props), value
	}
	p := Props{}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return p, value
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return p, value
	}
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			name = strings.ToLower(field.Name[:1]) + field.Name[1:]
		}
		p[name] = rv.Field(i).Interface()
	}
	return p, value
}

func collectSlots(children []*VNode) (map[string][]*VNode, map[string]func(Props) *VNode) {
	slots := map[string][]*VNode{"default": nil}
	scoped := map[string]func(Props) *VNode{}
	for _, child := range children {
		if child == nil {
			continue
		}
		if child.Kind != KindSlot {
			slots["default"] = append(slots["default"], child)
			continue
		}
		name := child.slotName
		if name == "" {
			name = "default"
		}
		if child.scopedSlot != nil {
			scoped[name] = child.scopedSlot
		} else {
			slots[name] = append(slots[name], child.Children...)
		}
	}
	return slots, scoped
}
