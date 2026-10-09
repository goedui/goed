// stringList / fmtSprint 的常驻回归。
//
// stringList 是 select / popover 拿 items 的**唯一**入口，而 items 有两条来源：
//
//   - Go 侧直接写 `Items: []string{...}`；
//   - 宿主桥（JS / JSON 数组）递进来的是 `[]any{...}`，元素可能是字符串、
//     数字、布尔，甚至 nil（JSON 的 null）。
//
// 第二条分支靠 fmtSprint 逐个转字符串。删掉它 stringList 会**静默返回 nil**：
// Select 拿到 len(items)==0，点开是个空面板；Popover 的上下键选择也直接失效。
// 不报错、不 panic，只是「点了没反应」——所以这条分支必须有人守。
package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

func TestStringListKeepsStringSliceIdentity(t *testing.T) {
	out := stringList([]string{"a", "b"})
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("[]string 应原样返回，得到 %#v", out)
	}
}

func TestStringListCoercesAnySlice(t *testing.T) {
	out := stringList([]any{"a", 1, nil, true, 3.5})
	want := []string{"a", "1", "", "true", "3.5"}
	if len(out) != len(want) {
		t.Fatalf("[]any 转出 %d 条，期望 %d：%#v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("out[%d] = %q，期望 %q（整份 %#v）", i, out[i], want[i], out)
		}
	}
}

func TestStringListRejectsOtherShapes(t *testing.T) {
	for _, v := range []any{nil, "not-a-list", 42, []int{1, 2}} {
		if out := stringList(v); out != nil {
			t.Fatalf("stringList(%#v) = %#v，期望 nil（否则 select 会画出假条目）", v, out)
		}
	}
}

func TestFmtSprintNilIsEmptyString(t *testing.T) {
	if got := fmtSprint(nil); got != "" {
		t.Fatalf("fmtSprint(nil) = %q，期望空串：nil 条目要占一个空行，不能变成 \"<nil>\"", got)
	}
	if got := fmtSprint(42); got != "42" {
		t.Fatalf("fmtSprint(42) = %q，期望 \"42\"", got)
	}
}

// TestSelectRendersBridgeItems 是 []any 分支的端到端证明：桥递进来的数组必须
// 真的画成下拉条目。只测 []string 的那条用例（chrome_test.go 里）在分支被删掉时
// 照样绿，这条会红。
func TestSelectRendersBridgeItems(t *testing.T) {
	const id = "bridge-items"
	SetSelectOpen(id, false)
	t.Cleanup(func() { SetSelectOpen(id, false) })

	// 先置展开状态再构造：Select 在构造时就决定要不要挂 popover。
	SetSelectOpen(id, true)
	n := Select(runtime.Props{
		Style: runtime.Style{Width: 160, Height: 34},
		ID:    id,
		Value: "甲",
		Items: []any{"甲", "乙", "丙"},
	})

	ctx := &mockContext{w: 640, h: 480}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Light)

	for _, want := range []string{"甲", "乙", "丙"} {
		found := false
		for _, s := range ctx.texts {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("下拉里没画出桥递进来的条目 %q：texts=%v", want, ctx.texts)
		}
	}
}
