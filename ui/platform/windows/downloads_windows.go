package windows

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// DownloadsDirectory resolves the Windows known folder, including redirected
// Downloads folders. Older systems fall back to the user's Downloads path.
func DownloadsDirectory() string {
	type guid struct {
		Data1        uint32
		Data2, Data3 uint16
		Data4        [8]byte
	}
	id := guid{0x374DE290, 0x123F, 0x4565, [8]byte{0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B}}
	proc := syscall.NewLazyDLL("shell32.dll").NewProc("SHGetKnownFolderPath")
	if proc.Find() == nil {
		var ptr *uint16
		hr, _, _ := proc.Call(uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&ptr)))
		if ptr != nil {
			defer syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(ptr)))
			if int32(hr) >= 0 {
				var units []uint16
				for n := 0; n < 32768; n++ {
					v := *(*uint16)(unsafe.Add(unsafe.Pointer(ptr), n*2))
					if v == 0 {
						break
					}
					units = append(units, v)
				}
				if len(units) > 0 {
					return syscall.UTF16ToString(units)
				}
			}
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads")
}
