package components

import (
	"github.com/goedui/goed/ui/core"
	"strings"
	"testing"
)

func TestInputMultilineSelectionAndPaste(t *testing.T) {
	i := NewInput("猫")
	i.Multiline = true
	i.SetFocused(true)
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: core.KeyEnter})
	i.HandleEvent(core.Event{Type: core.EventKeyChar, Char: '狗'})
	if i.Value != "猫\n狗" {
		t.Fatalf("value=%q", i.Value)
	}
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A', Ctrl: true})
	i.ReadClipboard = func() string { return "hello\r\nworld" }
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'V', Ctrl: true})
	if i.Value != "hello\nworld" {
		t.Fatalf("paste=%q", i.Value)
	}
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: core.KeyLeft, Shift: true})
	i.HandleEvent(core.Event{Type: core.EventKeyChar, Char: '猫'})
	if i.Value != "hello\nworl猫" {
		t.Fatalf("selection replacement=%q", i.Value)
	}
}
func TestInputPasswordNeverPaintsSecret(t *testing.T) {
	i := NewInput("secret-key")
	i.Password = true
	i.SetFocused(true)
	i.SetBounds(core.Rect{W: 200, H: 32})
	c := &variableCtx{}
	i.Render(c)
	for _, text := range c.texts {
		if strings.Contains(text, "secret") {
			t.Fatal("secret drawn")
		}
	}
	copied := false
	i.WriteClipboard = func(string) { copied = true }
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'A', Ctrl: true})
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: 'C', Ctrl: true})
	if copied {
		t.Fatal("masked key copied")
	}
}
func TestInputWrapAndCaretStayInField(t *testing.T) {
	i := NewInput("一二三四五六七八九十")
	i.Multiline = true
	i.SetFocused(true)
	i.SetBounds(core.Rect{W: 54, H: 60})
	c := &variableCtx{}
	i.Render(c)
	if len(i.lines) < 3 || i.scrollY <= 0 {
		t.Fatalf("lines=%d scroll=%d", len(i.lines), i.scrollY)
	}
	i.HandleEvent(core.Event{Type: core.EventKeyDown, Key: core.KeyHome, Ctrl: true})
	i.Render(c)
	if i.scrollY != 0 {
		t.Fatal("caret did not scroll home")
	}
}
