package runtime

import "sync"

// Dispatcher 把后台任务投递到 UI 线程。对应 Vue 的 nextTick 队列。
// post 为 nil 时（测试）只入队，调用 Drain 同步执行。
type Dispatcher struct {
	post  func(func())
	mu    sync.Mutex
	queue []func()
}

// NewDispatcher 创建调度器。应用层通常传入 platform.Post。
func NewDispatcher(post func(func())) *Dispatcher {
	return &Dispatcher{post: post}
}

// SetPost 替换投递函数。应用在窗口就绪后把 platform.Post 接进来。
func (d *Dispatcher) SetPost(post func(func())) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.post = post
	d.mu.Unlock()
}

// Post 在 UI 线程运行 fn。已关闭的调用方应自行判断。
func (d *Dispatcher) Post(fn func()) {
	if d == nil || fn == nil {
		return
	}
	d.mu.Lock()
	post := d.post
	d.mu.Unlock()
	if post != nil {
		post(fn)
		return
	}
	d.mu.Lock()
	d.queue = append(d.queue, fn)
	d.mu.Unlock()
}

// Drain 执行本地队列。有真实 post 时通常为空操作。
func (d *Dispatcher) Drain() {
	if d == nil {
		return
	}
	d.mu.Lock()
	q := d.queue
	d.queue = nil
	d.mu.Unlock()
	for _, fn := range q {
		fn()
	}
}
