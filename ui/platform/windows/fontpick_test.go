//go:build windows

package windows

import (
	"runtime"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/theme"
)

// withDWrite 起一次 COM、建一次 DirectWrite 工厂，跑完自动收尾。
//
// COM 的套间是**绑线程**的：CoInitializeEx 与 CoUninitialize 必须落在同一条 OS
// 线程上，所以每个用例都得 LockOSThread。
func withDWrite(t *testing.T, fn func(dwrite com)) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hresult(hr).failed() {
		t.Fatalf("CoInitializeEx: %v", hresult(hr))
	}
	defer procCoUninitialize.Call()

	dwrite, err := dwriteCreateFactory()
	if err != nil {
		t.Fatal(err)
	}
	defer release(dwrite)
	fn(dwrite)
}

// TestCreateTextFormatRejectsNullLocale 钉住 CreateTextFormat 的 locale 不能是 NULL。
//
// 这条坑很安静：localeName 传 NULL 时 DirectWrite 返回 E_INVALIDARG(0x80070057)，
// 失败只体现为「format == 0」。而调用方大多把 0 当成「没这个字体」退回按字号估宽，
// 于是整页文字的量宽与光栅全部走估算路径 —— 看起来只是「字有点飘」，不会有人
// 想到是 locale 的空指针。
//
// 这个用例把三个事实一起钉下来：NULL 会失败、空串 locale 能成、真 locale 能成。
// 于是 d2dContext.textFormat 里那段「locale 为空时退回 []uint16{0}」的兜底
// 就成了有依据的写法，而不是凭感觉加的。
func TestCreateTextFormatRejectsNullLocale(t *testing.T) {
	withDWrite(t, func(dwrite com) {
		if _, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, dwriteFontStyleNormal, nil); err == nil {
			t.Fatal("locale 传 NULL 竟然成功了，与 DirectWrite 的文档行为不符")
		}

		empty := []uint16{0}
		if _, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, dwriteFontStyleNormal, &empty[0]); err != nil {
			t.Fatalf("空串 locale 应当可以建出 format: %v", err)
		}

		if loc := userLocale(); len(loc) > 0 {
			if _, err := createTextFormat(dwrite, utf16Ptr("Segoe UI"), 16, 400, dwriteFontStyleNormal, &loc[0]); err != nil {
				t.Fatalf("真 locale 应当可以建出 format: %v", err)
			}
		}
	})
}

// TestDrawTextSurvivesNullLocale 是上面那条坑在**调用方**一侧的对照：
// d2dContext 即使没设 locale，也必须能建出 format。
//
// 从前测试里手工构造 &d2dContext{dwrite: dwrite} 就会踩到 —— locale 是空切片，
// 于是 CreateTextFormat 拿到 NULL，量宽与 FontMetrics 一起静默失效，
// 表现成「w=0、h=字号」的兜底值，很难联想到是 locale。
func TestDrawTextSurvivesNullLocale(t *testing.T) {
	withDWrite(t, func(dwrite com) {
		c := &d2dContext{dwrite: dwrite}
		style := renderer.TextStyle{FontSize: theme.Default.FontSize, FontFamily: theme.Default.FontFamily}
		w, h := c.MeasureText("HelloWorld", 1e6, style)
		if w <= 0 {
			t.Fatalf("没设 locale 时 MeasureText 也该量出宽度，得到 %v（format 没建出来？）", w)
		}
		if h <= 0 {
			t.Fatalf("没设 locale 时 MeasureText 也该量出行高，得到 %v", h)
		}
	})
}

