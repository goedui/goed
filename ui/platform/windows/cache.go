//go:build windows

package windows

// 文本格式与位图缓存上限。超出后按 LRU 释放 COM 对象，
// 避免 formats / images 随运行时间单向增长。
var (
	maxTextFormats = 64
	maxBitmaps     = 16
	maxBitmapBytes = 32 << 20
	releaseCOM     = release
)

type cachedBitmap struct {
	bmp   com
	bytes int
}

func touchKey[K comparable](order []K, key K) []K {
	for i, k := range order {
		if k == key {
			return append(order[:i], append(order[i+1:], key)...)
		}
	}
	return append(order, key)
}

func (c *d2dContext) rememberFormat(key formatKey, f com) {
	if c.formats == nil {
		c.formats = make(map[formatKey]com)
	}
	if old, ok := c.formats[key]; ok && old != 0 && old != f {
		releaseCOM(old)
	}
	c.formats[key] = f
	c.formatOrder = touchKey(c.formatOrder, key)
	for len(c.formatOrder) > maxTextFormats {
		oldest := c.formatOrder[0]
		c.formatOrder = c.formatOrder[1:]
		if oldest == key {
			continue
		}
		if old, ok := c.formats[oldest]; ok {
			releaseCOM(old)
			delete(c.formats, oldest)
		}
	}
}

func (c *d2dContext) rememberBitmap(key uint64, bmp com, bytes int) {
	if c.images == nil {
		c.images = make(map[uint64]cachedBitmap)
	}
	if old, ok := c.images[key]; ok && old.bmp != 0 && old.bmp != bmp {
		releaseCOM(old.bmp)
		c.imageBytes -= old.bytes
		if c.imageBytes < 0 {
			c.imageBytes = 0
		}
	}
	c.images[key] = cachedBitmap{bmp: bmp, bytes: bytes}
	c.imageBytes += bytes
	c.imageOrder = touchKey(c.imageOrder, key)
	for (len(c.imageOrder) > maxBitmaps || c.imageBytes > maxBitmapBytes) && len(c.imageOrder) > 1 {
		oldest := c.imageOrder[0]
		c.imageOrder = c.imageOrder[1:]
		if oldest == key {
			c.imageOrder = append(c.imageOrder, oldest)
			if len(c.imageOrder) <= 1 {
				break
			}
			continue
		}
		if old, ok := c.images[oldest]; ok {
			releaseCOM(old.bmp)
			c.imageBytes -= old.bytes
			if c.imageBytes < 0 {
				c.imageBytes = 0
			}
			delete(c.images, oldest)
		}
	}
}

func quantizeFontSize(size float32) float32 {
	if size <= 0 {
		return size
	}
	// 收成 0.5 DIP 一档，避免 13.000001 这类键撑爆格式缓存。
	return float32(int(size*2+0.5)) / 2
}
