package renderer

// StrokePoint 是圆头路径上的一个点。Curve 为真时，从上一点到 (X,Y) 走
// 三次贝塞尔，控制点是 (C1X,C1Y)、(C2X,C2Y)；否则是直线。
type StrokePoint struct {
	X, Y     float32
	C1X, C1Y float32
	C2X, C2Y float32
	Curve    bool
}

// RoundStrokePainter 由能画「圆头线帽 + 圆角连接 + 贝塞尔」的后端实现。
// 没有这个接口时图标退回 DrawLine 折线，画面仍在，只是转角是尖的。
type RoundStrokePainter interface {
	DrawRoundStroke(pts []StrokePoint, width float32, c Color, closed bool) bool
}

// RoundedRectStroker 由能直接描圆角矩形的后端实现。
// 线骑在矩形边上，直边和四角同粗。没有这个接口时退回路径或点阵。
type RoundedRectStroker interface {
	DrawRoundedRect(x, y, w, h, r float32, c Color, width float32)
}

// DrawRoundStroke 画一条圆头路径。后端不支持时返回 false。
func DrawRoundStroke(ctx Context, pts []StrokePoint, width float32, c Color, closed bool) bool {
	p, ok := ctx.(RoundStrokePainter)
	if !ok || p == nil {
		return false
	}
	return p.DrawRoundStroke(pts, width, c, closed)
}
