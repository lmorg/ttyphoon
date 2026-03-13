package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

type DrawOp string

const (
	DrawOpFrame       DrawOp = "frame"
	DrawOpCell        DrawOp = "cell"
	DrawOpGaugeH      DrawOp = "gauge_h"
	DrawOpGaugeV      DrawOp = "gauge_v"
	DrawOpBlockChrome DrawOp = "block_chrome"
)

type DrawCommand struct {
	Op        DrawOp        `json:"op"`
	X         int32         `json:"x"`
	Y         int32         `json:"y"`
	Height    int32         `json:"height"`
	EndX      int32         `json:"endX"`
	Char      string        `json:"char,omitempty"`
	Fg        *types.Colour `json:"fg,omitempty"`
	Bg        *types.Colour `json:"bg,omitempty"`
	Bold      bool          `json:"bold,omitempty"`
	Italic    bool          `json:"italic,omitempty"`
	Underline bool          `json:"underline,omitempty"`
	Strike    bool          `json:"strike,omitempty"`
	Width     int32         `json:"width"`
	Value     int32         `json:"value"`
	Max       int32         `json:"max"`
	Folded    bool          `json:"folded,omitempty"`
}

func sgrOpts(sgr *types.Sgr, forceBg bool) (fg *types.Colour, bg *types.Colour) {
	if sgr == nil {
		return types.SGR_COLOR_FOREGROUND, nil
	}

	if sgr.Bitwise.Is(types.SGR_INVERT) {
		bg, fg = sgr.Fg, sgr.Bg
	} else {
		fg, bg = sgr.Fg, sgr.Bg
	}

	if bg == types.SGR_COLOR_BACKGROUND && !forceBg {
		bg = nil
	}

	return fg, bg
}

func (wr *webkitRender) PrintCell(tile types.Tile, cell *types.Cell, cellPos *types.XY) {
	if cell == nil || cellPos == nil || cell.Char == 0 || cellPos.X < 0 || cellPos.Y < 0 {
		return
	}

	if tile.GetTerm() != nil {
		tileSize := tile.GetTerm().GetSize()
		if cellPos.X >= tileSize.X || cellPos.Y >= tileSize.Y {
			return
		}
	}

	if cell.Sgr == nil {
		return
	}

	pos := types.XY{
		X: cellPos.X + tile.Left(),
		Y: cellPos.Y + tile.Top(),
	}

	fg, bg := sgrOpts(cell.Sgr, false)
	width := int32(1)
	if cell.Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		width = 2
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:        DrawOpCell,
		X:         pos.X,
		Y:         pos.Y,
		Char:      string(cell.Char),
		Fg:        fg,
		Bg:        bg,
		Bold:      cell.Sgr.Bitwise.Is(types.SGR_BOLD),
		Italic:    cell.Sgr.Bitwise.Is(types.SGR_ITALIC),
		Underline: cell.Sgr.Bitwise.Is(types.SGR_UNDERLINE),
		Strike:    cell.Sgr.Bitwise.Is(types.SGR_STRIKETHROUGH),
		Width:     width,
	})
}

func (wr *webkitRender) PrintRow(tile types.Tile, cells []*types.Cell, cellPos *types.XY) {
	if cellPos == nil {
		return
	}

	length := int32(len(cells))

	if tile.GetTerm() != nil && tile.GetTerm().GetSize().X <= cellPos.X+length {
		length = tile.GetTerm().GetSize().X - cellPos.X
	}

	pos := &types.XY{X: cellPos.X, Y: cellPos.Y}
	for ; pos.X < length; pos.X++ {
		if cells[pos.X] == nil {
			continue
		}
		wr.PrintCell(tile, cells[pos.X], pos)
	}
}
