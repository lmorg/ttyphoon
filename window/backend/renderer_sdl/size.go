package rendersdl

import (
	"fmt"
	"log"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
)

/*func (sr *sdlRender) convertPxToCellXY(x, y int32) (*types.XY, bool) {
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

	if xy.X < sr.termWin.Active.TopLeft.X || xy.X > sr.termWin.Active.BottomRight.X ||
		xy.Y < sr.termWin.Active.TopLeft.Y || xy.Y > sr.termWin.Active.BottomRight.Y {
		return xy, false
	}

	xy.X -= sr.termWin.Active.TopLeft.X
	xy.Y -= sr.termWin.Active.TopLeft.Y

	return xy, true
}

func (sr *sdlRender) convertPxToCellXYNegX(x, y int32) (*types.XY, bool) {
	xy := &types.XY{
		X: (x - _PANE_LEFT_MARGIN) / sr.glyphSize.X,
		Y: (y - _PANE_TOP_MARGIN) / sr.glyphSize.Y,
	}

	if xy.X < 0 || x < _PANE_LEFT_MARGIN { // TODO
		xy.X = -1
	} else if xy.X >= sr.winCellSize.X {
		xy.X = sr.winCellSize.X - 1
	}
	if xy.Y < 0 {
		xy.Y = 0
	} else if xy.Y >= sr.winCellSize.Y {
		xy.Y = sr.winCellSize.Y - 1
	}

	if xy.X < sr.termWin.Active.TopLeft.X || xy.X > sr.termWin.Active.BottomRight.X ||
		xy.Y < sr.termWin.Active.TopLeft.Y || xy.Y > sr.termWin.Active.BottomRight.Y {
		return xy, false
	}

	xy.X -= sr.termWin.Active.TopLeft.X
	xy.Y -= sr.termWin.Active.TopLeft.Y

	return xy, true
}*/

func (sr *sdlRender) convertPxToCellXYTile(tile *types.Tile, x, y int32) *types.XY {
	xy := &types.XY{
		X: ((x - _PANE_LEFT_MARGIN) / sr.glyphSize.X) - tile.TopLeft.X,
		Y: ((y - _PANE_TOP_MARGIN) / sr.glyphSize.Y) - tile.TopLeft.Y,
	}

	termSize := tile.Term.GetSize()

	if xy.X < 0 {
		xy.X = 0
	} else if xy.X >= termSize.X {
		xy.X = termSize.X - 1
	}
	if xy.Y < 0 {
		xy.Y = 0
	} else if xy.Y >= termSize.Y {
		xy.Y = termSize.Y - 1
	}

	return xy
}

func (sr *sdlRender) convertPxToCellXYNegXTile(tile *types.Tile, x, y int32) *types.XY {
	xy := &types.XY{
		X: ((x - _PANE_LEFT_MARGIN) / sr.glyphSize.X) - tile.TopLeft.X,
		Y: ((y - _PANE_TOP_MARGIN) / sr.glyphSize.Y) - tile.TopLeft.Y,
	}

	termSize := tile.Term.GetSize()

	if xy.X < 0 {
		xy.X = -1
	} else if xy.X >= termSize.X {
		xy.X = termSize.X - 1
	}
	if xy.Y < 0 {
		xy.Y = 0
	} else if xy.Y >= termSize.Y {
		xy.Y = termSize.Y - 1
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
	newRect := &sdl.Rect{
		X: (rect.X - _PANE_LEFT_MARGIN) / sr.glyphSize.X,
		Y: (rect.Y - _PANE_TOP_MARGIN) / sr.glyphSize.Y,
		W: ((rect.X + rect.W - _PANE_LEFT_MARGIN) / sr.glyphSize.X),
		H: ((rect.Y + rect.H - _PANE_TOP_MARGIN) / sr.glyphSize.Y),
	}

	return newRect
}

func (sr *sdlRender) getTileFromPxOrActive(x, y int32) *types.Tile {
	x = (x - _PANE_LEFT_MARGIN) / sr.glyphSize.X
	y = (y - _PANE_TOP_MARGIN) / sr.glyphSize.Y

	for _, tile := range sr.termWin.Tiles {
		if tile.Term == nil {
			continue
		}

		if x >= tile.TopLeft.X && x <= tile.BottomRight.X &&
			y >= tile.TopLeft.Y && y <= tile.BottomRight.Y {
			return tile
		}
	}

	return sr.termWin.Active
}

// GetTermSize only exists so that elements can get the terminal size without
// having access to the term interface.
func (sr *sdlRender) GetTermSize(tileId types.TileId) *types.XY {
	debug.Log(tileId)
	return sr.termWin.Tiles[tileId].Term.GetSize()
}

/*func (sr *sdlRender) GetWinSize() *types.XY {
	return sr.winCellSize
}*/

// GetWindowSizeCells should only be called upon terminal resizing.
// All other checks for terminal size should come from winCellSize
func (sr *sdlRender) GetWindowSizeCells() *types.XY {
	x, y, err := sr.renderer.GetOutputSize()
	if err != nil {
		log.Println("i don't know how big the terminal window is")
		x, y = sr.window.GetSize()
	}
	//x, y := sr.window.GetSize()

	sr.winCellSize = &types.XY{
		X: ((x - _PANE_LEFT_MARGIN) / sr.glyphSize.X),
		Y: ((y - _PANE_TOP_MARGIN) / sr.glyphSize.Y) - sr.footer,
	}

	debug.Log(sr.winCellSize)
	debug.Log(sr.footer)

	return sr.winCellSize
}

///// resize

func (sr *sdlRender) windowResized() {
	size := sr.GetWindowSizeCells()

	if sr.termWin == nil {
		return
	}

	if !config.Config.Tmux.Enabled {
		sr.termWin.Active.Term.Resize(size)
		return
	}

	if sr.windowTabs == nil {
		debug.Log("sr.windowTabs is unset")
		return
	}

	//winId := sr.windowTabs.windows[sr.windowTabs.active].Id
	//sr.windowTabs = nil
	//err := sr.tmux.SelectAndResizeWindow(winId, size)
	err := sr.tmux.SelectAndResizeWindow(sr.windowTabs.windows[sr.windowTabs.active].Id, size)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Unable to resize window: %v", err))
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
