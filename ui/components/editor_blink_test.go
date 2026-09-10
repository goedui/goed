package components

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestCursorBlinkTogglesVisibility(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello"}
	e.VisibleLines = 10
	if !e.CursorVisible() {
		t.Fatal("cursor should start visible")
	}

	// 无宿主（OnRedraw == nil）时手动翻转相位。
	e.BlinkNow()
	if e.CursorVisible() {
		t.Fatal("cursor should be hidden after first blink toggle")
	}
	e.BlinkNow()
	if !e.CursorVisible() {
		t.Fatal("cursor should be visible again after second toggle")
	}
}

func TestCursorBlinkResetsOnEdit(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello"}

	// 翻转到隐藏相位，随后输入应立即恢复可见。
	e.BlinkNow()
	if e.CursorVisible() {
		t.Fatal("setup: cursor should be hidden")
	}

	e.InsertChar('a')
	if !e.CursorVisible() {
		t.Fatal("cursor must be visible immediately after typing (VSCode behavior)")
	}
}

func TestCursorBlinkResetsOnCursorMove(t *testing.T) {
	e := NewEditor()
	e.Lines = []string{"hello", "world"}

	e.BlinkNow()
	if e.CursorVisible() {
		t.Fatal("setup: cursor should be hidden")
	}

	e.MoveCursor(1, 0)
	if !e.CursorVisible() {
		t.Fatal("cursor must be visible after arrow-key move")
	}

	e.BlinkNow()
	e.MoveCursorToLineEnd()
	if !e.CursorVisible() {
		t.Fatal("cursor must be visible after End move")
	}
}

func TestCursorBlinkTimerFiresRedraw(t *testing.T) {
	var redraws int32
	e := NewEditor()
	e.SetOnRedraw(func() { atomic.AddInt32(&redraws, 1) })

	// 等待略长于一个闪烁周期：AfterFunc 应翻转并触发重绘。
	deadline := time.Now().Add(2 * cursorBlinkInterval)
	for time.Now().Before(deadline) && atomic.LoadInt32(&redraws) == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&redraws) == 0 {
		t.Fatal("blink timer did not trigger OnRedraw within one interval")
	}
	if e.CursorVisible() {
		t.Fatal("cursor should have toggled to hidden phase")
	}
}

func TestResetCursorBlinkRestartsTimer(t *testing.T) {
	var redraws int32
	e := NewEditor()
	e.SetOnRedraw(func() { atomic.AddInt32(&redraws, 1) })

	// 反复复位计时器：短时间内不应有任何翻转重绘。
	deadline := time.Now().Add(3 * cursorBlinkInterval / 4)
	for time.Now().Before(deadline) {
		e.ResetCursorBlink()
		time.Sleep(30 * time.Millisecond)
	}
	if n := atomic.LoadInt32(&redraws); n != 0 {
		t.Fatalf("expected no blink redraws while typing, got %d", n)
	}
	if !e.CursorVisible() {
		t.Fatal("cursor should stay visible while resetting")
	}
}
