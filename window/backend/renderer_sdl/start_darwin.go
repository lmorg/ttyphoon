//go:build darwin
// +build darwin

package rendersdl

func (sr *sdlRender) registerHotkey(hks ...*hotkeyFuncT) {
	go sr._registerHotkey(hks...)
}
