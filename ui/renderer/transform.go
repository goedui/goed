package renderer

// TransformPainter 由支持「绕点旋转后续绘制」的后端实现：Windows 上是
// ID2D1RenderTarget::SetTransform（槽位 30，与已验证的 Clear=47 / PushLayer=40
// 同一张 d2d1.h 序号表），Web 上是 canvas 2D 的 setTransform。
//
// 为什么做成可选接口、而不是给 Context 加方法：与 ImagePainter 同一个理由 ——
// Context 是公开接口，加一个方法就把所有外部实现（以及测试里的假后端）全打断了。
// 旋转目前只有水印一处用，后端按自己的条件实现：软件光栅器要变换每一个图元，
// 代价不成比例，不实现；调用方探测不到就退回水平画法 —— 能看见，只是不转，
// 与圆角图片缺 DrawImageRounded 时退回直角是同一个降级哲学。
type TransformPainter interface {
	// RotateAt 把后续绘制绕 (cx, cy) 旋转 deg 度。坐标系 y 向下，deg 为正是
	// 顺时针（水印的 315 → 文字从左下往右上斜，与 Word 一致）。
	// 连续调用会**替换**上一次的旋转，不是叠加 —— 调用方负责用 ResetTransform
	// 恢复，两者必须成对出现。
	RotateAt(deg, cx, cy float32)
	// ResetTransform 撤销旋转，回到后端的基准变换（无旋转，DPI 缩放照旧）。
	ResetTransform()
}

// RotateAt 让后续绘制绕 (cx, cy) 旋转 deg 度。返回后端是否真支持旋转；
// 不支持时调用方应当照常按水平画出内容 —— 什么都不画比不旋转更糟。
func RotateAt(ctx Context, deg, cx, cy float32) bool {
	if deg == 0 || ctx == nil {
		return false
	}
	if t, ok := ctx.(TransformPainter); ok {
		t.RotateAt(deg, cx, cy)
		return true
	}
	return false
}

// ResetTransform 撤销 RotateAt 的旋转。不支持旋转的上下文是空操作。
func ResetTransform(ctx Context) {
	if t, ok := ctx.(TransformPainter); ok {
		t.ResetTransform()
	}
}
