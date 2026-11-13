package components

// 本文件负责「码字 → 模块矩阵」：功能图形、格式 / 版本信息、数据排布、掩码与罚分。
// 编码（分段 / RS / 交织）见 qr.go。

// qrECFormatCode 是写入格式信息的 2 位纠错码（ISO 18004 的顺序，不是枚举顺序）。
var qrECFormatCode = [4]int{1, 0, 3, 2}

// qrBuildMatrix 组装最终矩阵。test 为 true 时格式 / 版本信息全部写成浅色，
// 与标准里「选掩码时先按占位模块打分」的做法一致（打分不能受真实格式位影响）。
func qrBuildMatrix(version int, level Level, codewords []byte, mask int, test bool) *QRMatrix {
	size := version*4 + 17
	m := &QRMatrix{Version: version, Level: level, Mask: mask, Size: size, mods: make([]bool, size*size)}
	fn := make([]bool, size*size)
	set := func(r, c int, dark bool) {
		if r < 0 || c < 0 || r >= size || c >= size {
			return
		}
		m.mods[r*size+c] = dark
		fn[r*size+c] = true
	}
	reserve := func(r, c int) {
		if r < 0 || c < 0 || r >= size || c >= size {
			return
		}
		fn[r*size+c] = true
	}
	write := func(r, c int, dark bool) {
		if r < 0 || c < 0 || r >= size || c >= size {
			return
		}
		m.mods[r*size+c] = dark
	}

	for _, p := range [][2]int{{0, 0}, {size - 7, 0}, {0, size - 7}} {
		qrDrawFinder(set, p[0], p[1], size)
	}
	pos := qrAlignmentPositions(version)
	for _, r := range pos {
		for _, c := range pos {
			if fn[r*size+c] {
				continue
			}
			qrDrawAlignment(set, r, c)
		}
	}
	for i := 8; i < size-8; i++ {
		if !fn[i*size+6] {
			set(i, 6, i%2 == 0)
		}
		if !fn[6*size+i] {
			set(6, i, i%2 == 0)
		}
	}
	// 格式信息 / 版本信息 / 固定暗模块占位：数据区必须跳过，稍后再写值。
	for i := 0; i < 15; i++ {
		switch {
		case i < 6:
			reserve(i, 8)
		case i < 8:
			reserve(i+1, 8)
		default:
			reserve(size-15+i, 8)
		}
		switch {
		case i < 8:
			reserve(8, size-i-1)
		case i < 9:
			reserve(8, 15-i)
		default:
			reserve(8, 15-i-1)
		}
	}
	set(size-8, 8, !test)
	if version >= 7 {
		for i := 0; i < 18; i++ {
			reserve(i/3, size-11+i%3)
			reserve(size-11+i%3, i/3)
		}
	}

	qrPlaceData(m, fn, codewords, mask)

	bits := qrBCHTypeInfo((qrECFormatCode[level] << 3) | mask)
	for i := 0; i < 15; i++ {
		dark := !test && (bits>>uint(i))&1 == 1
		switch {
		case i < 6:
			write(i, 8, dark)
		case i < 8:
			write(i+1, 8, dark)
		default:
			write(size-15+i, 8, dark)
		}
		switch {
		case i < 8:
			write(8, size-i-1, dark)
		case i < 9:
			write(8, 15-i, dark)
		default:
			write(8, 15-i-1, dark)
		}
	}
	if version >= 7 {
		vbits := qrBCHTypeNumber(version)
		for i := 0; i < 18; i++ {
			dark := !test && (vbits>>uint(i))&1 == 1
			write(i/3, size-11+i%3, dark)
			write(size-11+i%3, i/3, dark)
		}
	}
	return m
}

func qrDrawFinder(set func(r, c int, dark bool), row, col, size int) {
	for r := -1; r <= 7; r++ {
		for c := -1; c <= 7; c++ {
			if row+r < 0 || row+r >= size || col+c < 0 || col+c >= size {
				continue
			}
			dark := (r >= 0 && r <= 6 && (c == 0 || c == 6)) ||
				(c >= 0 && c <= 6 && (r == 0 || r == 6)) ||
				(r >= 2 && r <= 4 && c >= 2 && c <= 4)
			set(row+r, col+c, dark)
		}
	}
}

func qrDrawAlignment(set func(r, c int, dark bool), row, col int) {
	for r := -2; r <= 2; r++ {
		for c := -2; c <= 2; c++ {
			dark := r == -2 || r == 2 || c == -2 || c == 2 || (r == 0 && c == 0)
			set(row+r, col+c, dark)
		}
	}
}

