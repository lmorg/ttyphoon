package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/types"
)

func (wr *webkitRender) DrawTable(tile types.Tile, pos *types.XY, height int32, boundaries []int32) {
	if tile == nil || pos == nil || len(boundaries) == 0 {
		return
	}

	if height <= 0 {
		return
	}

	fg := types.SGR_COLOR_FOREGROUND

	X := pos.X + tile.Left() + 1
	Y := pos.Y + tile.Top()
	tableHeight := height + 1

	wr.TriggerDeallocation(func() {
		// Send a single table command with all boundaries information
		wr.enqueueDrawCommand(DrawCommand{
			Op:         DrawOpTable,
			X:          X,
			Y:          Y,
			Height:     tableHeight,
			Width:      boundaries[len(boundaries)-1],
			Boundaries: boundaries,
			Fg:         fg,
		})
	})
}
