//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestDragContinueIgnoresStaleKeyState(t *testing.T) {
	// ReleaseCapture 之后 grfKeyState 经常是 0，但左键还按着。
	if got := dragContinue(0, 0, true); got != sOK {
		t.Fatalf("物理左键按下应继续拖: %#x", got)
	}
	if got := dragContinue(0, mkLButton, false); got != sOK {
		t.Fatalf("消息里的左键也应继续拖: %#x", got)
	}
	if got := dragContinue(0, 0, false); got != ddSDrop {
		t.Fatalf("左键松开应放下: %#x", got)
	}
	if got := dragContinue(1, mkLButton, true); got != ddSCancel {
		t.Fatalf("Esc 应取消: %#x", got)
	}
}

func TestShellDataObjectFromRealFile(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	done, err := oleInit()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	f, err := os.CreateTemp("", "banzip-drag-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	obj, err := shellDataObject([]string{name})
	if err != nil || obj == 0 {
		t.Fatalf("Shell 数据对象: obj=%#x err=%v", obj, err)
	}
	releasePtr(obj)
}

type dragTestFormatEtc struct {
	cfFormat uint16
	_        [6]byte
	ptd      uintptr
	dwAspect uint32
	lindex   int32
	tymed    uint32
	_        uint32
}

type dragTestStgMedium struct {
	tymed uint32
	_     uint32
	data  uintptr
	unk   uintptr
}

// readHDrop 向 IDataObject 要 CF_HDROP，返回拖放句柄。
func readHDrop(obj uintptr) (uintptr, func(), error) {
	format := dragTestFormatEtc{cfFormat: 15, dwAspect: 1, lindex: -1, tymed: 1}
	var medium dragTestStgMedium
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	getData := *(*uintptr)(unsafe.Pointer(vtbl + 3*unsafe.Sizeof(uintptr(0))))
	hr, _, _ := syscall.SyscallN(getData, obj, uintptr(unsafe.Pointer(&format)), uintptr(unsafe.Pointer(&medium)))
	if hresult(hr).failed() {
		return 0, nil, hresult(hr)
	}
	return medium.data, func() { procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&medium))) }, nil
}

// TestShellDataObjectProvidesHDrop 直接向资源管理器会调用的 GetData 取
// CF_HDROP，确认拖出数据真的能还原成磁盘路径。
func TestShellDataObjectProvidesHDrop(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	done, err := oleInit()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	dir := t.TempDir()
	paths := make([]string, 0, 3)
	for _, name := range []string{"a.txt", "中文.txt"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	paths = append(paths, sub)
	obj, err := shellDataObject(paths)
	if err != nil || obj == 0 {
		t.Fatalf("Shell 数据对象: obj=%#x err=%v", obj, err)
	}
	defer releasePtr(obj)

	hdrop, release, err := readHDrop(obj)
	if err != nil {
		t.Fatalf("GetData(CF_HDROP): %v", err)
	}
	defer release()
	n, _, _ := procDragQueryFileW.Call(hdrop, 0xFFFFFFFF, 0, 0)
	if n != uintptr(len(paths)) {
		t.Fatalf("CF_HDROP 条目数 = %d, 期望 %d", n, len(paths))
	}
	for i := range paths {
		size, _, _ := procDragQueryFileW.Call(hdrop, uintptr(i), 0, 0)
		buf := make([]uint16, size+1)
		procDragQueryFileW.Call(hdrop, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), size+1)
		got := syscall.UTF16ToString(buf[:size])
		want, err := os.Stat(paths[i])
		if err != nil {
			t.Fatal(err)
		}
		have, err := os.Stat(got)
		if err != nil {
			t.Fatalf("第 %d 条路径 %q 不可访问: %v", i, got, err)
		}
		if !os.SameFile(want, have) {
			t.Fatalf("第 %d 条路径 = %q, 期望 %q", i, got, paths[i])
		}
	}
}

var procReleaseStgMedium = modOle32.NewProc("ReleaseStgMedium")
