//go:build linux

package linux

import "testing"

func TestMapKeySym(t *testing.T) {
	cases := []struct {
		sym  uint64
		key  int
		char rune
	}{
		{xkReturn, vkEnter, 0},
		{xkEscape, vkEscape, 0},
		{xkBackSpace, vkBackspace, 0},
		{xkLeft, vkLeft, 0},
		{xkF11, vkF11, 0},
		{'a', 'A', 'a'},
		{'A', 'A', 'A'},
		{xkSpace, vkSpace, ' '},
		{0x4e2d, 0, 0}, // CJK keysym：没有拉丁映射
	}
	for _, c := range cases {
		key, char := mapKeySym(c.sym)
		if key != c.key || char != c.char {
			t.Fatalf("sym %x: 得到 key=%d char=%q，期望 %d %q", c.sym, key, char, c.key, c.char)
		}
	}
}
