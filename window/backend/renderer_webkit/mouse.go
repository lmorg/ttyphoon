package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) HandleMouseButton(cellX, cellY int32, button types.MouseButtonT, clicks uint8, state types.ButtonStateT) {
	tile := wr.getTileFromCellOrActive(cellX, cellY)
	if tile == nil || tile.GetTerm() == nil {
		return
	}

	if wr.termWin != nil && wr.termWin.Active != nil && wr.termWin.Active.GetTerm() != nil {
		wr.termWin.Active.GetTerm().HasFocus(false)
	}

	tile.GetTerm().HasFocus(true)
	if wr.termWin != nil {
		wr.termWin.Active = tile
	}

	pos := wr.convertCellToTileXYNegX(tile, cellX, cellY)
	tile.GetTerm().MouseClick(pos, button, clicks, state, func() {})
}

func (wr *webkitRender) HandleMouseWheel(cellX, cellY, moveX, moveY int32) {
	tile := wr.getTileFromCellOrActive(cellX, cellY)
	if tile == nil || tile.GetTerm() == nil {
		return
	}

	pos := wr.convertCellToTileXY(tile, cellX, cellY)
	tile.GetTerm().MouseWheel(pos, &types.XY{X: moveX, Y: moveY})
}

func (wr *webkitRender) HandleMouseMotion(cellX, cellY, relX, relY, state int32) {
	tile := wr.getTileFromCellOrActive(cellX, cellY)
	if tile == nil || tile.GetTerm() == nil {
		return
	}

	pos := wr.convertCellToTileXYNegX(tile, cellX, cellY)
	callback := func() {}
	if state == 0 {
		callback = wr.termMouseMotionCallback
	}

	tile.GetTerm().MouseMotion(pos, &types.XY{X: relX, Y: relY}, callback)
}

func (wr *webkitRender) termMouseMotionCallback() {}

func (wr *webkitRender) getTileFromCellOrActive(cellX, cellY int32) types.Tile {
	if wr.termWin == nil {
		return nil
	}

	for _, tile := range wr.termWin.Tiles {
		if tile.GetTerm() == nil {
			continue
		}

		if cellX >= tile.Left()-1 && cellX <= tile.Right() &&
			cellY >= tile.Top() && cellY <= tile.Bottom() {
			return tile
		}
	}

	if wr.termWin.Active != nil {
		return wr.termWin.Active
	}

	if len(wr.termWin.Tiles) == 0 {
		return nil
	}

	return wr.termWin.Tiles[0]
}

func (wr *webkitRender) convertCellToTileXY(tile types.Tile, cellX, cellY int32) *types.XY {
	xy := &types.XY{
		X: cellX - tile.Left(),
		Y: cellY - tile.Top(),
	}

	termSize := tile.GetTerm().GetSize()

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

func (wr *webkitRender) convertCellToTileXYNegX(tile types.Tile, cellX, cellY int32) *types.XY {
	xy := &types.XY{
		X: cellX - tile.Left() - 1,
		Y: cellY - tile.Top(),
	}

	termSize := tile.GetTerm().GetSize()

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
