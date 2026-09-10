// Package composables - 组合式函数
// 代码折叠功能
package composables

// FoldRegion 折叠区域
type FoldRegion struct {
	StartLine int
	EndLine   int
	Folded    bool
	Level     int    // 缩进层级
	Type      string // "indent" | "brace" | "comment"
}

// FoldingManager 折叠管理器
type FoldingManager struct {
	Regions []FoldRegion

	// foldedMask 缓存「已折叠区域覆盖的行号位图」。大文件可能产生数以
	// 万计的折叠区域，IsFolded 若对每行线性扫描全部区域，会随
	// visibleLineMap 的逐行调用退化为 O(行数×区域数)，滚动直接卡死。
	// 位图让 IsFolded 变成 O(1) 查询。Regions 变化时置 foldedMaskDirty。
	foldedMask      []uint64
	foldedMaskDirty bool
}

// NewFoldingManager 创建折叠管理器
func NewFoldingManager() *FoldingManager {
	return &FoldingManager{
		Regions: []FoldRegion{},
	}
}

// DetectFoldRegions 检测可折叠区域（基于缩进）
func (fm *FoldingManager) DetectFoldRegions(lines []string) {
	fm.InvalidateFoldCache()
	fm.Regions = []FoldRegion{}

	if len(lines) == 0 {
		return
	}

	// 基于缩进检测
	stack := []int{} // 存储各层级的起始行
	prevIndent := 0

	for i, line := range lines {
		// 跳过空行
		if len(line) == 0 {
			continue
		}

		// 计算缩进
		indent := countIndent(line)

		// 如果缩进增加，可能是新的折叠区域开始
		if indent > prevIndent {
			// 上一行是折叠区域的开始
			if i > 0 {
				stack = append(stack, i-1)
			}
		} else if indent < prevIndent {
			// 缩进减少，关闭之前的折叠区域
			for len(stack) > 0 && indent <= countIndent(lines[stack[len(stack)-1]]) {
				startLine := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				// 创建折叠区域（至少3行才折叠）
				if i-startLine >= 2 {
					fm.Regions = append(fm.Regions, FoldRegion{
						StartLine: startLine,
						EndLine:   i - 1,
						Folded:    false,
						Level:     countIndent(lines[startLine]) / 4, // 假设每级缩进4空格
						Type:      "indent",
					})
				}
			}
		}

		prevIndent = indent
	}

	// 处理剩余的stack
	for len(stack) > 0 {
		startLine := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if len(lines)-1-startLine >= 2 {
			fm.Regions = append(fm.Regions, FoldRegion{
				StartLine: startLine,
				EndLine:   len(lines) - 1,
				Folded:    false,
				Level:     countIndent(lines[startLine]) / 4,
				Type:      "indent",
			})
		}
	}
}

// DetectFoldRegionsBrace 检测可折叠区域（基于花括号）
func (fm *FoldingManager) DetectFoldRegionsBrace(lines []string) {
	fm.InvalidateFoldCache()
	fm.Regions = []FoldRegion{}

	stack := []int{} // 存储 '{' 所在行

	for i, line := range lines {
		// 查找 '{'
		for _, ch := range line {
			if ch == '{' {
				stack = append(stack, i)
			} else if ch == '}' {
				if len(stack) > 0 {
					startLine := stack[len(stack)-1]
					stack = stack[:len(stack)-1]

					// 创建折叠区域
					if i-startLine >= 2 {
						fm.Regions = append(fm.Regions, FoldRegion{
							StartLine: startLine,
							EndLine:   i,
							Folded:    false,
							Level:     len(stack),
							Type:      "brace",
						})
					}
				}
			}
		}
	}
}

// ToggleFold 切换指定行的折叠状态
func (fm *FoldingManager) ToggleFold(line int) bool {
	for i := range fm.Regions {
		if fm.Regions[i].StartLine == line {
			fm.Regions[i].Folded = !fm.Regions[i].Folded
			fm.InvalidateFoldCache()
			return fm.Regions[i].Folded
		}
	}
	return false
}

// IsFolded 检查指定行是否被折叠。覆盖判断经 foldedMask 位图缓存，
// 避免大文件下每次都线性扫描全部折叠区域（外部可能直接改 Regions[i]
// 的字段，故缓存只能按「脏标记 + 惰性重建」失效）。
func (fm *FoldingManager) IsFolded(line int) bool {
	if fm.foldedMaskDirty || fm.foldedMask == nil {
		fm.rebuildFoldedMask()
	}
	if line < 0 || line >= len(fm.foldedMask)*64 {
		return false
	}
	return fm.foldedMask[line>>6]&(1<<(uint(line)&63)) != 0
}

// InvalidateFoldCache 在 Regions 或其 Folded 字段被外部直接修改后调用，
// 强制 IsFolded 重建位图。
func (fm *FoldingManager) InvalidateFoldCache() {
	fm.foldedMaskDirty = true
}

// rebuildFoldedMask 依据当前 Regions 重建折叠覆盖位图。
func (fm *FoldingManager) rebuildFoldedMask() {
	need := 0
	for i := range fm.Regions {
		if fm.Regions[i].Folded && fm.Regions[i].EndLine+1 > need {
			need = fm.Regions[i].EndLine + 1
		}
	}
	mask := make([]uint64, (need+63)/64)
	for i := range fm.Regions {
		if !fm.Regions[i].Folded {
			continue
		}
		for line := fm.Regions[i].StartLine + 1; line <= fm.Regions[i].EndLine; line++ {
			mask[line>>6] |= 1 << (uint(line) & 63)
		}
	}
	fm.foldedMask = mask
	fm.foldedMaskDirty = false
}

// GetFoldRegionAt 获取指定行的折叠区域
func (fm *FoldingManager) GetFoldRegionAt(line int) *FoldRegion {
	for i := range fm.Regions {
		if fm.Regions[i].StartLine == line {
			return &fm.Regions[i]
		}
	}
	return nil
}

// FoldAll 折叠所有区域
func (fm *FoldingManager) FoldAll() {
	for i := range fm.Regions {
		fm.Regions[i].Folded = true
	}
	fm.InvalidateFoldCache()
}

// UnfoldAll 展开所有区域
func (fm *FoldingManager) UnfoldAll() {
	for i := range fm.Regions {
		fm.Regions[i].Folded = false
	}
	fm.InvalidateFoldCache()
}

// GetVisibleLines 获取可见行号列表（跳过折叠的行）
func (fm *FoldingManager) GetVisibleLines(totalLines int) []int {
	visible := []int{}

	for i := 0; i < totalLines; i++ {
		if !fm.IsFolded(i) {
			visible = append(visible, i)
		}
	}

	return visible
}

// countIndent 计算行首空格数
func countIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4 // 制表符算4个空格
		} else {
			break
		}
	}
	return count
}
