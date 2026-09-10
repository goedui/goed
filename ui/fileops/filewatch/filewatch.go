// Package filewatch contains the platform file-change notification used by
// the editor to detect external document modifications. The abstract
// FileWatcher keeps platform code behind a narrow interface; only the
// Windows backend uses ReadDirectoryChangesW today.
package filewatch

import (
	"path/filepath"
	"sync"
)

// EventKind classifies a watched filesystem change.
type EventKind uint8

const (
	EventModified EventKind = iota
	EventDeleted
	EventRenamed
	// EventOverflow 表示平台通知缓冲溢出、具体事件已丢失，
	// 消费方应执行全量刷新（如目录树重建）兜底。
	EventOverflow
)

// Event describes one external change relevant to open documents.
type Event struct {
	Path string
	Kind EventKind
}

// Watcher receives change events. Callbacks fire on the watcher's internal
// thread; consumers must marshal state changes back to the UI thread
// themselves (v0.2 baseline §7 thread boundary).
type Callback func(Event)

// FileWatcher is the platform abstraction over directory change notification.
type FileWatcher interface {
	// WatchDir starts reporting changes under dir (recursive on Windows).
	WatchDir(dir string) error
	// WatchFile registers a single-file interest.
	WatchFile(path string) error
	// Close stops all watches and releases platform resources.
	Close() error
}

// Dispatcher 对事件去抖并把路径分发给注册的回调。平台实现只负责产出
// 原始路径，过滤与回调选择统一在这里。
type Dispatcher struct {
	mu       sync.Mutex
	watched  map[string]Callback // 规范化路径 → 回调
	delegate func(Callback)      // 平台注入的启动函数
}

func NewDispatcher(start func(Callback)) *Dispatcher {
	return &Dispatcher{watched: map[string]Callback{}, delegate: start}
}

// On 注册对单个文件路径感兴趣的处理回调。
func (d *Dispatcher) On(path string, fn Callback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if fn == nil {
		delete(d.watched, normalize(path))
	} else {
		d.watched[normalize(path)] = fn
	}
}

// Count 返回当前注册的文件数。
func (d *Dispatcher) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.watched)
}

// Start 启动平台后端并开始分发。
func (d *Dispatcher) Start() {
	if d.delegate != nil {
		d.delegate(d.dispatch)
	}
}

// dispatch 由平台线程调用；只分发命中已注册文件的路径。
func (d *Dispatcher) dispatch(ev Event) {
	d.mu.Lock()
	fn, ok := d.watched[normalize(ev.Path)]
	d.mu.Unlock()
	if ok && fn != nil {
		fn(ev)
	}
}

func normalize(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

// globalDispatch 由平台 watcher 调用的事件出口；由 DefaultDispatcher 设置。
var globalDispatch = func(Event) bool { return false }

// SetGlobalSink 把平台 watcher 的事件接到一个过滤函数（返回 false 表示
// watcher 应停止解析当前批次）。
func SetGlobalSink(sink func(Event) bool) {
	globalDispatch = sink
}

func joinPath(dir, name string) string {
	return filepath.Join(dir, name)
}

func filepathDir(path string) string {
	return filepath.Dir(path)
}
