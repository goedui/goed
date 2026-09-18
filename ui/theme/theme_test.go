package theme

import (
	"testing"

	"github.com/goedui/goed/ui/runtime"
)

func TestLookupBuiltin(t *testing.T) {
	if Lookup("dark").Name != "dark" {
		t.Fatal("dark 主题解析失败")
	}
	if !Lookup("DARK").Dark {
		t.Fatal("名称应大小写不敏感")
	}
	if Lookup("unknown").Name != Default.Name {
		t.Fatal("未知主题应回退到 Default")
	}
}

func TestVSCodeLightPlus(t *testing.T) {
	if Light.Background != hex("#FFFFFF") {
		t.Fatalf("editor.background: %+v", Light.Background)
	}
	if Light.Foreground != hex("#000000") {
		t.Fatalf("editor.foreground: %+v", Light.Foreground)
	}
	if Light.Titlebar != hex("#DDDDDD") {
		t.Fatalf("titleBar.activeBackground: %+v", Light.Titlebar)
	}
	if Light.Button != hex("#007ACC") {
		t.Fatalf("button.background: %+v", Light.Button)
	}
	if Light.ButtonForeground != hex("#FFFFFF") {
		t.Fatal("button.foreground 应为白")
	}
}

func TestVSCodeDarkPlus(t *testing.T) {
	if Dark.Background != hex("#1E1E1E") {
		t.Fatalf("editor.background: %+v", Dark.Background)
	}
	if Dark.Foreground != hex("#D4D4D4") {
		t.Fatalf("editor.foreground: %+v", Dark.Foreground)
	}
	if Dark.Titlebar != hex("#3C3C3C") {
		t.Fatalf("titleBar.activeBackground: %+v", Dark.Titlebar)
	}
	if Dark.Button != hex("#0E639C") {
		t.Fatalf("button.background: %+v", Dark.Button)
	}
}

func TestHex(t *testing.T) {
	if hex("#007ACC") != runtime.RGB(0, 122, 204) {
		t.Fatal("#RRGGBB")
	}
	if hex("#FFF") != runtime.RGB(255, 255, 255) {
		t.Fatal("#RGB")
	}
}

func TestRegisterCustom(t *testing.T) {
	Register(Theme{
		Name:       "brand",
		Dark:       true,
		Background: runtime.RGB(10, 20, 30),
		Foreground: runtime.RGB(240, 240, 240),
		FontFamily: "Segoe UI",
		FontSize:   18,
		Padding:    16,
	})
	got := Lookup("brand")
	if got.Name != "brand" || !got.Dark || got.FontSize != 18 {
		t.Fatalf("自定义主题未登记: %+v", got)
	}
	RegisterAs("alias", got)
	if Lookup("alias").FontSize != 18 {
		t.Fatal("RegisterAs 未生效")
	}
}

func TestSetDefault(t *testing.T) {
	prev := Default
	t.Cleanup(func() { SetDefault(prev) })
	SetDefault(Theme{Name: "fallback", FontSize: 20, Padding: 8})
	if Lookup("no-such").FontSize != 20 {
		t.Fatal("SetDefault 未作为 Lookup 回退")
	}
}
