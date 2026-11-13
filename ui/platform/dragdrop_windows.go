//go:build windows

package platform

import "github.com/goedui/goed/ui/platform/windows"

// bindDragDrop 把「拖出」能力交给平台层。窗口就绪时调用一次。
func bindDragDrop() {
	DoDragDropFiles = windows.DoDragDropFiles
}
