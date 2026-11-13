package components

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func renderFixture(m *QRMatrix) []string {
	rows := make([]string, m.Size)
	for y := 0; y < m.Size; y++ {
		var b strings.Builder
		for x := 0; x < m.Size; x++ {
			if m.Dark(x, y) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		rows[y] = b.String()
	}
	return rows
}

func assertMatrix(t *testing.T, got *QRMatrix, tc qrFixture) {
	t.Helper()
	if got.Version != tc.version {
		t.Fatalf("%q @ %s 版本 = %d，期望 %d", tc.text, tc.level, got.Version, tc.version)
	}
	if got.Mask != tc.mask {
		t.Fatalf("%q @ %s 掩码 = %d，期望 %d", tc.text, tc.level, got.Mask, tc.mask)
	}
	rows := renderFixture(got)
	if len(rows) != len(tc.rows) {
		t.Fatalf("%q 行数 = %d，期望 %d", tc.text, len(rows), len(tc.rows))
	}
	for i := range rows {
		if rows[i] != tc.rows[i] {
			t.Fatalf("%q @ %s 第 %d 行不一致\n got %s\nwant %s", tc.text, tc.level, i, rows[i], tc.rows[i])
		}
	}
}

// TestEncodeQRMatchesReference 覆盖选版、选掩码、RS 纠错、交织与格式信息：
// 任何一处算错都会让某一行的模块对不上。
func TestEncodeQRMatchesReference(t *testing.T) {
	for _, tc := range qrReferenceCases {
		m, err := EncodeQR(tc.text, tc.level)
		if err != nil {
			t.Fatalf("%q 编码失败: %v", tc.text, err)
		}
		assertMatrix(t, m, tc)
	}
}

// TestEncodeQRFixedMasks 固定版本逐掩码比对，掩码公式写错时能定位到具体哪一格。
func TestEncodeQRFixedMasks(t *testing.T) {
	for _, tc := range qrMaskCases {
		m, err := EncodeQRVersion(tc.text, tc.level, tc.version, tc.mask)
		if err != nil {
			t.Fatalf("mask %d 编码失败: %v", tc.mask, err)
		}
		assertMatrix(t, m, tc)
	}
}

func TestEncodeQREmptyAndCapacity(t *testing.T) {
	m, err := EncodeQR("", LevelM)
	if err != nil {
		t.Fatalf("空内容应能编码: %v", err)
	}
	if m.Version != 1 || m.Size != 21 {
		t.Fatalf("空内容应为 1 版 21x21，实际 %d/%d", m.Version, m.Size)
	}
	if _, err := EncodeQR(strings.Repeat("a", 3000), LevelL); err == nil {
		t.Fatal("超长内容应报错")
	}
	if _, err := EncodeQRVersion("https://d.cn", LevelH, 1, 0); err == nil {
		t.Fatal("1 版 H 级放不下 11 字节，应报错")
	}
	if _, err := EncodeQRVersion("x", LevelM, 0, 9); err == nil {
		t.Fatal("非法掩码应报错")
	}
}

// TestQRMatrixDarkBounds 越界读取必须安全：绘制时会按整块行合并，末列 +1 会读到外面。
func TestQRMatrixDarkBounds(t *testing.T) {
	m, err := EncodeQR("https://d.cn", LevelM)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {m.Size, 0}, {0, m.Size}, {m.Size, m.Size}} {
		if m.Dark(p[0], p[1]) {
			t.Fatalf("越界 (%d,%d) 不应是深色", p[0], p[1])
		}
	}
	var nilM *QRMatrix
	if nilM.Dark(0, 0) {
		t.Fatal("nil 矩阵应返回 false")
	}
}

func TestParseLevel(t *testing.T) {
	for _, s := range []string{"l", " L ", "L"} {
		if got := ParseLevel(s); got != LevelL {
			t.Fatalf("ParseLevel(%q) = %v，期望 L", s, got)
		}
	}
	if ParseLevel("H") != LevelH || ParseLevel("q") != LevelQ || ParseLevel("") != LevelM || ParseLevel("?") != LevelM {
		t.Fatal("ParseLevel 映射不对")
	}
	if LevelQ.String() != "Q" || Level(9).String() != "M" {
		t.Fatal("Level.String 映射不对")
	}
}

// TestQRCodePNGEncodes 确认 PNG 尺寸 = (模块数 + 2*静区) * 像素比例。
func TestQRCodePNGEncodes(t *testing.T) {
	data, err := QRCodePNG("https://d.cn", LevelM, 4, 2)
	if err != nil {
		t.Fatalf("生成 PNG 失败: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("PNG 解码失败: %v", err)
	}
	want := (21 + 4) * 4
	if img.Bounds().Dx() != want || img.Bounds().Dy() != want {
		t.Fatalf("PNG 尺寸 = %d x %d，期望 %d", img.Bounds().Dx(), img.Bounds().Dy(), want)
	}
	// 左上角静区必须是白，中心附近的定位图形应该是深色。
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
		t.Fatal("静区应为白色")
	}
	if _, err := QRCodePNG(strings.Repeat("x", 3000), LevelL, 2, 0); err == nil {
		t.Fatal("超长内容应返回错误")
	}
}

