// Package caret 提供「文本光标闪烁」这一个进程级状态。
//
// 闪烁全局唯一（屏幕上只可能有一个可见光标），相位不该每个控件各存一份：输入框
// （ui/components 的 textedit）与代码编辑区（editor 的 mcode）共用这里，才不会一快
// 一慢地各闪各的。单独成包是为了避开循环依赖 —— 请求重绘要用 ui/platform，而
// platform 依赖 renderer 定义的 Context。
package caret

import (
	"sync"
	"time"

	"github.com/goedui/goed/ui/platform"
)

var (
	mu sync.Mutex
	// on 是当前相位：true 表示这一帧该把光标画出来。
	on = true
	// used 是最近一次「有人来问相位」的时刻。闪烁循环靠它判断还有没有人在画
	// 光标，没人问就自己停 —— 外面不必记着通知它收工。
	used    = time.Now()
	running bool
	// period 是闪烁半周期：亮一个 period、灭一个 period，一个完整周期是它的两倍。
	// 530ms 是 Windows 的默认值（GetCaretBlinkTime）与 VS Code 的取值。
	period = 530 * time.Millisecond
	// frameHook 是「请求宿主重绘一帧」，默认 platform.RequestFrame（宿主在 Run 里
	// 绑定的带锁入口）。
	//
	// period 与 frameHook 都放在锁里，而不是做成裸的导出变量：后台循环每轮都要读
	// 它们，而测试要替换它们 —— 裸变量的话 -race 直接判数据竞争
	// （实测：editor 的 TestEditorClickResetsBlink，测试收尾改 Period 撞上循环里
	// 的 time.Sleep(Period)）。
	frameHook = platform.RequestFrame
)

// SetPeriod 设置闪烁半周期并返回原值，方便调用方 defer 恢复。
//
// 存在的意义是测试：压到毫秒级才能真正观察到相位翻转，否则验证「会闪」只能等半秒
// 或断言「调用过」这种抓不住 bug 的假绿。
func SetPeriod(d time.Duration) time.Duration {
	mu.Lock()
	old := period
	period = d
	mu.Unlock()
	return old
}

// SetRequestFrame 替换重绘钩子并返回原值。同样是测试用：要数「相位翻了到底有没有
// 请求重绘」—— 只测相位翻转对「在翻但没人重绘」完全免疫，而那在用户眼里和「根本
// 没相位」一样，都是一根静止的竖条。
func SetRequestFrame(fn func()) func() {
	mu.Lock()
	old := frameHook
	frameHook = fn
	mu.Unlock()
	return old
}

// Visible 报告这一帧光标该不该画，并登记「有人在画光标」。
//
// 每帧都要调用它。第一帧会顺带启动后台循环，此后每半个周期翻转一次相位并请求重绘。
func Visible() bool {
	mu.Lock()
	used = time.Now()
	cur := on
	start := !running
	running = true
	mu.Unlock()
	if start {
		go loop()
	}
	return cur
}

// Reset 让光标立刻回到「亮」相，并从此刻重新计时。
//
// 光标一动、或者用户点了编辑区，就要调一次：不然连打的时候字在往前走、
// 光标却正好落在灭相上，看着像光标丢了。
func Reset() {
	mu.Lock()
	on = true
	used = time.Now()
	mu.Unlock()
}

func loop() {
	for {
		// 半周期每轮从锁里取一次：测试会改它，裸读是数据竞争。
		mu.Lock()
		p := period
		mu.Unlock()
		time.Sleep(p)

		mu.Lock()
		// 两个周期没人来问相位，说明光标已经不在画了（控件不可见 / 被卸载），
		// 循环收工。下次 Visible 会重新起一个。
		if time.Since(used) > 2*p {
			running = false
			mu.Unlock()
			return
		}
		on = !on
		mu.Unlock()
		// 这里是后台 goroutine，必须走 requestFrame（默认 platform.RequestFrame
		// 这个带锁入口）—— 宿主在 Run 里绑定钩子，直接读全局变量会与它构成数据竞争。
		requestFrame()
	}
}

func requestFrame() {
	mu.Lock()
	fn := frameHook
	mu.Unlock()
	if fn != nil {
		fn()
	}
}
