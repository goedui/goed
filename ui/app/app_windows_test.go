//go:build windows

package app

import (
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/goedui/goed/ui/components"
	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

var (
	testUser32       = syscall.NewLazyDLL("user32.dll")
	procSendMessageW = testUser32.NewProc("SendMessageW")
	procGetDpi       = testUser32.NewProc("GetDpiForWindow")
)

const (
	testWMMouseMove   = 0x0200
	testWMLButtonDown = 0x0201
	testWMLButtonUp   = 0x0202
	testWMClose       = 0x0010
	testMKLButton     = 0x0001
)

// TestClickAfterFirstPaint 覆盖一个真实踩到过的坑：命中区如果早于布局采集
// （首帧时节点的 X/Y/W/H 还都是 0），窗口会整个点不动。这里开一个真实窗口，
// 在第一帧之后按真实客户区坐标合成一次点击，断言按钮回调被执行。
func TestClickAfterFirstPaint(t *testing.T) {
	if testing.Short() {
		t.Skip("需要真实窗口")
	}
	var clicks int32
	root := runtime.DefineComponent("Root", func(_ *runtime.SetupContext) runtime.RenderFunc {
		return func() []*runtime.VNode {
			return []*runtime.VNode{
				components.VStack(
					runtime.Props{Style: runtime.Style{Flex: 1}},
					components.Button(runtime.Props{
						Style: runtime.Style{Width: 200, Height: 60},
						ID:    "btn",
						Label: "点我",
						OnClick: func() {
							atomic.AddInt32(&clicks, 1)
						},
					}, "点我"),
				),
			}
		}
	})

	var frames int32
	testPaint = func(renderer.Context) {
		if atomic.AddInt32(&frames, 1) != 1 {
			return
		}
		// 第一帧画完再点，点完再关窗口。
		go func() {
			time.Sleep(300 * time.Millisecond)
			// 按钮在标题栏（32 DIP）下方的左上角，尺寸 200x60。
			clickClient(100, 62)
			time.Sleep(300 * time.Millisecond)
			procSendMessageW.Call(platform.HWND, testWMClose, 0, 0)
		}()
	}
	defer func() { testPaint = nil }()

	if err := CreateApp(root).Title("github.com/goedui/goed-app-test").Size(400, 300).UseTheme(theme.Dark).Run(); err != nil {
		t.Fatalf("Run 失败：%v", err)
	}
	if atomic.LoadInt32(&clicks) == 0 {
		t.Fatal("首帧之后的合成点击没有触发回调：命中区可能为空（采集早于布局）")
	}
}

// clickClient 在客户区 DIP 坐标上合成一次「移动 + 按下 + 抬起」。
func clickClient(x, y int) {
	if platform.HWND == 0 {
		return
	}
	dpi, _, _ := procGetDpi.Call(platform.HWND)
	if dpi == 0 {
		dpi = 96
	}
	px := x * int(dpi) / 96
	py := y * int(dpi) / 96
	lp := uintptr(py<<16 | (px & 0xFFFF))
	procSendMessageW.Call(platform.HWND, testWMMouseMove, 0, lp)
	procSendMessageW.Call(platform.HWND, testWMLButtonDown, testMKLButton, lp)
	procSendMessageW.Call(platform.HWND, testWMLButtonUp, 0, lp)
}
