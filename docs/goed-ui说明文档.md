# goed UI 使用说明

应用层 API 入口：[`ui`](../ui/ui.go)

模块：[`github.com/goedui/goed`](../go.mod)

最低 Go 版本：Go 1.23

goed UI 是用 Go 编写的声明式 UI 引擎。应用通过 `VNode` 描述界面，引擎负责布局、绘制、输入与命中。应用代码通常只需导入 `github.com/goedui/goed/ui`；导入 `ui` 会自动注册内置控件。

## 安装

在应用自己的 Go module 目录执行：

```sh
go get github.com/goedui/goed/ui@latest
```

在源码中导入：

```go
import "github.com/goedui/goed/ui"
```

goed UI 是库，不是可执行命令，因此不要用 `go install` 安装。已有 Go module 的项目直接执行 `go get`；新项目先运行 `go mod init example.com/myapp`。本仓库内的示例可从仓库根目录运行：

```sh
go run ./helloworld
```

Windows 10/11 是主要桌面平台；Linux 使用 X11 后端，Web 使用 `js/wasm` 后端。macOS 后端尚未实现。

## 快速使用

一个界面由根组件、控件节点和 `CreateApp` 创建的应用组成。组件的 `setup` 只执行一次；返回的渲染函数在界面需要更新时重新生成节点。

```go
var App = ui.Component(func(_ *ui.SetupContext) func() *ui.VNode {
	return func() *ui.VNode {
		return ui.VStack(
			ui.Props{Style: ui.Style{Flex: 1, Padding: 20, Gap: 12}},
			ui.Text(ui.Props{Style: ui.Style{Height: 24}}, "欢迎使用 goed UI"),
			ui.Button(ui.Props{OnClick: func() {}}, "点击"),
		)
	}
})

func main() {
	if err := ui.CreateApp(App).
		Title("示例应用").
		Theme("light").
		Size(800, 600).
		Run(); err != nil {
		panic(err)
	}
}
```

普通事件驱动界面不需要启用连续帧。根布局通常设置 `Style.Flex: 1` 以填满客户区；尺寸和坐标单位为 DIP（96 DIP = 1 英寸）。文本节点要给足 `Style.Height`，否则 DirectWrite 可能裁掉字形下半部分。

应用配置常用方法：

| 方法 | 说明 |
|---|---|
| `Title(string)` | 窗口标题 |
| `Size(width, height)` | 客户区逻辑尺寸，默认 800×600 DIP |
| `Theme("light" / "dark")` | 选择内置主题；未知名称回退默认主题 |
| `UseTheme(ui.Theme)` | 直接使用主题值 |
| `Continuous(bool)` | 每帧重建根组件；实时动画或游戏场景使用 |
| `Icon([]byte)` | 设置标准 ICO 窗口图标 |
| `DropFiles(func([]string))` | 接收拖入窗口的文件路径 |
| `HiddenTitlebar()` | 隐藏框架标题栏，由应用绘制顶部区域 |
| `Run()` | 创建窗口并进入消息循环，必须在主 goroutine 调用 |

## HelloWorld

下面的示例包含响应式计数器。`Ref` 在组件 `setup` 中创建；调用 `Set` 会请求重绘。

```go
package main

import "github.com/goedui/goed/ui"

var HelloWorld = ui.Component(func(_ *ui.SetupContext) func() *ui.VNode {
	count := ui.Ref(0)
	return func() *ui.VNode {
		return ui.VStack(
			ui.Props{Style: ui.Style{Flex: 1, Padding: 24, Gap: 12}},
			ui.Text(ui.Props{Style: ui.Style{Height: 32, FontSize: 24, Weight: 600}}, "Hello, goed UI"),
			ui.Text(ui.Props{Style: ui.Style{Height: 24}}, "点击次数：", count),
			ui.Button(ui.Props{
				OnClick: func() { count.Set(count.Get() + 1) },
			}, "点击我"),
		)
	}
})

func main() {
	if err := ui.CreateApp(HelloWorld).
		Title("HelloWorld").
		Theme("light").
		Size(800, 600).
		Run(); err != nil {
		panic(err)
	}
}
```

## 组件基础

### 节点与子内容

`ui.H` 是通用节点工厂，类型可以是引擎标签名或自定义 `*ui.ComponentDef`。`ui.Text` 创建文本节点；`ui.VStack`、`ui.HStack` 等创建内置布局节点。

