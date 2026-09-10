package syntax

import "testing"

func TestDetectLanguages(t *testing.T) {
	for path, want := range map[string]string{"main.go": "go", "README.md": "markdown", "notes.txt": "generic"} {
		if got := Detect(path); got != want {
			t.Fatalf("Detect(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestGoTokens(t *testing.T) {
	lines := Highlight([]string{"func main() {", "\treturn \"ok\" // done"}, "main.go")
	if !hasToken(lines[0], Keyword, "func") || !hasToken(lines[0], Function, "main") {
		t.Fatalf("unexpected Go tokens: %#v", lines[0])
	}
	if !hasKind(lines[1], String) || !hasKind(lines[1], Comment) {
		t.Fatalf("unexpected Go string/comment tokens: %#v", lines[1])
	}
}

func TestMarkdownTokens(t *testing.T) {
	lines := Highlight([]string{"# Title", "see [docs](https://example.com)"}, "README.md")
	if !hasKind(lines[0], Heading) || !hasKind(lines[1], Link) {
		t.Fatalf("unexpected Markdown tokens: %#v", lines)
	}
}

func TestGenericCodeTokensAndUnicode(t *testing.T) {
	lines := Highlight([]string{"def привет(name): # greeting", "    return \"你好\""}, "script.py")
	if !hasToken(lines[0], Keyword, "def") || !hasKind(lines[0], Comment) {
		t.Fatalf("unexpected generic code tokens: %#v", lines[0])
	}
	if !hasToken(lines[1], String, "\"你好\"") {
		t.Fatalf("Unicode string should remain intact: %#v", lines[1])
	}
}

func TestHighlightToleratesMalformedInput(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Highlight panicked on malformed input: %v", recovered)
		}
	}()
	lines := Highlight([]string{"value = \"unterminated\\", "raw\x00text"}, "broken.py")
	if len(lines) != 2 {
		t.Fatalf("got %d highlighted lines, want 2", len(lines))
	}
}

func hasKind(tokens []Token, kind Kind) bool {
	for _, token := range tokens {
		if token.Kind == kind {
			return true
		}
	}
	return false
}

func hasToken(tokens []Token, kind Kind, text string) bool {
	for _, token := range tokens {
		if token.Kind == kind && token.Text == text {
			return true
		}
	}
	return false
}
