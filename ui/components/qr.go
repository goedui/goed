package components

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Level 是二维码纠错等级（L 最低、H 最高）。等级越高，能容忍的污损越多，
// 同样内容需要的版本（尺寸）也越大。
type Level int

const (
	LevelL Level = iota
	LevelM
	LevelQ
	LevelH
)

func (l Level) String() string {
	switch l {
	case LevelL:
		return "L"
	case LevelQ:
		return "Q"
	case LevelH:
		return "H"
	default:
		return "M"
	}
}

func (l Level) valid() bool { return l >= LevelL && l <= LevelH }

// ParseLevel 解析 "L"/"M"/"Q"/"H"（大小写不敏感），未知值返回 LevelM。
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "L":
		return LevelL
	case "Q":
		return LevelQ
	case "H":
		return LevelH
	default:
		return LevelM
	}
}

// QRMatrix 是编码后的二维码模块矩阵。Dark(x,y) 的 y 轴从上往下，和屏幕坐标一致。
type QRMatrix struct {
	Version int
	Level   Level
	Mask    int
	Size    int
	mods    []bool
}

// Dark 报告 (x,y) 处是否是要画的深色模块。越界返回 false。
func (m *QRMatrix) Dark(x, y int) bool {
	if m == nil || x < 0 || y < 0 || x >= m.Size || y >= m.Size {
		return false
	}
	return m.mods[y*m.Size+x]
}

// EncodeQR 按内容与纠错等级编码，自动选最小版本与最优掩码。
func EncodeQR(text string, level Level) (*QRMatrix, error) {
	return EncodeQRVersion(text, level, 0, -1)
}

// EncodeQRVersion 指定版本 / 掩码编码。version 0 表示自动选最小版本，
// mask 小于 0 表示按标准罚分自动选。固定二者主要供测试与复现使用。
func EncodeQRVersion(text string, level Level, version, mask int) (*QRMatrix, error) {
	if !level.valid() {
		level = LevelM
	}
	data := []byte(text)
	if len(data) > 2953 {
		return nil, errors.New("qrcode: 内容过长（字节模式上限 2953 字节）")
	}
	if version < 0 || version > 40 {
		return nil, fmt.Errorf("qrcode: 版本 %d 超出 0..40", version)
	}
	if version == 0 {
		version = qrFitVersion(len(data), level)
		if version == 0 {
			return nil, errors.New("qrcode: 内容过长，40 版也放不下")
		}
	} else if !qrFits(len(data), version, level) {
		return nil, fmt.Errorf("qrcode: 内容放不进版本 %d（等级 %s）", version, level)
	}
	if mask < -1 || mask > 7 {
		return nil, fmt.Errorf("qrcode: 掩码 %d 超出 0..7", mask)
	}
	payload := qrCodewords(data, version, level)
	if mask < 0 {
		mask = qrBestMask(version, level, payload)
	}
	return qrBuildMatrix(version, level, payload, mask, false), nil
}

// ===== 容量与版本 ========================================================

func qrFitVersion(nbytes int, level Level) int {
	for v := 1; v <= 40; v++ {
		if qrFits(nbytes, v, level) {
			return v
		}
	}
	return 0
}

func qrFits(nbytes, version int, level Level) bool {
	countBits := 8
	if version >= 10 {
		countBits = 16
	}
	return 4+countBits+nbytes*8 <= qrDataCodewords(version, level)*8
}

// qrTotalCodewords 是某个版本的全部码字数，由模块总数扣掉功能图形后除以 8。
func qrTotalCodewords(version int) int {
	result := (16*version+128)*version + 64
	if version >= 2 {
		numAlign := version/7 + 2
		result -= (25*numAlign-10)*numAlign - 55
		if version >= 7 {
			result -= 36
		}
	}
	return result / 8
}

func qrDataCodewords(version int, level Level) int {
	total := qrTotalCodewords(version)
	return total - qrECPerBlock[level][version-1]*qrNumBlocks[level][version-1]
}

// qrAlignmentPositions 返回校正图形中心坐标（行 / 列共用同一组）。
func qrAlignmentPositions(version int) []int {
	if version == 1 {
		return nil
	}
	num := version/7 + 2
	step := 26
	if version != 32 {
		step = ((version*4 + num*2 + 1) / (num*2 - 2)) * 2
	}
	pos := make([]int, num)
	pos[0] = 6
	p := version*4 + 10
	for i := num - 1; i >= 1; i-- {
		pos[i] = p
		p -= step
	}
	return pos
}