```go
ui.H("layer", ui.Props{Style: ui.Style{Flex: 1}}, content, overlay)
ui.Text(ui.Props{Style: ui.Style{Color: ui.RGB(100, 100, 100), Height: 22}}, "说明")
```

通用节点工厂的第一个参数可为 `ui.Props`，之后传 `*ui.VNode`、`[]*ui.VNode` 或文本子项。不要将 `[]any` 与 `Props` 一起展开。`Button` 是例外，签名为 `ui.Button(props?, label)`，不是通用子节点工厂。

### Props

控件均可使用 `ui.Props` 的通用字段；每个组件只读取与自身相关的字段。回调字段 `OnChange`、`OnOpen`、`OnClose`、`OnSelect` 等是 `any`，实际函数签名见对应组件条目。

| 类别 | 字段 |
|---|---|
| 布局与身份 | `Style`、`Key`、`ID` |
| 文案与标识 | `Label`、`Placeholder`、`Variant`、`Title`、`Hint`、`Icon`、`Name`、`Badge`、`Message`、`Path`、`Open`、`PopupId`、`Anchor`、`AlignX`、`Shortcut`、`TextAlign`、`Footer` |
| 状态 | `Disabled`、`Selected`、`Password`、`Compact`、`Multiline`、`Readonly`、`Nowrap`、`Borderless`、`Dashed`、`Vertical`、`CloseOnEnter`、`Modal`、`Focus`、`WideBar`、`NoWheelX`、`Ellipsis`、`Accent` |
| 事件 | `OnClick`、`OnMouseEnter`、`OnMouseLeave`、`OnChange`、`OnInput`、`OnSubmit`、`OnBlur`、`OnKeyDown`、`OnClose`、`OnOpen`、`OnSelect`、`OnEmpty`、`OnFooter`、`OnEnter`、`OnWheel`、`OnScroll`、`OnPaint`、`OnPointerDown`、`OnContextMenu` |
| 数据 | `Value`、`Items`、`Menus`、`Data`、`State`、`Dispatcher`、`Term`、`ContentWidth` |
| 颜色 | `SelectedBackground`、`SelectedColor`、`ItemBackground`、`ItemColor`、`ItemBorder`、`CloseColor`、`OpenBackground`、`OpenColor`、`OnBG`、`OffBG`、`Knob` |
| 视觉效果与坐标 | `Gradient`、`Shadow`、`AtX`、`AtY` |

`OnClick` 表示节点矩形可命中并执行回调；它不是组件的 `OnChange`，也不会自动向父组件发送事件。值类控件是受控的：回调收到新值后，应用应更新自己的状态并在下一次渲染时传回 `Value`。

`OnKeyDown` 接收 `ui.KeyEvent` 并返回 `bool`；返回 `true` 表示按键已处理。事件包含 `Key`、`Char`、`Ctrl`、`Shift`、`Alt`、`Down`。`OnContextMenu` 的签名是 `func(x, y float32)`，坐标为客户区 DIP。

### Style

`Style` 放在 `Props.Style` 中。布局尺寸、边距、字体等数值均为 DIP。颜色可用 `ui.RGB(r, g, b)` 或 `ui.RGBA(r, g, b, a)` 创建；颜色的 `A == 0` 表示未设置，走主题默认。

| 字段 | 说明 |
|---|---|
| `Flex` | 按剩余主轴空间分配尺寸；根节点设为 `1` 可填满窗口 |
| `Width`、`Height`、`MinWidth`、`MinHeight`、`MaxWidth`、`MaxHeight` | 尺寸与约束 |
| `Padding`、`Margin`、`Gap` | 内边距、外边距、子项间距 |
| `MarginAuto`、`MarginLeftAuto` | 水平自动居中；HStack 中将本项及后续项推至右侧 |
| `FontSize`、`FontFamily`、`Weight` | 字体字号、字体族、字重 |
| `Align`、`Justify` | 交叉轴与主轴对齐；可用 `ui.Start`、`Center`、`End`、`Stretch`、`SpaceBetween` |
| `Overflow`、`OverflowX`、`OverflowY` | `ui.OverflowVisible`、`Hidden`、`Auto`、`Scroll`；轴向字段优先 |
| `Position`、`Top`、`Right`、`Bottom`、`Left`、`Inset` | 相对、绝对或 sticky 定位及边缘偏移 |
| `Color`、`Background`、`Border`、`Radius` | 前景、背景、描边与圆角 |

