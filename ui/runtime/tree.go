package runtime

// Walk 深度优先遍历节点。fn 返回 true 时停止。
func Walk(nodes []*VNode, fn func(*VNode) bool) {
	if fn == nil {
		return
	}
	var walk func([]*VNode) bool
	walk = func(list []*VNode) bool {
		for _, n := range list {
			if n == nil {
				continue
			}
			if fn(n) {
				return true
			}
			if len(n.Children) > 0 && walk(n.Children) {
				return true
			}
		}
		return false
	}
	walk(nodes)
}

// FindByID 按 Attrs["id"] 查找第一个节点。
func FindByID(nodes []*VNode, id string) *VNode {
	if id == "" {
		return nil
	}
	var found *VNode
	Walk(nodes, func(n *VNode) bool {
		if n.ID() == id {
			found = n
			return true
		}
		return false
	})
	return found
}
