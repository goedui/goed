//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	beginUpdate    = kernel32.NewProc("BeginUpdateResourceW")
	updateResource = kernel32.NewProc("UpdateResourceW")
	endUpdate      = kernel32.NewProc("EndUpdateResourceW")
)

func stamp(exe, icoPath string) error {
	ico, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}
	entries := parseICO(ico)
	if len(entries) == 0 {
		return fmt.Errorf("%s 里没有图像", icoPath)
	}
	path, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	h, _, callErr := beginUpdate.Call(uintptr(unsafe.Pointer(path)), 0)
	if h == 0 {
		return fmt.Errorf("打开 %s 失败: %v", exe, callErr)
	}
	ok := false
	defer func() {
		if !ok {
			endUpdate.Call(h, 1)
		}
	}()
	for _, e := range entries {
		if err := writeRes(h, 3, uintptr(e.id), e.data); err != nil {
			return err
		}
	}
	if err := writeRes(h, 14, 1, groupIcon(entries)); err != nil {
		return err
	}
	if r, _, callErr := endUpdate.Call(h, 0); r == 0 {
		return fmt.Errorf("写入图标失败: %v", callErr)
	}
	ok = true
	return nil
}

func writeRes(h, typ, name uintptr, data []byte) error {
	r, _, callErr := updateResource.Call(
		h, typ, name, 0x0409,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)
	if r == 0 {
		return fmt.Errorf("UpdateResource: %v", callErr)
	}
	return nil
}
