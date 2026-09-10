package components

import (
	"bytes"
	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// ImageHost retains decoded pixels across reactive renders and scrolling.
type ImageHost struct {
	core.BaseComponent
	View          *ImageView
	data          []byte
	name, failure string
}

func (i *ImageHost) MeasureContent(core.DrawingContext) { i.Bounds.W, i.Bounds.H = 160, 160 }
func (i *ImageHost) Render(ctx core.DrawingContext) {
	if !i.Visible {
		return
	}
	core.WithClip(ctx, i.Bounds, func() {
		if i.View.Ready() {
			i.View.Render(ctx, i.Bounds.X, i.Bounds.Y, i.Bounds.W, i.Bounds.H)
		} else if i.failure != "" {
			ctx.DrawText(i.Bounds.X, i.Bounds.Y, "预览失败："+i.failure, theme.GetTheme().TitleBarFg)
		}
	})
}

type imageElement struct{}

func (imageElement) Tag() string { return "image" }
func (imageElement) Create(p runtime.Props) core.Component {
	i := &ImageHost{BaseComponent: *core.NewBaseComponent(), View: NewImageView()}
	imageElement{}.Update(i, nil, p)
	return i
}
func (imageElement) Update(c core.Component, _, p runtime.Props) {
	i := c.(*ImageHost)
	data, _ := p["data"].([]byte)
	name := p.Str("name", "")
	if bytes.Equal(data, i.data) && name == i.name {
		return
	}
	i.data, i.name = data, name
	i.failure = ""
	if len(data) == 0 {
		i.View.Reset()
		return
	}
	if err := i.View.LoadBytes(name, data); err != nil {
		i.failure = err.Error()
	}
}
func init() { runtime.RegisterElement(imageElement{}) }
