package components

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/goedui/goed/ui/core"
	"github.com/goedui/goed/ui/theme"
)

// maxImageFileBytes 图片文件读取上限（128MB），防止异常大文件耗尽内存。
const maxImageFileBytes = 128 << 20

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// ImageView 只读图片预览：PNG/JPEG/GIF 解码，等比缩放居中显示。
// 缩放在 Go 侧完成（双线性，预乘 alpha 域），GDI 仅做 1:1 块传输，
// 规避部分驱动 StretchDIBits 大图缩放静默失败的问题。缩放结果与
// 背景合成一起按（背景色, 目标尺寸）缓存。
type ImageView struct {
	Path      string
	Format    string // PNG / JPEG / GIF
	SrcW      int32
	SrcH      int32
	SizeBytes int64

	pixels     []byte // 预乘 alpha 的 RGBA（image.NewRGBA 保证 4 字节紧凑）
	composited []byte // 缩放 + 背景合成后的不透明 BGRA（平台绘制格式）
	cacheW     int32
	cacheH     int32
	cacheBg    uint32
	cacheOK    bool
}

func NewImageView() *ImageView { return &ImageView{} }

// Reset 释放像素与缓存缓冲区，恢复到未加载状态。切换文档或加载失败时
// 调用，确保 Ready() 与缓存状态一致，避免旧数据污染下一次渲染。
func (v *ImageView) Reset() {
	v.pixels = nil
	v.composited = nil
	v.cacheW, v.cacheH, v.cacheBg = 0, 0, 0
	v.cacheOK = false
	v.Path, v.Format = "", ""
	v.SrcW, v.SrcH, v.SizeBytes = 0, 0, 0
}

func (v *ImageView) Ready() bool { return v.pixels != nil }

// Load 读取并解码图片文件。格式经魔数嗅探（避免仅凭扩展名误判）。
// 进入即 Reset：失败时不残留上一次加载的像素与合成缓存，保证 Ready()
// 与实际状态一致（修复重复打开/回退十六进制预览时的黑屏与状态污染）。
func (v *ImageView) Load(path string) error {
	v.Reset()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageFileBytes+1))
	if err != nil {
		return err
	}
	return v.LoadBytes(path, data)
}

