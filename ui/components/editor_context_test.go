package components

import (
	"testing"

	"github.com/goedui/goed/editor/composables"
)

func TestEditorContextMenuAndClipboardProvider(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello"}
	e.Selection = composables.NewSelection(
		composables.Position{Line: 0, Column: 0},
		composables.Position{Line: 0, Column: 5},
	)
	clipboardText := ""
	e.SetClipboardProvider(func() string { return clipboardText }, func(value string) {
		clipboardText = value
	})

	// The editor's internal button contract uses 2 for right click; app maps
	// Win32's button value 1 before forwarding the event.
	e.HandleMouseDown(0, 0, 2, false, false, 1)
	if !e.ContextMenu.IsVisible {
		t.Fatal("right click did not open the editor context menu")
	}

	for _, item := range e.ContextMenu.Items {
		if item.Shortcut == "Ctrl+C" {
			if !item.Enabled || item.OnClick == nil {
				t.Fatal("copy menu item should be enabled for a selection")
			}
			item.OnClick()
			if clipboardText != "hello" {
				t.Fatalf("clipboard provider received %q, want %q", clipboardText, "hello")
			}
			return
		}
	}
	t.Fatal("copy menu item was not present")
}

func TestEditorMouseDragUpdatesSelection(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello world"}
	e.HandleMouseDown(28, 0, 0, false, false, 1)
	e.HandleMouseDrag(73, 0)
	e.HandleMouseUp()

	if got := e.GetSelectedText(); got != "hello" {
		t.Fatalf("drag selection = %q, want %q", got, "hello")
	}
}

func TestEditorContextMenuIncludesExplorerAction(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello"}
	e.FilePath = "C:\\work\\hello.go"
	called := false
	e.OnShowInExplorer = func() { called = true }
	e.HandleRightClick(0, 0)

	for _, item := range e.ContextMenu.Items {
		if item.Shortcut == "" && item.OnClick != nil {
			if !item.Enabled {
				t.Fatal("explorer action should be enabled for a saved document")
			}
			item.OnClick()
			if !called {
				t.Fatal("explorer action did not invoke its callback")
			}
			return
		}
	}
	t.Fatal("explorer action was not present")
}

func TestEditorMultilineSelectionClipboardActions(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello", "world", "tail"}
	e.Selection = composables.NewSelection(
		composables.Position{Line: 0, Column: 2},
		composables.Position{Line: 1, Column: 3},
	)
	clipboardText := ""
	e.SetClipboardProvider(func() string { return clipboardText }, func(value string) {
		clipboardText = value
	})

	if got := e.GetSelectedText(); got != "llo\nwor" {
		t.Fatalf("selected text = %q, want %q", got, "llo\nwor")
	}
	e.Cut()
	if clipboardText != "llo\nwor" {
		t.Fatalf("cut clipboard = %q, want %q", clipboardText, "llo\nwor")
	}
	if got := e.Lines; len(got) != 2 || got[0] != "held" || got[1] != "tail" {
		t.Fatalf("lines after cut = %#v, want [held tail]", got)
	}

	e.Paste()
	if got := e.Lines; len(got) != 3 || got[0] != "hello" || got[1] != "world" || got[2] != "tail" {
		t.Fatalf("lines after paste = %#v, want [hello world tail]", got)
	}
}
