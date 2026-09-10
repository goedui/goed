package runtime

import (
	"testing"

	"github.com/goedui/goed/ui/reactive"
)

// TestReactiveAggregateAliasesAreInterchangeable 验证聚合层类型别名与
// reactive 包类型完全互通（同一标识类型），这是渐进迁移的根基。
func TestReactiveAggregateAliasesAreInterchangeable(t *testing.T) {
	// runtime.Ref 产出的 ref 可被 reactive 包的 API 直接消费。
	ref := Ref(42) // *runtime.RefValue[int] == *reactive.RefValue[int]
	var asReactive *reactive.RefValue[int] = ref
	if asReactive.Get() != 42 {
		t.Fatalf("aliased ref value = %d, want 42", asReactive.Get())
	}

	// reactive.Ref 产出的 ref 也可作为 runtime.RefValue 使用。
	fromReactive := reactive.Ref("hello")
	var asRuntime *RefValue[string] = fromReactive
	asRuntime.Set("world")
	if fromReactive.Get() != "world" {
		t.Fatalf("reactive ref not shared with alias: %q", fromReactive.Get())
	}

	// Computed 别名互通。
	sum := NewComputed(func() int { return ref.Get() * 2 })
	var reactiveComputed *reactive.Computed[int] = sum
	if reactiveComputed.Get() != 84 {
		t.Fatalf("computed = %d, want 84", reactiveComputed.Get())
	}
}

// TestReactiveAggregateBehavior 抽烟测试聚合入口的响应式行为。
func TestReactiveAggregateBehavior(t *testing.T) {
	count := Ref(0)
	runs := 0
	stop := Watch(func() {
		_ = count.Get()
		runs++
	})

	count.Set(1)
	Flush()
	stop()

	if runs != 2 { // 初次运行 + 一次变更
		t.Fatalf("effect runs = %d, want 2", runs)
	}
	if Pending() {
		t.Fatal("queue should be empty after Flush")
	}
}
