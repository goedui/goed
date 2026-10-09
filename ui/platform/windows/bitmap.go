//go:build windows

package windows

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"runtime"
	"syscall"
	"unsafe"
)

func (c *d2dContext) bitmap(data []byte) com {
	if c.rt == 0 || len(data) == 0 {
		return 0
	}
	key := hashBytes(data)
	if c.images == nil {
		c.images = make(map[uint64]cachedBitmap)
	}
	if cached, ok := c.images[key]; ok {
		c.imageOrder = touchKey(c.imageOrder, key)
		return cached.bmp
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w < 1 || h < 1 {
		return 0
	}
	needed := w * h * 4
	bgra := c.scratch
	if needed > 8<<20 {
		bgra = make([]byte, needed)
	} else {
		if cap(c.scratch) < needed {
			c.scratch = make([]byte, needed)
		}
		bgra = c.scratch
	}
	bgra = bgra[:needed]
	copyImageBGRA(bgra, img)
	bmp := c.createBitmap(w, h, bgra)
	if bmp == 0 {
		return 0
	}
	c.rememberBitmap(key, bmp, needed)
	return bmp
}

// createBitmap 从预乘 BGRA 像素创建位图：返回时像素已被拷贝，bgra 缓冲即可复用。
// 失败返回 0。
func (c *d2dContext) createBitmap(w, h int, bgra []byte) com {
	if c.rt == 0 || w < 1 || h < 1 || len(bgra) < w*h*4 {
		return 0
	}
	props := d2d1BitmapProperties{
		pixelFormat: d2d1PixelFormat{
			format:    dxgiFormatB8G8R8A8Unorm,
			alphaMode: d2d1AlphaModePremultiplied,
		},
		dpiX: c.dpi,
		dpiY: c.dpi,
	}
	if props.dpiX <= 0 {
		props.dpiX = 96
		props.dpiY = 96
	}
	size := uintptr(uint32(w)) | uintptr(uint32(h))<<32
	var bmp com
	hr, _, _ := syscall.SyscallN(
		c.rt.slot(idxCreateBitmap),
		uintptr(c.rt),
		size,
		uintptr(unsafe.Pointer(&bgra[0])),
		uintptr(w*4),
		uintptr(unsafe.Pointer(&props)),
		uintptr(unsafe.Pointer(&bmp)),
	)
	runtime.KeepAlive(bgra)
	runtime.KeepAlive(props)
	if hresult(hr).failed() || bmp == 0 {
		return 0
	}
	return bmp
}

func copyImageBGRA(dst []byte, img image.Image) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	switch src := img.(type) {
	case *image.RGBA:
		for y := 0; y < h; y++ {
			row := src.Pix[y*src.Stride : y*src.Stride+w*4]
			out := dst[y*w*4 : (y+1)*w*4]
			for x := 0; x < w; x++ {
				i := x * 4
				r, g, b, a := row[i], row[i+1], row[i+2], row[i+3]
				out[i], out[i+1], out[i+2], out[i+3] = b, g, r, a
			}
		}
	case *image.NRGBA:
		for y := 0; y < h; y++ {
			row := src.Pix[y*src.Stride : y*src.Stride+w*4]
			out := dst[y*w*4 : (y+1)*w*4]
			for x := 0; x < w; x++ {
				i := x * 4
				r, g, b, a := row[i], row[i+1], row[i+2], row[i+3]
				out[i], out[i+1], out[i+2], out[i+3] = b, g, r, a
			}
		}
	default:
		tmp := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(tmp, tmp.Bounds(), img, bounds.Min, draw.Src)
		copyImageBGRA(dst, tmp)
	}
}

func (c *d2dContext) releaseImages() {
	for k, cached := range c.images {
		release(cached.bmp)
		delete(c.images, k)
	}
	c.imageOrder = nil
	c.imageBytes = 0
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}
