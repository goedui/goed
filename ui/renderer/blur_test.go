package renderer

import "testing"

func TestBlurSpreadsAndKeepsMass(t *testing.T) {
	const w, h = 21, 21
	src := make([]byte, w*h*4)
	o := (10*w + 10) * 4
	src[o], src[o+1], src[o+2], src[o+3] = 0, 0, 255, 255
	out := BlurPremultiplied(src, w, h, 2)
	if out == nil || len(out) != len(src) {
		t.Fatal("模糊应返回等长缓冲")
	}
	center := (10*w + 10) * 4
	edge := (10*w + 0) * 4
	if out[center+3] <= out[edge+3] {
		t.Fatalf("中心应比边缘更实：center=%d edge=%d", out[center+3], out[edge+3])
	}
	// 左右对称：高斯核是偶函数。
	left := (10*w + 8) * 4
	right := (10*w + 12) * 4
	if out[left+3] != out[right+3] {
		t.Fatalf("模糊应左右对称：%d vs %d", out[left+3], out[right+3])
	}
	var sum int
	for i := 3; i < len(out); i += 4 {
		sum += int(out[i])
	}
	if sum < 200 || sum > 280 {
		t.Fatalf("alpha 质量应接近 255，实际 %d", sum)
	}
}