// LoadBytes 从内存字节加载图片（供网络响应/内嵌资源等无文件场景复用）。
// path 仅作展示标识。进入即 Reset，语义与 Load 一致。
func (v *ImageView) LoadBytes(path string, data []byte) error {
	v.Reset()
	if int64(len(data)) > maxImageFileBytes {
		return fmt.Errorf("图片超过 %d MB，不支持预览", maxImageFileBytes>>20)
	}
	img, format, err := decodeImage(data)
	if err != nil {
		return err
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	v.pixels = rgba.Pix
	v.SrcW, v.SrcH = int32(b.Dx()), int32(b.Dy())
	v.Format = format
	v.Path = path
	v.SizeBytes = int64(len(data))
	// 彻底失效合成缓存：旧缓冲区必须丢弃，不能仅置 cacheOK=false，
	// 否则同尺寸重开时可能复用旧图数据。
	v.composited = nil
	v.cacheW, v.cacheH, v.cacheBg = 0, 0, 0
	v.cacheOK = false
	return nil
}

// decodeImage 按魔数分发到对应解码器。
func decodeImage(data []byte) (image.Image, string, error) {
	switch {
	case len(data) >= len(pngMagic) && bytes.Equal(data[:len(pngMagic)], pngMagic):
		img, err := png.Decode(bytes.NewReader(data))
		return img, "PNG", err
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		img, err := jpeg.Decode(bytes.NewReader(data))
		return img, "JPEG", err
	case len(data) >= 6 && string(data[:3]) == "GIF":
		img, err := gif.Decode(bytes.NewReader(data))
		return img, "GIF", err
	}
	return nil, "", fmt.Errorf("不支持的图片格式（支持 PNG/JPEG/GIF）")
}

// resampleBilinear 双线性重采样 RGBA 像素（非预乘域，合成时再与背景混合）。
// 采样网格：1:1 时逐像素精确映射；放大时采样点对齐源像素中点，
// 使过渡像素为相邻源像素的等权混合；缩小时采用标准中心对齐网格。
func resampleBilinear(src []byte, sw, sh, dw, dh int32) []byte {
	dst := make([]byte, int(dw)*int(dh)*4)
	if sw <= 0 || sh <= 0 {
		return dst
	}
	sx := float32(sw) / float32(dw)
	sy := float32(sh) / float32(dh)
	// sampleCoord 把目标坐标 d 映射为源坐标对 (c0,c1) 与权重 t：
	// f 为浮点采样位置，c0=floor(f)，t=f-c0。
	sampleCoord := func(d int32, scale float32, limit int32) (c0, c1 int32, t float32) {
		var f float32
		if scale <= 1 {
			// 1:1 / 放大：右缘对齐映射 f=(d+1)*scale-1，使 1:1 精确
			// 恒等，且放大过渡像素恰为相邻源像素的等权混合。
			f = (float32(d)+1)*scale - 1
		} else {
			// 缩小：中心对齐降采样网格。
			f = (float32(d)+0.5)*scale - 0.5
		}
		if f < 0 {
			f = 0
		}
		c0 = int32(f)
		if c0 > limit-1 {
			c0 = limit - 1
		}
		c1 = c0 + 1
		if c1 > limit-1 {
			c1 = limit - 1
		}
		t = f - float32(c0)
		if t < 0 {
			t = 0
		}
		return c0, c1, t
	}
	for dy := int32(0); dy < dh; dy++ {
		y0, y1, ty := sampleCoord(dy, sy, sh)
		for dx := int32(0); dx < dw; dx++ {
			x0, x1, tx := sampleCoord(dx, sx, sw)
			// 双线性混合 4 个源像素（逐通道）。插值在 float32 域进行，
			// 避免 uint32 相减下溢造成像素错误。
			di := (dy*dw + dx) * 4
			for c := int32(0); c < 4; c++ {
				p00 := float32(src[(y0*sw+x0)*4+c])
				p01 := float32(src[(y0*sw+x1)*4+c])
				p10 := float32(src[(y1*sw+x0)*4+c])
				p11 := float32(src[(y1*sw+x1)*4+c])
				top := p00 + (p01-p00)*tx
				bot := p10 + (p11-p10)*tx
				out := top + (bot-top)*ty
				if out < 0 {
					out = 0
				} else if out > 255 {
					out = 255
				}
				dst[di+c] = byte(out + 0.5)
			}
		}
	}
	return dst
}

// ensureComposite 把源像素缩放到 (dw,dh) 并与背景色合成为不透明 BGRA。
// 结果按（背景色, 目标尺寸）缓存；透明区域显示为标准灰白棋盘格，
// 避免透明底图片在深色编辑器背景上"看不见"。
func (v *ImageView) ensureComposite(bg uint32, dw, dh int32) {
	if v.cacheOK && v.cacheBg == bg && v.cacheW == dw && v.cacheH == dh {
		return
	}
	if dw <= 0 || dh <= 0 {
		return
	}
	scaled := resampleBilinear(v.pixels, v.SrcW, v.SrcH, dw, dh)
	buf := make([]byte, len(scaled))
	for i := 0; i+3 < len(scaled); i += 4 {
		px := int32(i/4) % dw
		py := int32(i/4) / dw
		// 8px 棋盘格：(0,0) 起始为深色格 (204,204,204)，相邻格浅色 (240,240,240)。
		var cellR, cellG, cellB uint32 = 204, 204, 204
		if ((px/8)+(py/8))%2 == 1 {
			cellR, cellG, cellB = 240, 240, 240
		}
		// 非预乘 alpha 与背景合成：out = src + cell*(1-a)。
		// scaled 为非预乘域像素（draw.Draw 到 image.RGBA 的产物）。
		r, g, b, a := uint32(scaled[i]), uint32(scaled[i+1]), uint32(scaled[i+2]), uint32(scaled[i+3])
		inv := 255 - a
		buf[i] = byte((b*255 + cellB*inv) / 255)   // B
		buf[i+1] = byte((g*255 + cellG*inv) / 255) // G
		buf[i+2] = byte((r*255 + cellR*inv) / 255) // R
		buf[i+3] = 0xFF
	}
	v.composited, v.cacheW, v.cacheH = buf, dw, dh
	v.cacheBg, v.cacheOK = bg, true
}

// fitRect 计算等比缩放后的居中矩形（最多放大 4 倍）。
func (v *ImageView) fitRect(x, y, w, h int32) core.Rect {
	availW, availH := w-16, h-16
	if availW < 1 || availH < 1 || v.SrcW < 1 || v.SrcH < 1 {
		return core.Rect{}
	}
	scale := float32(availW) / float32(v.SrcW)
	if sh := float32(availH) / float32(v.SrcH); sh < scale {
		scale = sh
	}
	if scale > 4 {
		scale = 4
	}
	if scale <= 0 {
		scale = 0.01
	}
	dw := int32(float32(v.SrcW) * scale)
	dh := int32(float32(v.SrcH) * scale)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	return core.Rect{X: x + (w-dw)/2, Y: y + (h-dh)/2, W: dw, H: dh}
}

func (v *ImageView) Render(ctx core.DrawingContext, x, y, w, h int32) {
	if !v.Ready() {
		return
	}
	th := theme.GetTheme()
	ctx.DrawRect(x, y, w, h, th.EditorBg, true)
	if ic, ok := ctx.(core.ImageDrawingContext); ok {
		if r := v.fitRect(x, y, w, h); r.W > 0 && r.H > 0 {
			// 缩放在 Go 侧完成，绘制为 1:1 块传输（规避驱动缩放怪癖）。
			v.ensureComposite(th.EditorBg, r.W, r.H)
			ic.DrawImageBGRA(v.composited, r.W, r.H, r.X, r.Y, r.W, r.H)
		}
	} else {
		// 平台无位图绘制能力：文字回退。
		ctx.SetFont(th.UIFontFamily, th.FontBody())
		ctx.DrawText(x+16, y+24, "当前平台不支持图片预览", th.TextSecondary)
	}
}
