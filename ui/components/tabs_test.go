package components

import "testing"

func TestTabsCloseKeepsItemsAndActiveIndexInSync(t *testing.T) {
	tabs := NewTabs()
	tabs.Open("one")
	tabs.Open("two")
	tabs.Open("three")
	tabs.Active = 1

	if !tabs.Close("two") {
		t.Fatal("Close should remove an existing tab")
	}
	if len(tabs.Items) != 2 || tabs.Items[0].Path != "one" || tabs.Items[1].Path != "three" {
		t.Fatalf("items after middle close = %#v", tabs.Items)
	}
	if tabs.Active != 1 {
		t.Fatalf("active index after middle close = %d, want 1", tabs.Active)
	}

	if !tabs.Close("three") || tabs.Active != 0 {
		t.Fatalf("active index after last close = %d, want 0", tabs.Active)
	}
	if !tabs.Close("one") || tabs.Active != -1 || len(tabs.Items) != 0 {
		t.Fatalf("empty tabs state = len:%d active:%d", len(tabs.Items), tabs.Active)
	}
	if tabs.Close("missing") {
		t.Fatal("Close should report false for an unknown path")
	}
}
