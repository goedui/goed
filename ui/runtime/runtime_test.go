package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/reactive"
)

// countingCtx 记录绘制调用的假绘图上下文。
type countingCtx struct {
	rects int
	texts []string
}

func (c *countingCtx) DrawRect(x, y, w, h int32, color uint32, filled bool) { c.rects++ }
func (c *countingCtx) DrawLine(x1, y1, x2, y2 int32, color uint32, width int32) {
}
func (c *countingCtx) DrawRoundedRect(x, y, w, h, radius int32, color uint32, filled bool) {
}
func (c *countingCtx) DrawEllipse(x, y, w, h int32, color uint32, filled bool)    {}
func (c *countingCtx) DrawPolygon(points []core.Point, color uint32, filled bool) {}
func (c *countingCtx) DrawText(x, y int32, text string, color uint32) {
	c.texts = append(c.texts, text)
}
func (c *countingCtx) MeasureText(text string) (width, height int32) { return int32(len(text)) * 8, 16 }
func (c *countingCtx) SetFont(name string, size int32)               {}
func (c *countingCtx) SetBkMode(transparent bool)                    {}
func (c *countingCtx) BeginDraw()                                    {}
func (c *countingCtx) EndDraw()                                      {}
func (c *countingCtx) SetClipRect(x, y, w, h int32)                  {}
func (c *countingCtx) ResetClip()                                    {}

// boxElement 测试用元素：绘制背景矩形。
type boxElement struct{}

func (boxElement) Tag() string { return "box" }

func (boxElement) Create(props Props) core.Component {
	b := &BoxHost{Bg: props.Color("bg", 0)}
	b.Visible = true
	return b
}

func (boxElement) Update(host core.Component, old, new Props) {
	b := host.(*BoxHost)
	if nb := new.Color("bg", 0); nb != old.Color("bg", 0) {
		b.Bg = nb
	}
}

// BoxHost 测试用宿主。
type BoxHost struct {
	core.BaseComponent
	Bg uint32
}

func (b *BoxHost) Render(ctx core.DrawingContext) {
	if !b.Visible {
		return
	}
	if b.Layout != nil {
		b.Layout.Arrange(b.Bounds)
	}
	ctx.DrawRect(b.Bounds.X, b.Bounds.Y, b.Bounds.W, b.Bounds.H, b.Bg, true)
	for _, child := range b.Children {
		if child != nil {
			child.Render(ctx)
		}
	}
}

func init() {
	RegisterElement(boxElement{})
}

func findBox(root *core.BaseComponent) []*BoxHost {
	var out []*BoxHost
	var walk func(c core.Component)
	walk = func(c core.Component) {
		if b, ok := c.(*BoxHost); ok {
			out = append(out, b)
		}
		if base, ok := c.(*core.BaseComponent); ok {
			for _, ch := range base.Children {
				walk(ch)
			}
		}
	}
	walk(root)
	return out
}

func findTexts(root *core.BaseComponent) []*TextHost {
	var out []*TextHost
	var walk func(c core.Component)
	walk = func(c core.Component) {
		if t, ok := c.(*TextHost); ok {
			out = append(out, t)
		}
		if base, ok := c.(*core.BaseComponent); ok {
			for _, ch := range base.Children {
				walk(ch)
			}
		}
	}
	walk(root)
	return out
}

func TestMountAndReactiveUpdate(t *testing.T) {
	label := reactive.Ref("hello")
	show := reactive.Ref(true)

	root := core.NewBaseComponent()
	r := NewRenderer(root)
	invalidations := 0
	r.OnInvalidate = func() { invalidations++ }

	r.Mount(func() []*VNode {
		nodes := []*VNode{
			T(label.Get()),
		}
		if show.Get() {
			nodes = append(nodes, H("box", Props{"bg": uint32(0xFF0000)}))
		}
		return nodes
	})

	var texts []*TextHost
	WalkTree(root, func(c core.Component) bool {
		if text, ok := c.(*TextHost); ok {
			texts = append(texts, text)
		}
		return false
	})
	if len(texts) != 1 || texts[0].Text != "hello" {
		t.Fatalf("初始文本错误: %v", texts)
	}
	if boxes := findBox(root); len(boxes) != 1 {
		t.Fatalf("初始 box 数量 = %d, 期望 1", len(boxes))
	}

	// 响应式更新：改文本
	label.Set("world")
	texts = findTexts(root)
	if len(texts) != 1 || texts[0].Text != "world" {
		t.Fatalf("更新后文本错误: %v", texts)
	}

	// 响应式更新：隐藏 box（卸载）
	show.Set(false)
	if boxes := findBox(root); len(boxes) != 0 {
		t.Fatalf("隐藏后 box 数量 = %d, 期望 0", len(boxes))
	}

	// 响应式更新：重新显示（挂载）
	show.Set(true)
	if boxes := findBox(root); len(boxes) != 1 {
		t.Fatalf("重新显示后 box 数量 = %d, 期望 1", len(boxes))
	}

	if invalidations < 3 {
		t.Fatalf("invalidations = %d, 期望 >= 3", invalidations)
	}
}

