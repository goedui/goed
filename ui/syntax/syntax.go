// Package syntax provides small, dependency-free syntax tokenizers for the
// editor. Tokenizers are line based so they can be used by any renderer.
package syntax

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind uint8

const (
	Plain Kind = iota
	Keyword
	String
	Comment
	Number
	Function
	Type
	Operator
	Heading
	Link
)

type Token struct {
	Text string
	Kind Kind
}

type Language struct {
	Name  string
	Match func(path string) bool
	Token func(lines []string) [][]Token
}

var languages = []Language{
	{Name: "go", Match: matchExtensions(".go"), Token: tokenizeGo},
	{Name: "markdown", Match: matchExtensions(".md", ".markdown", ".mdown", ".mkdn"), Token: tokenizeMarkdown},
	{Name: "code", Match: matchExtensions(
		".js", ".jsx", ".ts", ".tsx", ".json", ".css", ".java", ".c", ".h", ".cc", ".cpp", ".cxx",
		".rs", ".swift", ".kt", ".kts", ".scala", ".py", ".rb", ".php", ".lua", ".sh", ".bash",
		".zsh", ".fish", ".yaml", ".yml", ".toml", ".ini", ".conf", ".sql", ".html", ".htm",
		".xml", ".xhtml", ".vue", ".svelte",
	), Token: tokenizeGenericCode},
	{Name: "generic", Match: func(string) bool { return true }, Token: tokenizeGeneric},
}

func Detect(path string) string {
	for _, language := range languages {
		if language.Match(path) {
			return language.Name
		}
	}
	return "generic"
}

func Highlight(lines []string, path string) [][]Token {
	languageName := Detect(path)
	for _, language := range languages {
		if language.Name == languageName {
			return language.Token(lines)
		}
	}
	return tokenizeGeneric(lines)
}

func Register(language Language) {
	if language.Name == "" || language.Token == nil || language.Match == nil {
		return
	}
	for i := range languages {
		if languages[i].Name == language.Name {
			languages[i] = language
			return
		}
	}
	languages = append([]Language{language}, languages...)
}

func matchExtensions(extensions ...string) func(string) bool {
	return func(path string) bool {
		ext := strings.ToLower(filepath.Ext(path))
		for _, candidate := range extensions {
			if ext == candidate {
				return true
			}
		}
		return false
	}
}

func tokenizeGeneric(lines []string) [][]Token {
	result := make([][]Token, len(lines))
	for i, line := range lines {
		result[i] = []Token{{Text: line, Kind: Plain}}
	}
	return result
}

func tokenizeGenericCode(lines []string) [][]Token {
	result := make([][]Token, len(lines))
	inBlockComment := false
	for i, line := range lines {
		result[i], inBlockComment = tokenizeCodeLine(line, inBlockComment, genericKeywords, genericTypes, true)
	}
	return result
}

// The generic profile intentionally combines the common vocabulary used by
// C-like, scripting, and configuration languages. It provides useful color
// hints for files without requiring a separate parser for every extension.
var genericKeywords = map[string]bool{
	"as": true, "async": true, "await": true, "break": true, "case": true, "catch": true,
	"class": true, "const": true, "continue": true, "def": true, "default": true,
	"defer": true, "delete": true, "do": true, "else": true, "enum": true, "export": true,
	"extends": true, "fallthrough": true, "finally": true, "for": true, "from": true,
	"fn": true, "func": true, "function": true, "go": true, "if": true, "import": true,
	"in": true, "interface": true, "let": true, "match": true, "module": true, "namespace": true,
	"new": true, "package": true, "pass": true, "private": true, "protected": true,
	"public": true, "range": true, "return": true, "select": true, "static": true,
	"struct": true, "switch": true, "throw": true, "trait": true, "try": true, "type": true,
	"var": true, "when": true, "while": true, "with": true, "yield": true,
}

var genericTypes = map[string]bool{
	"any": true, "bool": true, "boolean": true, "byte": true, "char": true, "double": true,
	"error": true, "float": true, "float32": true, "float64": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "integer": true, "list": true, "map": true,
	"number": true, "object": true, "rune": true, "set": true, "string": true, "str": true,
	"tuple": true, "uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
}

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

var goTypes = map[string]bool{
	"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true, "int": true, "int8": true, "int16": true,
	"int32": true, "int64": true, "rune": true, "string": true, "uint": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
}

func tokenizeGo(lines []string) [][]Token {
	result := make([][]Token, len(lines))
	inBlockComment := false
	for i, line := range lines {
		var tokens []Token
		tokens, inBlockComment = tokenizeGoLine(line, inBlockComment)
		retokenizeGoFuncNames(tokens)
		result[i] = tokens
	}
	return result
}

// retokenizeGoFuncNames 把 func 声明名标为 Function。泛型声明
// （func Name[T any](...）的名字后面是 '['，走不到调用判定的 '(' 规则，
// 因此按 "func 关键字后的第一个标识符" 后处理补齐 VSCode 的声明名黄色。
func retokenizeGoFuncNames(tokens []Token) {
	expect := false
	for i := range tokens {
		switch {
		case tokens[i].Kind == Keyword && strings.TrimSpace(tokens[i].Text) == "func":
			expect = true
		case expect && tokens[i].Kind == Plain && strings.TrimSpace(tokens[i].Text) == "":
			// func 与名字之间的空白，继续等待。
		case expect && (tokens[i].Kind == Type || tokens[i].Kind == Function || tokens[i].Kind == Plain):
			if name := strings.TrimSpace(tokens[i].Text); name != "" && isIdentStart(name[0]) {
				tokens[i].Kind = Function
			}
			expect = false
		default:
			expect = false
		}
	}
}

