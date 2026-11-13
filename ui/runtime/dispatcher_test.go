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

// TestLazyDispatcherResolvesPostLate 是「先建状态、后建窗口」的回归测试。
//
// 宿主要在 Run 里才绑定 platform.Post；构造时把 platform.Post 拷进
// Dispatcher 读到的是 nil，投递就永远只入队、没人执行 —— 后台消息到不了
// 界面。惰性解析必须在 post 生效之后立刻能投递。
func TestLazyDispatcherResolvesPostLate(t *testing.T) {
	var post func(func())
	d := NewLazyDispatcher(func() func(func()) { return post })

	// 绑定之前：只入队。
	ran := 0
	d.Post(func() { ran++ })
	if ran != 0 {
		t.Fatalf("绑定前就执行了 %d 次", ran)
	}

	// 模拟宿主在 Run 里绑定。
	post = func(fn func()) { fn() }
	d.Post(func() { ran++ })
	if ran != 1 {
		t.Fatalf("绑定后应立刻投递，实际执行 %d 次", ran)
	}

	// 绑定前积压的那次仍在队列里，Drain 之后才执行。
	d.Drain()
	if ran != 2 {
		t.Fatalf("Drain 后应为 2 次，实际 %d", ran)
	}
}

// TestLazyDispatcherSetPostOverrides 确认 SetPost 之后不再走 resolve，
// 否则应用层显式注入的投递函数会被宿主的 platform.Post 盖掉。
func TestLazyDispatcherSetPostOverrides(t *testing.T) {
	d := NewLazyDispatcher(func() func(func()) { return func(fn func()) { fn() } })
	var viaSetPost int
	d.SetPost(func(fn func()) { viaSetPost++; fn() })
	d.Post(func() {})
	if viaSetPost != 1 {
		t.Fatalf("SetPost 应接管投递，实际 %d 次", viaSetPost)
	}
}
