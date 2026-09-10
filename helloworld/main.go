package main
import (
	"github.com/goedui/goed/ui/app"
	"github.com/goedui/goed/ui/runtime"
)
var HelloWorld = runtime.DefineComponent("HelloWorld", func(_ *runtime.SetupContext) runtime.RenderFunc{
    return func() []*runtime.VNode{
        return []*runtime.VNode{
		   runtime.Text("HelloWorld"),
		}
	}
})

func main(){
	newApp := app.CreateApp(HelloWorld).Title("HelloWorld").Size(800,600)
	if err := newApp.Run();err!=nil {
		panic(err)
	}
}