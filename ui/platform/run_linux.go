//go:build linux

package platform

import "github.com/goedui/goed/ui/platform/linux"

// Run 启动 X11 宿主窗口并进入消息循环。
func Run(opts Options) error {
	return linux.Run(linux.Options{
		Title:       opts.Title,
		Width:       opts.Width,
		Height:      opts.Height,
		Dark:        opts.Dark,
		Icon16:      opts.Icon16,
		Icon32:      opts.Icon32,
		Paint:       opts.Paint,
		Pointer:     opts.Pointer,
		Key:         opts.Key,
		Wheel:       opts.Wheel,
		Composition: opts.Composition,
		Caret:       opts.Caret,
		BindRequestFrame: bindRequestFrame,
		SetWindowTitle: func(title string) {
			if HWND != 0 {
				linux.SetWindowTitle(HWND, title)
			}
		},
		BindPost: func(fn func(func())) {
			Post = fn
		},
		BindHWND: func(hwnd uintptr) {
			HWND = hwnd
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
