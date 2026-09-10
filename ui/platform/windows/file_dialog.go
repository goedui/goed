package windows

import (
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	comdlg32             = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
)

const (
	ofFileMustExist   = 0x00001000
	ofPathMustExist   = 0x00000800
	ofNoChangeDir     = 0x00000008
	ofOverwritePrompt = 0x00000002
)

type openFileName struct {
	StructSize    uint32
	Owner         syscall.Handle
	Instance      syscall.Handle
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExtension  *uint16
	CustData      uintptr
	Hook          uintptr
	TemplateName  *uint16
	Reserved      *uint16
	Reserved2     uint32
	ExFlags       uint32
}

func (w *FramelessWindow) Handle() syscall.Handle {
	if w == nil {
		return 0
	}
	return w.hwnd
}

func OpenFileDialog(owner syscall.Handle, initialDir string) (string, bool) {
	return fileDialog(owner, initialDir, false)
}

func SaveFileDialog(owner syscall.Handle, initialDir string) (string, bool) {
	return fileDialog(owner, initialDir, true)
}

func fileDialog(owner syscall.Handle, initialDir string, save bool) (string, bool) {
	buffer := make([]uint16, 32768)
	filter := utf16Buffer("Text files (*.txt;*.md;*.go)\x00*.txt;*.md;*.go\x00All files (*.*)\x00*.*")
	dir := syscall.StringToUTF16Ptr(initialDir)
	title := syscall.StringToUTF16Ptr("Open File")
	if save {
		title = syscall.StringToUTF16Ptr("Save File As")
	}
	of := openFileName{
		StructSize: uint32(unsafe.Sizeof(openFileName{})), Owner: owner,
		Filter: &filter[0], FilterIndex: 1, File: &buffer[0], MaxFile: uint32(len(buffer)),
		InitialDir: dir, Title: title, Flags: ofNoChangeDir | ofPathMustExist,
	}
	if save {
		of.Flags |= ofOverwritePrompt
	} else {
		of.Flags |= ofFileMustExist
	}
	proc := procGetOpenFileNameW
	if save {
		proc = procGetSaveFileNameW
	}
	if result, _, _ := proc.Call(uintptr(unsafe.Pointer(&of))); result == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buffer), true
}

func utf16Buffer(value string) []uint16 {
	value = strings.TrimRight(value, "\x00")
	runes := []rune(value)
	result := utf16.Encode(runes)
	result = append(result, 0, 0)
	return result
}