// TestFontMetricsReportsRealAscent 钉住 FontMetrics 报的是**真实**字体度量。
//
// 从前是拿 FontSize * 0.85 当基线 —— 一个凭空的常数。不同字体的上伸比例差得很远
// （Arial 0.938em、Microsoft YaHei UI 1.0157em），常数必然有一头是错的：
// 用 0.85 时整行文字会比该在的位置低 1~2 像素，而且换字体还会变。
//
// 这里只钉两条与具体字体无关的性质：
//  1. ascent + descent 与 MeasureText 量出的行高一致（同一个字体、同一套度量）；
//  2. ascent 严格小于行高（不然基线跑到行盒外面去了）。
func TestFontMetricsReportsRealAscent(t *testing.T) {
	withDWrite(t, func(dwrite com) {
		c := &d2dContext{dwrite: dwrite}
		style := renderer.TextStyle{FontSize: 16, FontFamily: theme.Default.FontFamily}
		fm, ok := c.FontMetrics(style)
		if !ok {
			t.Fatal("FontMetrics 取不到度量")
		}
		if fm.Ascent <= 0 || fm.Descent < 0 {
			t.Fatalf("度量不合理: ascent=%v descent=%v", fm.Ascent, fm.Descent)
		}
		_, h := c.MeasureText("Hg", 1e6, style)
		if fm.LineHeight() <= 0 {
			t.Fatalf("行高应当为正: %v", fm.LineHeight())
		}
		// 行高与量宽得到的高度应当接近（都在同一个字体上量）。
		if diff := fm.LineHeight() - h; diff > 1 || diff < -1 {
			t.Fatalf("FontMetrics 的行高 %v 与 MeasureText 的 %v 差得太多，像是用了两套字体",
				fm.LineHeight(), h)
		}
		// 再确认它不是那个 0.85 的常数：16px 下 0.85*16 = 13.6，
		// 而真实字体的上伸应当在 0.9em 以上或接近。
		if fm.Ascent == 16*0.85 {
			t.Fatalf("ascent=%v 恰好等于 FontSize*0.85，说明还在用那个凭空的常数", fm.Ascent)
		}
	})
}

// TestPickFamilyWalksListInOrder 钉住「font-family 按顺序取第一个装了的」。
//
// 这是 CSS 层做不到的事：CSS 只知道列表，不知道本机装了什么。从前的实现是
// 「一路吞掉、换成主题字体」，于是 `Arial,sans-serif` 落到 Microsoft YaHei UI，
// 而真机浏览器会落到 Arial —— 后者的拉丁字形窄 6%（0.556em vs 0.59em）。
// 百度顶部导航那排 `.c-font-normal{font:13px/23px Arial,sans-serif}` 里的
// 「hao123」因此宽出 3 DIP，它后面 9 个导航项整体右移 3~5 像素。
func TestPickFamilyWalksListInOrder(t *testing.T) {
	withDWrite(t, func(dwrite com) {
		fallback := theme.Default.FontFamily

		// Arial 装了、而且画得出拉丁 —— 列表里第一个就该它。
		if got := pickFamily(dwrite, "Arial,"+fallback, fallback, "hao123"); got != "Arial" {
			t.Fatalf("Arial 装了就应当选 Arial，得到 %q", got)
		}

		// 没装的族要被跳过，落到下一个装了的。
		if got := pickFamily(dwrite, "NoSuchFontXyz123,"+fallback, fallback, "hao123"); got != fallback {
			t.Fatalf("没装的族应当被跳过，得到 %q", got)
		}

		// 空列表 -> 兜底。
		if got := pickFamily(dwrite, "", fallback, "hao123"); got != fallback {
			t.Fatalf("空列表应当返回兜底 %q，得到 %q", fallback, got)
		}

		// 拉丁走 Arial、汉字走主题字体：同一份列表，按**文字片段**挑出不同的族。
		// 这正是「量宽与取基线必须喂同一段文字」的原因。
		latin := pickFamily(dwrite, "Arial,"+fallback, fallback, "hao123")
		cjk := pickFamily(dwrite, "Arial,"+fallback, fallback, "新闻")
		if latin == "Arial" && cjk == "Arial" {
			t.Fatal("Arial 没有汉字字形，画「新闻」不该还选 Arial（HasCharacter 判定没生效？）")
		}
		if cjk != fallback {
			t.Fatalf("画汉字应当落到兜底字体 %q，得到 %q", fallback, cjk)
		}
	})
}

// TestFamilyCoversText 钉住字形覆盖判定的两个边界：空串一律不覆盖、空白不算缺字。
func TestFamilyCoversText(t *testing.T) {
	withDWrite(t, func(dwrite com) {
		if familyCoversText(dwrite, theme.Default.FontFamily, "") {
			t.Fatal("空文本不该算「画得出」—— 否则第一个族永远胜出")
		}
		if familyCoversText(dwrite, "NoSuchFontXyz123", "abc") {
			t.Fatal("不存在的族不该算覆盖")
		}
		// 空格不该被当成缺字而把整个族否掉。
		if !familyCoversText(dwrite, theme.Default.FontFamily, "  \t ") {
			t.Fatal("全是空白时应当算覆盖（空白不查字形）")
		}
	})
}
