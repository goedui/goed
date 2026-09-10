// Package runtime 提供 Vue3 风格的声明式组件运行时。
//
// 三层职责：
//   - VNode：render 函数的输出，描述"界面应该是什么样"
//   - patch：对比新旧 VNode 树，对宿主组件树做最小更新
//   - Renderer：持有宿主组件树（core.Component），驱动 mount/patch/unmount
//
// 与 Vue3 的对应关系：
//
//	h()             -> H / T
//	defineComponent -> DefineComponent
//	setup()         -> SetupFunc
//	render()        -> RenderFunc
//	props/emits     -> Props + SetupContext.Emit
//	provide/inject  -> Provide / Inject
package runtime

import (
	"fmt"
	"github.com/goedui/goed/ui/core"
	"reflect"
	"strings"
)

// VNodeKind 节点类型。
type VNodeKind int

const (
	KindElement   VNodeKind = iota // 宿主元素（box/text/button...）
	KindComponent                  // 声明式组件（有 setup/render）
	KindText                       // 纯文本
	KindSlot                       // 仅用于组件 children 的插槽声明
)

// Props 属性表。值类型任意，由元素/组件自行解释。
type Props map[string]any

// Int 读取整型属性，缺失返回 def。
func (p Props) Int(key string, def int32) int32 {
	if v, ok := p[key]; ok {
		if n, ok := v.(int32); ok {
			return n
		}
		if n, ok := v.(int); ok {
			return int32(n)
		}
	}
	return def
}

// Bool 读取布尔属性，缺失返回 def。
func (p Props) Bool(key string, def bool) bool {
	if v, ok := p[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// Str 读取字符串属性，缺失返回 def。
func (p Props) Str(key string, def string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// Color 读取颜色属性（uint32），缺失返回 def。
func (p Props) Color(key string, def uint32) uint32 {
	if v, ok := p[key]; ok {
		if c, ok := v.(uint32); ok {
			return c
		}
	}
	return def
}

// Fn 读取无参函数属性，缺失返回 nil。
func (p Props) Fn(key string) func() {
	if f, ok := p[key].(func()); ok {
		return f
	}
	return nil
}

// StringFn reads a callback receiving one string value, used by input-like
// controls for Vue-style onInput/onChange handlers.
func (p Props) StringFn(key string) func(string) {
	if f, ok := p[key].(func(string)); ok {
		return f
	}
	return nil
}

// Container 可容纳子组件的宿主接口。*core.BaseComponent 与所有嵌入它的
// 宿主元素（BoxHost 等）都满足该接口；runtime 只依赖此接口，
// 不对具体宿主类型做断言。
type Container interface {
	core.Component
	AddChild(child core.Component)
	RemoveChild(child core.Component)
}

// VNode 虚拟节点。
type VNode struct {
	Kind       VNodeKind
	Tag        string   // 元素标签（element）
	Def        *Def     // 组件定义（component）
	Key        string   // 列表复用键（对应 v-for :key）
	Props      Props    // 属性
	Children   []*VNode // 子节点
	Text       string   // 文本内容（text）
	rawProps   any      // typed component props, retained alongside normalized Props
	slotName   string
	scopedSlot func(Props) *VNode

	// 运行时状态（patch 使用）
	instance *Instance
	host     core.Component // 挂载后的宿主组件
}

// H 创建元素节点：H("hstack", Props{"gap": 8}, childA, childB)
func H(tag string, props Props, children ...*VNode) *VNode {
	return &VNode{Kind: KindElement, Tag: tag, Props: props, Children: children}
}

// Input creates the built-in text input element. Keeping this small helper
// makes the common control read like Vue's h("input", props) while the host
// implementation remains in the components package.
func Input(props Props) *VNode { return H("input", props) }

// C 创建组件节点：C(MyComponent, Props{"title": "hi"})
func C(def *Def, props any, children ...*VNode) *VNode {
	p, raw := normalizeProps(props)
	return &VNode{Kind: KindComponent, Def: def, Props: p, rawProps: raw, Children: children}
}

// Text creates one text node. Multiple values are concatenated, which is
// convenient for Vue-style interpolation: Text("你点击了 ", count, " 次").
func Text(values ...any) *VNode {
	var text strings.Builder
	for _, value := range values {
		text.WriteString(textValue(value))
	}
	return &VNode{Kind: KindText, Text: text.String()}
}

// T is the short compatibility form of Text. New code should prefer Text so
// the node type is explicit at the call site.
func T(values ...any) *VNode { return Text(values...) }

// textValue is the render-function equivalent of Vue interpolation. In
// addition to strings, it accepts a getter or a reactive Ref directly. A Ref
// getter is evaluated while the component effect is active, so it remains
// reactive without requiring fmt.Sprintf at every call site.
func textValue(value any) string {
	if value == nil {
		return ""
	}
	if getter, ok := value.(func() string); ok {
		return getter()
	}
	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}
	method := reflect.ValueOf(value).MethodByName("Get")
	if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
		result := method.Call(nil)[0].Interface()
		return fmt.Sprint(result)
	}
	return fmt.Sprint(value)
}

// Slot declares a named slot when passed as a component child. Ordinary
// component children are collected into the default slot automatically.
func Slot(name string, children ...*VNode) *VNode {
	return &VNode{Kind: KindSlot, slotName: name, Children: children}
}

// ScopedSlot declares a slot whose renderer receives the scope exposed by the
// child component.
func ScopedSlot(name string, render func(Props) *VNode) *VNode {
	return &VNode{Kind: KindSlot, slotName: name, scopedSlot: render}
}

// WithKey 为节点设置复用键（链式）。
func (n *VNode) WithKey(key string) *VNode {
	n.Key = key
	return n
}

// VIf is the render-function equivalent of v-if.
func VIf(condition bool, node *VNode) *VNode {
	if condition {
		return node
	}
	return nil
}

// VElseIf selects the first matching branch, or elseNode when none match.
func VElseIf(conditions []bool, nodes []*VNode, elseNode *VNode) *VNode {
	limit := len(conditions)
	if len(nodes) < limit {
		limit = len(nodes)
	}
	for i := 0; i < limit; i++ {
		if conditions[i] {
			return nodes[i]
		}
	}
	return elseNode
}

// VShow keeps a node mounted while toggling its visibility prop.
func VShow(condition bool, node *VNode) *VNode {
	if node == nil || condition {
		return node
	}
	clone := *node
	clone.Props = cloneProps(node.Props)
	if clone.Props == nil {
		clone.Props = Props{}
	}
	clone.Props["visible"] = false
	return &clone
}

// VFor is the render-function equivalent of v-for.
func VFor[T any](items []T, render func(T, int) *VNode) []*VNode {
	nodes := make([]*VNode, 0, len(items))
	for i, item := range items {
		if node := render(item, i); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// VForObject iterates a map and supplies value, key and iteration index.
func VForObject[K comparable, V any](items map[K]V, render func(V, K, int) *VNode) []*VNode {
	nodes := make([]*VNode, 0, len(items))
	i := 0
	for key, value := range items {
		if node := render(value, key, i); node != nil {
			nodes = append(nodes, node)
		}
		i++
	}
	return nodes
}

func cloneProps(props Props) Props {
	if props == nil {
		return nil
	}
	clone := make(Props, len(props))
	for key, value := range props {
		clone[key] = value
	}
	return clone
}
