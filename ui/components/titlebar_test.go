package components

import (
	"testing"

	"github.com/goedui/goed/ui/platform"
	"github.com/goedui/goed/ui/runtime"
)

func TestTitleBarRendersFrame(t *testing.T) {
	platform.SetFrame(platform.FrameState{
		Title:     "HelloWorld",
		Maximized: true,
		Active:    true,
		Hover:     platform.CaptionClose,
	})
	inst := runtime.Mount(TitleBar, nil)
	nodes := inst.Render()
	if len(nodes) != 1 || nodes[0].Tag != "titlebar" {
		t.Fatalf("期望 titlebar 元素，得到 %+v", nodes)
	}
	if nodes[0].Attrs["title"] != "HelloWorld" {
		t.Fatalf("标题未传入: %+v", nodes[0].Attrs)
	}
	if nodes[0].Attrs["maximized"] != true {
		t.Fatal("最大化状态未传入")
	}
	if nodes[0].Attrs["hover"] != int(platform.CaptionClose) {
		t.Fatal("关闭按钮悬停状态未传入")
	}
	if nodes[0].Attrs["icon"] == nil {
		t.Fatal("默认应用图标未传入标题栏")
	}
}
