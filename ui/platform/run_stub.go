//go:build !windows && !linux

package platform

import "fmt"

// Run 在非 Windows 平台尚未实现。渲染后端目前是 Direct2D。
func Run(opts Options) error {
	_ = opts
	return fmt.Errorf("github.com/goedui/goed/ui: 当前仅支持 Windows（Direct2D 渲染后端）")
}
