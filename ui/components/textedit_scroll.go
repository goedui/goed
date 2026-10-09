package components

// TextareaOffsetY 返回编辑框某字节偏移对应的内容纵坐标（相对文本顶）。
func TextareaOffsetY(id string, off int) float32 {
	st := textareaState(id)
	if st == nil || st.layout == nil {
		return 0
	}
	lay := st.layout
	if lay.sparse() || len(lay.points) == 0 {
		return float32(lay.lineForOffset(off)) * lay.lineH
	}
	return float32(lay.pointAt(off).line) * lay.lineH
}

// TextareaLineHeight 返回编辑框当前行高。
func TextareaLineHeight(id string) float32 {
	st := textareaState(id)
	if st == nil || st.layout == nil {
		return 0
	}
	return st.layout.lineH
}
