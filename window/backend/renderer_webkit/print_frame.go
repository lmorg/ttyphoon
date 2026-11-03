package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

const (
	frameFlagResultMask int32 = 0x3
	frameFlagTopHalf    int32 = 1 << 2
	frameFlagBottomHalf int32 = 1 << 3
)

func (wr *webkitRender) DrawFrame(tile types.Tile) {
	var flags int32
	if pane, ok := tile.(interface{ InteractiveActionChrome() *types.Colour }); ok {
		switch pane.InteractiveActionChrome() {
		case types.COLOR_OK:
			flags = 1
		case types.COLOR_ERROR:
			flags = 2
		}
	}
	if wr.termWin != nil {
		for _, other := range wr.termWin.Tiles {
			if other == nil || other == tile {
				continue
			}
			if tile.Right() <= other.Left() || other.Right() <= tile.Left() {
				continue
			}
			if other.Top() == tile.Bottom()+2 {
				flags |= frameFlagBottomHalf
			}
			if tile.Top() == other.Bottom()+2 {
				flags |= frameFlagTopHalf
			}
		}
	}
	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpFrame,
		Flags:  flags,
		X:      tile.Left(),
		Y:      tile.Top() - 1,
		Width:  tile.Right() - tile.Left() + 2,
		Height: tile.Bottom() - tile.Top() + 2,
	})
}
