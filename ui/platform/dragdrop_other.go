//go:build !windows

package platform

// bindDragDrop 非 Windows 平台没有系统级文件拖放，保持 nil。
func bindDragDrop() {}
