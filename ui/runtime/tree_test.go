package runtime

import "testing"

func TestFindByID(t *testing.T) {
	tree := []*VNode{
		H("vstack", Props{ID: "root"},
			H("button", Props{ID: "ok"}, "确定"),
			Text("x"),
		),
	}
	if FindByID(tree, "ok") == nil {
		t.Fatal("未找到 ok")
	}
	if FindByID(tree, "missing") != nil {
		t.Fatal("不应找到 missing")
	}
}
