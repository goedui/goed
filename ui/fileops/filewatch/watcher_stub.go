//go:build !windows

package filewatch

import "errors"

// stubWatcher 在非 Windows 平台提供接口占位：监视不产出事件，
// 编辑器其余路径（保存时 mtime 对比）仍然可用。
type stubWatcher struct{}

// New 返回非 Windows 平台的占位实现。
func New() (FileWatcher, error) {
	return stubWatcher{}, nil
}

func (stubWatcher) WatchDir(string) error  { return errors.New("filewatch: not supported on this platform") }
func (stubWatcher) WatchFile(string) error { return errors.New("filewatch: not supported on this platform") }
func (stubWatcher) Close() error           { return nil }
