package renderer

var (
	pointerX, pointerY float32
	pointerDown        bool
	pointerSeen        bool
)

// SetPointer 由应用在每帧绘制前写入，供按钮 hover / pressed 使用。单位 DIP。
func SetPointer(x, y float32, down bool) {
	pointerX, pointerY, pointerDown = x, y, down
	pointerSeen = true
}

// PointerPos 返回当前指针位置，单位 DIP。控件的 OnClick 用它换算局部坐标。
func PointerPos() (x, y float32) { return pointerX, pointerY }

func pointerIn(x, y, w, h float32) bool {
	return pointerX >= x && pointerY >= y && pointerX < x+w && pointerY < y+h
}

// Pointer 返回最近一次指针事件的客户区坐标（DIP）。控件可以用它把点击
// 位置换算成插入点。返回的 ok 为 false 表示还没有指针事件。
func Pointer() (x, y float32, ok bool) {
	if pointerX == 0 && pointerY == 0 && !pointerSeen {
		return 0, 0, false
	}
	return pointerX, pointerY, true
}
