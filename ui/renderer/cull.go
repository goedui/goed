package renderer

import "github.com/goedui/goed/ui/runtime"

type cullRect struct {
	x0, y0, x1, y1 float32
}

var cullStack []cullRect

func resetCullClip() {
	cullStack = cullStack[:0]
}

// PushCullClip 记录绘制裁剪，供离屏节点跳过 DrawText。
// 与 Context.PushClip 成对使用：GPU 裁像素，这里裁遍历。
func PushCullClip(x, y, w, h float32) {
	r := cullRect{x, y, x + w, y + h}
	if n := len(cullStack); n > 0 {
		p := cullStack[n-1]
		if r.x0 < p.x0 {
			r.x0 = p.x0
		}
		if r.y0 < p.y0 {
			r.y0 = p.y0
		}
		if r.x1 > p.x1 {
			r.x1 = p.x1
		}
		if r.y1 > p.y1 {
			r.y1 = p.y1
		}
	}
	cullStack = append(cullStack, r)
}

// PopCullClip 弹出绘制裁剪。
func PopCullClip() {
	if n := len(cullStack); n > 0 {
		cullStack = cullStack[:n-1]
	}
}

func nodeInCullClip(n *runtime.VNode) bool {
	if n == nil {
		return false
	}
	if len(cullStack) == 0 {
		return true
	}
	c := cullStack[len(cullStack)-1]
	return n.X < c.x1 && n.Y < c.y1 && n.X+n.W > c.x0 && n.Y+n.H > c.y0
}
