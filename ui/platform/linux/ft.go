//go:build linux

package linux

/*
#cgo LDFLAGS: -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

static void *ftlib;

int ft_load(void) {
	ftlib = dlopen("libfreetype.so.6", RTLD_NOW | RTLD_GLOBAL);
	if (!ftlib) {
		ftlib = dlopen("libfreetype.so", RTLD_NOW | RTLD_GLOBAL);
	}
	return ftlib != 0;
}

void *ft_sym(const char *n) { return ftlib ? dlsym(ftlib, n) : 0; }

// FreeType 的结构体只按公开头文件抄下用得到的字段前缀，前面字段的偏移由 C 编译器
// 算（不在 Go 里硬编码）；字段顺序与 freetype/fttypes.h、freetype.h 一致，布局 ABI 稳定。

typedef struct { void *data; void (*finalizer)(void *); } ft_generic;

typedef struct {
	long width;
	long height;
	long hori_bearing_x;
	long hori_bearing_y;
	long hori_advance;
	long vert_bearing_x;
	long vert_bearing_y;
	long vert_advance;
} ft_glyph_metrics;

typedef struct {
	unsigned int rows;
	unsigned int width;
	int pitch;
	unsigned char *buffer;
	unsigned short num_grays;
	unsigned char pixel_mode;
	unsigned char palette_mode;
	void *palette;
} ft_bitmap;

typedef struct {
	void *library;
	void *face;
	void *next;
	unsigned int reserved;
	ft_generic generic;
	ft_glyph_metrics metrics;
	long linear_hori_advance;
	long linear_vert_advance;
	long advance_x;
	long advance_y;
	int format;
	ft_bitmap bitmap;
	int bitmap_left;
	int bitmap_top;
} ft_slot;

typedef struct { long x_min, y_min, x_max, y_max; } ft_bbox;

typedef struct {
	long num_faces;
	long face_index;
	long face_flags;
	long style_flags;
	long num_glyphs;
	char *family_name;
	char *style_name;
	int num_fixed_sizes;
	void *available_sizes;
	int num_charmaps;
	void *charmaps;
	ft_generic generic;
	ft_bbox bbox;
	unsigned short units_per_em;
	short ascender;
	short descender;
	short height;
	short max_advance_width;
	short max_advance_height;
	short underline_position;
	short underline_thickness;
	void *glyph;
	void *size;
	void *charmap;
} ft_face;

typedef struct {
	unsigned short x_ppem;
	unsigned short y_ppem;
	long x_scale;
	long y_scale;
	long ascender;
	long descender;
	long height;
	long max_advance;
} ft_size_metrics;

typedef struct {
	void *face;
	ft_generic generic;
	ft_size_metrics metrics;
} ft_size;

// 下面这些包装把函数指针转型后再调，调用方只要传 dlsym 拿到的地址；
// 创建出来的对象按 void* 返回，Go 那边不用 uintptr 转来转去。

void *ft_init(void *fn) {
	void *lib = 0;
	if (((int (*)(void **)) fn)(&lib) != 0) return 0;
	return lib;
}

void *ft_new_face(void *fn, void *lib, const char *path, long index) {
	void *face = 0;
	if (((int (*)(void *, const char *, long, void **)) fn)(lib, path, index, &face) != 0) return 0;
	return face;
}

int ft_set_pixel_sizes(void *fn, void *face, unsigned int w, unsigned int h) {
	return ((int (*)(void *, unsigned int, unsigned int)) fn)(face, w, h);
}

int ft_select_charmap(void *fn, void *face, uint32_t enc) {
	return ((int (*)(void *, uint32_t)) fn)(face, enc);
}

unsigned int ft_char_index(void *fn, void *face, unsigned long ch) {
	return ((unsigned int (*)(void *, unsigned long)) fn)(face, ch);
}

int ft_load_char(void *fn, void *face, unsigned long ch, int32_t flags) {
	return ((int (*)(void *, unsigned long, int32_t)) fn)(face, ch, flags);
}

// ft_size_metrics_get 取当前字号的 ascent / height（像素）。没设过字号返回 0。
int ft_size_metrics_get(void *face, int *ascent, int *height) {
	ft_size *sz = (ft_size *) ((ft_face *) face)->size;
	if (!sz) return 0;
	*ascent = (int) (sz->metrics.ascender >> 6);
	*height = (int) (sz->metrics.height >> 6);
	return 1;
}

// ft_glyph_out 把 FT_Load_Char 之后的字形信息摊平，省得在 Go 里读 C 结构体。
typedef struct {
	int width;
	int height;
	int pitch;
	int left;
	int top;
	int advance;
	int mode;
	unsigned char *buffer;
} ft_glyph_out;

// ft_glyph_out_new 交给 C 分配，Go 那边就不用去算结构体大小。
ft_glyph_out *ft_glyph_out_new(void) { return (ft_glyph_out *) calloc(1, sizeof(ft_glyph_out)); }

void ft_glyph_info(void *face, ft_glyph_out *out) {
	ft_slot *slot = (ft_slot *) ((ft_face *) face)->glyph;
	out->width = (int) slot->bitmap.width;
	out->height = (int) slot->bitmap.rows;
	out->pitch = slot->bitmap.pitch;
	out->left = slot->bitmap_left;
	out->top = slot->bitmap_top;
	out->advance = (int) (slot->metrics.hori_advance >> 6);
	out->mode = (int) slot->bitmap.pixel_mode;
	out->buffer = slot->bitmap.buffer;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// 走系统字体栈：字库文件（/usr/share/fonts 下的 ttf/otf/ttc）交给系统装的
// libfreetype 解析、光栅化，进程里不再带字体解析器。
//
// 用 dlopen 而不是 cgo 链接 freetype（同 x11.go）：编译期不需要开发头文件和 .so
// 符号，运行时缺库则整条系统字体路径作废，上层退回内嵌点阵。

const (
	ftLoadRender  = 1 << 2     // FT_LOAD_RENDER
	ftEncodingUni = 0x756E6963 // FT_ENCODING_UNICODE = 'unic'

	ftPixelMono = 0 // FT_PIXEL_MODE_MONO
)

type ftAPI struct {
	init      unsafe.Pointer // FT_Init_FreeType
	newFace   unsafe.Pointer // FT_New_Face
	setSize   unsafe.Pointer // FT_Set_Pixel_Sizes
	selectMap unsafe.Pointer // FT_Select_Charmap
	charIndex unsafe.Pointer // FT_Get_Char_Index
	loadChar  unsafe.Pointer // FT_Load_Char
}

var (
	ftOnce    sync.Once
	ftFn      ftAPI
	ftLibrary unsafe.Pointer // 进程共用一个 FT_Library，所有 face 挂在它下面
	ftReady   bool
	ftOut     *C.ft_glyph_out
)

// loadFT 加载 libfreetype 并抓齐符号，缺一个就整体不可用。
func loadFT() bool {
	ftOnce.Do(func() {
		if C.ft_load() == 0 {
			return
		}
		for _, s := range []struct {
			name string
			dst  *unsafe.Pointer
		}{
			{"FT_Init_FreeType", &ftFn.init},
			{"FT_New_Face", &ftFn.newFace},
			{"FT_Set_Pixel_Sizes", &ftFn.setSize},
			{"FT_Select_Charmap", &ftFn.selectMap},
			{"FT_Get_Char_Index", &ftFn.charIndex},
			{"FT_Load_Char", &ftFn.loadChar},
		} {
			p := ftSym(s.name)
			if p == nil {
				return
			}
			*s.dst = p
		}
		lib := C.ft_init(ftFn.init)
		if lib == nil {
			return
		}
		ftLibrary = lib
		ftOut = C.ft_glyph_out_new()
		if ftOut == nil {
			return
		}
		ftReady = true
	})
	return ftReady
}

func ftSym(name string) unsafe.Pointer {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.ft_sym(cname)
}

// ftFace 是一个打开的字库文件。FT_Face 自带「当前字号」，换字号得调
// FT_Set_Pixel_Sizes，所以记下 px 省掉重复设置。
type ftFace struct {
	face unsafe.Pointer
	px   int
}

// openFTFace 打开字库文件的第 index 个 face（.ttc 是一个文件装多个 face）。
func openFTFace(path string, index int) (*ftFace, error) {
	if !loadFT() {
		return nil, errors.New("libfreetype 不可用")
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	face := C.ft_new_face(ftFn.newFace, ftLibrary, cpath, C.long(index))
	if face == nil {
		return nil, fmt.Errorf("打不开字库 %s", path)
	}
	// 小端 Unicode charmap，CJK 的 .ttc 默认未必选中它。
	C.ft_select_charmap(ftFn.selectMap, face, C.uint32_t(ftEncodingUni))
	return &ftFace{face: face}, nil
}

// setSize 把 face 切到 px 像素的字号，返回 ascent / height（像素）。
func (f *ftFace) setSize(px int) (int, int, error) {
	if f == nil || f.face == nil {
		return 0, 0, errors.New("字库未打开")
	}
	if f.px != px {
		if C.ft_set_pixel_sizes(ftFn.setSize, f.face, C.uint(0), C.uint(px)) != 0 {
			return 0, 0, fmt.Errorf("FT_Set_Pixel_Sizes(%d) 失败", px)
		}
		f.px = px
	}
	var ascent, height C.int
	if C.ft_size_metrics_get(f.face, &ascent, &height) == 0 {
		return 0, 0, errors.New("字库没有字号度量")
	}
	return int(ascent), int(height), nil
}

// ftGlyph 是一次 FT_Load_Char 的结果。buffer 指向 face 的字形槽，下次加载
// 字符就被覆盖，所以拿到就画。
type ftGlyph struct {
	buf   unsafe.Pointer
	w, h  int
	pitch int
	left  int
	top   int
	adv   int
	mode  int
}

// glyph 光栅化一个字符。render 为 false 时只取度量，不做位图。
func (f *ftFace) glyph(r rune, px int) (ftGlyph, bool) {
	return f.load(r, px, true)
}

// advance 只要步进宽度，不生成位图。
func (f *ftFace) advance(r rune, px int) int {
	g, ok := f.load(r, px, false)
	if !ok {
		return 0
	}
	return g.adv
}

func (f *ftFace) load(r rune, px int, render bool) (ftGlyph, bool) {
	if f == nil || f.face == nil || ftOut == nil {
		return ftGlyph{}, false
	}
	if _, _, err := f.setSize(px); err != nil {
		return ftGlyph{}, false
	}
	if C.ft_char_index(ftFn.charIndex, f.face, C.ulong(r)) == 0 {
		// 字库里没有这个字符，交给上层走缺字处理。
		return ftGlyph{}, false
	}
	flags := C.int32_t(0)
	if render {
		flags = C.int32_t(ftLoadRender)
	}
	if C.ft_load_char(ftFn.loadChar, f.face, C.ulong(r), flags) != 0 {
		return ftGlyph{}, false
	}
	C.ft_glyph_info(f.face, ftOut)
	return ftGlyph{
		buf:   unsafe.Pointer(ftOut.buffer),
		w:     int(ftOut.width),
		h:     int(ftOut.height),
		pitch: int(ftOut.pitch),
		left:  int(ftOut.left),
		top:   int(ftOut.top),
		adv:   int(ftOut.advance),
		mode:  int(ftOut.mode),
	}, true
}
