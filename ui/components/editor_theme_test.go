package components

import (
	"testing"

	"github.com/goedui/goed/ui/theme"
)

func TestThemeRegistryCyclesThroughBuiltins(t *testing.T) {
	prev := theme.GetTheme()
	defer theme.SetTheme(prev)

	theme.SetTheme(theme.Dark())
	names := map[string]bool{}
	for i := 0; i < len(theme.Names()); i++ {
		th := theme.Next()
		if th == nil || th.Name == "" {
			t.Fatal("Next returned nil or unnamed theme")
		}
		if names[th.Name] {
			t.Fatalf("theme %s repeated within one cycle", th.Name)
		}
		names[th.Name] = true
	}
	if !names["Dark"] || !names["Light"] || !names["Tabby"] {
		t.Fatalf("builtin themes missing, got %v", names)
	}
}

func TestThemeByNameResolvesBuiltins(t *testing.T) {
	if th := theme.ByName("Light"); th == nil || th.Name != "Light" {
		t.Fatalf("ByName(Light) = %#v", th)
	}
	if th := theme.ByName("Nope"); th != nil {
		t.Fatalf("ByName(Nope) = %#v, want nil", th)
	}
}


