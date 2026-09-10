package windows

import (
	"sync"
	"syscall"
)

const goedIconLogicalSize = 32

type goedIconSet struct {
	large syscall.Handle
	small syscall.Handle
}

var (
	goedIconsOnce sync.Once
	goedIcons     goedIconSet
)

// loadGoedIcons creates the application icon from pixels compiled into the
// binary. Keeping the icon here avoids a runtime dependency on an .ico file.
func loadGoedIcons() goedIconSet {
	goedIconsOnce.Do(func() {
		goedIcons.large = createGoedIcon(32)
		goedIcons.small = createGoedIcon(16)
	})
	return goedIcons
}

func createGoedIcon(size int) syscall.Handle {
	if size <= 0 {
		return 0
	}
	maskStride := ((size + 31) / 32) * 4
	andBits := make([]byte, maskStride*size)
	xorBits := make([]byte, size*size*4)

	for y := 0; y < size; y++ {
		// CreateIcon expects DIB rows bottom-up.
		row := size - 1 - y
		for x := 0; x < size; x++ {
			logicalX := x * goedIconLogicalSize / size
			logicalY := y * goedIconLogicalSize / size
			r, g, b, transparent := goedIconPixel(logicalX, logicalY)
			if transparent {
				andBits[row*maskStride+x/8] |= 1 << uint(7-x%8)
				continue
			}
			pixel := (row*size + x) * 4
			xorBits[pixel] = b
			xorBits[pixel+1] = g
			xorBits[pixel+2] = r
			xorBits[pixel+3] = 0xff
		}
	}

	return createIcon(size, size, andBits, xorBits)
}

func goedIconPixel(x, y int) (r, g, b byte, transparent bool) {
	const (
		backgroundR = 28
		backgroundG = 38
		backgroundB = 53
		accentR     = 69
		accentG     = 211
		accentB     = 194
		cursorR     = 238
		cursorG     = 242
		cursorB     = 246
	)

	// Rounded corners keep the icon legible against light and dark taskbars.
	const radius = 5
	if (x < radius && y < radius && cornerOutside(radius-1-x, radius-1-y, radius)) ||
		(x >= goedIconLogicalSize-radius && y < radius && cornerOutside(x-(goedIconLogicalSize-radius), radius-1-y, radius)) ||
		(x < radius && y >= goedIconLogicalSize-radius && cornerOutside(radius-1-x, y-(goedIconLogicalSize-radius), radius)) ||
		(x >= goedIconLogicalSize-radius && y >= goedIconLogicalSize-radius && cornerOutside(x-(goedIconLogicalSize-radius), y-(goedIconLogicalSize-radius), radius)) {
		return 0, 0, 0, true
	}

	r, g, b = backgroundR, backgroundG, backgroundB
	if x <= 2 || x >= goedIconLogicalSize-3 || y <= 2 || y >= goedIconLogicalSize-3 {
		r, g, b = accentR, accentG, accentB
	}

	// A blocky G mark mirrors the editor's document-and-caret identity.
	dx, dy := x-15, y-15
	distance := dx*dx + dy*dy
	if distance >= 49 && distance <= 100 && !(x > 18 && y < 15) {
		r, g, b = accentR, accentG, accentB
	}
	if x >= 15 && x <= 23 && y >= 14 && y <= 17 {
		r, g, b = accentR, accentG, accentB
	}
	if x >= 24 && x <= 26 && y >= 9 && y <= 22 {
		r, g, b = cursorR, cursorG, cursorB
	}
	return r, g, b, false
}

func cornerOutside(dx, dy, radius int) bool {
	return dx*dx+dy*dy > radius*radius
}
