package runtime

import (
	"sync"
	"testing"
)

func TestDispatcherRunsWorkersOnlyWhenDrained(t *testing.T) {
	d := NewDispatcher(nil)
	var wg sync.WaitGroup
	calls := 0
	for n := 0; n < 40; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); d.Post(func() { calls++ }) }()
	}
	wg.Wait()
	if calls != 0 {
		t.Fatal("worker modified UI state")
	}
	d.Drain()
	if calls != 40 {
		t.Fatal("lost worker completion")
	}
	d.Close()
	d.Post(func() { t.Fatal("closed dispatcher executed a job") })
	d.Drain()
}
