//go:build !windows

package platform

func init() {
	if SetWindowTitle == nil {
		SetWindowTitle = func(string) {}
	}
}

func init() {
	SaveImage = func(string, string) (string, bool, error) { return "", false, nil }
	OpenImage = func() (string, bool, error) { return "", false, nil }
	OpenFile = func(string, string) (string, bool, error) { return "", false, nil }
	SaveFile = func(string, string, string, string) (string, bool, error) { return "", false, nil }
	OpenFolder = func(string) (string, bool, error) { return "", false, nil }
	ClipboardGet = func() string { return "" }
	ClipboardSet = func(string) {}
	if Post == nil {
		Post = func(fn func()) {
			if fn != nil {
				fn()
			}
		}
	}
}
