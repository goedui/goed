package theme

import (
	"math"
	"testing"

	"github.com/goedui/goed/ui/runtime"
)

// relLum 是 WCAG 2.x 的相对亮度。
func relLum(c runtime.Color) float64 {
	ch := func(v float32) float64 {
		f := float64(v)
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// contrast 是 WCAG 对比度，范围 1..21。
func contrast(a, b runtime.Color) float64 {
	la, lb := relLum(a), relLum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// 内置主题必须显式写死禁用色，不能只靠 DisabledColors 的回退。
//
// 回退是给「手写主题」准备的兜底（声明了 Dark 就能拿到对的一侧），但内置的两套
// 调色板是设计产物：值要能被 grep 到、能被 review，也不该因为回退常量的微调而
// 悄悄变样。
func TestBuiltinThemesSetDisabledExplicitly(t *testing.T) {
	for name, th := range map[string]Theme{"Light": Light, "Dark": Dark} {
		if th.Disabled.A == 0 {
			t.Errorf("%s.Disabled 没设（A=0，会走回退）", name)
		}
		if th.DisabledForeground.A == 0 {
			t.Errorf("%s.DisabledForeground 没设（A=0，会走回退）", name)
		}
	}
}

// 零值主题的回退要按 Dark 分两侧：手写主题只声明 Dark 就该拿到暗色那一档。
func TestDisabledColorsFallbackFollowsDark(t *testing.T) {
	lbg, lfg := Theme{}.DisabledColors()
	dbg, dfg := Theme{Dark: true}.DisabledColors()

	if lbg == dbg {
		t.Errorf("明暗两侧的回退填充相同（%+v），暗色下会变成浅色块", dbg)
	}
	if lfg == dfg {
		t.Errorf("明暗两侧的回退文字色相同（%+v）", dfg)
	}
	if relLum(dbg) >= relLum(lbg) {
		t.Errorf("暗色回退填充（亮度 %.4f）应比浅色（%.4f）暗", relLum(dbg), relLum(lbg))
	}
}

// 显式设了就轮不到回退。
func TestDisabledColorsExplicitOverrideWins(t *testing.T) {
	wantBG := runtime.RGB(1, 2, 3)
	wantFG := runtime.RGB(4, 5, 6)
	// 故意带上 Dark：如果实现先按 Dark 选回退再判断显式值，这里就会露馅。
	th := Theme{Dark: true, Disabled: wantBG, DisabledForeground: wantFG}

	bg, fg := th.DisabledColors()
	if bg != wantBG {
		t.Errorf("填充被回退覆盖：got %+v want %+v", bg, wantBG)
	}
	if fg != wantFG {
		t.Errorf("文字色被回退覆盖：got %+v want %+v", fg, wantFG)
	}
}

// 禁用文字压在自己的禁用底上要过 AA（4.5:1）。
//
// 这一条是有来历的：填充从 #E8EAEE 改到 #E0E3E8 时，原来的文字色 #5F6673 从
// 4.80:1 掉到 4.49:1 —— 刚好差一点点掉出 AA。填充一改，文字色必须重算。
func TestDisabledColorsContrastOnOwnFill(t *testing.T) {
	for name, th := range map[string]Theme{"Light": Light, "Dark": Dark} {
		bg, fg := th.DisabledColors()
		if got := contrast(bg, fg); got < 4.5 {
			t.Errorf("%s 禁用文字对比度 %.2f:1 < 4.5:1（填充 #%02x%02x%02x 文字 #%02x%02x%02x）",
				name, got, to8(bg.R), to8(bg.G), to8(bg.B), to8(fg.R), to8(fg.G), to8(fg.B))
		}
	}
}

// 禁用填充和它周围的面要分得开。
//
// 两侧的失败形态不一样，所以分别断言：
//
//   - 浅色：填充比卡片深一档。改回 #E8EAEE 会让对白卡的对比度从 1.287 掉到 1.204，
//     于是「禁用」和「可用」只剩字色在变。
//   - 暗色：填充必须比背景**暗得有限**。旧值 #E8EAEE 叠在 #1E1E1E 上是 13.84:1，
//     深色页面里禁用控件是一整块在发光的白。
func TestDisabledFillSeparatesFromSurfaces(t *testing.T) {
	lbg, _ := Light.DisabledColors()
	if got := contrast(lbg, Light.Background); got < 1.25 {
		t.Errorf("浅色禁用填充对背景只有 %.3f:1，和可用态几乎分不出来（应 >= 1.25）", got)
	}

	dbg, _ := Dark.DisabledColors()
	if got := contrast(dbg, Dark.Background); got > 2.0 {
		t.Errorf("暗色禁用填充对背景有 %.2f:1，在深色页面里会是一块亮斑（应 <= 2.0）", got)
	}
}

func to8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}
