package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) DrawHighlightRect(tile types.Tile, _topLeftCell, bottomRightCell *types.XY) {
	if tile == nil || _topLeftCell == nil || bottomRightCell == nil {
		return
	}
	if bottomRightCell.X <= 0 || bottomRightCell.Y <= 0 {
		return
	}

	topLeftCell := &types.XY{
		X: _topLeftCell.X + tile.Left() + 1,
		Y: _topLeftCell.Y + tile.Top(),
	}

	wr.TriggerDeallocation(func() {
		wr.enqueueDrawCommand(DrawCommand{
			Op:     DrawOpHighlight,
			X:      topLeftCell.X,
			Y:      topLeftCell.Y,
			Width:  bottomRightCell.X,
			Height: bottomRightCell.Y,
			Fg:     highlightBorderColour,
			Bg:     highlightFillColour,
		})
	})
}

func (wr *webkitRender) DrawRectWithColour(tile types.Tile, _topLeftCell, _bottomRightCell *types.XY, colour *types.Colour, incLeftMargin bool) {
	if tile == nil || _topLeftCell == nil || _bottomRightCell == nil || colour == nil {
		return
	}

	topLeftCell := &types.XY{
		X: _topLeftCell.X,
		Y: max(_topLeftCell.Y, 0),
	}

	bottomRightCell := &types.XY{
		X: _bottomRightCell.X,
		Y: _bottomRightCell.Y + min(_topLeftCell.Y, 0),
	}

	if tile.GetTerm() != nil && bottomRightCell.Y+topLeftCell.Y > tile.GetTerm().GetSize().Y {
		bottomRightCell.Y = tile.GetTerm().GetSize().Y - topLeftCell.Y
	}

	if bottomRightCell.X <= 0 || bottomRightCell.Y <= 0 {
		return
	}

	leftOffset := int32(1)
	if incLeftMargin {
		leftOffset = 0
	}

	wr.TriggerDeallocation(func() {
		wr.enqueueDrawCommand(DrawCommand{
			Op:     DrawOpRectColour,
			X:      topLeftCell.X + tile.Left() + leftOffset,
			Y:      topLeftCell.Y + tile.Top(),
			Width:  bottomRightCell.X,
			Height: bottomRightCell.Y,
			Fg:     colour,
			Bg:     colour,
		})
	})
}
