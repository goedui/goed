// Package reactive 提供 Vue3 风格的响应式内核。
//
// 核心思想与 Vue3 一致：谁在读取响应式数据，谁就是它的依赖。
// Effect 运行期间读取 Ref/Computed 会自动建立依赖关系；之后任何一次
// Ref.Set 都会把该 Effect 重新排入调度队列，实现"状态一变、精确更新"。
// Computed 本身也是订阅者：求值期间它以订阅者身份注册到上游 Ref，
// 上游变化时先标记自身为脏，再向下游订阅者传播。
//
// 线程模型：所有读写必须发生在 UI 线程（Windows 消息循环线程）。
// 跨线程更新请通过 app 层的 Enqueue 投递回 UI 线程。
package reactive

// subscriber 是依赖的接收方。Effect 与 Computed 都实现它：
// Effect 收到通知后重新入队，Computed 收到通知后标记脏并继续向下游传播。
type subscriber interface {
	notifyChanged()
	addDep(d *depSet)
}

// depSet 是一个可被追踪的依赖集合（Ref 或 Computed 各持有一个）。
type depSet struct {
	subs map[subscriber]struct{}
}

func (d *depSet) track(s subscriber) {
	if d.subs == nil {
		d.subs = make(map[subscriber]struct{})
	}
	d.subs[s] = struct{}{}
}

func (d *depSet) untrack(s subscriber) {
	delete(d.subs, s)
}

func (d *depSet) notify() {
	if len(d.subs) == 0 {
		return
	}
	// 复制一份再通知，避免回调中新增/删除订阅影响本次遍历。
	subs := make([]subscriber, 0, len(d.subs))
	for s := range d.subs {
		subs = append(subs, s)
	}
	for _, s := range subs {
		s.notifyChanged()
	}
}

// current 是正在读取响应式数据的订阅者（UI 单线程，全局指针即安全）。
var current subscriber

// RefValue is the concrete storage behind a reactive.Ref call.
type RefValue[T any] struct {
	value T
	deps  depSet
}

// Ref creates a reactive reference, corresponding to Vue 3's ref().
func Ref[T any](initial T) *RefValue[T] {
	return &RefValue[T]{value: initial}
}

// Get 读取值。若当前处于 Effect/Computed 求值中，则建立依赖。
func (r *RefValue[T]) Get() T {
	if current != nil {
		current.addDep(&r.deps)
	}
	return r.value
}

// Set 写入值并通知所有依赖它的订阅者。
func (r *RefValue[T]) Set(v T) {
	if equalAny(v, r.value) {
		return
	}
	r.value = v
	r.deps.notify()
}

// SetSilent 写入值但不触发更新（用于初始化或批量场景）。
func (r *RefValue[T]) SetSilent(v T) {
	r.value = v
}

// Peek 读取值但不建立依赖。
func (r *RefValue[T]) Peek() T {
	return r.value
}

// equalAny 比较两个任意值；不可比较类型（slice/map/func）视为不等。
func equalAny(a, b any) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}

// computedBase 是 Computed 的非泛型基座，让 Ref/Effect 可以
// 以统一接口持有它（Go 泛型无法对 *Computed[T] 做类型断言）。
type computedBase struct {
	dirty     bool
	computing bool
	subs      depSet    // 谁订阅了我（我变脏时通知谁）
	myDeps    []*depSet // 我订阅了谁（重算前解除，支持动态依赖）
}

func (b *computedBase) notifyChanged() {
	if !b.dirty {
		b.dirty = true
		b.subs.notify()
	}
}

func (b *computedBase) addDep(d *depSet) {
	d.track(b)
	b.myDeps = append(b.myDeps, d)
}

func (b *computedBase) clearDeps() {
	for _, d := range b.myDeps {
		d.untrack(b)
	}
	b.myDeps = b.myDeps[:0]
}

// Computed 计算属性，对应 Vue3 的 computed()。
// 惰性求值 + 脏标记：只有依赖变化后第一次 Get 才重新计算。
type Computed[T any] struct {
	computedBase
	compute func() T
	value   T
}

// NewComputed 创建计算属性。
func NewComputed[T any](compute func() T) *Computed[T] {
	c := &Computed[T]{compute: compute}
	c.dirty = true
	return c
}

