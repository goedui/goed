// helloworld 是最小的 goed UI 程序（一个静态文本框），用来验证窗口能起来。
// API 用法见 ui/ui.go 的包注释；带交互的示例见 clicktest。
package main

import "github.com/goedui/goed/ui"

var HelloWorld = ui.Component(func(_ *ui.SetupContext) func() *ui.VNode {
	return func() *ui.VNode {
		return ui.VStack(
			ui.Text("HelloWorld"),
		)
	}
})

func main() {
	if err := ui.CreateApp(HelloWorld).Title("HelloWorld").Theme("dark").Size(800, 600).Run(); err != nil {
		panic(err)
	}
}
