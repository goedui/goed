package components

import (
	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"strings"
	"unicode"
)

var codeKeywords = map[string]bool{}

func init() {
	RegisterGlyph("copy", func(c renderer.Context, x, y, s float32, fg renderer.Color) {
		stroke(c, x, y, s, fg, 8, 8, 21, 8, 21, 21, 8, 21, 8, 8)
		stroke(c, x, y, s, fg, 16, 8, 16, 3, 3, 3, 3, 16, 8, 16)
	})
	for _, word := range strings.Fields("package import func type struct interface map chan go defer select range var const if else for switch case default return break continue class function let new async await export from try catch throw public private static void int string bool float64 nil null true false def with as in is not and or lambda yield pass extends implements") {
		codeKeywords[word] = true
	}
}

func highlightCode(line, language string) []TextRun {
	plain := runtime.RGB(212, 212, 212)
	if language == "diff" {
		if strings.HasPrefix(line, "+") {
			return []TextRun{{Text: line, Color: runtime.RGB(106, 187, 134)}}
		}
		if strings.HasPrefix(line, "-") {
			return []TextRun{{Text: line, Color: runtime.RGB(225, 125, 125)}}
		}
	}
	runes := []rune(line)
	var out []TextRun
	for i := 0; i < len(runes); {
		start := i
		color := plain
		r := runes[i]
		switch {
		case r == '\'' || r == '"' || r == '`':
			quote := r
			i++
			for i < len(runes) {
				if runes[i] == '\\' {
					i += 2
					continue
				}
				if runes[i] == quote {
					i++
					break
				}
				i++
			}
			if i > len(runes) {
				i = len(runes)
			}
			color = runtime.RGB(206, 145, 120)
		case r == '#' || (r == '/' && i+1 < len(runes) && runes[i+1] == '/'):
			i = len(runes)
			color = runtime.RGB(106, 153, 85)
		case unicode.IsDigit(r):
			i++
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			color = runtime.RGB(181, 206, 168)
		case unicode.IsLetter(r) || r == '_':
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			if codeKeywords[string(runes[start:i])] {
				color = runtime.RGB(86, 156, 214)
			}
		default:
			i++
		}
		out = append(out, TextRun{Text: string(runes[start:i]), Color: color})
	}
	return out
}
