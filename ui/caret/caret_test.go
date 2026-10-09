package caret

import (
	"sync/atomic"
	"testing"
	"time"
)

// 这一组测的是「光标到底会不会闪」。
//
// 默认半周期 530ms，直接拿它跑测试要么慢，要么只能断言「Visible 被调用过」这种
// 抓不住 bug 的假绿 —— 一个永远返回 true 的实现照样全绿，而「永远返回 true」就是
// 「光标不闪」这个 bug 本身。所以把 Period 压到毫秒级，真正观察相位的翻转。

// resetForTest 把包级状态恢复成「刚启动」，并把半周期压到 half。
//
// 收尾必须等后台循环自己退出：它退出时会置 running=false，不等的话它会带着被改小的
// 半周期去翻下一条测试的相位，把结果变成随机的。
func resetForTest(t *testing.T, half time.Duration) {
	t.Helper()
	old := SetPeriod(half)
	clear := func() {
		mu.Lock()
		on, used, running = true, time.Now(), false
		mu.Unlock()
	}
	t.Cleanup(func() {
		waitIdle(t)
		clear()
		SetPeriod(old)
	})
	clear()
}

// waitIdle 等后台循环收工：两个周期没人来问相位，它就该自己退出了。
func waitIdle(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		r := running
		mu.Unlock()
		if !r {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("没人画光标了，后台循环还在跑：它会一直每半秒唤醒 UI 线程重绘")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestVisibleStartsOn：第一次问相位必须是「亮」的。
//
// 反过来的话，控件画出来的头一帧光标就是暗的，用户看到的是「点了没反应」。
func TestVisibleStartsOn(t *testing.T) {
	resetForTest(t, 5*time.Millisecond)
	if !Visible() {
		t.Fatal("第一次问相位就是「灭」：光标刚出现就看不见")
	}
	waitIdle(t)
}

// TestBlinkTogglesOverTime 是这个包存在的理由：相位真的会翻转。
//
// 只断言「Visible 能被调用」是没用的 —— 永远返回 true 的实现也能过，
// 而那正是「光标不闪」本身。
func TestBlinkTogglesOverTime(t *testing.T) {
	resetForTest(t, 5*time.Millisecond)

	var sawOn, sawOff bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !(sawOn && sawOff) {
		if Visible() {
			sawOn = true
		} else {
			sawOff = true
		}
		time.Sleep(time.Millisecond)
	}
	if !sawOn || !sawOff {
		t.Fatalf("相位一直没翻转（亮=%v 灭=%v）：光标不会闪", sawOn, sawOff)
	}
	waitIdle(t)
}

// TestResetBringsCaretBackToOn：Reset 之后必须立刻是亮的。
//
// 光标移动、点击编辑区都会调 Reset，为的是「字在动的时候光标别正好落在灭相上」。
func TestResetBringsCaretBackToOn(t *testing.T) {
	resetForTest(t, 5*time.Millisecond)

	// 先等它翻到灭相。
	deadline := time.Now().Add(3 * time.Second)
	for Visible() {
		if time.Now().After(deadline) {
			t.Fatal("相位一直没翻到灭相")
		}
		time.Sleep(time.Millisecond)
	}
	Reset()
	if !Visible() {
		t.Fatal("Reset 之后相位还是灭的：移动光标后光标会继续暗着")
	}
	waitIdle(t)
}

// TestLoopStopsWhenNobodyPaints：没人再问相位，循环要自己收工。
//
// 不收工的话，一个被卸载的编辑框会永远每半秒唤醒一次 UI 线程请求重绘。
func TestLoopStopsWhenNobodyPaints(t *testing.T) {
	resetForTest(t, 5*time.Millisecond)
	Visible()   // 起循环
	waitIdle(t) // 之后不再调用 Visible，它该自己停
}

// TestLoopRequestsRepaint：相位每翻一次，就要请求一次重绘。
//
// 这是「光标不闪」里最阴的一种形态：相位明明在翻，但没人重绘，屏幕上就是一根
// 静止的竖条。只测相位翻转的用例对它完全免疫 —— 而它和「根本没相位」在用户眼里
// 一模一样。所以要单独钉住这一条。
func TestLoopRequestsRepaint(t *testing.T) {
	resetForTest(t, 5*time.Millisecond)

	var n int32
	old := SetRequestFrame(func() { atomic.AddInt32(&n, 1) })
	t.Cleanup(func() { SetRequestFrame(old) })

	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadInt32(&n) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("相位翻了几轮却一次都没请求重绘：屏幕上的光标是静止的")
		}
		Visible() // 一直有人问相位，循环才会继续翻
		time.Sleep(time.Millisecond)
	}
	waitIdle(t)
}
