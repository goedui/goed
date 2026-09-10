package components

import "testing"

func TestDialogConfirmAcceptsAndCancels(t *testing.T) {
	dialog := NewDialog("", "")
	confirmed, cancelled := false, false
	dialog.ShowConfirm("删除", "将 main.go 移入回收站？", func() { confirmed = true })
	dialog.OnCancel = func() { cancelled = true }
	if !dialog.Confirm || !dialog.IsOpen() {
		t.Fatal("ShowConfirm should open in confirm mode")
	}
	panel := dialog.Bounds(800, 600)
	confirm := dialog.buttonBounds(panel)
	if !dialog.HandleMouseDown(confirm.X+1, confirm.Y+1, 800, 600) || dialog.IsOpen() || !confirmed || cancelled {
		t.Fatal("confirm button should invoke OnConfirm only")
	}

	dialog.ShowConfirm("删除", "将 main.go 移入回收站？", func() { confirmed = true })
	if !dialog.HandleKeyDown(0x1B) || dialog.IsOpen() || !cancelled {
		t.Fatal("Escape should invoke OnCancel")
	}
}

func TestDialogShowResetsConfirmMode(t *testing.T) {
	dialog := NewDialog("", "")
	dialog.ShowConfirm("删除", "msg", func() {})
	dialog.Show()
	if dialog.Confirm {
		t.Fatal("plain Show should restore single-button mode")
	}
}