func TestKeyedListDiff(t *testing.T) {
	items := reactive.Ref([]string{"a", "b", "c"})

	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		var nodes []*VNode
		for _, it := range items.Get() {
			nodes = append(nodes, H("box", Props{"bg": uint32(1)}).WithKey(it))
		}
		return nodes
	})

	boxes := findBox(root)
	if len(boxes) != 3 {
		t.Fatalf("初始 box 数 = %d, 期望 3", len(boxes))
	}
	// 记录实例指针验证复用：key=b 的宿主是原第二个
	second := boxes[1]

	// 删除 a，新增 d：b、c 的宿主应被复用
	items.Set([]string{"b", "c", "d"})
	boxes = findBox(root)
	if len(boxes) != 3 {
		t.Fatalf("更新后 box 数 = %d, 期望 3", len(boxes))
	}
	if boxes[0] != second {
		t.Fatal("key=b 的宿主应被复用（原第二个）")
	}
}

func TestComponentLifecycle(t *testing.T) {
	mounted := false
	unmounted := false

	counter := DefineComponent("Counter", func(ctx *SetupContext) RenderFunc {
		ctx.OnMounted(func() { mounted = true })
		ctx.OnUnmounted(func() { unmounted = true })
		count := reactive.Ref(0)
		return func() []*VNode {
			return []*VNode{T("count=" + string(rune('0'+count.Peek())))}
		}
	})

	visible := reactive.Ref(true)
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		if visible.Get() {
			return []*VNode{C(counter, nil)}
		}
		return nil
	})

	if !mounted {
		t.Fatal("OnMounted 未触发")
	}
	if unmounted {
		t.Fatal("OnUnmounted 不应提前触发")
	}
	visible.Set(false)
	if !unmounted {
		t.Fatal("OnUnmounted 未触发")
	}
}

func TestProvideInject(t *testing.T) {
	var got string

	child := DefineComponent("Child", func(ctx *SetupContext) RenderFunc {
		got = ctx.Inject("lang", "none").(string)
		return func() []*VNode { return nil }
	})

	parent := DefineComponent("Parent", func(ctx *SetupContext) RenderFunc {
		ctx.Provide("lang", "go")
		return func() []*VNode { return []*VNode{C(child, nil)} }
	})

	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode { return []*VNode{C(parent, nil)} })

	if got != "go" {
		t.Fatalf("inject = %q, 期望 go", got)
	}
}

func TestUnmountStopsEffect(t *testing.T) {
	label := reactive.Ref("v1")
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode { return []*VNode{T(label.Get())} })

	r.Unmount()
	// 卸载后修改状态不应 panic，也不应重建宿主
	label.Set("v2")
	if texts := findTexts(root); len(texts) != 0 {
		t.Fatalf("卸载后仍有文本宿主: %d", len(texts))
	}
}

func TestRenderOutput(t *testing.T) {
	label := reactive.Ref("渲染")
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		return []*VNode{
			H("box", Props{"bg": uint32(0x123456)},
				T(label.Get()),
			),
		}
	})

	ctx := &countingCtx{}
	root.SetBounds(core.Rect{W: 100, H: 100})
	root.Render(ctx)
	if len(ctx.texts) != 1 || ctx.texts[0] != "渲染" {
		t.Fatalf("渲染文本 = %v", ctx.texts)
	}
	if ctx.rects != 1 {
		t.Fatalf("绘制矩形数 = %d, 期望 1", ctx.rects)
	}
}

