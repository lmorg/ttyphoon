//go:build darwin
// +build darwin

package globalhotkeys

func registerHotkey(hks ...*hotkeyFuncT) {
	go _registerHotkey(hks...)
}
