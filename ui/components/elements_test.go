package components

import (
	"testing"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/reactive"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/style"
)

// runtime.TextHost 别名，测试断言用。
type TextHost = runtime.TextHost

type fakeCtx struct {
	rects int
	texts []string
}

type variableCtx struct{ fakeCtx }

func (c *variableCtx) MeasureText(text string) (width, height int32) {
	return int32(len([]rune(text))) * 8, 16
}

func (c *fakeCtx) DrawRect(x, y, w, h int32, color uint32, filled bool) { c.rects++ }
func (c *fakeCtx) DrawLine(x1, y1, x2, y2 int32, color uint32, width int32) {
}
func (c *fakeCtx) DrawRoundedRect(x, y, w, h, radius int32, color uint32, filled bool) {
}
func (c *fakeCtx) DrawEllipse(x, y, w, h int32, color uint32, filled bool)    {}
func (c *fakeCtx) DrawPolygon(points []core.Point, color uint32, filled bool) {}
func (c *fakeCtx) DrawText(x, y int32, text string, color uint32)             { c.texts = append(c.texts, text) }
func (c *fakeCtx) MeasureText(text string) (width, height int32)              { return 40, 16 }
func (c *fakeCtx) SetFont(name string, size int32)                            {}
func (c *fakeCtx) SetBkMode(transparent bool)                                 {}
func (c *fakeCtx) BeginDraw()                                                 {}
func (c *fakeCtx) EndDraw()                                                   {}
func (c *fakeCtx) SetClipRect(x, y, w, h int32)                               {}
func (c *fakeCtx) ResetClip()                                                 {}

func TestElementsRegistered(t *testing.T) {
	for _, tag := range []string{"box", "vstack", "hstack", "text", "button", "titlebar"} {
		if !runtime.HasElement(tag) {
			t.Errorf("元素 %q 未注册", tag)
		}
	}
	if !runtime.HasElement("input") {
		t.Error("元素 \"input\" 未注册")
	}
}

func TestInputFocusAndKeyboardEditing(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	var input *InputHost
	seen := ""
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.Input(runtime.Props{
			"value":       seen,
			"placeholder": "输入内容",
			"onInput":     func(value string) { seen = value },
		})}
	})
	input, _ = root.Children[0].(*InputHost)
	if input == nil {
		t.Fatal("input host not created")
	}
	root.SetBounds(core.Rect{W: 300, H: 40})
	input.SetBounds(core.Rect{W: 300, H: 32})
	if !runtime.DispatchEvent(root, core.Event{Type: core.EventMouseDown, X: 20, Y: 16}) || !input.Focused() {
		t.Fatal("click should focus input")
	}
	runtime.DispatchEvent(root, core.Event{Type: core.EventKeyChar, Char: 'a'})
	runtime.DispatchEvent(root, core.Event{Type: core.EventKeyChar, Char: 'b'})
	if input.Value != "ab" || seen != "ab" {
		t.Fatalf("typed value = %q, callback value = %q", input.Value, seen)
	}
	runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: core.KeyBackspace})
	if input.Value != "a" {
		t.Fatalf("backspace value = %q", input.Value)
	}
	r.Unmount()
	if runtime.DispatchEvent(root, core.Event{Type: core.EventKeyChar, Char: 'x'}) {
		t.Fatal("unmounted input should not receive keyboard events")
	}
}

func TestInputTabFocusTraversal(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.H("hstack", nil,
			runtime.Input(nil), runtime.Input(nil))}
	})
	stack := root.Children[0]
	stack.SetBounds(core.Rect{W: 200, H: 40})
	first, _ := childrenOf(stack)[0].(*InputHost)
	second, _ := childrenOf(stack)[1].(*InputHost)
	first.SetBounds(core.Rect{W: 90, H: 32})
	second.SetBounds(core.Rect{X: 100, W: 90, H: 32})
	runtime.DispatchEvent(root, core.Event{Type: core.EventMouseDown, X: 20, Y: 16})
	if !first.Focused() {
		t.Fatal("first input should be focused")
	}
	if !runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: core.KeyTab}) || !second.Focused() || first.Focused() {
		t.Fatal("tab should move focus to the next input")
	}
	if !runtime.DispatchEvent(root, core.Event{Type: core.EventKeyDown, Key: core.KeyTab, Shift: true}) || !first.Focused() {
		t.Fatal("shift+tab should move focus back")
	}
	r.Unmount()
}

