//go:build windows

package platform

import "github.com/goedui/goed/ui/platform/windows"

// Run 启动 Direct2D 宿主窗口并进入消息循环。
func Run(opts Options) error {
	return windows.Run(windows.Options{
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
				windows.SetWindowText(HWND, title)
			}
		},
		SetWindowSize: func(fn func(w, h int) bool) {
			SetWindowSize = func(w, h int) bool { return requestClientSize(fn, w, h) }
		},
		GetWindowSize: func(fn func() (w, h int, ok bool)) {
			GetWindowSize = fn
		},
		UIThreadID: SetUIThread,
		BindPost: func(fn func(func())) {
			Post = fn
		},
		BindHWND: func(hwnd uintptr) {
			HWND = hwnd
		},
		FrameUpdate: func(maximized, active bool, hover, pressed int) {
			st := Frame()
			// 标题可能已被 platform.SetTitle 改过（编辑器打开文件后），
			// 这里只在还没有标题时用启动标题兜底。
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
