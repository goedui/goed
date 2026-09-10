package reactive

import "testing"

func TestRefEffect(t *testing.T) {
	count := Ref(0)
	runs := 0
	e := NewEffect(func() {
		_ = count.Get()
		runs++
	})
	e.Run()

	if runs != 1 {
		t.Fatalf("初始运行次数 = %d, 期望 1", runs)
	}
	count.Set(1)
	if runs != 2 {
		t.Fatalf("Set 后运行次数 = %d, 期望 2", runs)
	}
	count.Set(2)
	if runs != 3 {
		t.Fatalf("再次 Set 后运行次数 = %d, 期望 3", runs)
	}
}

func TestRefSameValueNoNotify(t *testing.T) {
	v := Ref(42)
	runs := 0
	e := NewEffect(func() { _ = v.Get() })
	e.Run()

	v.Set(42) // 相同值不应触发
	if runs != 0 && v.Peek() != 42 {
		t.Fatal("不应有变化")
	}
	// 通过 Watch 验证不触发
	triggered := false
	stop := Watch(func() {
		_ = v.Get()
		triggered = true
	})
	defer stop()
	triggered = false
	v.Set(42)
	if triggered {
		t.Fatal("相同值不应触发 Watch")
	}
}

func TestEffectStop(t *testing.T) {
	v := Ref(1)
	runs := 0
	e := NewEffect(func() {
		_ = v.Get()
		runs++
	})
	e.Run()
	e.Stop()
	v.Set(2)
	if runs != 1 {
		t.Fatalf("Stop 后运行次数 = %d, 期望 1", runs)
	}
}

func TestComputedLazy(t *testing.T) {
	src := Ref(1)
	calls := 0
	c := NewComputed(func() int {
		calls++
		return src.Get() * 2
	})

	if got := c.Get(); got != 2 {
		t.Fatalf("首次 Get = %d, 期望 2", got)
	}
	if got := c.Get(); got != 2 {
		t.Fatalf("二次 Get = %d, 期望 2（缓存）", got)
	}
	if calls != 1 {
		t.Fatalf("计算次数 = %d, 期望 1（惰性缓存）", calls)
	}

	// 依赖变化只标记脏，不立即计算
	src.Set(5)
	if calls != 1 {
		t.Fatalf("Set 后计算次数 = %d, 期望 1（惰性）", calls)
	}
	if got := c.Get(); got != 10 {
		t.Fatalf("依赖变化后 Get = %d, 期望 10", got)
	}
	if calls != 2 {
		t.Fatalf("计算次数 = %d, 期望 2", calls)
	}
}

func TestComputedInEffect(t *testing.T) {
	src := Ref(2)
	double := NewComputed(func() int { return src.Get() * 2 })
	runs := 0
	e := NewEffect(func() {
		_ = double.Get()
		runs++
	})
	e.Run()

	if runs != 1 {
		t.Fatalf("初始运行 = %d", runs)
	}
	src.Set(3)
	if runs != 2 {
		t.Fatalf("源变化后运行 = %d, 期望 2", runs)
	}
}

func TestBatch(t *testing.T) {
	a := Ref(0)
	b := Ref(0)
	runs := 0
	e := NewEffect(func() {
		_ = a.Get()
		_ = b.Get()
		runs++
	})
	e.Run()

	Batch(func() {
		a.Set(1)
		b.Set(1)
		a.Set(2)
	})
	// Batch 结束后 flush 一次
	if runs != 2 {
		t.Fatalf("Batch 后运行次数 = %d, 期望 2（合并为一次）", runs)
	}
}

func TestDynamicDeps(t *testing.T) {
	cond := Ref(true)
	a := Ref(1)
	b := Ref(2)
	runs := 0
	e := NewEffect(func() {
		if cond.Get() {
			_ = a.Get()
		} else {
			_ = b.Get()
		}
		runs++
	})
	e.Run()

	// b 未被读取，修改 b 不应触发
	b.Set(99)
	if runs != 1 {
		t.Fatalf("修改未依赖的 b 后运行 = %d, 期望 1", runs)
	}
	// 切换条件后依赖变为 b
	cond.Set(false)
	if runs != 2 {
		t.Fatalf("切换条件后运行 = %d, 期望 2", runs)
	}
	// a 不再被依赖
	a.Set(100)
	if runs != 2 {
		t.Fatalf("修改已脱离依赖的 a 后运行 = %d, 期望 2", runs)
	}
	// b 现在是依赖
	b.Set(200)
	if runs != 3 {
		t.Fatalf("修改新依赖 b 后运行 = %d, 期望 3", runs)
	}
}

func TestWatch(t *testing.T) {
	v := Ref("init")
	got := ""
	stop := Watch(func() {
		got = v.Get()
	})
	defer stop()

	if got != "init" {
		t.Fatalf("Watch 立即执行 got = %q", got)
	}
	v.Set("next")
	if got != "next" {
		t.Fatalf("Watch 更新 got = %q, 期望 next", got)
	}
	stop()
	v.Set("after-stop")
	if got != "next" {
		t.Fatalf("Stop 后不应更新, got = %q", got)
	}
}

func TestComputedChain(t *testing.T) {
	base := Ref(1)
	mid := NewComputed(func() int { return base.Get() + 1 })
	top := NewComputed(func() int { return mid.Get() * 10 })

	if got := top.Get(); got != 20 {
		t.Fatalf("链式计算 = %d, 期望 20", got)
	}
	base.Set(2)
	if got := top.Get(); got != 30 {
		t.Fatalf("链式更新 = %d, 期望 30", got)
	}
}