func TestTitleBarButtonsAndDragRegion(t *testing.T) {
	bar := &TitleBarHost{Title: "Goed"}
	minimized, maximized, closed := false, false, false
	bar.OnMinimize = func() { minimized = true }
	bar.OnMaximize = func() { maximized = true }
	bar.OnClose = func() { closed = true }
	bar.Visible = true
	bar.SetBounds(core.Rect{W: 400, H: TitleBarHeight})

	if !bar.IsDragRegion(40, 20) || bar.IsDragRegion(390, 20) {
		t.Fatal("titlebar drag region should exclude the three command buttons")
	}

	for i, seen := range []*bool{&minimized, &maximized, &closed} {
		x := int32(400) - TitleBarButtonWidth*int32(3-i) + TitleBarButtonWidth/2
		bar.HandleEvent(core.Event{Type: core.EventMouseDown, X: x, Y: 20})
		bar.HandleEvent(core.Event{Type: core.EventMouseUp, X: x, Y: 20})
		if !*seen {
			t.Fatalf("titlebar button %d callback was not invoked", i)
		}
	}
}

// TestVueStyleCounter 端到端验证 Vue 式闭环：
// 点击按钮 → ref 变化 → 文本自动更新。
func TestVueStyleCounter(t *testing.T) {
	count := reactive.Ref(0)

	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{
			runtime.H("hstack", runtime.Props{"gap": 8},
				runtime.T("count: "+itoa(count.Get())),
				runtime.H("button", runtime.Props{
					"label":   "+1",
					"onClick": func() { count.Set(count.Get() + 1) },
				}),
			),
		}
	})

	// 找到按钮宿主并模拟点击
	btn := findButton(root)
	if btn == nil {
		t.Fatal("未找到按钮宿主")
	}
	btn.SetBounds(core.Rect{X: 0, Y: 0, W: 80, H: 30})

	// 按下 + 释放
	btn.HandleEvent(core.Event{Type: core.EventMouseDown, X: 40, Y: 15})
	btn.HandleEvent(core.Event{Type: core.EventMouseUp, X: 40, Y: 15})

	if count.Get() != 1 {
		t.Fatalf("点击后 count = %d, 期望 1", count.Get())
	}

	// 响应式更新后文本应变化
	texts := findTexts(root)
	if len(texts) != 1 || texts[0].Text != "count: 1" {
		t.Fatalf("点击后文本 = %v, 期望 [count: 1]", texts)
	}
}

func TestStackLayoutArrange(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{
			runtime.H("vstack", runtime.Props{"gap": 4},
				runtime.H("box", runtime.Props{"bg": uint32(1)}),
				runtime.H("box", runtime.Props{"bg": uint32(2)}),
			),
		}
	})

	// 手动给 vstack 容器布局
	stack, ok := root.Children[0].(*BoxHost)
	if !ok {
		t.Fatalf("根子组件类型 = %T, 期望 *BoxHost", root.Children[0])
	}
	stack.SetBounds(core.Rect{X: 0, Y: 0, W: 100, H: 100})
	// 给子 box 设置初始尺寸供布局计算
	for _, child := range stack.Children {
		child.SetBounds(core.Rect{W: 100, H: 10})
	}
	stack.Render(&fakeCtx{})

	if len(stack.Children) != 2 {
		t.Fatalf("子组件数 = %d, 期望 2", len(stack.Children))
	}
	first := stack.Children[0].GetBounds()
	second := stack.Children[1].GetBounds()
	if first.Y != 0 {
		t.Fatalf("第一个子组件 Y = %d, 期望 0", first.Y)
	}
	// gap=4：第二个应在 10+4=14
	if second.Y != 14 {
		t.Fatalf("第二个子组件 Y = %d, 期望 14", second.Y)
	}
}