func tokenizeGoLine(line string, inBlockComment bool) ([]Token, bool) {
	return tokenizeCodeLine(line, inBlockComment, goKeywords, goTypes, false)
}

func tokenizeCodeLine(line string, inBlockComment bool, keywords, types map[string]bool, hashComments bool) ([]Token, bool) {
	var result []Token
	flush := func(text string, kind Kind) {
		if text != "" {
			if len(result) > 0 && result[len(result)-1].Kind == kind {
				result[len(result)-1].Text += text
			} else {
				result = append(result, Token{Text: text, Kind: kind})
			}
		}
	}
	for pos := 0; pos < len(line); {
		if inBlockComment {
			end := strings.Index(line[pos:], "*/")
			if end < 0 {
				flush(line[pos:], Comment)
				return result, true
			}
			end += pos + 2
			flush(line[pos:end], Comment)
			pos = end
			inBlockComment = false
			continue
		}
		if strings.HasPrefix(line[pos:], "//") || (hashComments && line[pos] == '#' && (pos == 0 || unicode.IsSpace(rune(line[pos-1])))) {
			flush(line[pos:], Comment)
			break
		}
		if strings.HasPrefix(line[pos:], "/*") {
			inBlockComment = true
			continue
		}
		if line[pos] == '"' || line[pos] == '\'' || line[pos] == '`' {
			quote := line[pos]
			start := pos
			pos++
			for pos < len(line) {
				if line[pos] == '\\' && quote != '`' {
					if pos+1 < len(line) {
						pos += 2
					} else {
						// Keep an unterminated trailing escape inside the string token.
						pos++
					}
					continue
				}
				if line[pos] == quote {
					pos++
					break
				}
				pos++
			}
			flush(line[start:pos], String)
			continue
		}
		if isDigit(line[pos]) {
			start := pos
			for pos < len(line) && (isDigit(line[pos]) || strings.ContainsRune("._", rune(line[pos]))) {
				pos++
			}
			flush(line[start:pos], Number)
			continue
		}
		if line[pos] == ' ' || line[pos] == '\t' {
			start := pos
			for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
				pos++
			}
			flush(line[start:pos], Plain)
			continue
		}
		if isIdentStart(line[pos]) {
			start := pos
			for pos < len(line) && isIdentPart(line[pos]) {
				pos++
			}
			word := line[start:pos]
			kind := Plain
			if keywords[word] {
				kind = Keyword
			} else if pos < len(line) && strings.TrimSpace(line[pos:]) != "" && line[pos] == '(' {
				// 函数调用/声明（foo(、x.Get(、func bar(）在类型判定之前检查，
				// 否则大写开头的函数名会被短路成 Type，丢失 VSCode 的函数黄色。
				kind = Function
			} else if types[word] || (len(word) > 0 && unicode.IsUpper(rune(word[0]))) {
				kind = Type
			}
			flush(word, kind)
			continue
		}
		// Decode one complete rune so non-ASCII text remains valid UTF-8 when
		// it is sent to the drawing backend. Unknown Unicode characters are
		// plain text; punctuation stays an operator.
		r, size := utf8.DecodeRuneInString(line[pos:])
		if size == 0 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			flush(line[pos:pos+size], Plain)
		} else {
			flush(line[pos:pos+size], Operator)
		}
		pos += size
	}
	return result, inBlockComment
}

func tokenizeMarkdown(lines []string) [][]Token {
	result := make([][]Token, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			prefix := len(line) - len(trimmed)
			result[i] = []Token{{Text: line[:prefix], Kind: Plain}, {Text: trimmed, Kind: Heading}}
			continue
		}
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			prefix := len(line) - len(trimmed)
			result[i] = []Token{{Text: line[:prefix], Kind: Plain}, {Text: trimmed[:1], Kind: Keyword}, {Text: trimmed[1:], Kind: Plain}}
			continue
		}
		result[i] = tokenizeMarkdownInline(line)
	}
	return result
}

func tokenizeMarkdownInline(line string) []Token {
	var result []Token
	for pos := 0; pos < len(line); {
		start := pos
		if line[pos] == '[' {
			if close := strings.IndexByte(line[pos+1:], ']'); close >= 0 {
				close += pos + 1
				if close+1 < len(line) && line[close+1] == '(' {
					if end := strings.IndexByte(line[close+2:], ')'); end >= 0 {
						end += close + 3
						result = append(result, Token{Text: line[pos:end], Kind: Link})
						pos = end
						continue
					}
				}
			}
			result = append(result, Token{Text: line[pos : pos+1], Kind: Plain})
			pos++
			continue
		}
		if strings.ContainsRune("`*_", rune(line[pos])) {
			marker := line[pos]
			if end := strings.IndexByte(line[pos+1:], marker); end >= 0 {
				end += pos + 2
				result = append(result, Token{Text: line[pos:end], Kind: String})
				pos = end
				continue
			}
			result = append(result, Token{Text: line[pos : pos+1], Kind: Plain})
			pos++
			continue
		}
		for pos < len(line) && !strings.ContainsRune("[`*_", rune(line[pos])) {
			pos++
		}
		if pos > start {
			result = append(result, Token{Text: line[start:pos], Kind: Plain})
		}
	}
	return result
}

func isDigit(b byte) bool      { return b >= '0' && b <= '9' }
func isIdentStart(b byte) bool { return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }
func isIdentPart(b byte) bool  { return isIdentStart(b) || isDigit(b) }
