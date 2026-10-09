package components

import (
	"testing"

	"github.com/goedui/goed/ui/renderer"
	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/theme"
)

// lineRecorder 记下每次 DrawLine（对勾这类「线段几何」断言用）。
type checkboxLineRecorder struct {
	mockContext
	lines [][4]float32
}

func (r *checkboxLineRecorder) DrawLine(x1, y1, x2, y2 float32, _ renderer.Color, _ float32) {
	r.lines = append(r.lines, [4]float32{x1, y1, x2, y2})
}

// TestCheckboxBasics 复选框：勾选态画主题色实心框 + 两段对勾；未勾选画底框 +
// 描边、没有对勾；标签跟在框后；点击翻转并回调新值；禁用不吃点击。
func TestCheckboxBasics(t *testing.T) {
	got := []bool{}
	off := Checkbox(runtime.Props{
		ID:       "cb-off",
		Value:    false,
		Label:    "显示裁切标记",
		OnChange: func(v bool) { got = append(got, v) },
	})
	rec := &checkboxLineRecorder{mockContext: mockContext{w: 300, h: 60}}
	renderer.PaintTree(rec, []*runtime.VNode{off}, theme.Light)

	// 未勾选：方框底 + 描边，没有对勾线。
	if len(rec.rounds) != 1 {
		t.Fatalf("未勾选应只有一个圆角方框，实际 %d", len(rec.rounds))
	}
	if diff(rec.rounds[0].c, renderer.ColorFrom(theme.Light.Background)) > 0.01 {
		t.Fatalf("未勾选方框底色应为主题背景色：%+v", rec.rounds[0].c)
	}
	if len(rec.strokes) != 1 {
		t.Fatalf("未勾选应有方框描边，实际 %d", len(rec.strokes))
	}
	if len(rec.lines) != 0 {
		t.Fatalf("未勾选不应画对勾：%v", rec.lines)
	}
	found := false
	for _, s := range rec.texts {
		if s == "显示裁切标记" {
			found = true
		}
	}
	if !found {
		t.Fatalf("标签没画出来：%v", rec.texts)
	}
	// 自然宽度（MeasureNode，而不是布局后被拉伸的槽宽）：带标签 = 方框 + 间距 + 文字。
	mw, _ := renderer.MeasureNode(rec, off, 300, 0, renderer.TextStyle{}, theme.Light)
	if mw <= checkboxBox+checkboxGap+1 {
		t.Fatalf("自然宽度应包含标签：w=%.1f", mw)
	}

	// 点击 → 回调新值 true。
	off.OnClick()
	if len(got) != 1 || got[0] != true {
		t.Fatalf("点击应回调 true：%+v", got)
	}

	// 勾选态：主题按钮色实心框 + 两段对勾线，没有描边。
	on := Checkbox(runtime.Props{ID: "cb-on", Value: true, Label: "显示裁切标记"})
	rec2 := &checkboxLineRecorder{mockContext: mockContext{w: 300, h: 60}}
	renderer.PaintTree(rec2, []*runtime.VNode{on}, theme.Light)
	if len(rec2.rounds) != 1 {
		t.Fatalf("勾选态应只有一个方框，实际 %d", len(rec2.rounds))
	}
	if diff(rec2.rounds[0].c, renderer.ColorFrom(theme.Light.Button)) > 0.01 {
		t.Fatalf("勾选态方框应为主题按钮色：%+v", rec2.rounds[0].c)
	}
	if len(rec2.lines) != 2 {
		t.Fatalf("对勾应是两段线，实际 %d", len(rec2.lines))
	}

	// 禁用：不吃点击。
	got2 := []bool{}
	dis := Checkbox(runtime.Props{
		ID:       "cb-disabled",
		Value:    false,
		Disabled: true,
		OnChange: func(v bool) { got2 = append(got2, v) },
	})
	dis.OnClick()
	if len(got2) != 0 {
		t.Fatal("禁用态点击不应回调")
	}

	// 无标签：自然宽度就是方框 + 间距。
	bare := Checkbox(runtime.Props{ID: "cb-bare", Value: false})
	mw, _ = renderer.MeasureNode(&mockContext{w: 100, h: 40}, bare, 300, 0, renderer.TextStyle{}, theme.Light)
	if mw > checkboxBox+checkboxGap+1 {
		t.Fatalf("无标签时不应为文字留宽度：w=%.1f", mw)
	}
}
