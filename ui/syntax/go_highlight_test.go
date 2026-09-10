package syntax

import (
	"testing"
)

// TestGoHighlightMatchesVSCodeDarkPlusSemantics 对齐 VSCode Dark+ 的
// 语义着色：函数名/调用为 Function（黄），大写开头类型为 Type（青），
// 字符串为 String（橙），关键字 Keyword（蓝）。历史上的回归是"大写开头
// → Type"短路了"后跟 ( → Function"，导致所有函数名显示为青蓝色。
func TestGoHighlightMatchesVSCodeDarkPlusSemantics(t *testing.T) {
	lines := []string{
		"package composables",
		"",
		"import (",
		"\t\"log\"",
		"\t\"github.com/goedui/goed/ui/reactive\"",
		")",
		"",
		"func UseStorage[T any](store *storage.Store, key string, def T) (*reactive.RefValue[T], func(T)) {",
		"\tvalue := reactive.Ref(def)",
		"\tif raw, ok := store.Get(key); ok {",
		"\t\tvalue.SetSilent(typed)",
		"\t}",
		"\treturn value",
		"}",
	}
	tokens := flatten(Highlight(lines, "useStorage.go"))

	cases := []struct {
		text string
		kind Kind
	}{
		{"package", Keyword},
		{"import", Keyword},
		{"\"log\"", String},
		{"\"github.com/goedui/goed/ui/reactive\"", String},
		{"UseStorage", Function}, // 泛型函数声明名：名字后是 '['
		{"UseStorage", Function},
		{"Store", Type},
		{"RefValue", Type},
		{"Ref", Function},     // 大写开头但被调用 → Function 而非 Type
		{"Get", Function},     // 方法调用
		{"SetSilent", Function},
		{"string", Type},
		{"if", Keyword},
		{"return", Keyword},
	}
	for _, c := range cases {
		if got := tokens[c.text]; got != c.kind {
			t.Errorf("%q kind = %d, want %d", c.text, got, c.kind)
		}
	}
}

// flatten 收集 token 文本 → kind 的最后取值（同名 token 可能出现多次）。
func flatten(tokens [][]Token) map[string]Kind {
	out := map[string]Kind{}
	for _, line := range tokens {
		for _, tok := range line {
			out[tok.Text] = tok.Kind
		}
	}
	return out
}
