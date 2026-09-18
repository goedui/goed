package runtime

import "fmt"

// 当前正在执行 setup 的组件。Ref 在此时创建，才能把 Set 接到 Invalidate。
var current *SetupContext

// Ref 对应 Vue 3 的 ref<T>。用 NewRef 构造；应用层更常见的是 ui.Ref。
type Ref[T any] struct {
	value      T
	invalidate func()
}

// NewRef 创建响应式值。必须在 setup 里调用，Set 才会触发重绘。
func NewRef[T any](v T) *Ref[T] {
	r := &Ref[T]{value: v}
	if current != nil {
		r.invalidate = current.invalidate
	}
	return r
}

// Bind 把 Set 接到指定的 Invalidate。测试或延迟创建的 Ref 用这个。
func (r *Ref[T]) Bind(invalidate func()) {
	if r != nil {
		r.invalidate = invalidate
	}
}

// Get 对应 Vue 的 ref.value 读取。
func (r *Ref[T]) Get() T {
	if r == nil {
		var z T
		return z
	}
	return r.value
}

// Set 对应 Vue 的 ref.value = v，并请求下一帧。
func (r *Ref[T]) Set(v T) {
	if r == nil {
		return
	}
	r.value = v
	if r.invalidate != nil {
		r.invalidate()
	}
}

// String 让 Text("点击了 ", count, " 次") 能直接插值。
func (r *Ref[T]) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprint(r.value)
}
