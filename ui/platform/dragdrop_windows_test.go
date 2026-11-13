//go:build windows

package platform

import "testing"

func TestBindDragDropInstallsHook(t *testing.T) {
	bindDragDrop()
	if DoDragDropFiles == nil {
		t.Fatal("Windows 上应挂上系统拖放")
	}
	Unbind()
	if DoDragDropFiles != nil {
		t.Fatal("Unbind 应摘掉拖放钩子")
	}
}
