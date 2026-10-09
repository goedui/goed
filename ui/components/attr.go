package components

import (
	"fmt"

	"github.com/goedui/goed/ui/runtime"
)

func setAttr(n *runtime.VNode, key string, v any) {
	if n == nil {
		return
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	n.Attrs[key] = v
}

func mergeAttrs(n *runtime.VNode, extra runtime.Attrs) {
	if n == nil || extra == nil {
		return
	}
	if n.Attrs == nil {
		n.Attrs = runtime.Attrs{}
	}
	for k, v := range extra {
		n.Attrs[k] = v
	}
}

func stringList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, fmtSprint(x))
		}
		return out
	default:
		return nil
	}
}

func fmtSprint(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func onString(n *runtime.VNode, key string) func(string) {
	if n == nil || n.Attrs == nil {
		return nil
	}
	if fn, ok := n.Attrs[key].(func(string)); ok {
		return fn
	}
	return nil
}

func onBool(n *runtime.VNode, key string) func(bool) {
	if n == nil || n.Attrs == nil {
		return nil
	}
	if fn, ok := n.Attrs[key].(func(bool)); ok {
		return fn
	}
	return nil
}

func onFloat(n *runtime.VNode, key string) func(float32) {
	if n == nil || n.Attrs == nil {
		return nil
	}
	if fn, ok := n.Attrs[key].(func(float32)); ok {
		return fn
	}
	return nil
}

func onVoid(n *runtime.VNode, key string) func() {
	if n == nil || n.Attrs == nil {
		return nil
	}
	if fn, ok := n.Attrs[key].(func()); ok {
		return fn
	}
	return nil
}
