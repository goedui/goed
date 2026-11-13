package composables

import "testing"

func TestFindUnicodeCaseFold(t *testing.T) {
	results := Find("\u212a k", "k", SearchOptions{})
	if len(results) != 2 || results[0].Text != "\u212a" || results[0].Position.Column != 0 || results[1].Position.Column != 2 {
		t.Fatalf("unexpected Unicode matches: %+v", results)
	}
	if got := Replace("\u212a k", "k", "x", SearchOptions{}); got != "x x" {
		t.Fatalf("Unicode replacement: %q", got)
	}
}

func TestReplaceOverlappingMatches(t *testing.T) {
	if got := Replace("aaa", "aa", "", SearchOptions{CaseSensitive: true}); got != "a" {
		t.Fatalf("overlapping replacement: %q", got)
	}
	if got := Replace("aaaaa", "aa", "x", SearchOptions{}); got != "xxa" {
		t.Fatalf("multiple overlapping replacements: %q", got)
	}
}

func TestFindWholeWordUnicodeAndRegex(t *testing.T) {
	for _, options := range []SearchOptions{{WholeWord: true}, {WholeWord: true, Regex: true}} {
		results := Find("one \u4e2done one_ one", "one", options)
		if len(results) != 2 || results[0].Position.Column != 0 || results[1].Position.Column != 14 {
			t.Fatalf("whole word matches: %+v", results)
		}
	}
}
