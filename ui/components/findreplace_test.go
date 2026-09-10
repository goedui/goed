package components

import (
	"fmt"
	"testing"
)

func newTestFindReplace(text string) (*FindReplace, *Editor) {
	e := NewEditor()
	e.SetText(text)
	f := NewFindReplace(
		func() string { return e.GetText() },
		func(line, col, length int, rep string) { e.ReplaceRange(line, col, length, rep) },
		func(line, col, length int) { e.GotoPosition(line, col, length) },
	)
	return f, e
}

func TestFindReplaceSearchAndNavigate(t *testing.T) {
	f, e := newTestFindReplace("foo bar\nbaz foo\nfoo")
	f.Show("")

	// 输入查询字符
	for _, ch := range "foo" {
		f.HandleChar(ch)
	}
	if len(f.Matches) != 3 {
		t.Fatalf("matches = %d, want 3", len(f.Matches))
	}
	if f.Current != 0 {
		t.Fatalf("current = %d, want 0", f.Current)
	}
	// 首个匹配应把光标定位到第一行
	if e.CursorLine != 0 {
		t.Fatalf("cursor line = %d, want 0", e.CursorLine)
	}

	f.Next()
	if f.Current != 1 {
		t.Fatalf("after Next current = %d, want 1", f.Current)
	}
	if e.CursorLine != 1 {
		t.Fatalf("cursor line = %d, want 1", e.CursorLine)
	}
	f.Next()
	if f.Current != 2 {
		t.Fatalf("after Next current = %d, want 2", f.Current)
	}
	f.Next() // 循环回 0
	if f.Current != 0 {
		t.Fatalf("after wrap current = %d, want 0", f.Current)
	}
	f.Prev() // 回绕到最后一个
	if f.Current != 2 {
		t.Fatalf("after Prev wrap current = %d, want 2", f.Current)
	}
	if got := f.StatusText(); got != "3/3" {
		t.Fatalf("status = %q, want 3/3", got)
	}
}

func TestFindReplaceCaseSensitivity(t *testing.T) {
	f, _ := newTestFindReplace("Foo foo FOO")
	f.Show("")
	f.SetPlacement(0, 0, 800, 500) // 初始化按钮几何
	for _, ch := range "foo" {
		f.HandleChar(ch)
	}
	if len(f.Matches) != 3 {
		t.Fatalf("case-insensitive matches = %d, want 3", len(f.Matches))
	}
	// 点击 Aa 切换区分大小写
	f.HandleMouseDown(f.btnCase.X+1, f.btnCase.Y+1)
	if !f.MatchCase {
		t.Fatal("MatchCase should be true after toggle")
	}
	if len(f.Matches) != 1 {
		t.Fatalf("case-sensitive matches = %d, want 1", len(f.Matches))
	}
}

func TestFindReplaceOne(t *testing.T) {
	f, e := newTestFindReplace("cat cat cat")
	f.Show("")
	for _, ch := range "cat" {
		f.HandleChar(ch)
	}
	f.HandleChar(' ') // 追加空格使查询为 "cat "？不，直接改查询验证替换
	f.Query = "cat"
	f.Changed()
	if len(f.Matches) != 3 {
		t.Fatalf("matches = %d, want 3", len(f.Matches))
	}
	f.Replace = "dog"
	f.ReplaceCurrent()
	if got := e.GetText(); got != "dog cat cat" {
		t.Fatalf("text after replace one = %q", got)
	}
	if len(f.Matches) != 2 {
		t.Fatalf("matches after replace = %d, want 2", len(f.Matches))
	}
}

func TestFindReplaceAll(t *testing.T) {
	f, e := newTestFindReplace("cat cat cat")
	f.Show("")
	f.Query = "cat"
	f.Changed()
	f.Replace = "dog"
	if n := f.ReplaceAll(); n != 3 {
		t.Fatalf("ReplaceAll = %d, want 3", n)
	}
	if got := e.GetText(); got != "dog dog dog" {
		t.Fatalf("text after replace all = %q", got)
	}
	if !e.Modified {
		t.Fatal("editor should be marked modified")
	}
}

func TestFindReplaceEscapeCloses(t *testing.T) {
	f, _ := newTestFindReplace("hello world")
	f.Show("hello")
	if !f.IsOpen() {
		t.Fatal("should be open after Show")
	}
	f.HandleKeyDown(0x1B)
	if f.IsOpen() {
		t.Fatal("Esc should close the widget")
	}
}

