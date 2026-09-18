package theme

import (
	"strings"

	"github.com/goedui/goed/ui/runtime"
)

// Theme 是一套命名视觉配置。零值颜色（A=0）表示未设置，绘制时回退到相邻默认。
type Theme struct {
	Name               string
	Dark               bool
	Background         runtime.Color
	Foreground         runtime.Color
	Titlebar           runtime.Color
	TitlebarForeground runtime.Color
	TitlebarHover      runtime.Color
	TitlebarInactive   runtime.Color
	CloseHover         runtime.Color
	Separator          runtime.Color
	Button             runtime.Color
	ButtonForeground   runtime.Color
	ButtonHover        runtime.Color
	ButtonPressed      runtime.Color
	FontFamily         string
	FontSize           float32
	Padding            float32
}

var (
	registry = map[string]Theme{}
	// Default 是 Lookup 找不到名称时的回退，也是 CreateApp 的初始主题。
	Default Theme
)

func init() {
	Register(Light)
	Register(Dark)
	Default = Light
}

func key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Register 按 t.Name 登记主题。同名覆盖。Name 为空则忽略。
func Register(t Theme) {
	k := key(t.Name)
	if k == "" {
		return
	}
	t.Name = k
	registry[k] = t
}

// RegisterAs 用指定名称登记，覆盖 t.Name。
func RegisterAs(name string, t Theme) {
	t.Name = name
	Register(t)
}

// Lookup 按名称取主题，未知名称回退到 Default。
func Lookup(name string) Theme {
	if t, ok := registry[key(name)]; ok {
		return t
	}
	if Default.FontSize > 0 {
		return Default
	}
	return Light
}

// SetDefault 把后续 Lookup 失败和 CreateApp 初始主题改成 t。
func SetDefault(t Theme) {
	if t.FontSize <= 0 {
		t.FontSize = Light.FontSize
	}
	Default = t
}

// DarkMode 报告是否应按深色窗口边框处理。
func (t Theme) DarkMode() bool {
	return t.Dark
}
