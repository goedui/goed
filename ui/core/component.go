package core

// Component 组件接口
type Component interface {
	Render(ctx DrawingContext)
	GetBounds() Rect
	SetBounds(rect Rect)
	HandleEvent(event Event) bool
}

// BaseComponent 组件基类
type BaseComponent struct {
	Bounds   Rect
	Children []Component
	Parent   Component
	Visible  bool
	Enabled  bool
	Layout   Layout
	ID       string
	spec     LayoutSpec
	declared bool
}

// ReorderChildren preserves host identity while following declarative key order.
func (c *BaseComponent) ReorderChildren(children []Component) { c.Children = children }

func (c *BaseComponent) SetLayoutSpec(spec LayoutSpec)        { c.spec, c.declared = spec, true }
func (c *BaseComponent) LayoutProperties() (LayoutSpec, bool) { return c.spec, c.declared }
func (c *BaseComponent) Flex() int32                          { return c.spec.Flex }
func (c *BaseComponent) ElementID() string                    { return c.ID }
func (c *BaseComponent) SetElementState(id string, visible, enabled bool) {
	c.ID, c.Visible, c.Enabled = id, visible, enabled
}
func (c *BaseComponent) IsVisible() bool { return c.Visible }
func (c *BaseComponent) IsEnabled() bool { return c.Enabled }

func NewBaseComponent() *BaseComponent {
	return &BaseComponent{
		Visible: true,
		Enabled: true,
	}
}

func (c *BaseComponent) GetBounds() Rect {
	return c.Bounds
}

func (c *BaseComponent) SetBounds(rect Rect) {
	c.Bounds = rect
}

func (c *BaseComponent) AddChild(child Component) {
	if child == nil {
		return
	}
	c.Children = append(c.Children, child)
	if base, ok := child.(*BaseComponent); ok {
		base.Parent = c
	}
}

func (c *BaseComponent) RemoveChild(child Component) {
	for i, ch := range c.Children {
		if ch == child {
			c.Children = append(c.Children[:i], c.Children[i+1:]...)
			break
		}
	}
}

func (c *BaseComponent) Render(ctx DrawingContext) {
	if !c.Visible {
		return
	}
	if c.Layout != nil {
		c.Layout.Arrange(c.Bounds)
	}
	for _, child := range c.Children {
		if child != nil {
			child.Render(ctx)
		}
	}
}

func (c *BaseComponent) HandleEvent(event Event) bool {
	// Base implementation - override in subclass
	return false
}

func (c *BaseComponent) Contains(x, y int32) bool {
	return x >= c.Bounds.X && x < c.Bounds.X+c.Bounds.W &&
		y >= c.Bounds.Y && y < c.Bounds.Y+c.Bounds.H
}

// GetChildren 以统一方法暴露子组件，让嵌入 BaseComponent 的宿主
// （BoxHost 等）可以被通用遍历逻辑穿透。
func (c *BaseComponent) GetChildren() []Component {
	return c.Children
}
