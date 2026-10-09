package components

// ScrollMax 返回滚动容器的可滚上限（内容高度 - 可视高度）。尚未布局时为 0。
func ScrollMax(id string) float32 {
	scrollMu.RLock()
	defer scrollMu.RUnlock()
	st, ok := scrollExtent[scrollKey(id, false)]
	if !ok || st.maxOff < 0 {
		return 0
	}
	return st.maxOff
}
