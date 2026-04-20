package main

import (
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

// ShowCommandPalette opens the command palette and sends all options to the
// frontend in one payload. Filtering is done in JS.
func (a *WApp) ShowCommandPalette() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	renderer.ShowCommandPalette()
}

// CommandPaletteSelect executes the chosen item via the renderer.
func (a *WApp) CommandPaletteSelect(index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	renderer.CommandPaletteSelect(index)
}