// Get 读取计算值。若当前处于 Effect/Computed 求值中，则建立依赖。
func (c *Computed[T]) Get() T {
	if current != nil {
		current.addDep(&c.subs)
	}
	if !c.dirty {
		return c.value
	}
	if c.computing {
		panic("reactive: computed 循环依赖")
	}
	c.computing = true
	prev := current
	current = &c.computedBase // 求值期间以上游订阅者身份收集依赖
	defer func() {
		current = prev
		c.computing = false
		// panic 时保持 dirty，下次 Get 重算。
	}()
	c.clearDeps()
	c.value = c.compute()
	c.dirty = false
	return c.value
}

// Peek 读取缓存值但不触发重算、不建立依赖。
func (c *Computed[T]) Peek() T {
	return c.value
}

// Invalidate 手动标记为脏并通知下游订阅者。
func (c *Computed[T]) Invalidate() {
	c.notifyChanged()
}

// Watch 手动订阅一个响应式变化（立即执行一次）。
// 返回取消函数。这是逃生舱，优先使用 Effect。
func Watch(fn func()) (stop func()) {
	e := NewEffect(fn)
	e.Run()
	return e.Stop
}

// Effect 副作用：运行 fn，自动收集 fn 读取的所有响应式依赖；
// 任一依赖变化时 fn 会被重新调度执行。
type Effect struct {
	fn      func()
	myDeps  []*depSet
	queued  bool
	stopped bool
}

// NewEffect 创建副作用（不立即运行）。
func NewEffect(fn func()) *Effect {
	return &Effect{fn: fn}
}

// Run 立即执行一次并收集依赖。
func (e *Effect) Run() {
	if e.stopped {
		return
	}
	e.clearDeps()
	prev := current
	current = e
	defer func() { current = prev }() // fn panic 时也恢复上下文
	e.fn()
}

func (e *Effect) clearDeps() {
	for _, d := range e.myDeps {
		d.untrack(e)
	}
	e.myDeps = e.myDeps[:0]
}

func (e *Effect) addDep(d *depSet) {
	d.track(e)
	e.myDeps = append(e.myDeps, d)
}

// notifyChanged 实现 subscriber：把自己交给调度器。
func (e *Effect) notifyChanged() {
	scheduler.enqueue(e)
}

// Stop 停止追踪，之后 Set 不再调度本 Effect。
func (e *Effect) Stop() {
	e.stopped = true
	e.clearDeps()
}

// Batch 将多次 Set 合并为一次调度（对应 Vue 的 nextTick 批处理）。
// fn 执行完毕后统一 flush。
func Batch(fn func()) {
	scheduler.batch(fn)
}

// SetFlushCallback 注入调度器的 flush 实现。
// 由宿主（窗口消息循环）调用：flushFn 负责在合适的时机触发重绘，
// 例如把 win.Invalidate 投递回 UI 线程。注入后必须由宿主调用 Flush。
func SetFlushCallback(fn func()) {
	scheduler.flushFn = fn
}

// Flush 立即执行所有排队的 Effect。宿主在消息循环空闲时调用。
func Flush() {
	scheduler.Flush()
}

// Pending 报告是否有待处理的 Effect。
func Pending() bool {
	return len(scheduler.queue) > 0
}

// scheduler 单例调度器。
type schedulerT struct {
	queue    []*Effect
	flushing bool
	batching int
	flushFn  func() // 宿主注入；nil 时同步 flush
}

var scheduler schedulerT

func (s *schedulerT) enqueue(effects ...*Effect) {
	for _, e := range effects {
		if e.stopped || e.queued {
			continue
		}
		e.queued = true
		s.queue = append(s.queue, e)
	}
	s.schedule()
}

// schedule 在非 flush、非 batch 作用域且队列非空时触发一次 flush。
func (s *schedulerT) schedule() {
	if s.flushing || s.batching > 0 || len(s.queue) == 0 {
		return
	}
	if s.flushFn != nil {
		s.flushFn()
		return
	}
	s.Flush()
}

// Flush 运行队列中的全部 Effect，包括运行期间新产生的连锁更新。
func (s *schedulerT) Flush() {
	if s.flushing {
		return
	}
	s.flushing = true
	defer func() {
		s.flushing = false
		s.schedule() // flush 后可能仍有遗留（flushFn 异步场景）
	}()
	for len(s.queue) > 0 {
		queue := s.queue
		s.queue = nil
		for _, e := range queue {
			e.queued = false
			e.Run()
		}
	}
}

func (s *schedulerT) batch(fn func()) {
	s.batching++
	defer func() {
		s.batching--
		s.schedule()
	}()
	fn()
}
