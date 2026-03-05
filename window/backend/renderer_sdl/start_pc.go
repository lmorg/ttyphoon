//go:build !darwin
// +build !darwin

package rendersdl

func (sr *sdlRender) registerHotkey(hks ...*hotkeyFuncT) {
	return // currently not supported on Linux
	sr._registerHotkey(hks...)
}
