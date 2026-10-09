//go:build js && wasm

package platform

import "github.com/goedui/goed/ui/platform/web"

// 浏览器里没有同步的文件对话框：<input type=file> 与 showSaveFilePicker 都是
// 异步的，而这里的签名要求「调用即返回路径」。所以文件对话框一律保持
// 「用户取消」的语义（ok=false）。直接保存走 SaveBytes（<a download>）。
func init() {
	if SetWindowTitle == nil {
		SetWindowTitle = web.SetDocumentTitle
	}

	SaveBytes = func(name, mime string, data []byte) (string, bool, error) {
		if err := web.DownloadBytes(name, mime, data); err != nil {
			return "", false, err
		}
		return name, true, nil
	}
	SaveImage = func(string, string) (string, bool, error) { return "", false, nil }
	OpenImage = func() (string, bool, error) { return "", false, nil }
	OpenFile = func(string, string) (string, bool, error) { return "", false, nil }
	SaveFile = func(string, string, string, string) (string, bool, error) { return "", false, nil }
	OpenFolder = func(string) (string, bool, error) { return "", false, nil }

	ClipboardGet = web.ClipboardText
	ClipboardSet = web.SetClipboardText

	// js/wasm 只有一个线程：所有 goroutine 都跑在这条线程上，
	// 「投递到 UI 线程」与直接调用是同一件事。
	if Post == nil {
		Post = func(fn func()) {
			if fn != nil {
				fn()
			}
		}
	}
}
