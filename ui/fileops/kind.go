package fileops

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileKind 文件内容类别：决定编辑器以何种视图打开。
type FileKind int

const (
	KindText   FileKind = iota // 文本文档（编辑器）
	KindImage                  // 图片（只读预览）
	KindBinary                 // 二进制（只读十六进制预览）
)

// 二进制/图片按扩展名快速判定的白名单；无扩展名或未列出的文件走
// 内容嗅探（首 8KB 含 NUL 即二进制）。
var (
	imageExts = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".bmp": true, ".webp": true, ".ico": true, ".tif": true, ".tiff": true,
	}
	binaryExts = map[string]bool{
		".exe": true, ".dll": true, ".sys": true, ".obj": true, ".o": true,
		".a": true, ".lib": true, ".so": true, ".dylib": true,
		".zip": true, ".gz": true, ".tar": true, ".7z": true, ".rar": true,
		".bz2": true, ".xz": true, ".zst": true,
		".pdf": true, ".class": true, ".jar": true,
		".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true, ".sqlite3": true,
		".pyc": true, ".wasm": true, ".exe.bak": true,
	}
)

const sniffSize = 8 * 1024

// DetectKind 判定文件类别。读取失败按文本处理（真正的错误由打开路径报告）。
func DetectKind(path string) FileKind {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case imageExts[ext]:
		return KindImage
	case binaryExts[ext]:
		return KindBinary
	}
	f, err := os.Open(path)
	if err != nil {
		return KindText
	}
	defer f.Close()
	head := make([]byte, sniffSize)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return KindText
	}
	if bytes.IndexByte(head[:n], 0) >= 0 {
		return KindBinary
	}
	return KindText
}

// LoadPreview 读取文件的只读预览：最多 maxBytes 字节，配合 Truncated
// 判断是否被截断。total 为磁盘上的完整大小。
func LoadPreview(path string, maxBytes int64) (data []byte, total int64, truncated bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer f.Close()
	if info, statErr := f.Stat(); statErr == nil {
		total = info.Size()
	}
	data = make([]byte, maxBytes+1)
	n, err := io.ReadFull(f, data)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return nil, total, false, err
	}
	if int64(n) > maxBytes {
		return data[:maxBytes], total, true, nil
	}
	return data[:n], total, false, nil
}

// FormatSize 人类可读的文件大小（B/KB/MB/GB）。
func FormatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