func TestComponentRootReceivesLayoutBounds(t *testing.T) {
	component := runtime.DefineComponent("LayoutProbe", func(*runtime.SetupContext) runtime.RenderFunc {
		return func() []*runtime.VNode {
			return []*runtime.VNode{runtime.H("vstack", runtime.Props{"gap": 6, "padding": 8},
				runtime.T("first"),
				runtime.T("second"),
			)}
		}
	})
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.H("box", runtime.Props{"direction": "column"},
			runtime.C(component, runtime.Props{"flex": 1}),
		)}
	})
	root.SetBounds(core.Rect{W: 240, H: 160})
	root.Children[0].SetBounds(core.Rect{W: 240, H: 160})
	ctx := &variableCtx{}
	root.Render(ctx)
	texts := findTexts(root)
	if len(texts) != 2 {
		t.Fatalf("text hosts = %d, want 2", len(texts))
	}
	if texts[0].GetBounds().W == 0 || texts[0].GetBounds().H == 0 {
		t.Fatalf("component root did not receive bounds: root=%#v child=%#v first=%#v second=%#v", root.GetBounds(), root.Children[0].GetBounds(), texts[0].GetBounds(), texts[1].GetBounds())
	}
	if texts[1].GetBounds().Y <= texts[0].GetBounds().Y {
		t.Fatalf("stack children overlap: first=%#v second=%#v", texts[0].GetBounds(), texts[1].GetBounds())
	}
}

func TestTextMeasurementRefreshesAfterReactiveUpdate(t *testing.T) {
	message := reactive.Ref("short")
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.H("vstack", runtime.Props{"gap": 8, "align": "start"},
			runtime.T(message.Get()), runtime.T("stable"),
		)}
	})
	root.Children[0].SetBounds(core.Rect{W: 300, H: 120})
	ctx := &variableCtx{}
	root.Render(ctx)
	texts := findTexts(root)
	if len(texts) != 2 {
		t.Fatalf("text hosts = %d, want 2", len(texts))
	}
	shortWidth := texts[0].GetBounds().W
	firstHost := texts[0]
	message.Set("this is a substantially longer reactive label")
	root.Render(ctx)
	texts = findTexts(root)
	if texts[0] != firstHost {
		t.Fatal("text host should be reused")
	}
	if texts[0].GetBounds().W <= shortWidth {
		t.Fatalf("text width was not refreshed: before=%d after=%d", shortWidth, texts[0].GetBounds().W)
	}
	if texts[1].GetBounds().Y <= texts[0].GetBounds().Y {
		t.Fatalf("updated text overlaps following row: first=%#v second=%#v", texts[0].GetBounds(), texts[1].GetBounds())
	}
}

func TestStylePropsDriveStackLayout(t *testing.T) {
	root := core.NewBaseComponent()
	r := runtime.NewRenderer(root)
	r.Mount(func() []*runtime.VNode {
		return []*runtime.VNode{runtime.H("vstack", runtime.Props{"style": style.Parse(map[string]string{
			"padding": "10px 20px", "gap": "12px", "align": "start",
		})}, runtime.T("a"), runtime.T("b"))}
	})
	root.SetBounds(core.Rect{W: 200, H: 100})
	root.Children[0].SetBounds(core.Rect{W: 200, H: 100})
	root.Render(&variableCtx{})
	texts := findTexts(root)
	if len(texts) != 2 {
		t.Fatalf("text hosts = %d, want 2", len(texts))
	}
	first, second := texts[0].GetBounds(), texts[1].GetBounds()
	if first.X != 20 || first.Y != 10 {
		t.Fatalf("style padding not applied: first=%#v", first)
	}
	if second.Y != first.Y+first.H+12 {
		t.Fatalf("style gap not applied: first=%#v second=%#v", first, second)
	}
}

func childrenOf(c core.Component) []core.Component {
	switch h := c.(type) {
	case *core.BaseComponent:
		return h.Children
	case *BoxHost:
		return h.Children
	case *runtime.TextHost:
		return h.Children
	case *ButtonHost:
		return h.Children
	}
	if h, ok := c.(interface{ GetChildren() []core.Component }); ok {
		return h.GetChildren()
	}
	return nil
}

func findButton(c core.Component) *ButtonHost {
	if b, ok := c.(*ButtonHost); ok {
		return b
	}
	for _, ch := range childrenOf(c) {
		if b := findButton(ch); b != nil {
			return b
		}
	}
	return nil
}

func findTexts(c core.Component) []*runtime.TextHost {
	var out []*runtime.TextHost
	var walk func(core.Component)
	walk = func(cc core.Component) {
		if t, ok := cc.(*runtime.TextHost); ok {
			out = append(out, t)
		}
		for _, ch := range childrenOf(cc) {
			walk(ch)
		}
	}
	walk(c)
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
