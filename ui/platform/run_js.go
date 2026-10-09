//go:build js && wasm

package platform

import "github.com/goedui/goed/ui/platform/web"

// Run 把引擎挂到页面上的 <canvas>，然后永不返回——浏览器里没有「窗口关闭」
// 这个事件，页面的生命周期由标签页决定。
//
// 与 windows / linux 后端的差别：
//   - 没有 HWND，HWND 保持 0；
//   - 不提供 SetWindowSize：画布尺寸由页面布局决定，程序改不了（与 linux 一致）；
//   - GetWindowSize 返回画布的 CSS 尺寸，单位 DIP。
func Run(opts Options) error {
	// 浏览器里没有「系统窗口」这回事，标题栏右侧的最小化 / 最大化 / 关闭
	// 没有对应物。告诉 TitleBar 别画那三个按钮，标题才会铺满整条。
	CaptionButtons = false
	return web.Run(web.Options{
		Title:            opts.Title,
		Width:            opts.Width,
		Height:           opts.Height,
		Dark:             opts.Dark,
		Icon16:           opts.Icon16,
		Icon32:           opts.Icon32,
		Paint:            opts.Paint,
		Pointer:          opts.Pointer,
		RightPointer:     opts.RightPointer,
		Key:              opts.Key,
		Wheel:            opts.Wheel,
		Composition:      opts.Composition,
		Caret:            opts.Caret,
		BindRequestFrame: bindRequestFrame,
		BindPost: func(fn func(func())) {
			Post = fn
		},
		SetWindowTitle: web.SetDocumentTitle,
		GetWindowSize: func(fn func() (w, h int, ok bool)) {
			GetWindowSize = fn
		},
		FrameUpdate: func(maximized, active bool, hover, pressed int) {
			st := Frame()
			if st.Title == "" {
				st.Title = opts.Title
			}
			st.Maximized = maximized
			st.Active = active
			st.Hover = CaptionButton(hover)
			st.Pressed = CaptionButton(pressed)
			SetFrame(st)
		},
	})
}
