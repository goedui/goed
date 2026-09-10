package components

import "testing"

func TestTitleToolbarDragRegions(t *testing.T) {
	toolbar := NewTitleToolbar("Goed")
	toolbar.width = 1200

	cases := []struct {
		x, y int32
		drag bool
	}{
		{100, 20, false},  // menu
		{260, 20, false},  // toolbar button
		{950, 20, false},  // search
		{1080, 20, false}, // window controls
		{700, 20, true},   // empty title-bar space
		{700, 50, false},  // editor area
	}
	for _, tc := range cases {
		if got := toolbar.IsDraggable(tc.x, tc.y); got != tc.drag {
			t.Errorf("IsDraggable(%d, %d) = %v, want %v", tc.x, tc.y, got, tc.drag)
		}
	}
}
