// Package fileops contains the platform-neutral document operations shared by
// github.com/goedui/goed's editor surfaces.
package fileops

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

var ErrPathRequired = errors.New("file path is required")

type Document struct {
	Path    string
	Content string
	// ModTime 是磁盘文件加载/保存时刻的修改时间，供外部变更检测比对。
	ModTime time.Time
}

func NewDocument() Document { return Document{} }

func Load(path string) (Document, error) {
	path = normalizePath(path)
	if path == "" {
		return Document{}, ErrPathRequired
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	modTime := time.Time{}
	if info, err := os.Stat(path); err == nil {
		modTime = info.ModTime()
	}
	return Document{Path: path, Content: string(data), ModTime: modTime}, nil
}

// Save writes an existing document while retaining its current file mode when
// possible. SaveAs creates a regular user file with mode 0644.
func Save(path, content string) error {
	path = normalizePath(path)
	if path == "" {
		return ErrPathRequired
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, []byte(content), mode)
}

func SaveAs(path, content string) (Document, error) {
	path = normalizePath(path)
	if err := Save(path, content); err != nil {
		return Document{}, err
	}
	modTime := time.Time{}
	if info, err := os.Stat(path); err == nil {
		modTime = info.ModTime()
	}
	return Document{Path: path, Content: content, ModTime: modTime}, nil
}

// StaleOnDisk 报告磁盘文件的修改时间是否晚于文档记录的加载/保存时间，
// 即文件在编辑器之外被改动过。modTime 为零值（未知）时返回 false。
func StaleOnDisk(doc Document) bool {
	if doc.Path == "" || doc.ModTime.IsZero() {
		return false
	}
	info, err := os.Stat(doc.Path)
	if err != nil {
		return false // 文件消失由调用方按删除处理
	}
	return info.ModTime().After(doc.ModTime)
}

// MissingOnDisk 报告文档对应的文件是否已从磁盘消失。
func MissingOnDisk(doc Document) bool {
	if doc.Path == "" {
		return false
	}
	_, err := os.Stat(doc.Path)
	return os.IsNotExist(err)
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	if absolute, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(path)
}
