// Package composables - 组合式函数
// 查找/替换功能
package composables

import (
	"regexp"
	"strings"
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

	results := make([]SearchResult, 0)
	lines := strings.Split(content, "\n")

	// 正则表达式模式
	if opts.Regex {
		pattern := query
		if !opts.CaseSensitive {
			pattern = "(?i)" + pattern
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil
		}

		for lineNum, line := range lines {
			matches := re.FindAllStringIndex(line, -1)
			for _, match := range matches {
				// 编辑器光标使用 rune 列；正则返回字节偏移需转换。
				results = append(results, SearchResult{
					Position: Position{Line: lineNum, Column: runeCol(line, match[0])},
					Length:   len([]rune(line[match[0]:match[1]])),
					Text:     line[match[0]:match[1]],
				})
			}
		}
		return results
	}

	// 普通文本查找
	searchText := query
	if !opts.CaseSensitive {
		searchText = strings.ToLower(searchText)
	}

	for lineNum, line := range lines {
		searchLine := line
		if !opts.CaseSensitive {
			searchLine = strings.ToLower(searchLine)
		}

		index := 0
		for {
			pos := strings.Index(searchLine[index:], searchText)
			if pos == -1 {
				break
			}

			actualPos := index + pos

			// 全字匹配检查
			if opts.WholeWord {
				// 检查前一个字符
				if actualPos > 0 && isWordChar(rune(line[actualPos-1])) {
					index = actualPos + 1
					continue
				}

				// 检查后一个字符
				endPos := actualPos + len(query)
				if endPos < len(line) && isWordChar(rune(line[endPos])) {
					index = actualPos + 1
					continue
				}
			}

			results = append(results, SearchResult{
				Position: Position{Line: lineNum, Column: runeCol(line, actualPos)},
				Length:   len([]rune(query)),
				Text:     line[actualPos : actualPos+len(query)],
			})

			index = actualPos + 1
		}
	}

	return results
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
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}
