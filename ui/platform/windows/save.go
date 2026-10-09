//go:build windows

package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveBytes 把字节写入「下载」文件夹，不弹对话框。实现 platform.SaveBytes。
// 同名文件自动加 " (2)" 后缀，避免覆盖。
func SaveBytes(name, mime string, data []byte) (string, bool, error) {
	_ = mime
	dir := downloadsDir()
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, "Downloads")
		}
	}
	if dir == "" {
		return "", false, fmt.Errorf("找不到下载文件夹")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	path, err := writeBytesToDir(dir, name, data)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

func writeBytesToDir(dir, name string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("没有要保存的内容")
	}
	name = sanitizeFileName(name)
	path := uniquePath(filepath.Join(dir, name))
	tmp, err := os.CreateTemp(dir, ".goed-save-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; i < 10000; i++ {
		cand := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
	return fmt.Sprintf("%s-%d%s", base, os.Getpid(), ext)
}
