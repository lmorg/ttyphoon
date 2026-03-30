package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) DrawOutputBlockChrome(tile types.Tile, _start, n int32, c *types.Colour, folded bool) {
	if tile == nil || tile.GetTerm() == nil || c == nil {
		return
	}

	termHeight := tile.GetTerm().GetSize().Y
	if _start >= termHeight {
		return
	}

	start := _start + tile.Top()
	height := n
	if _start+n >= termHeight {
		height = termHeight - _start - 1
	}
	if height < 0 {
		return
	}

	cmd := DrawCommand{
		Op:     DrawOpBlockChrome,
		X:      tile.Left(),
		Y:      start,
		Height: height + 1,
		EndX:   tile.Right() + 2,
		Fg:     c,
		Folded: folded,
	}

	wr.enqueueDrawCommand(cmd)
}