func TestQRCodeNodeReadsTextFromChildren(t *testing.T) {
	n := QRCode("https://d.cn")
	if got := renderer.AttrString(n, "text"); got != "https://d.cn" {
		t.Fatalf("children 里的文本应转成 text 属性，实际 %q", got)
	}
	if len(n.Children) != 0 {
		t.Fatal("纯文本子节点不应留在 children 里参与布局")
	}
	p := QRCode(runtime.Props{Label: "https://d.cn"})
	if got := qrText(p); got != "https://d.cn" {
		t.Fatalf("label 兜底失败: %q", got)
	}
	if qrQuiet(p) != 4 {
		t.Fatal("默认静区应为 4")
	}
	p.Attrs["quiet"] = 1
	if qrQuiet(p) != 1 {
		t.Fatal("quiet 属性未生效")
	}
	if qrLevel(p) != LevelM {
		t.Fatal("未写 level 时应回退 M")
	}
	p.Attrs["level"] = "h"
	if qrLevel(p) != LevelH {
		t.Fatal("level 属性未生效")
	}
	// Level 类型的值也要认（QRCodeWith 之外的应用可能直接塞枚举）。
	p.Attrs["level"] = LevelQ
	if qrLevel(p) != LevelQ {
		t.Fatal("Level 枚举值未生效")
	}
	p.Attrs["level"] = Level(42)
	if qrLevel(p) != LevelM {
		t.Fatal("非法 Level 应回退 M")
	}
}

func TestMeasureQRCodeIsSquare(t *testing.T) {
	ctx := &mockContext{w: 400, h: 400}
	n := QRCode(runtime.Props{Style: runtime.Style{Width: 200, Height: 120}})
	w, h := measureQRCode(ctx, n, 400, 400, renderer.TextStyle{FontSize: 13}, theme.Light)
	if w != 120 || h != 120 {
		t.Fatalf("应按短边成正方形，实际 %v x %v", w, h)
	}
	// 只给 Height 时宽要跟着高走，不能退回可用宽度。
	n = QRCode(runtime.Props{Style: runtime.Style{Height: 90}})
	w, h = measureQRCode(ctx, n, 400, 400, renderer.TextStyle{FontSize: 13}, theme.Light)
	if w != 90 || h != 90 {
		t.Fatalf("只有高时按高成正方形，实际 %v x %v", w, h)
	}
	n = QRCode()
	w, h = measureQRCode(ctx, n, 240, 0, renderer.TextStyle{FontSize: 13}, theme.Light)
	if w != 240 || h != 240 {
		t.Fatalf("无显式尺寸时应跟可用宽度，实际 %v x %v", w, h)
	}
	w, h = measureQRCode(ctx, QRCode(), 0, 0, renderer.TextStyle{FontSize: 13}, theme.Light)
	if w != 160 || h != 160 {
		t.Fatalf("无约束时应回退 160，实际 %v x %v", w, h)
	}
}

// TestQuietAttributeVariants 应用层可能塞 int / float32（JSON 回来的数字），都要认。
func TestQuietAttributeVariants(t *testing.T) {
	for _, tc := range []struct {
		v    any
		want int
	}{
		{2, 2},
		{int32(3), 3},
		{float32(5), 5},
		{float64(6), 6},
		{99, 16},
		{-3, 0},
		{"4", 4},
	} {
		n := QRCode()
		n.Attrs["quiet"] = tc.v
		if got := qrQuiet(n); got != tc.want {
			t.Fatalf("quiet=%v 得到 %d，期望 %d", tc.v, got, tc.want)
		}
	}
}

// TestPaintQRCodeDrawsModulesAndQuietZone 深色模块要落在静区之后，并按格对齐：
// 174 / (21 + 2*4) = 6 DIP 一格，第一个定位图形从 (24,24) 起、宽 7 格。
func TestPaintQRCodeDrawsModulesAndQuietZone(t *testing.T) {
	ctx := &mockContext{w: 200, h: 200}
	n := QRCode(runtime.Props{Style: runtime.Style{Width: 174, Height: 174}}, "https://d.cn")
	n.X, n.Y, n.W, n.H = 0, 0, 174, 174
	paintQRCode(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if ctx.rects < 2 {
		t.Fatalf("应至少画底色 + 若干深色模块，实际 %d", ctx.rects)
	}
	found := false
	for _, f := range ctx.fills[1:] {
		if near(f.x, 24) && near(f.y, 24) && near(f.w, 42) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("未在静区之后找到 7 格宽的定位图形块: %+v", ctx.fills[:minInt(len(ctx.fills), 4)])
	}
	for _, f := range ctx.fills[1:] {
		if f.x < 24-0.01 || f.y < 24-0.01 {
			t.Fatalf("深色模块画进静区了: %+v", f)
		}
	}
}

func TestPaintQRCodeBadgeAndFallback(t *testing.T) {
	ctx := &mockContext{w: 240, h: 240}
	n := QRCode(runtime.Props{Style: runtime.Style{Width: 220, Height: 220}, Badge: "d.cn"}, "https://d.cn")
	n.X, n.Y, n.W, n.H = 0, 0, 220, 220
	paintQRCode(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if len(ctx.texts) != 1 || ctx.texts[0] != "d.cn" {
		t.Fatalf("徽标文字未画出: %v", ctx.texts)
	}
	// 太小的二维码不画徽标，否则会盖掉太多模块。
	ctx = &mockContext{w: 80, h: 80}
	n.W, n.H = 80, 80
	paintQRCode(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if len(ctx.texts) != 0 {
		t.Fatal("小尺寸不应画徽标")
	}
	// 编码失败时只画底色，不 panic。
	ctx = &mockContext{w: 100, h: 100}
	n = QRCode(runtime.Props{Style: runtime.Style{Width: 100, Height: 100}}, strings.Repeat("x", 3000))
	n.X, n.Y, n.W, n.H = 0, 0, 100, 100
	paintQRCode(ctx, n, renderer.TextStyle{FontSize: 13}, theme.Light)
	if ctx.rects != 1 {
		t.Fatalf("失败时只应画底色，实际 %d 次填充", ctx.rects)
	}
}

func near(v, want float32) bool {
	d := v - want
	return d > -0.01 && d < 0.01
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