### Text

创建文本节点。可选的第一个参数是 `ui.Props`，后续任意值会按顺序转成文本并拼接；字体、颜色和对齐由 `Style` 控制。显式设置足够的 `Style.Height`，避免绘制后端裁掉字形。

```go
ui.Text(ui.Props{Style: ui.Style{
	Height: 24, FontSize: 14, Weight: 500, Color: ui.RGB(70, 78, 90),
}}, "你好，", "goed UI")
```

## 布局与容器

### VStack

垂直排列子项。`Style.Gap` 控制间隔，`Align` 控制水平对齐，`Justify` 控制纵向剩余空间；子项 `Flex` 分配剩余高度。

```go
ui.VStack(ui.Props{Style: ui.Style{Flex: 1, Gap: 8, Padding: 12}},
			ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "标题"),
	ui.Button("确定"),
)
```

### HStack

水平排列子项。子项 `Flex` 按权重分配剩余宽度，`Style.Gap` 设置间距；子项 `MarginLeftAuto: true` 可将操作区推到最右侧。

```go
ui.HStack(ui.Props{Style: ui.Style{Width: 600, Gap: 8, Align: ui.Center}},
	ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "名称"),
	ui.Input(ui.Props{Style: ui.Style{Flex: 1}, Placeholder: "输入名称"}),
	ui.Button("保存"),
)
```

### Wrap

横向流式排列，空间不足时换行。子项保持自然尺寸，子项 `Flex` 不参与分配。

```go
ui.Wrap(ui.Props{Style: ui.Style{Width: 400, Gap: 8}},
	ui.Tag(ui.Props{Label: "Go"}),
	ui.Tag(ui.Props{Label: "Desktop"}),
	ui.Tag(ui.Props{Label: "UI"}),
)
```

### Box

矩形容器，按自身 `Style` 绘制背景、描边和圆角，子项排在内边距区域。

```go
ui.Box(ui.Props{Style: ui.Style{
	Flex: 1, Padding: 16, Background: ui.RGB(245, 247, 250), Radius: 6,
}}, ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "卡片内容"))
```

Box 将子项布局到相同的内边距区域；需要纵向或横向排列多个子项时，在 Box 内再放 VStack 或 HStack。

### Layer

叠放容器，通过 `ui.H("layer", ...)` 创建。第一个普通子项铺满容器，其余子项按自然尺寸叠放在左上角；Popover、Dialog、ContextMenu、ItemMenu 作为浮层参与布局。

```go
ui.H("layer", ui.Props{Style: ui.Style{Flex: 1}},
	ui.Image(ui.Props{Style: ui.Style{Flex: 1}, Data: imageBytes}),
	ui.Tag(ui.Props{Label: "预览"}),
)
```

### Table

按列对齐的网格。直接子项是行，行的子项是单元格。列宽按各行自然宽度取最大值，剩余宽度按单元格 `Flex` 分配；所有单元格都没有 `Flex` 时均分。空单元格仍占列。

```go
ui.Table(ui.Props{Style: ui.Style{Flex: 1, Gap: 4}},
	ui.HStack(ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "名称"), ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "状态")),
	ui.HStack(ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "worker"), ui.Text(ui.Props{Style: ui.Style{Height: 22}}, "运行中")),
)
```

### Scroll

可滚动容器，纵向默认 `OverflowAuto`。用稳定的 `ID`（或 `Key`）保存滚动位置。`WideBar` 加宽滚动条并预留滑槽；`NoWheelX` 禁止滚轮驱动横向滚动。

```go
ui.Scroll(ui.Props{ID: "results", Style: ui.Style{Flex: 1}, WideBar: true}, resultList)

offset := ui.ScrollOffset("results")
ui.SetScrollOffset("results", offset)
maxOffset := ui.ScrollMax("results")
ui.ScrollBy("results", 120)
ui.ScrollAxis("results", 60, true) // true 表示横向
```

普通容器也可设置 `Style.OverflowY: ui.OverflowAuto`。滚动状态按 ID 保存。

### Divider