// ===== 码字：分段 / 填充 / RS / 交织 ====================================

type qrBitWriter struct {
	buf []byte
	n   int // 已写入的位数
}

func (w *qrBitWriter) put(val, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if (val>>uint(i))&1 == 1 {
			w.buf[len(w.buf)-1] |= 1 << uint(7-w.n%8)
		}
		w.n++
	}
}

// qrCodewords 生成数据码字 + 纠错码字，按标准交织后的最终码字序列。
// 统一使用字节模式（UTF-8），不做数字 / 字母数字分段优化。
func qrCodewords(data []byte, version int, level Level) []byte {
	capacity := qrDataCodewords(version, level)
	w := &qrBitWriter{}
	w.put(0b0100, 4) // 字节模式
	if version <= 9 {
		w.put(len(data), 8)
	} else {
		w.put(len(data), 16)
	}
	for _, b := range data {
		w.put(int(b), 8)
	}
	total := capacity * 8
	for i := 0; i < 4 && w.n < total; i++ {
		w.put(0, 1)
	}
	for w.n%8 != 0 {
		w.put(0, 1)
	}
	for pad := 0; len(w.buf) < capacity; pad++ {
		if pad%2 == 0 {
			w.put(0xEC, 8)
		} else {
			w.put(0x11, 8)
		}
	}
	return qrInterleave(w.buf, version, level)
}

func qrInterleave(data []byte, version int, level Level) []byte {
	blocks := qrNumBlocks[level][version-1]
	ecc := qrECPerBlock[level][version-1]
	dataTotal := qrDataCodewords(version, level)
	shortLen := dataTotal / blocks
	numLong := dataTotal % blocks
	divisor := qrRSDivisor(ecc)

	datas := make([][]byte, blocks)
	eccs := make([][]byte, blocks)
	off := 0
	for i := 0; i < blocks; i++ {
		n := shortLen
		if i >= blocks-numLong {
			n++
		}
		datas[i] = data[off : off+n]
		off += n
		eccs[i] = qrRSRemainder(datas[i], divisor)
	}
	out := make([]byte, 0, qrTotalCodewords(version))
	for i := 0; i <= shortLen; i++ {
		for b := 0; b < blocks; b++ {
			if i < len(datas[b]) {
				out = append(out, datas[b][i])
			}
		}
	}
	for i := 0; i < ecc; i++ {
		for b := 0; b < blocks; b++ {
			out = append(out, eccs[b][i])
		}
	}
	return out
}

var qrGFExp [512]byte
var qrGFLog [256]byte

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		qrGFExp[i] = byte(x)
		qrGFLog[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < len(qrGFExp); i++ {
		qrGFExp[i] = qrGFExp[i-255]
	}
}

func qrGFMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return qrGFExp[int(qrGFLog[a])+int(qrGFLog[b])]
}

// qrRSDivisor 生成 degree 次的 Reed-Solomon 生成多项式（省略最高次项）。
func qrRSDivisor(degree int) []byte {
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := 0; j < degree; j++ {
			result[j] = qrGFMul(result[j], root)
			if j+1 < degree {
				result[j] ^= result[j+1]
			}
		}
		root = qrGFMul(root, 2)
	}
	return result
}

func qrRSRemainder(data, divisor []byte) []byte {
	result := make([]byte, len(divisor))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i := range result {
			result[i] ^= qrGFMul(divisor[i], factor)
		}
	}
	return result
}

// ===== 编码缓存 ========================================================

var qrMatrixCache = struct {
	sync.Mutex
	m map[string]*QRMatrix
}{m: map[string]*QRMatrix{}}

func qrLookup(text string, level Level) (*QRMatrix, error) {
	key := level.String() + "\x00" + text
	qrMatrixCache.Lock()
	if m, ok := qrMatrixCache.m[key]; ok {
		qrMatrixCache.Unlock()
		return m, nil
	}
	qrMatrixCache.Unlock()
	m, err := EncodeQR(text, level)
	if err != nil {
		return nil, err
	}
	qrMatrixCache.Lock()
	if len(qrMatrixCache.m) >= 64 {
		qrMatrixCache.m = map[string]*QRMatrix{}
	}
	qrMatrixCache.m[key] = m
	qrMatrixCache.Unlock()
	return m, nil
}
