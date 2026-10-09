package renderer

import "math"

// BlurPremultiplied 对预乘 BGRA 缓冲做可分离高斯模糊，返回新缓冲。
// sigma 是标准差（像素）。sigma<=0 时原样拷贝。
// 这是没有 GPU 模糊时的阴影核：调用方把结果上传成位图即可。
func BlurPremultiplied(pix []byte, w, h int, sigma float32) []byte {
	n := w * h * 4
	if w < 1 || h < 1 || len(pix) < n {
		return nil
	}
	out := make([]byte, n)
	copy(out, pix[:n])
	if sigma < 0.35 {
		return out
	}
	radius := int(sigma*3 + 0.5)
	if radius < 1 {
		radius = 1
	}
	if radius > 64 {
		radius = 64
	}
	k := gaussKernel(radius, sigma)
	tmp := make([]byte, n)
	blurAxis(out, tmp, w, h, k, true)
	blurAxis(tmp, out, w, h, k, false)
	return out
}

func gaussKernel(radius int, sigma float32) []float32 {
	k := make([]float32, radius*2+1)
	s2 := 2 * sigma * sigma
	var sum float32
	for i := -radius; i <= radius; i++ {
		w := float32(math.Exp(-float64(i*i) / float64(s2)))
		k[i+radius] = w
		sum += w
	}
	if sum <= 0 {
		return k
	}
	for i := range k {
		k[i] /= sum
	}
	return k
}

func blurAxis(src, dst []byte, w, h int, k []float32, horizontal bool) {
	radius := len(k) / 2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var b, g, r, a float32
			for i, wgt := range k {
				sx, sy := x, y
				if horizontal {
					sx = x + i - radius
					if sx < 0 {
						sx = 0
					} else if sx >= w {
						sx = w - 1
					}
				} else {
					sy = y + i - radius
					if sy < 0 {
						sy = 0
					} else if sy >= h {
						sy = h - 1
					}
				}
				o := (sy*w + sx) * 4
				b += float32(src[o]) * wgt
				g += float32(src[o+1]) * wgt
				r += float32(src[o+2]) * wgt
				a += float32(src[o+3]) * wgt
			}
			o := (y*w + x) * 4
			dst[o] = sat8(b)
			dst[o+1] = sat8(g)
			dst[o+2] = sat8(r)
			dst[o+3] = sat8(a)
		}
	}
}

func sat8(v float32) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}
