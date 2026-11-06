package mainwindow

import (
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) ShowTooltip(text string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.ShowTooltip(text)
}

func (a *WApp) CloseTooltip() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.CloseTooltip()
}
