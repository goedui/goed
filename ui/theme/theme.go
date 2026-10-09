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
	// Disabled / DisabledForeground 是禁用控件的填充与文字色。零值表示未设置，
	// 取用时走 DisabledColors 的内置回退（按 Dark 分两侧）。
	Disabled           runtime.Color
	DisabledForeground runtime.Color
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

// 内置的禁用态默认值。不是「随便挑的灰」，两个约束都是可验的：
//
//   - 填充要和卡片底（light 的 #FFFFFF / dark 的 #1E1E1E）分得出来，
//     否则「灰掉」这件事根本看不见；
//   - 文字要在填充上过 WCAG AA 正文的 4.5:1 —— 生成过程中的计时文案
//     就是禁用态文字，读不清就是缺陷。
var (
	disabledLightBG = hex("#E0E3E8")
	disabledLightFG = hex("#5C6370") // on #E0E3E8 = 4.70:1
	disabledDarkBG  = hex("#2A2D33")
	disabledDarkFG  = hex("#96A0AF") // on #2A2D33 = 5.22:1
)

// DisabledColors 返回禁用控件该用的填充色与文字色。
//
// 单列一个方法而不是让 button / select / textedit 各自回退：「禁用态长什么样」是三处
// 共用的语义。以前三处各抄一份字面量（#E8EAEE / #5F6673），暗色主题下那块近白色叠在
// #1E1E1E 上（13.84:1）像在发光，而三处都以为别处会跟主题走，谁也没改。
//
// 回退按 Dark 分两侧：手写主题只要声明了 Dark 就能拿到对的一侧，不必逐个补色。
func (t Theme) DisabledColors() (bg, fg runtime.Color) {
	bg, fg = t.Disabled, t.DisabledForeground
	if bg.A == 0 {
		bg = disabledLightBG
		if t.Dark {
			bg = disabledDarkBG
		}
	}
	if fg.A == 0 {
		fg = disabledLightFG
		if t.Dark {
			fg = disabledDarkFG
		}
	}
	return bg, fg
}
