# 声明式原生 UI

`helloworld/main.go` 是最小示例，`imgen/src/views/generator/view.go` 展示表单、后台任务与图片预览的组合。导入 `ui/app` 会注册 `ui/components` 的内置宿主。

```go
var Counter = runtime.DefineComponent("Counter", func(ctx *runtime.SetupContext) runtime.RenderFunc {
    count := runtime.Ref(0)
    return func() []*runtime.VNode {
        return []*runtime.VNode{
            runtime.H("vstack", runtime.Props{"padding": 24, "gap": 12},
                runtime.Text("次数：", count),
                runtime.H("button", runtime.Props{
                    "label": "增加", "radius": 16,
                    "onClick": func() { count.Set(count.Get() + 1) },
                }),
            ),
        }
    }
})
```

## 内置元素

| 元素 | 常用属性 | 行为 |
| --- | --- | --- |
| `vstack` / `hstack` | `gap`, `padding`, `align`, `justify`, `bg`, `border`, `radius` | 按内容测量，支持弹性尺寸和嵌套卡片 |
| `box` | 同上，另有 `direction` | 有方向时排列，无方向时子元素铺满 |
| `text` | `value`, `color`, `ellipsis` | 自然尺寸测量、超长省略 |
| `input` | `value`, `onInput`, `onChange`, `placeholder`, `multiline`, `password`, `radius` | 单行/多行、选择、剪贴板、键盘编辑 |
| `button` | `label`, `onClick`, `variant`, `selected`, `radius` | 鼠标/键盘激活、聚焦、禁用和悬停清理；`secondary` 使用次要按钮配色 |
| `switch` | `value`, `onChange func(bool)` | 布尔开关，支持鼠标及 Space/Enter |
| `image` | `data []byte`, `name`, `height` | 保留解码与缩放缓存，内容不变时滚动/重新渲染不会丢失预览 |
| `scroll` | `flex`, `scrollToEnd` | 测量内容、裁剪绘制及鼠标命中、滚轮滚动、聚焦自动滚入视野 |
| `layer` | `onKeyDown func(core.Event) bool` | 普通内容与浮层组合，支持页面快捷键 |
| `popover` | `anchor`, `items []string`, `onSelect`, `onClose` | 按 `id` 锚定，最多六项可视，支持方向键、Enter、Esc、点击外部关闭和滚轮 |
| `titlebar` | `title`, `onMinimize`, `onMaximize`, `onClose` | 原生窗口标题栏，默认由应用外壳提供 |

宿主元素共有 `id`、`width`、`height`、`flex`、`visible` 和 `enabled` 属性。尺寸表示请求尺寸，框架将其与最终布局矩形分开保存，避免窗口缩放或条件节点变化后沿用旧尺寸。`ref func(core.Component)` 在挂载时得到宿主，卸载时得到 `nil`；一般表单只需要值和回调绑定。

`scroll` 内通常放一个 `vstack`。固定标签栏和状态栏放在 `scroll` 外。修改 `scrollToEnd` 的整数版本值，可以在新结果出现后滚到底部；普通状态刷新不改变滚动位置。

`layer` 的第一个子节点是普通内容，末尾可放一个条件 `popover`；弹层通过 `anchor` 查找同一层中的元素 `id`。弹层优先处理输入，避免点击透传。

## 状态与生命周期

- 在 `DefineComponent` 的 setup 中创建状态和事件回调，在 render 中读取状态并返回 VNodes。
- 条件节点可以返回 `nil`；动态列表使用稳定的 `.WithKey(...)`，重排时保留输入框、图片和其他宿主实例。
- 删除聚焦节点会清理焦点；Tab 遍历会跳过禁用和隐藏的控件，并让滚动容器中的目标可见。
- `ctx.OnUnmounted` 适合取消页面请求。网络回调不应直接修改响应式状态或组件树。

## 后台任务回到 UI 线程

`ui/app` 给根组件传入 `dispatcher *runtime.Dispatcher` 和原生 `window`。向子页面传递这些 props 后，后台任务可将完成处理投递到 UI 线程：

```go
dispatcher := ctx.Props()["dispatcher"].(*runtime.Dispatcher)
status := runtime.Ref("等待")
go func() {
    result := doWork()
    dispatcher.Post(func() { status.Set(result) })
}()
```

`Post` 可被多个 worker 调用；应用在绘制前 `Drain`。应用退出会关闭 dispatcher，页面自身还应取消任务或检查已卸载状态。`imgen` 的生成流程示范了取消、不可变请求快照和卸载后的回调丢弃。

动态挂载的元素也会通过 `Renderer.OnHostMounted` 获得平台服务，声明式输入框可自动使用 Windows 剪贴板。手动创建 renderer 时可以自行提供该钩子。

## 验证

运行 `go test ./...`、`go test -race ./...` 和 `go vet ./...`。布局、key 重排、输入、滚动命中、浮层和后台完成都有回归覆盖；Windows 图片回归使用原生 GDI 离屏绘制，不需要打开交互窗口。