// qrPlaceData 从右下角起、按两列一组蛇形向上 / 向下填码字，跳过所有功能模块。
func qrPlaceData(m *QRMatrix, fn []bool, codewords []byte, mask int) {
	size := m.Size
	idx, bit := 0, 7
	row, inc := size-1, -1
	for base := size - 1; base > 0; base -= 2 {
		col := base
		if col <= 6 {
			col--
		}
		for {
			for _, c := range [2]int{col, col - 1} {
				pos := row*size + c
				if fn[pos] {
					continue
				}
				dark := false
				if idx < len(codewords) {
					dark = (codewords[idx]>>uint(bit))&1 == 1
				}
				if qrMaskAt(mask, row, c) {
					dark = !dark
				}
				m.mods[pos] = dark
				bit--
				if bit < 0 {
					bit = 7
					idx++
				}
			}
			row += inc
			if row < 0 || row >= size {
				row -= inc
				inc = -inc
				break
			}
		}
	}
}

func qrMaskAt(mask, i, j int) bool {
	switch mask {
	case 0:
		return (i+j)%2 == 0
	case 1:
		return i%2 == 0
	case 2:
		return j%3 == 0
	case 3:
		return (i+j)%3 == 0
	case 4:
		return (i/2+j/3)%2 == 0
	case 5:
		return (i*j)%2+(i*j)%3 == 0
	case 6:
		return ((i*j)%2+(i*j)%3)%2 == 0
	case 7:
		return ((i*j)%3+(i+j)%2)%2 == 0
	}
	return false
}

const (
	qrG15     = 0x537
	qrG15Mask = 0x5412
	qrG18     = 0x1F25
)

func qrBCHDigit(v int) int {
	d := 0
	for v != 0 {
		d++
		v >>= 1
	}
	return d
}

func qrBCHTypeInfo(data int) int {
	d := data << 10
	for qrBCHDigit(d)-qrBCHDigit(qrG15) >= 0 {
		d ^= qrG15 << uint(qrBCHDigit(d)-qrBCHDigit(qrG15))
	}
	return ((data << 10) | d) ^ qrG15Mask
}

func qrBCHTypeNumber(version int) int {
	d := version << 12
	for qrBCHDigit(d)-qrBCHDigit(qrG18) >= 0 {
		d ^= qrG18 << uint(qrBCHDigit(d)-qrBCHDigit(qrG18))
	}
	return (version << 12) | d
}

// ===== 掩码选择 ========================================================

func qrBestMask(version int, level Level, payload []byte) int {
	best, bestScore := 0, 0
	for mask := 0; mask < 8; mask++ {
		m := qrBuildMatrix(version, level, payload, mask, true)
		score := qrPenalty(m.mods, m.Size)
		if mask == 0 || score < bestScore {
			best, bestScore = mask, score
		}
	}
	return best
}

// qrPenalty 是标准四条罚分规则，与 python-qrcode 的 lost_point 等价。
func qrPenalty(mods []bool, size int) int {
	score := 0
	container := make([]int, size+1)
	countRuns := func(get func(i int) bool) {
		prev := get(0)
		length := 0
		for i := 0; i < size; i++ {
			cur := get(i)
			if cur == prev {
				length++
				continue
			}
			if length >= 5 {
				container[length]++
			}
			length = 1
			prev = cur
		}
		if length >= 5 {
			container[length]++
		}
	}
	for r := 0; r < size; r++ {
		row := r
		countRuns(func(i int) bool { return mods[row*size+i] })
	}
	for c := 0; c < size; c++ {
		col := c
		countRuns(func(i int) bool { return mods[i*size+col] })
	}
	for length := 5; length <= size; length++ {
		score += container[length] * (length - 2)
	}

	// 2x2 同色块。
	for r := 0; r < size-1; r++ {
		skip := false
		for c := 0; c < size-1; c++ {
			if skip {
				skip = false
				continue
			}
			tr := mods[r*size+c+1]
			if tr != mods[(r+1)*size+c+1] {
				skip = true
				continue
			}
			if tr != mods[r*size+c] || tr != mods[(r+1)*size+c] {
				continue
			}
			score += 3
		}
	}

	// 1:1:3:1:1 的类定位图形（前后带 4 个浅色模块）。
	var window [11]bool
	check := func() bool {
		return !window[1] && window[4] && !window[5] && window[6] && !window[9] &&
			((window[0] && window[2] && window[3] && !window[7] && !window[8] && !window[10]) ||
				(!window[0] && !window[2] && !window[3] && window[7] && window[8] && window[10]))
	}
	for r := 0; r < size; r++ {
		skip := false
		for c := 0; c+10 < size; c++ {
			if skip {
				skip = false
				continue
			}
			for k := 0; k < 11; k++ {
				window[k] = mods[r*size+c+k]
			}
			if check() {
				score += 40
			}
			if window[10] {
				skip = true
			}
		}
	}
	for c := 0; c < size; c++ {
		skip := false
		for r := 0; r+10 < size; r++ {
			if skip {
				skip = false
				continue
			}
			for k := 0; k < 11; k++ {
				window[k] = mods[(r+k)*size+c]
			}
			if check() {
				score += 40
			}
			if window[10] {
				skip = true
			}
		}
	}

	// 深色比例偏离 50% 的幅度，每 5% 计 10 分。
	dark := 0
	for _, d := range mods {
		if d {
			dark++
		}
	}
	percent := float64(dark) / float64(size*size)
	rating := int(absFloat(percent*100-50) / 5)
	return score + rating*10
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
