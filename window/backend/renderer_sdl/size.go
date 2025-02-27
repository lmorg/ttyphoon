package rendersdl

import (
	"log"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) convertPxToCellXY(x, y int32) *types.XY {
	xy := &types.XY{
		X: (x - _PANE_LEFT_MARGIN) / sr.glyphSize.X,
		Y: (y - _PANE_TOP_MARGIN) / sr.glyphSize.Y,
	}

	if xy.X < 0 {
		xy.X = 0
	} else if xy.X >= sr.winCellSize.X {
		xy.X = sr.winCellSize.X - 1
	}
	if xy.Y < 0 {
		xy.Y = 0
	} else if xy.Y >= sr.winCellSize.Y {
		xy.Y = sr.winCellSize.Y - 1
	}

	return xy
}

func (sr *sdlRender) convertPxToCellXYNegX(x, y int32) *types.XY {
	xy := &types.XY{
		X: (x - _PANE_LEFT_MARGIN) / sr.glyphSize.X,
		Y: (y - _PANE_TOP_MARGIN) / sr.glyphSize.Y,
	}

	if xy.X < 0 || x < _PANE_LEFT_MARGIN {
		xy.X = -1
	} else if xy.X >= sr.winCellSize.X {
		xy.X = sr.winCellSize.X - 1
	}
	if xy.Y < 0 {
		xy.Y = 0
	} else if xy.Y >= sr.winCellSize.Y {
		xy.Y = sr.winCellSize.Y - 1
	}

	return xy
}

func normaliseRect(rect *sdl.Rect) {
	if rect.W < 0 {
		rect.X += rect.W
		rect.W = -rect.W
	}

	if rect.H < 0 {
		rect.Y += rect.H
		rect.H = -rect.H
	}
}

func (sr *sdlRender) rectPxToCells(rect *sdl.Rect) *sdl.Rect {
	return &sdl.Rect{
		X: (rect.X - _PANE_LEFT_MARGIN) / sr.glyphSize.X,
		Y: (rect.Y - _PANE_TOP_MARGIN) / sr.glyphSize.Y,
		W: ((rect.X + rect.W - _PANE_LEFT_MARGIN) / sr.glyphSize.X),
		H: ((rect.Y + rect.H - _PANE_TOP_MARGIN) / sr.glyphSize.Y),
	}
}

// GetTermSize only exists so that elements can get the terminal size without
// having access to the term interface.
func (sr *sdlRender) GetTermSize(tileId types.TileId) *types.XY {
	return sr.termWin.Tiles[tileId].Term.GetSize()
}

func (sr *sdlRender) GetWinSize() *types.XY {
	return sr.winCellSize
}

// GetWindowSizeCells should only be called upon terminal resizing.
// All other checks for terminal size should come from winCellSize
func (sr *sdlRender) GetWindowSizeCells() *types.XY {
	x, y, err := sr.renderer.GetOutputSize()
	if err != nil {
		log.Println("i don't know how big the terminal window is")
		x, y = sr.window.GetSize()
	}

	sr.winCellSize = &types.XY{
		X: ((x - _PANE_LEFT_MARGIN) / sr.glyphSize.X),
		Y: ((y - _PANE_TOP_MARGIN) / sr.glyphSize.Y) - sr.footer,
	}

	debug.Log(sr.winCellSize)

	return sr.winCellSize
}

///// resize

func (sr *sdlRender) windowResized() {
	sr.windowTabs = nil
	if sr.termWin != nil {
		sr.termWin.Active.Resize(sr.GetWindowSizeCells())
	}
}

func (sr *sdlRender) ResizeWindow(size *types.XY) {
	go func() { sr._resize <- size }()
}

func (sr *sdlRender) _resizeWindow(size *types.XY) {
	w := (size.X * sr.glyphSize.X) + _PANE_LEFT_MARGIN
	h := ((size.Y + sr.footer) * sr.glyphSize.Y) + _PANE_TOP_MARGIN
	sr.window.SetSize(w, h)
	sr.RefreshWindowList()
}