分隔线，默认水平；传 `true` 或设置 `Vertical: true` 绘制竖线。`Style.Width`、`Height`、`Background` 可调整尺寸和颜色。

```go
ui.Divider()
ui.Divider(true)
```

## 输入与选择

### Button

可点击按钮。签名是 `ui.Button(props?, label)`；`Variant` 可设为 `"secondary"` 或 `"ghost"`，`Selected` 使用主题强调色，`Icon` 设置左侧图标。支持 `Disabled`、`OnClick`、`Style`。默认高度至少 32 DIP，默认水平和垂直内边距分别为 16 和 8 DIP。

```go
ui.Button(ui.Props{
	Variant: "secondary",
	Icon:    "save",
	OnClick: save,
}, "保存")
```

### Input

单行文本输入，支持密码、只读和可选多行模式。`Value` 为当前字符串；输入时通过 `OnInput func(string)` 通知应用。`OnSubmit func(string)` 在单行框按 Enter 时触发，`OnBlur` 在失焦时触发。`ID` 应唯一且保持稳定。未指定尺寸时宽 160、高 32 DIP。

```go
account := ui.Ref("") // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Input(ui.Props{
		ID: "account", Value: account.Get(), Placeholder: "账号",
		OnInput: account.Set,
		OnSubmit: func(v string) { login(v) },
	})
}
```

可选参数：`Password`、`Readonly`、`Multiline`、`Borderless`、`Disabled` 和 `Style`。编辑状态按 ID 保存。

### Textarea

多行文本编辑器，和 Input 共用光标、选区、IME、撤销重做等编辑核心。`OnInput` 接收新文本；设有 `OnSubmit` 时 Enter 提交，Shift+Enter 换行，Ctrl+Enter 也可提交。未设置 `OnSubmit` 时 Enter 插入换行。`OnScroll func(float32)` 接收纵向滚动偏移。

```go
document := ui.Ref("") // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Textarea(ui.Props{
		ID: "editor", Value: document.Get(),
		Style: ui.Style{Flex: 1, Padding: 8, FontFamily: "Consolas"},
		Placeholder: "开始输入…", Borderless: true,
		OnInput: document.Set,
	})
}
```

`Readonly`、`Placeholder`、`Borderless`、`Disabled` 和 `Style` 同样适用。切换不同文档但复用同一 ID 时，调用 `ui.ResetInputState(id)` 清除旧光标和撤销状态。编辑辅助 API：`CaretLineCol`、`MoveCaret`、`SelectedRange`、`SelectRange`、`SelectAll`、`EditOp`、`EditOpID`、`FocusedCaretRect`。

### Select

下拉选择，`Items` 为 `[]string`，`Value` 为当前选项。`OnChange func(string)` 返回所选值；`OnEmpty func()` 在空列表时点击；`OnOpen func(bool)` 通知开合状态。`Compact` 只绘制箭头，适用于输入框旁的紧凑触发器。

```go
quality := ui.Ref("标准") // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Select(ui.Props{
		ID: "quality", Value: quality.Get(),
		Items: []string{"标准", "高质量"}, Placeholder: "选择质量",
		OnChange: quality.Set,
		OnOpen: func(open bool) { logOpenState(open) },
	})
}
```

Select 的开合状态按 ID 保存，因此每个 Select 都应设置非空且唯一的 `ID`。需要程序控制时调用 `ui.SetSelectOpen(id, open)`。`Compact` 模式默认宽 44、高 32 DIP。

### Switch

布尔开关，使用 `Value bool` 和 `OnChange func(bool)`。`OnBG`、`OffBG`、`Knob` 可接受 `ui.Color` 或 `"#RRGGBB"` 字符串覆盖颜色；`Style.Width`、`Height` 覆盖默认尺寸，默认 40×22 DIP。

```go
enabled := ui.Ref(false) // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Switch(ui.Props{
		ID: "enabled", Value: enabled.Get(),
		OnChange: enabled.Set,
		OnBG: "#35A66F",
	})
}
```

### Checkbox

复选框，`Value bool` 表示勾选状态，`Label` 是右侧文案，`OnChange func(bool)` 返回点击后的值。`Disabled` 禁止交互；不设置 Label 时只画方框。方框为 14 DIP，默认整行最小高度为 20 DIP。

