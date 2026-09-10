package filewatch

import (
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestDispatcherRoutesOnlyRegisteredPaths(t *testing.T) {
	var hits int32
	d := NewDispatcher(func(fn Callback) {
		// 平台后端把 dispatch 函数接给 watcher；测试直接用 dispatch。
		go func() {
			// noop 启动：测试手动调用 d.dispatch
		}()
		_ = fn
	})
	d.Start()
	d.On(filepath.Join("a", "b.txt"), func(Event) { atomic.AddInt32(&hits, 1) })
	d.On("c.txt", nil) // nil 注销

	d.dispatch(Event{Path: filepath.Join("a", "b.txt"), Kind: EventModified})
	d.dispatch(Event{Path: "other.txt", Kind: EventModified})
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("callback hits = %d, want 1", got)
	}
	if d.Count() != 1 {
		t.Fatalf("watched count = %d, want 1", d.Count())
	}
}

func TestDispatcherNormalizesPaths(t *testing.T) {
	var hits int32
	d := NewDispatcher(nil)
	d.Start()
	d.On("x.txt", func(Event) { atomic.AddInt32(&hits, 1) })
	d.dispatch(Event{Path: filepath.Join(".", "x.txt")})
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("hits = %d, want 1 (path normalization)", got)
	}
}
