package platform

// SaveImage 打开“另存为”对话框。取消时 ok=false。
var SaveImage func(name, format string) (path string, ok bool, err error)

// OpenImage 打开“选择图片”对话框。
var OpenImage func() (path string, ok bool, err error)

// OpenFile 打开“选择文件”对话框。filter 为空时列出所有文件。
var OpenFile func(title, filter string) (path string, ok bool, err error)

// SaveFile 打开“保存文件”对话框。
var SaveFile func(title, name, filter, ext string) (path string, ok bool, err error)

// OpenFolder 打开“选择文件夹”对话框。
var OpenFolder func(title string) (path string, ok bool, err error)

// ClipboardGet 读取剪贴板文本。
var ClipboardGet func() string

// ClipboardSet 写入剪贴板文本。
var ClipboardSet func(string)
