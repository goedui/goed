package app

import (
	"testing"

	"github.com/goedui/goed/ui/runtime"
)

func TestCreateAppBuilder(t *testing.T) {
	component := runtime.DefineComponent("Probe", func(*runtime.SetupContext) runtime.RenderFunc {
		return func() []*runtime.VNode { return nil }
	})
	a := CreateApp().Mount(component).Title("Probe").Size(320, 240).HideTitleBar(true)
	if a.component != component || a.title != "Probe" || a.width != 320 || a.height != 240 || !a.options.HideTitleBar {
		t.Fatalf("builder state = %#v", a)
	}
}

func TestApplicationRunRequiresRoot(t *testing.T) {
	if err := CreateApp().Run(); err == nil {
		t.Fatal("Run without a root component should fail")
	}
}
