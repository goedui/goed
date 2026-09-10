// Package composables contains small, reusable pieces of reactive state.
// The API follows VueUse's useXxx convention while keeping ownership explicit
// and friendly to Go's type system.
package composables

import "github.com/goedui/goed/ui/reactive"

// UseCounter creates a counter ref and the operations that mutate it.
// Mutations are deliberately returned as functions so they can be passed
// directly to event props, just like Vue composables return methods.
func UseCounter(initial int) (count *reactive.RefValue[int], increment, decrement func()) {
	count = reactive.Ref(initial)
	increment = func() { count.Set(count.Get() + 1) }
	decrement = func() { count.Set(count.Get() - 1) }
	return count, increment, decrement
}

// UseToggle creates a boolean ref and a toggle operation.
func UseToggle(initial bool) (value *reactive.RefValue[bool], toggle func()) {
	value = reactive.Ref(initial)
	toggle = func() { value.Set(!value.Get()) }
	return value, toggle
}
