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
