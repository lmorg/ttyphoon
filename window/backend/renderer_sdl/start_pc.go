//go:build !darwin
// +build !darwin

package rendersdl

func (sr *sdlRender) registerHotkey(hks ...*hotkeyFuncT) {
	sr._registerHotkey(hks...)
}
