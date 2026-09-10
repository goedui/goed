package windows

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestGuidFromStringPacksCLSIDFileOpenDialog(t *testing.T) {
	// CLSID_FileOpenDialog = {DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}
	got := guidFromString(clsidFileOpenDialog)
	want := guid{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	if got != want {
		t.Fatalf("guidFromString(CLSID_FileOpenDialog) = %08X-%04X-%04X-%02X, want %08X-%04X-%04X-%02X",
			got.Data1, got.Data2, got.Data3, got.Data4,
			want.Data1, want.Data2, want.Data3, want.Data4)
	}
}

func TestGuidFromStringPacksIIDIFileDialog(t *testing.T) {
	// IID_IFileDialog = {42F85136-DB7E-439C-85F1-E4075D135FC8}
	got := guidFromString(iidIFileDialog)
	want := guid{0x42F85136, 0xDB7E, 0x439C, [8]byte{0x85, 0xF1, 0xE4, 0x07, 0x5D, 0x13, 0x5F, 0xC8}}
	if got != want {
		t.Fatalf("guidFromString(IID_IFileDialog) = %08X-%04X-%04X-%02X, want %08X-%04X-%04X-%02X",
			got.Data1, got.Data2, got.Data3, got.Data4,
			want.Data1, want.Data2, want.Data3, want.Data4)
	}
}

func TestIFileDialogVtblSlotsMatchShobjidl(t *testing.T) {
	// shobjidl_core.h IFileDialog 槽位：IUnknown 0-2 → Show 3 →
	// SetOptions 9 / GetOptions 10 / SetDefaultFolder 11 / SetFolder 12 /
	// GetFolder 13 → IFileOpenDialog::GetResult 20。
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"setOptions", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.setOptions)), 9},
		{"getOptions", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.getOptions)), 10},
		{"setDefaultFolder", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.setDefaultFolder)), 11},
		{"setFolder", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.setFolder)), 12},
		{"getFolder", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.getFolder)), 13},
		{"getResult", vtblSlot(unsafe.Offsetof(iFileDialogVtbl{}.getResult)), 20},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("vtable slot %s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

// TestCreateFileOpenDialogCOM 验证 COM 绑定端到端可用：GUID 打包正确、
// vtable 槽位正确（GetOptions 成功返回默认选项）。
func TestCreateFileOpenDialogCOM(t *testing.T) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartment)
	if uhr := uint32(hr); uhr != 0 && uhr != hrSFalse && uhr != hrRPCChangedMode {
		t.Fatalf("CoInitializeEx failed: 0x%08x", hr)
	}
	dialog, err := createFileOpenDialog()
	if err != nil {
		t.Fatalf("createFileOpenDialog: %v", err)
	}
	vtbl := (*iFileDialogVtbl)(vtblOf(dialog))
	defer syscall.SyscallN(vtbl.release, uintptr(unsafe.Pointer(dialog)))

	var options uint32
	if hr := comMethod(vtbl.getOptions, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(&options))); hr != 0 {
		t.Fatalf("GetOptions failed: 0x%08x (vtable slot likely wrong)", hr)
	}
	options |= fosPickFolders
	if hr := comMethod(vtbl.setOptions, uintptr(unsafe.Pointer(dialog)), uintptr(options)); hr != 0 {
		t.Fatalf("SetOptions failed: 0x%08x", hr)
	}
}