func TestTypedPropsLifecycleAndSlots(t *testing.T) {
	type panelProps struct{ Title string }
	var events []string
	panel := DefineComponentWithProps[panelProps]("Panel", func(ctx *SetupContext, _ panelProps) RenderFunc {
		ctx.OnBeforeMount(func() { events = append(events, "before-mount") })
		ctx.OnMounted(func() { events = append(events, "mounted") })
		ctx.OnBeforeUpdate(func() { events = append(events, "before-update") })
		ctx.OnUpdated(func() { events = append(events, "updated") })
		ctx.OnBeforeUnmount(func() { events = append(events, "before-unmount") })
		ctx.OnUnmounted(func() { events = append(events, "unmounted") })
		return func() []*VNode {
			props := MustProps[panelProps](ctx)
			children := []*VNode{T(props.Title)}
			children = append(children, ctx.RenderSlot("default")...)
			children = append(children, ctx.RenderSlot("footer")...)
			return []*VNode{H("box", nil, children...)}
		}
	})
	title := reactive.Ref("one")
	visible := reactive.Ref(true)
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		if !visible.Get() {
			return nil
		}
		return []*VNode{C(panel, panelProps{Title: title.Get()}, T("body"), Slot("footer", T("foot")))}
	})
	if got := strings.Join(events, ","); got != "before-mount,mounted" {
		t.Fatalf("mount hooks = %q", got)
	}
	var texts []*TextHost
	WalkTree(root, func(c core.Component) bool {
		if text, ok := c.(*TextHost); ok {
			texts = append(texts, text)
		}
		return false
	})
	if len(texts) != 3 || texts[0].Text != "one" || texts[2].Text != "foot" {
		t.Fatalf("slot/text output = %#v", texts)
	}
	title.Set("two")
	if got := strings.Join(events, ","); got != "before-mount,mounted,before-update,updated" {
		t.Fatalf("update hooks = %q", got)
	}
	texts = texts[:0]
	WalkTree(root, func(c core.Component) bool {
		if text, ok := c.(*TextHost); ok {
			texts = append(texts, text)
		}
		return false
	})
	if len(texts) == 0 || texts[0].Text != "two" {
		t.Fatal("typed props did not update")
	}
	visible.Set(false)
	if got := strings.Join(events, ","); got != "before-mount,mounted,before-update,updated,before-unmount,unmounted" {
		t.Fatalf("unmount hooks = %q", got)
	}
}

func TestRenderHelpers(t *testing.T) {
	if VIf(false, T("hidden")) != nil {
		t.Fatal("VIf false should return nil")
	}
	node := VShow(false, H("text", Props{"value": "kept"}))
	if node == nil || node.Props.Bool("visible", true) {
		t.Fatal("VShow should set visible=false")
	}
	nodes := VFor([]string{"a", "b"}, func(value string, index int) *VNode {
		return T(value).WithKey(fmt.Sprint(index))
	})
	if len(nodes) != 2 || nodes[1].Key != "1" {
		t.Fatalf("VFor = %#v", nodes)
	}
}

func TestTextSources(t *testing.T) {
	count := reactive.Ref(2)
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		return []*VNode{T(count)}
	})
	texts := findTexts(root)
	if len(texts) != 1 || texts[0].Text != "2" {
		t.Fatalf("ref text = %#v", texts)
	}
	count.Set(3)
	if texts[0].Text != "3" {
		t.Fatalf("ref text update = %q", texts[0].Text)
	}
	message := reactive.NewComputed(func() string { return fmt.Sprintf("count=%d", count.Get()) })
	r.Mount(func() []*VNode { return []*VNode{T(message)} })
	if got := findTexts(root)[0].Text; got != "count=3" {
		t.Fatalf("computed text = %q", got)
	}
	joined := T("count=", count, ", ready")
	if joined.Text != "count=3, ready" {
		t.Fatalf("joined text = %q", joined.Text)
	}
}

func TestVShowTogglesTextWithoutUnmounting(t *testing.T) {
	show := reactive.Ref(false)
	root := core.NewBaseComponent()
	r := NewRenderer(root)
	r.Mount(func() []*VNode {
		return []*VNode{VShow(show.Get(), T("persistent"))}
	})
	var text *TextHost
	WalkTree(root, func(c core.Component) bool {
		if candidate, ok := c.(*TextHost); ok {
			text = candidate
		}
		return false
	})
	if text == nil || text.Visible {
		t.Fatal("VShow(false) should hide the mounted text")
	}
	show.Set(true)
	if text == nil || !text.Visible {
		t.Fatal("VShow(true) should reveal the same text host")
	}
}

func TestErrorCapturedBoundary(t *testing.T) {
	caught := false
	child := DefineComponent("Broken", func(*SetupContext) RenderFunc {
		return func() []*VNode { panic("render failed") }
	})
	parent := DefineComponent("Boundary", func(ctx *SetupContext) RenderFunc {
		ctx.OnErrorCaptured(func(err error) bool {
			caught = err.Error() == "render failed"
			return true
		})
		return func() []*VNode { return []*VNode{C(child, nil)} }
	})
	root := core.NewBaseComponent()
	NewRenderer(root).Mount(func() []*VNode { return []*VNode{C(parent, nil)} })
	if !caught {
		t.Fatal("OnErrorCaptured did not receive child render error")
	}
}

func TestSetupWatch(t *testing.T) {
	var source *reactive.RefValue[string]
	var changes []string
	watcher := DefineComponent("Watcher", func(ctx *SetupContext) RenderFunc {
		source = reactive.Ref("first")
		ctx.Watch(source, func(newValue, oldValue string) {
			changes = append(changes, oldValue+"->"+newValue)
		})
		return func() []*VNode { return []*VNode{T(source.Get())} }
	})
	root := core.NewBaseComponent()
	NewRenderer(root).Mount(func() []*VNode { return []*VNode{C(watcher, nil)} })
	source.Set("second")
	if len(changes) != 1 || changes[0] != "first->second" {
		t.Fatalf("watch changes = %v", changes)
	}
}
