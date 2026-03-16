package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) DrawGaugeH(tile types.Tile, topLeft *types.XY, width int32, value, max int, c *types.Colour) {
	if tile == nil || topLeft == nil || c == nil || width <= 0 || max <= 0 {
		return
	}

	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:    DrawOpGaugeH,
		X:     topLeft.X + tile.Left() + 1,
		Y:     topLeft.Y + tile.Top(),
		Width: width,
		Value: int32(value),
		Max:   int32(max),
		Fg:    c,
	})
}

func (wr *webkitRender) DrawGaugeV(tile types.Tile, topLeft *types.XY, height int32, value, max int, c *types.Colour) {
	if tile == nil || topLeft == nil || c == nil || height <= 0 || max <= 0 {
		return
	}

	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpGaugeV,
		X:      topLeft.X + tile.Left(),
		Y:      topLeft.Y + tile.Top(),
		Height: height,
		Value:  int32(value),
		Max:    int32(max),
		Fg:     c,
	})
}
