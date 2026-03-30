package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) DrawFrame(tile types.Tile) {
	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpFrame,
		X:      tile.Left(),
		Y:      tile.Top(),
		Width:  tile.Right() - tile.Left() + 2,
		Height: tile.Bottom() - tile.Top() + 1,
	})
}
