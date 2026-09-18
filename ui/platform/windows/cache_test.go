//go:build windows

package windows

import (
	"image"
	"image/color"
	"testing"
)

func TestTouchKeyMovesExistingToEnd(t *testing.T) {
	order := []int{1, 2, 3}
	got := touchKey(order, 1)
	if len(got) != 3 || got[2] != 1 || got[0] != 2 {
		t.Fatalf("touchKey = %v", got)
	}
	got = touchKey(got, 4)
	if len(got) != 4 || got[3] != 4 {
		t.Fatalf("append = %v", got)
	}
}

func TestQuantizeFontSize(t *testing.T) {
	if got := quantizeFontSize(13.000001); got != 13 {
		t.Fatalf("quantize 13.000001 = %v", got)
	}
	if got := quantizeFontSize(13.4); got != 13.5 {
		t.Fatalf("quantize 13.4 = %v", got)
	}
	if got := quantizeFontSize(0); got != 0 {
		t.Fatalf("quantize 0 = %v", got)
	}
}

func TestRememberFormatEvictsOldest(t *testing.T) {
	oldRelease := releaseCOM
	releaseCOM = func(com) {}
	old := maxTextFormats
	maxTextFormats = 2
	t.Cleanup(func() {
		maxTextFormats = old
		releaseCOM = oldRelease
	})

	c := &d2dContext{formats: map[formatKey]com{}}
	k1 := formatKey{family: "a", size: 12}
	k2 := formatKey{family: "b", size: 12}
	k3 := formatKey{family: "c", size: 12}
	c.rememberFormat(k1, 1)
	c.rememberFormat(k2, 2)
	c.rememberFormat(k3, 3)
	if len(c.formats) != 2 {
		t.Fatalf("formats = %d, want 2", len(c.formats))
	}
	if _, ok := c.formats[k1]; ok {
		t.Fatal("oldest format should be evicted")
	}
	if _, ok := c.formats[k3]; !ok {
		t.Fatal("newest format should remain")
	}
}

func TestRememberBitmapEvictsByCount(t *testing.T) {
	oldRelease := releaseCOM
	releaseCOM = func(com) {}
	oldN, oldB := maxBitmaps, maxBitmapBytes
	maxBitmaps, maxBitmapBytes = 2, 1<<30
	t.Cleanup(func() {
		maxBitmaps, maxBitmapBytes = oldN, oldB
		releaseCOM = oldRelease
	})

	c := &d2dContext{images: map[uint64]cachedBitmap{}}
	c.rememberBitmap(1, 11, 100)
	c.rememberBitmap(2, 22, 100)
	c.rememberBitmap(3, 33, 100)
	if len(c.images) != 2 {
		t.Fatalf("images = %d, want 2", len(c.images))
	}
	if _, ok := c.images[1]; ok {
		t.Fatal("oldest bitmap should be evicted")
	}
	if c.imageBytes != 200 {
		t.Fatalf("imageBytes = %d", c.imageBytes)
	}
}

func TestCopyImageBGRAReusesBuffer(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	src.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	dst := make([]byte, 8)
	copyImageBGRA(dst, src)
	if dst[0] != 0 || dst[1] != 0 || dst[2] != 255 || dst[3] != 255 {
		t.Fatalf("pixel0 = %v, want BGRA red", dst[:4])
	}
	if dst[4] != 0 || dst[5] != 255 || dst[6] != 0 || dst[7] != 255 {
		t.Fatalf("pixel1 = %v, want BGRA green", dst[4:])
	}
}
