package components

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// encodeTestPNG 生成 2×2 半透明测试 PNG（左上不透明红，右上半透明绿，
// 下排透明）。
func encodeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	img.SetRGBA(1, 0, color.RGBA{G: 255, A: 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestImageViewLoadDecode 验证魔数嗅探解码与字段填充。
func TestImageViewLoadDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tiny.png")
	if err := os.WriteFile(path, encodeTestPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewImageView()
	if err := v.Load(path); err != nil {
		t.Fatal(err)
	}
	if v.Format != "PNG" || v.SrcW != 2 || v.SrcH != 2 {
		t.Fatalf("format=%s size=%dx%d", v.Format, v.SrcW, v.SrcH)
	}
	if !v.Ready() || len(v.pixels) != 2*2*4 {
		t.Fatalf("pixels = %d bytes", len(v.pixels))
	}
}

// TestImageViewLoadRejectsUnknown 验证非图片内容报错。
func TestImageViewLoadRejectsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.png")
	if err := os.WriteFile(path, []byte("not an image at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewImageView()
	if err := v.Load(path); err == nil {
		t.Fatal("expected decode error")
	}
	if v.Ready() {
		t.Fatal("component should stay empty after failure")
	}
}

// TestImageViewComposite 验证 alpha 与背景合成（BGRA 字节序、不透明输出）。
func TestImageViewComposite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tiny.png")
	if err := os.WriteFile(path, encodeTestPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewImageView()
	if err := v.Load(path); err != nil {
		t.Fatal(err)
	}
	bg := uint32(0x00201F1F) // EditorBg 类深色 COLORREF
	v.ensureComposite(bg, 2, 2)

	// 像素 0 (0,0)：alpha=255 → 纯红（BGRA = 00,00,FF,FF）。
	i := 0
	if v.composited[i] != 0 || v.composited[i+1] != 0 || v.composited[i+2] != 255 {
		t.Fatalf("opaque red = %v", v.composited[i:i+4])
	}
	// 像素 1 (1,0)：绿色 a=128 → 与棋盘格 (204,204,204) 各半（±1 舍入）。
	i = 4
	wantG := (uint32(v.pixels[i+1])*255 + 204*(255-uint32(v.pixels[i+3]))) / 255
	gotG := uint32(v.composited[i+1])
	if gotG < wantG-1 || gotG > wantG+1 {
		t.Fatalf("half green = %d, want %d (±1)", gotG, wantG)
	}
	// 像素 2 (0,1)：全透明 → 棋盘格深色格 (204,204,204)。
	i = 8
	if v.composited[i] != 204 || v.composited[i+1] != 204 || v.composited[i+2] != 204 {
		t.Fatalf("transparent on dark cell = %v", v.composited[i:i+4])
	}
	// 缓存：同参数二次调用不重建。
	buf := v.composited
	v.ensureComposite(bg, 2, 2)
	if &buf[0] != &v.composited[0] {
		t.Fatal("cache should be reused for same bg/size")
	}
	// 尺寸变化：重建。
	v.ensureComposite(bg, 4, 4)
	if len(v.composited) != 4*4*4 {
		t.Fatalf("resized cache = %d bytes", len(v.composited))
	}
}

// TestResampleBilinear 验证双线性缩放：1:1 恒等与 2 倍放大取值。
func TestResampleBilinear(t *testing.T) {
	src := []byte{
		0, 0, 0, 255, 200, 200, 200, 255, // 黑、浅灰
		200, 200, 200, 255, 0, 0, 0, 255, // 浅灰、黑
	}
	id := resampleBilinear(src, 2, 2, 2, 2)
	for i := range src {
		if id[i] != src[i] {
			t.Fatalf("identity mismatch at %d: %d != %d", i, id[i], src[i])
		}
	}
	up := resampleBilinear(src, 2, 2, 4, 4)
	if len(up) != 4*4*4 {
		t.Fatalf("upscaled len = %d", len(up))
	}
	mid := up[(2*4+2)*4] // (2,2)：四角混合，期望 ≈100（±2 舍入）
	if mid > 102 || mid < 98 {
		t.Fatalf("center ≈ 100 expected, got %d", mid)
	}
}

// TestImageViewFitRect 验证等比缩放居中（缩小与放大上限）。
func TestImageViewFitRect(t *testing.T) {
	v := NewImageView()
	v.SrcW, v.SrcH = 100, 50
	r := v.fitRect(0, 0, 400, 300)
	if r.W != 384 || r.H != 192 || r.X != 8 || r.Y != 54 {
		t.Fatalf("down fit = %+v", r)
	}
	// 小图标放大上限 4 倍。
	v.SrcW, v.SrcH = 8, 8
	r = v.fitRect(0, 0, 400, 300)
	if r.W != 32 || r.H != 32 {
		t.Fatalf("up fit = %+v", r)
	}
}
