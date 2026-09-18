package runtime

import (
	"sync"
	"testing"
)

func TestDispatcherQueuesUntilDrain(t *testing.T) {
	d := NewDispatcher(nil)
	n := 0
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Post(func() { n++ })
		}()
	}
	wg.Wait()
	if n != 0 {
		t.Fatal("未 Drain 前不应执行")
	}
	d.Drain()
	if n != 20 {
		t.Fatalf("Drain 后得到 %d", n)
	}
}

func TestDispatcherPostUsesCallback(t *testing.T) {
	var got int
	d := NewDispatcher(func(fn func()) { fn(); got++ })
	d.Post(func() {})
	d.Post(func() {})
	if got != 2 {
		t.Fatalf("post 回调次数 = %d", got)
	}
	d.Drain()
}