```go
includeHidden := ui.Ref(false) // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Checkbox(ui.Props{
		Value: includeHidden.Get(), Label: "包含隐藏文件",
		OnChange: includeHidden.Set,
	})
}
```

### Tabs

水平标签栏。`Items` 为 `[]ui.Tab`，`Value` 是选中的 tab ID；`OnChange func(string)` 通知切换，`OnClose func(string)` 通知关闭。`Variant: "pill"` 使用胶囊样式。颜色字段包括 `SelectedBackground`、`SelectedColor`、`ItemBackground`、`ItemColor`、`ItemBorder`、`CloseColor`。默认矩形标签高度为 28 DIP，pill 高度为 34 DIP。

```go
activeTab := ui.Ref("home") // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.Tabs(ui.Props{
		Items: []ui.Tab{
			{ID: "home", Label: "主页"},
			{ID: "log", Label: "日志", Closable: true, Icon: "edit"},
		},
		Value: activeTab.Get(), Variant: "pill",
		OnChange: activeTab.Set,
		OnClose: func(id string) { closeTab(id) },
	})
}
```

`ui.Tab` 字段：`ID`、`Label`、`Icon`、`Closable`、`Disabled`、`CloseID`。ID 为空时使用 Label。

## 展示组件

### Image

显示 PNG 或 JPEG 字节，按原图比例等比缩放并居中。`Data []byte` 是图片内容；`Badge` 在右上角显示角标；`Style.Width`、`Height`、`Radius` 控制尺寸和圆角。未指定尺寸时默认 56×56 DIP。

```go
ui.Image(ui.Props{
	Data: pngBytes, Badge: "3",
	Style: ui.Style{Width: 180, Height: 120, Radius: 6},
})
```

### QRCode

把文本编码并绘制成二维码，节点保持正方形；同时指定宽高时取较短边。`Style.Color` 设置深色模块，`Style.Background` 设置底色；`Badge` 设置中央短标签。纠错级别默认 M，静区默认 4 个模块。纠错等级常量为 `ui.QRLevelL`、`ui.QRLevelM`、`ui.QRLevelQ`、`ui.QRLevelH`。

```go
qr := ui.QRCode(ui.Props{
	Style: ui.Style{Width: 220, Height: 220}, Badge: "goed",
}, "https://example.com")
qr.Attrs["level"] = ui.QRLevelH // 也可用 "H"；可选 L、M、Q、H
qr.Attrs["quiet"] = 4
```

显式版本：`ui.QRCodeWith(text, ui.QRLevelH, props)`。`ui.ParseQRLevel("h")` 解析级别，未知值回退 M；`ui.QRCodeInfo(text, level)` 返回版本号和模块边长；`ui.QRCodePNG(text, level, scale, quiet)` 导出 PNG，其中 `scale` 是每模块像素（小于等于 0 时默认 6），`quiet` 是静区模块数（负数按 0 处理，通常建议至少 4）。内容超出编码上限时返回错误；绘制节点不会 panic。

### Empty

空状态面板，包含图标、标题、说明和可选操作子节点。参数：`Title`、`Hint`、`Icon`、`Dashed`、`Accent` 和 `Style`。内置图标类型有 `image`、`history`、`spinner`；其它名字使用已注册的 Glyph，找不到时回退到 image 图标。

```go
ui.Empty(ui.Props{
	Title: "暂无结果", Hint: "调整条件后重试",
	Icon: "history", Dashed: true,
}, ui.Button("刷新"))
```

### Tag

显示短状态文案的胶囊标签。使用 `Label` 设置文案，`Style.FontSize`、`Width`、`Height`、`Radius` 控制外观；默认高度约 20 DIP。可选内部属性 `fg`、`bg` 设置文字与底色，类型为 `ui.Color`。

```go
tag := ui.Tag(ui.Props{Label: "运行中"})
tag.Attrs["fg"] = ui.RGB(35, 110, 70)
tag.Attrs["bg"] = ui.RGB(225, 244, 232)
```

### Glyph

绘制单个矢量图标。`Icon` 为图标名，`Style.Width`、`Height` 设置大小（默认 16 DIP），`Style.Color` 设置前景色。内置名称包括 `edit`、`link`、`shield`、`box`、`user`、`save`、`trash`、`eye`、`refresh`、`bolt`、`sparkle`、`upload`、`image`、`chevron`、`check`、`ellipsis`、`cloud`。

