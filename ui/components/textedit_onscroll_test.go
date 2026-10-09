// 编辑区翻页的视图跟随与 onScroll 通知。
//
// 两条此前没人守的行为：
//
//   - 大文档（sparse：> maxLayoutRun 个 rune 的 visual / ≥256KB 的 large）
//     的绘制不消费 scrollSync —— PageUp/PageDown 光标翻页了，视图却纹丝不动，
//     看起来就像「编辑区翻页没生效」（md-previewer 的 2730 行文档正是这个
//     场景）。现在两条绘制路径同一套光标跟随规则。
//   - 滚轮 / 光标跟随 / SetTextareaScrollY 三条滚动来源各改各的偏移，「滚动
//     变了要通知应用」无人负责 —— 「编辑区滚动 → 预览同步」的联动挂不上。
//     现在通知统一收在绘制出口，偏移没变不重复打扰。
package components

import (
	"strings"
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// TestPageDownOnSparseLayoutScrollsViewport 稀疏布局的翻页要连视图一起翻：
// 光标下移一页后重新绘制，滚动偏移应跟到光标处（此前 scrollSync 被无视，
// 按 PageDown 光标和视图都不动）。
func TestPageDownOnSparseLayoutScrollsViewport(t *testing.T) {
	n := paintArea(t, "tk-sparse-page", numberedLines(800))
	st := stateFor(n)
	if !st.layout.sparse() {
		t.Fatal("样本应走稀疏布局")
	}
	MoveCaret(n, 0, 0)
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyPageDown, Down: true})
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)
	step := pageStep(t, n)
	want := float32(step) * st.layout.lineH
	if absf(st.scrollY-want) > 1 {
		t.Fatalf("翻页应滚动一整页：scrollY=%.1f 期望 %.1f", st.scrollY, want)
	}
}

// TestPageUpOnSparseLayoutScrollsBackUp 上翻对称：翻回去后视图跟着回到顶部。
func TestPageUpOnSparseLayoutScrollsBackUp(t *testing.T) {
	n := paintArea(t, "tk-sparse-pageup", numberedLines(800))
	st := stateFor(n)
	MoveCaret(n, 0, 0)
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyPageDown, Down: true})
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)
	if st.scrollY <= 0 {
		t.Fatalf("前置条件失败：翻页后应已下滚，实际 scrollY=%.1f", st.scrollY)
	}
	runtime.DispatchKey([]*runtime.VNode{n}, runtime.KeyEvent{Key: runtime.KeyPageUp, Down: true})
	renderer.PaintTree(&mockContext{w: 500, h: 400}, []*runtime.VNode{n}, theme.Dark)
	if st.scrollY != 0 {
		t.Fatalf("上翻回顶后视图应回到顶部，实际 scrollY=%.1f", st.scrollY)
	}
}

// TestTextareaNotifiesOnScrollOncePerChange onScroll 只在滚动偏移真正变化时
// 通知一次：重复绘制不重复打扰，滚一次报一次。
func TestTextareaNotifiesOnScrollOncePerChange(t *testing.T) {
	var got []float32
	// 高度必须显式限住：不限高时 textarea 随内容自增，节点高 = 内容高、
	// 可滚空间为 0，滚轮偏移会被 clampScrollY 收回 0（滚动都滚不动，
	// 自然也无从通知）。
	n := Textarea(runtime.Props{
		ID: "tk-onscroll", Value: numberedLines(800),
		Style:    runtime.Style{Height: 400},
		OnScroll: func(y float32) { got = append(got, y) },
	})
	t.Cleanup(func() { ResetInputState(n.ID()) })
	ctx := &mockContext{w: 500, h: 400}

	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if len(got) != 0 {
		t.Fatalf("没滚动不应通知，实际 %v", got)
	}
	if onFloat(n, "onScroll") == nil {
		t.Fatal("OnScroll 应挂到节点的 onScroll 属性上")
	}

	// 滚轮 delta 负 = 向下滚，偏移 +|delta|*48。
	n.Attrs["onWheel"].(func(float32, float32, float32) bool)(0, 0, -2)
	if y := stateFor(n).scrollY; y != 96 {
		t.Fatalf("滚轮后 scrollY = %.1f，期望 96", y)
	}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if len(got) != 1 {
		t.Fatalf("滚动一次应通知一次，实际 %v", got)
	}
	if got[0] != 96 {
		t.Fatalf("通知值 = %.1f，期望 96（|delta|*48）", got[0])
	}

	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	if len(got) != 1 {
		t.Fatalf("偏移没变不应重复通知，实际 %v", got)
	}
}

func TestArrowKeysRevealCaretOnSparseLayout(t *testing.T) {
	const id = "tk-sparse-x"
	ResetInputState(id)
	t.Cleanup(func() {
		runtime.SetFocus("")
		ResetInputState(id)
	})
	line := strings.Repeat("a", 200)
	text := strings.Repeat(line+"\n", 200)
	n := Textarea(runtime.Props{
		ID: id, Value: text, Nowrap: true,
		Style: runtime.Style{Width: 120, Height: 200, FontSize: 14},
	})
	n.X, n.Y, n.W, n.H = 0, 0, 120, 200
	runtime.SetFocus(id)
	ctx := &mockContext{w: 400, h: 300}
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	st := stateFor(n)
	if st.layout == nil || !st.layout.sparse() {
		t.Fatal("sample should use the sparse layout")
	}
	MoveCaret(n, 0, 0)
	renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	for i := 0; i < 80; i++ {
		press(n, runtime.KeyRight)
		renderer.PaintTree(ctx, []*runtime.VNode{n}, theme.Dark)
	}
	if st.scrollX <= 1 {
		t.Fatalf("horizontal caret follow on sparse layout, scrollX=%.1f", st.scrollX)
	}
}