func TestFindReplaceToggleReplaceRow(t *testing.T) {
	f, e := newTestFindReplace("aa bb aa")
	f.Show("aa")
	if f.ShowReplace {
		t.Fatal("replace row should start hidden")
	}
	// 面板高度应只含查找行
	if h := f.panelHeight(); h != 40 {
		t.Fatalf("panel height without replace = %d, want 40", h)
	}
	f.ToggleReplace()
	if !f.ShowReplace {
		t.Fatal("replace row should be visible after toggle")
	}
	if h := f.panelHeight(); h != 76 {
		t.Fatalf("panel height with replace = %d, want 76", h)
	}
	// 全部替换验证替换行路径
	f.Replace = "cc"
	if n := f.ReplaceAll(); n != 2 {
		t.Fatalf("ReplaceAll = %d, want 2", n)
	}
	if got := e.GetText(); got != "cc bb cc" {
		t.Fatalf("text = %q", got)
	}
}

func TestFindReplacePlacementAnchorsTopRight(t *testing.T) {
	f, _ := newTestFindReplace("x")
	f.Show("x")
	// 编辑器位于 (300, 66)，宽 800 高 500
	f.SetPlacement(300, 66, 800, 500)
	p := f.Panel()
	if p.Y != 66+8 {
		t.Fatalf("panel Y = %d, want %d (below tab bar)", p.Y, 74)
	}
	// 右缘应距编辑器右缘 12px + 110px minimap 间距（与 SetPlacement 一致）
	wantRight := int32(300 + 800 - 12 - 110)
	if p.X+p.W != wantRight {
		t.Fatalf("panel right = %d, want %d", p.X+p.W, wantRight)
	}
	if !f.ContainsHit(p.X+10, p.Y+10) {
		t.Fatal("ContainsHit should be true inside panel")
	}
	if f.ContainsHit(p.X-50, p.Y+10) {
		t.Fatal("ContainsHit should be false outside panel")
	}
}

func TestFindReplaceQueryClearsOnEmpty(t *testing.T) {
	f, _ := newTestFindReplace("abc")
	f.Show("abc")
	f.Query = "abc"
	f.Changed()
	if len(f.Matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(f.Matches))
	}
	// 逐个退格清空
	for i := 0; i < 3; i++ {
		f.HandleKeyDown(0x08) // Backspace
	}
	if f.Query != "" {
		t.Fatalf("query = %q, want empty", f.Query)
	}
	if len(f.Matches) != 0 {
		t.Fatalf("matches after clear = %d, want 0", len(f.Matches))
	}
}

func TestFindReplaceSelectsMatchText(t *testing.T) {
	// 含中文行：验证 rune 列坐标系下光标列与选中文本都正确。
	f, e := newTestFindReplace("中文注释 foo tail\nfoo")
	f.Show("")
	for _, ch := range "foo" {
		f.HandleChar(ch)
	}
	if len(f.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(f.Matches))
	}
	// 首个匹配：行 0 列 5（"中文注释 " = 5 个 rune）
	if e.CursorLine != 0 || e.CursorCol != 5 {
		t.Fatalf("cursor = %d:%d, want 0:5", e.CursorLine, e.CursorCol)
	}
	if got := e.GetSelectedText(); got != "foo" {
		t.Fatalf("selected = %q, want %q", got, "foo")
	}
	// 下一个匹配在行 1
	f.Next()
	if e.CursorLine != 1 || e.CursorCol != 0 {
		t.Fatalf("cursor = %d:%d, want 1:0", e.CursorLine, e.CursorCol)
	}
	if got := e.GetSelectedText(); got != "foo" {
		t.Fatalf("selected = %q, want %q", got, "foo")
	}
}

func TestGotoPositionCentersLine(t *testing.T) {
	e := NewEditor()
	// 40 行文档，视口 10 行。
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	e.Lines = lines
	e.VisibleLines = 10
	e.visibleMapDirty = true

	e.GotoPosition(30, 0, 0)
	// 目标行应位于视口中部：起始可见索引约 30-5=25。
	if e.ScrollLine < 24 || e.ScrollLine > 27 {
		t.Fatalf("ScrollLine = %d, want ~25 (line 30 centered)", e.ScrollLine)
	}
}