```go
ui.Glyph(ui.Props{Icon: "save", Style: ui.Style{Width: 18, Height: 18, Color: ui.RGB(40, 100, 180)}})
```

应用可用 `ui.RegisterGlyph(name, paint)` 注册自定义图标，`ui.HasGlyph(name)` 查询名称。绘制函数类型来自 `ui/components` 的 `GlyphPaint`；Canvas 自绘中也可调用 `ui.PaintGlyphNamed`。

### Canvas

自定义绘制区域。通过 `OnPaint` 提供 `func(renderer.Context, x, y, width, height float32)`；使用绘图上下文需要额外导入 `github.com/goedui/goed/ui/renderer`。未指定尺寸时优先使用父级空间；无法从父级取得尺寸时回退到 400×300 DIP。

```go
import "github.com/goedui/goed/ui/renderer"

ui.Canvas(ui.Props{
	Style: ui.Style{Width: 240, Height: 120},
	OnPaint: func(ctx renderer.Context, x, y, w, h float32) {
		ctx.FillRect(x, y, w, h, renderer.RGB(240, 245, 250))
		ctx.DrawLine(x, y, x+w, y+h, renderer.RGB(40, 100, 180), 2)
	},
})
```

## 菜单与浮层

### MenuBar

顶层菜单栏，`Menus` 为 `[]ui.Menu`。`OnOpen func(string)` 接收展开菜单 ID，关闭时收到空字符串；`OnSelect func(string)` 接收选中项的显示文本。`PopupId` 可指定下拉节点 ID。

```go
ui.MenuBar(ui.Props{
	ID: "main-menu",
	Menus: []ui.Menu{{
		ID: "file", Label: "文件",
		Items: []ui.MenuItem{
			{Label: "新建", Shortcut: "Ctrl+N", Action: newFile},
			{Sep: true},
			{Label: "退出", Action: ui.CloseWindow},
		},
	}},
	OnOpen: func(id string) { openMenu = id },
})
```

`ui.Menu` 字段：`ID`、`Label`、`Items`。`ui.MenuItem` 字段：`Label`、`Shortcut`、`Sep`、`Checked`、`Heading`、`Action`。

### Popover

菜单或选项浮层。`Anchor` 指向锚点节点 ID；`Items` 为 `[]string`，`"-"` 或 `"---"` 表示分隔线；`AlignX` 可设 `"start"`、`"center"`、`"end"`。`OnSelect func(string)` 返回所选文本，`OnClose func()` 通知关闭。组件会按锚点定位、在边缘处调整位置，并支持 Esc 关闭。

```go
ui.Popover(ui.Props{
	ID: "actions-popup", Anchor: "actions-button",
	Items: []string{"复制", "---", "删除"}, AlignX: "end",
	OnSelect: func(v string) { runAction(v) },
	OnClose: func() { popupOpen = false },
})
```

### Dialog

居中对话框，含遮罩和内容卡片。可用 `Title`、`Message`、`ContentWidth`、子节点、`OnClose func()`、`OnEnter func()` 设置内容和动作。默认宽 400 DIP，最小宽 240 DIP。`Modal: true` 禁止点击遮罩关闭；`CloseOnEnter: true` 让 Enter 在没有 `OnEnter` 时关闭。Esc 调用 `OnClose`。

```go
ui.Dialog(ui.Props{
	Title: "确认删除", Message: "此操作无法撤销。",
	Modal: true, ContentWidth: 420,
	OnClose: func() { dialogOpen = false },
	OnEnter: func() { deleteItem() },
}, ui.HStack(ui.Button("取消"), ui.Button("删除")))
```

### ContextMenu

右键菜单，面板按指针位置弹出，点击外部或按 Esc 关闭。将 `OnContextMenu` 绑定到触发节点；回调坐标是客户区 DIP。`Items` 为 `[]ui.MenuItem`，`AtX`、`AtY` 指定菜单位置；每项可通过 `Action` 执行业务动作。可选 `OnClose`、`OnSelect`。

