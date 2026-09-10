package windows

import (
	"syscall"
	"unsafe"
)

func CreateFont(name string, height int32) syscall.Handle {
	return createFontQuality(name, height, fontAntialiased)
}

const (
	fontAntialiased uint32 = 4
	fontClearType   uint32 = 5
)

// createFontQuality keeps grayscale rendering as the default. Opaque UI
// surfaces copied 1:1 to the display can opt into sharper ClearType text.
func createFontQuality(name string, height int32, quality uint32) syscall.Handle {
	if name == "" {
		name = "Consolas"
	}

	// A negative height specifies glyph height in logical pixels, not points.
	if height > 0 {
		height = -height
	}

	// 常量定义
	const (
		OUT_TT_PRECIS       = 4    // TrueType 字体精度
		CLIP_DEFAULT_PRECIS = 0    // 默认裁剪精度
		FIXED_PITCH         = 1    // 固定宽度
		FF_MODERN           = 0x30 // 现代字体系列
	)

	// Create the requested face with the surface's selected rasterization.
	ret, _, _ := procCreateFontW.Call(
		uintptr(height),              // Height（负值 = 字符高度）
		0,                            // Width（0 = 自动）
		0,                            // Escapement（旋转角度）
		0,                            // Orientation（基线角度）
		400,                          // Weight（FW_NORMAL = 400）
		0,                            // Italic（0 = 不倾斜）
		0,                            // Underline（0 = 无下划线）
		0,                            // StrikeOut（0 = 无删除线）
		1,                            // CharSet（DEFAULT_CHARSET = 1）
		uintptr(OUT_TT_PRECIS),       // OutPrecision（TrueType 优先）
		uintptr(CLIP_DEFAULT_PRECIS), // ClipPrecision
		uintptr(quality),             // Quality
		fontPitch(name),              // PitchAndFamily
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
	)

	font := syscall.Handle(ret)

	// 如果是 Cascadia Code 且可能不存在，回退到 Consolas
	if font == 0 {
		if name != "" {
			// 尝试 Consolas 作为回退
			ret, _, _ = procCreateFontW.Call(
				uintptr(height), 0, 0, 0, 400, 0, 0, 0,
				1,
				uintptr(OUT_TT_PRECIS),
				uintptr(CLIP_DEFAULT_PRECIS),
				uintptr(quality),
				fontPitch("Segoe UI"),
				uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Segoe UI"))),
			)
			return syscall.Handle(ret)
		}
	}

	return font
}
