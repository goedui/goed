package components

import "testing"

func TestDialogModalLifecycle(t *testing.T) {
	dialog := NewDialog("About Goed", "Goed\nVersion: v0.2")
	if dialog.IsOpen() {
		t.Fatal("new dialog should be closed")
	}
	dialog.Show()
	if !dialog.IsOpen() {
		t.Fatal("Show should open the dialog")
	}
	if !dialog.HandleMouseDown(0, 0, 800, 600) || !dialog.IsOpen() {
		t.Fatal("outside clicks should be consumed without closing a modal")
	}
	panel := dialog.Bounds(800, 600)
	button := dialog.buttonBounds(panel)
	if !dialog.HandleMouseDown(button.X+1, button.Y+1, 800, 600) || dialog.IsOpen() {
		t.Fatal("clicking OK should close the dialog")
	}
}

func TestDialogEscapeCloses(t *testing.T) {
	dialog := NewDialog("About Goed", "Version: v0.2")
	dialog.Show()
	if !dialog.HandleKeyDown(0x1B) || dialog.IsOpen() {
		t.Fatal("Escape should close the dialog")
	}
}
