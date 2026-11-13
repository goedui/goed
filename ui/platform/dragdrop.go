package platform

// DoDragDropFiles 把一组磁盘路径交给系统拖放（Windows 上是 OLE DoDragDrop）。
//
// 阻塞到用户松手或取消。必须在 UI 线程上调用；调用前宿主会放开鼠标捕获，
// 否则松手事件会被本窗口吃掉、拖放永远不结束。
//
// 路径必须位于同一父目录：Shell 数据对象只接受同一文件夹下的子项。
// 未实现的平台为 nil，调用方要判空。
var DoDragDropFiles func(paths []string) error
