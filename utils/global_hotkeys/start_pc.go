//go:build !darwin
// +build !darwin

package globalhotkeys

func registerHotkey(hks ...*hotkeyFuncT) {
	_registerHotkey(hks...)
}
