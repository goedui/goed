package components

import "testing"

func TestInputDialogTypingAndConfirm(t *testing.T) {
	d := NewInputDialog()
	confirmed := ""
	d.OnConfirm = func(value string) { confirmed = value }
	d.Show("重命名", "位置: C:\\demo", "old.txt")
	if !d.IsOpen() {
		t.Fatal("Show should open the dialog")
	}
	for _, c := range "2" {
		d.HandleChar(c)
	}
	// 两次 Backspace 后再确认，验证编辑路径。
	d.HandleKeyDown(0x08)
	d.HandleKeyDown(0x08)
	d.HandleKeyDown(0x0D) // Enter 确认
	if d.IsOpen() {
		t.Fatal("Enter should close the dialog")
	}
	if confirmed != "old.tx" {
		t.Fatalf("confirmed = %q, want %q", confirmed, "old.tx")
	}
}

func TestInputDialogEmptyValueNotConfirmed(t *testing.T) {
	d := NewInputDialog()
	called := false
	d.OnConfirm = func(value string) { called = true }
	d.Show("新建文件", "位置: C:\\demo", "")
	d.HandleKeyDown(0x0D)
	if !d.IsOpen() || called {
		t.Fatal("empty value must not submit or close")
	}
	d.Value = []rune("   ")
	d.HandleKeyDown(0x0D)
	if !d.IsOpen() || called {
		t.Fatal("blank value must not submit or close")
	}
}

func TestInputDialogEscapeAndOutsideClickCancel(t *testing.T) {
	d := NewInputDialog()
	called := false
	d.OnConfirm = func(value string) { called = true }
	d.Show("新建文件夹", "位置: C:\\demo", "src")
	// 800x600 视口下面板居中于 (180,205)，(10,10) 在面板外。
	if !d.HandleMouseDown(10, 10, 800, 600) || d.IsOpen() || called {
		t.Fatal("outside click should cancel and be consumed")
	}
	d.Show("新建文件夹", "位置: C:\\demo", "src")
	if !d.HandleKeyDown(0x1B) || d.IsOpen() || called {
		t.Fatal("Escape should cancel")
	}
}

func TestInputDialogButtonHits(t *testing.T) {
	d := NewInputDialog()
	confirmed := ""
	d.OnConfirm = func(value string) { confirmed = value }
	d.Show("新建文件", "位置: C:\\demo", "a.go")
	panel := d.panelBounds(800, 600)
	confirm := d.confirmBounds(panel)
	if !d.HandleMouseDown(confirm.X+1, confirm.Y+1, 800, 600) || d.IsOpen() || confirmed != "a.go" {
		t.Fatal("confirm button should submit the value")
	}
	d.Show("新建文件", "位置: C:\\demo", "a.go")
	cancel := d.cancelBounds(panel)
	if !d.HandleMouseDown(cancel.X+1, cancel.Y+1, 800, 600) || d.IsOpen() || confirmed != "a.go" {
		t.Fatal("cancel button should close without submitting")
	}
}
