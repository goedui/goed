package platform

// SaveBytes 把字节保存到平台默认位置，不弹对话框。
// Windows 写入「下载」文件夹；浏览器按 name 触发一次 download。
// 取消或未实现时 ok=false。path 在桌面是完整路径，在 Web 上是下载文件名。
var SaveBytes func(name, mime string, data []byte) (path string, ok bool, err error)

// SaveImage 打开“另存为”对话框。取消时 ok=false。
var SaveImage func(name, format string) (path string, ok bool, err error)

// OpenImage 打开“选择图片”对话框。
var OpenImage func() (path string, ok bool, err error)

// OpenFile 打开“选择文件”对话框。filter 为空时列出所有文件。
var OpenFile func(title, filter string) (path string, ok bool, err error)

// SaveFile 打开“保存文件”对话框。
var SaveFile func(title, name, filter, ext string) (path string, ok bool, err error)

// OpenFolder 打开“选择文件夹”对话框。
//
// 返回 (路径, 选中了吗, 出错了吗)。**用户取消不算错**：ok=false、err=nil。
//
// Windows 上默认走老式对话框（shell32 的 SHBrowseForFolderW），但树的根钉在
// 「此电脑」（FOLDERID_ComputerFolder）而不是桌面，于是盘符是树的一级子项；另外带
// 一个能直接输路径的输入框（BIF_EDITBOX）。这两点合起来才是「所有盘和路径都能选」——
// 只钉根、不给输入框，遇到 UNC 路径或要粘一条长路径时在界面上仍然没有入口。
//
// 想换成 Vista+ 的 IFileOpenDialog（资源管理器那种带导航窗格、地址栏、面包屑的界面）：
// 把环境变量 GOED_FOLDER_DIALOG 设成 "modern"。它已经实现好了，只是默认不开 ——
// 现代 Common Item Dialog 在某些机器上建得出窗口却永不抽消息，Show 收不回来，
// 会把整个界面一起卡住。这一点用五组对照验证过**不是本库的问题**（Go/Python 两种宿主、
// 有无 DPI 感知、选文件夹/选文件、自建 COM/comdlg32），完整证据链见
// ui/platform/windows/input.go 里 folderDialogEnv 的说明。
var OpenFolder func(title string) (path string, ok bool, err error)

// ClipboardGet 读取剪贴板文本。
var ClipboardGet func() string

// ClipboardSet 写入剪贴板文本。
var ClipboardSet func(string)
