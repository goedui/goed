package components

import "github.com/goedui/goed/ui/platform/windows"

// ConfigureWindow gives declarative fields the standard native clipboard.
func (i *InputHost) ConfigureWindow(w *windows.FramelessWindow) {
	i.ReadClipboard = func() string { return windows.ReadClipboardText(w.Handle()) }
	i.WriteClipboard = func(text string) { windows.WriteClipboardText(w.Handle(), text) }
}