```go
menu := ui.Ref[*ui.VNode](nil) // 在 Component 的 setup 中创建
return func() *ui.VNode {
	return ui.H("layer", ui.Props{Style: ui.Style{Flex: 1}},
		ui.Box(ui.Props{OnContextMenu: func(x, y float32) {
			menu.Set(ui.ContextMenu(ui.Props{
				AtX: x, AtY: y,
				Items: []ui.MenuItem{
					{Label: "复制", Action: copySelection},
					{Label: "删除", Action: deleteSelection},
				},
				OnClose: func() { menu.Set(nil) },
			}))
		}}, content),
		menu.Get(),
	)
}
```

`ContextMenu` 为空 Items 时返回 nil。`OnSelect` 收到的是显示标签文本；需要业务分支时优先为 `MenuItem` 设置 `Action`。

### ItemMenu

富内容选择面板，选项由图标、名称和说明组成；贴在 `Anchor` 指定的节点旁，空间不足时可翻到锚点上方。`Items` 为 `[]ui.MenuRow`；`OnSelect func(string)` 返回选项 ID。支持 `Title`、`Hint`、`Footer`、`OnClose`、`OnFooter`、`SelectedBackground` 和 `SelectedColor`。

```go
ui.ItemMenu(ui.Props{
	Anchor: "mode-button", Title: "选择模式", Hint: "↑↓ 切换",
	Items: []ui.MenuRow{
		{ID: "fast", Icon: "bolt", Label: "快速", Desc: "更快完成任务", Active: true},
		{ID: "quality", Icon: "sparkle", Label: "高质量", Desc: "优先输出质量"},
	},
	Footer: "模式设置", OnFooter: openModeSettings,
	OnSelect: func(id string) { setMode(id) },
})
```

`ui.MenuRow` 字段：`ID`、`Icon`、`Label`、`Desc`、`Active`。空 Items 时返回 nil。点击面板外部或按 Esc 会调用 `OnClose`。

### Tooltip

锚点上方的单行提示，`Anchor` 为目标节点 ID，`Label` 为提示文字。Tooltip 由应用决定何时加入节点树；它本身只负责定位和绘制。

```go
ui.Tooltip(ui.Props{Anchor: "save-button", Label: "保存文件"})
```

### WindowCaption

窗口最小化、最大化和关闭按钮图形：`ui.WindowCaption()` 不接收参数。通常不需要手动创建，框架标题栏会自动处理。应用启用 `HiddenTitlebar()` 自绘顶栏时可将它放入顶栏；按钮点击与窗口命中由宿主处理。Web 平台不绘制系统窗口按钮。

## 自定义组件与状态

`ui.Component` 的 `setup` 只运行一次，返回的函数每次重绘时产生新的根节点。`ui.Ref` 应在 `setup` 中创建；组件内设置 `Ref` 会触发重绘。

```go
var Counter = ui.Component(func(ctx *ui.SetupContext) func() *ui.VNode {
	count := ui.Ref(0)
	ctx.OnUnmounted(func() { /* 清理资源 */ })
	return func() *ui.VNode {
		return ui.VStack(
			ui.Text(ui.Props{Style: ui.Style{Height: 24}}, "当前值：", count),
			ui.Button(ui.Props{OnClick: func() { count.Set(count.Get() + 1) }}, "增加"),
		)
	}
})
```

`SetupContext` 提供 `Invalidate`、`OnMounted`、`OnUnmounted`、`Props`、`Attr`、`String`、`Children`。自定义组件通过 `ui.H(MyComponent, props, children...)` 使用。`VNode.WithKey(string)` 设置列表节点 key；`VNode.ID()` 返回节点 ID。

需要焦点或编辑器控制时可使用 `ui.SetFocus(id)`、`ui.FocusedID()`、`ui.CaretLineCol`、`ui.MoveCaret`、`ui.SelectedRange`、`ui.SelectRange`、`ui.SelectAll`、`ui.EditOp`、`ui.EditOpID`。编辑命令字符串为 `undo`、`redo`、`cut`、`copy`、`paste`、`delete`、`selectall`。Textarea 滚动 API：`TextareaScrollY`、`TextareaScrollMax`、`SetTextareaScrollY`、`PageTextarea`、`TextareaOffsetY`、`TextareaLineHeight`。

## 参考

- 引擎应用 API：[`ui/ui.go`](../ui/ui.go)
- 控件实现：[`ui/components`](../ui/components)
- 最小应用：[`helloworld/main.go`](../helloworld/main.go)
- Go 模块：[`go.mod`](../go.mod)
