// Package composables - 组合式函数
// 查找/替换功能
package composables

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SearchOptions 查找选项
type SearchOptions struct {
	CaseSensitive bool // 区分大小写
	WholeWord     bool // 全字匹配
	Regex         bool // 正则表达式
}

// SearchResult 查找结果
type SearchResult struct {
	Position Position
	Length   int
	Text     string
}

// Find 查找文本
func Find(content string, query string, opts SearchOptions) []SearchResult {
	if query == "" {
		return nil
	}
	pattern := query
	if !opts.Regex {
		pattern = regexp.QuoteMeta(pattern)
	}
	if !opts.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	expression, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	results := make([]SearchResult, 0)
	for lineNum, line := range strings.Split(content, "\n") {
		if opts.Regex {
			for _, match := range expression.FindAllStringIndex(line, -1) {
				results = appendMatch(results, line, lineNum, match[0], match[1], opts.WholeWord)
			}
			continue
		}
		for offset := 0; offset < len(line); {
			match := expression.FindStringIndex(line[offset:])
			if match == nil {
				break
			}
			start, end := offset+match[0], offset+match[1]
			results = appendMatch(results, line, lineNum, start, end, opts.WholeWord)
			_, size := utf8.DecodeRuneInString(line[start:])
			offset = start + size
		}
	}
	return results
}

func appendMatch(results []SearchResult, line string, lineNum, start, end int, wholeWord bool) []SearchResult {
	if wholeWord {
		before, _ := utf8.DecodeLastRuneInString(line[:start])
		after, _ := utf8.DecodeRuneInString(line[end:])
		if (start > 0 && isWordChar(before)) || (end < len(line) && isWordChar(after)) {
			return results
		}
	}
	return append(results, SearchResult{
		Position: Position{Line: lineNum, Column: runeCol(line, start)},
		Length:   utf8.RuneCountInString(line[start:end]),
		Text:     line[start:end],
	})
}

// runeCol 把字节偏移转换为行内 rune 列（编辑器光标坐标系）。
func runeCol(line string, bytePos int) int {
	return len([]rune(line[:bytePos]))
}

// Replace 替换文本
func Replace(content string, query string, replacement string, opts SearchOptions) string {
	results := Find(content, query, opts)
	if len(results) == 0 {
		return content
	}
	nonOverlapping := make([]SearchResult, 0, len(results))
	previousLine, previousEnd := -1, 0
	for _, result := range results {
		if result.Position.Line == previousLine && result.Position.Column < previousEnd {
			continue
		}
		nonOverlapping = append(nonOverlapping, result)
		previousLine = result.Position.Line
		previousEnd = result.Position.Column + result.Length
	}
	results = nonOverlapping

	lines := strings.Split(content, "\n")

	// 从后往前替换（避免位置偏移）。Column/Length 均为 rune 坐标。
	for i := len(results) - 1; i >= 0; i-- {
		result := results[i]
		runes := []rune(lines[result.Position.Line])

		col := result.Position.Column
		newLine := string(runes[:col]) +
			replacement +
			string(runes[col+result.Length:])

		lines[result.Position.Line] = newLine
	}

	return strings.Join(lines, "\n")
}

// isWordChar 是否为单词字符
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
