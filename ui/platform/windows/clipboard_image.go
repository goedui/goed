//go:build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"unsafe"
)

const (
	cfDIB = 8
)

// ClipboardImage 读取剪贴板里的 DIB，转成 PNG。没有图片时 ok=false。
func ClipboardImage() ([]byte, bool) {
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return nil, false
	}
	defer procCloseClipboard.Call()
	avail, _, _ := procIsClipboardFormatAvailable.Call(cfDIB)
	if avail == 0 {
		return nil, false
	}
	h, _, _ := procGetClipboardData.Call(cfDIB)
	if h == 0 {
		return nil, false
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return nil, false
	}
	defer procGlobalUnlock.Call(h)
	size, _, _ := procGlobalSize.Call(h)
	if size < 40 {
		return nil, false
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(p)), size)
	buf := make([]byte, len(raw))
	copy(buf, raw)
	pngBytes, err := dibToPNG(buf)
	if err != nil || len(pngBytes) == 0 {
		return nil, false
	}
	return pngBytes, true
}

func dibToPNG(dib []byte) ([]byte, error) {
	if len(dib) < 40 {
		return nil, fmt.Errorf("dib too short")
	}
	header := binary.LittleEndian.Uint32(dib[0:4])
	if int(header) > len(dib) || header < 40 {
		return nil, fmt.Errorf("bad dib header")
	}
	w := int(int32(binary.LittleEndian.Uint32(dib[4:8])))
	h := int(int32(binary.LittleEndian.Uint32(dib[8:12])))
	bottomUp := h > 0
	if h < 0 {
		h = -h
	}
	if w <= 0 || h <= 0 || w > 16000 || h > 16000 {
		return nil, fmt.Errorf("bad dib size")
	}
	bpp := binary.LittleEndian.Uint16(dib[14:16])
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("unsupported dib bpp")
	}
	pixels := dib[header:]
	stride := ((w*int(bpp) + 31) / 32) * 4
	if len(pixels) < stride*h {
		return nil, fmt.Errorf("dib pixels truncated")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcY := y
		if bottomUp {
			srcY = h - 1 - y
		}
		row := pixels[srcY*stride : srcY*stride+stride]
		for x := 0; x < w; x++ {
			i := x * int(bpp) / 8
			o := img.PixOffset(x, y)
			img.Pix[o+2] = row[i]
			img.Pix[o+1] = row[i+1]
			img.Pix[o] = row[i+2]
			img.Pix[o+3] = 255
			if bpp == 32 && row[i+3] != 0 {
				img.Pix[o+3] = row[i+3]
			}
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
