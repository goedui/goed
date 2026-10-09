//go:build windows

package platform

import "github.com/goedui/goed/ui/platform/windows"

func init() {
	SaveBytes = windows.SaveBytes
	SaveImage = func(name, format string) (string, bool, error) {
		return windows.SaveImageDialog(HWND, name, format)
	}
	OpenImage = func() (string, bool, error) {
		return windows.OpenImageDialog(HWND)
	}
	OpenFile = func(title, filter string) (string, bool, error) {
		return windows.OpenFileDialog(HWND, title, filter)
	}
	SaveFile = func(title, name, filter, ext string) (string, bool, error) {
		return windows.SaveFileDialog(HWND, title, name, filter, ext)
	}
	OpenFolder = func(title string) (string, bool, error) {
		return windows.OpenFolderDialog(HWND, title)
	}
	ClipboardGet = windows.ClipboardText
	ClipboardSet = windows.SetClipboardText
}
