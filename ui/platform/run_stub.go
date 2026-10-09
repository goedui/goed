//go:build !windows && !linux && !js

package platform

import "fmt"

// Run 在这个平台尚未实现。渲染后端目前是 Direct2D（windows）、
// X11 + 软件光栅（linux）与 Canvas 2D（js/wasm）。
func Run(opts Options) error {
	_ = opts
	return fmt.Errorf("github.com/goedui/goed/ui: 当前平台没有渲染后端（支持 windows / linux / js+wasm）")
}
