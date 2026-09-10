package runtime

import "sync"

// Dispatcher transfers worker completions to the owning UI thread.
// Post is thread safe; Drain and all reactive state changes run on that thread.
type Dispatcher struct {
	mu      sync.Mutex
	pending []func()
	wake    func()
	closed  bool
}

func NewDispatcher(wake func()) *Dispatcher { return &Dispatcher{wake: wake} }
func (d *Dispatcher) Post(fn func()) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.pending = append(d.pending, fn)
	wake := d.wake
	d.mu.Unlock()
	if wake != nil {
		wake()
	}
}
func (d *Dispatcher) Drain() {
	d.mu.Lock()
	jobs := d.pending
	d.pending = nil
	d.mu.Unlock()
	for _, fn := range jobs {
		fn()
	}
}
func (d *Dispatcher) Close() { d.mu.Lock(); d.closed = true; d.pending = nil; d.mu.Unlock() }
