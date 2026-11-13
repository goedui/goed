package runtime

import "sync"

// Dispatcher 把后台任务投递到 UI 线程。对应 Vue 的 nextTick 队列。
// post 为 nil 时（测试）只入队，调用 Drain 同步执行。
type Dispatcher struct {
	post func(func())
	// resolve 非 nil 时每次投递都重新解析 post。宿主是在 Run 里才绑定
	// platform.Post 的，而应用状态往往在 Run 之前就建好了 —— 构造时把
	// platform.Post 拷进来的话，读到的是 nil，于是永远只入队、没人执行。
	resolve func() func(func())
	mu      sync.Mutex
	queue   []func()
}

// NewDispatcher 创建调度器。应用层通常传入 platform.Post。
func NewDispatcher(post func(func())) *Dispatcher {
	return &Dispatcher{post: post}
}

// NewLazyDispatcher 创建调度器，post 在每次投递时才解析。
//
// 给「先建状态、后建窗口」的应用用：resolve 返回 nil 时退回入队，
// 与 NewDispatcher(nil) 的行为一致。
func NewLazyDispatcher(resolve func() func(func())) *Dispatcher {
	return &Dispatcher{resolve: resolve}
}

// SetPost 替换投递函数。应用在窗口就绪后把 platform.Post 接进来。
func (d *Dispatcher) SetPost(post func(func())) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.post = post
	d.resolve = nil
	d.mu.Unlock()
}

// Post 在 UI 线程运行 fn。已关闭的调用方应自行判断。
func (d *Dispatcher) Post(fn func()) {
	if d == nil || fn == nil {
		return
	}
	d.mu.Lock()
	post := d.post
	if post == nil && d.resolve != nil {
		post = d.resolve()
	}
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
