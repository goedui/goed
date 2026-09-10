package runtime

import "github.com/goedui/goed/ui/reactive"

// 本文件是 Vue 3 聚合入口（'vue' 包）的对应物：用户代码只需 import
// runtime 即可获得完整的响应式 API，而不必知道内部拆分为 reactive
// （零依赖响应式内核，等价 @vue/reactivity）与 runtime（渲染调度器，
// 等价 @vue/runtime-core）两层。类型别名保证与 reactive 包的类型
// 完全互通，零运行时成本。

// RefValue 是 reactive.RefValue 的别名（等价 Vue 的 Ref<T>）。
type RefValue[T any] = reactive.RefValue[T]

// Computed 是 reactive.Computed 的别名（等价 Vue 的 ComputedRef<T>）。
type Computed[T any] = reactive.Computed[T]

// Effect 是 reactive.Effect 的别名。
type Effect = reactive.Effect

// Ref 创建响应式引用（等价 Vue 的 ref()）。
func Ref[T any](initial T) *RefValue[T] { return reactive.Ref(initial) }

// NewComputed 创建计算属性（等价 Vue 的 computed()）。
func NewComputed[T any](compute func() T) *Computed[T] { return reactive.NewComputed(compute) }

// NewEffect 创建副作用（不立即运行；等价 Vue 内部的 ReactiveEffect）。
func NewEffect(fn func()) *Effect { return reactive.NewEffect(fn) }

// Watch 创建并立即运行一个副作用，返回停止函数（等价 Vue 的 watchEffect）。
func Watch(fn func()) (stop func()) { return reactive.Watch(fn) }

// Batch 在同一批内合并多次变更，fn 返回后统一 flush（等价 Vue 的 nextTick 批处理语义）。
func Batch(fn func()) { reactive.Batch(fn) }

// SetFlushCallback 注入宿主 flush 回调（UI 线程重绘入口）。
func SetFlushCallback(fn func()) { reactive.SetFlushCallback(fn) }

// Flush 立即执行排队的 Effect。
func Flush() { reactive.Flush() }

// Pending 报告是否有待处理的 Effect。
func Pending() bool { return reactive.Pending() }
