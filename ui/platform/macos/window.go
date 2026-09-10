package macos

// Platform interface placeholder for macOS
// TODO: Implement in Phase 7

// Window macOS 窗口接口（预留）
type Window struct {
	// TODO: Cocoa implementation
}

// NewWindow 创建窗口（预留）
func NewWindow(title string, width, height int32) (*Window, error) {
	panic("macOS platform not implemented yet - Phase 7")
}

// NewFrameless 创建无边框窗口（预留）
func NewFrameless(title string, width, height int32) (*Window, error) {
	panic("macOS platform not implemented yet - Phase 7")
}
