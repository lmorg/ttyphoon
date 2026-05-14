package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

type DrawOp string

const (
	DrawOpFrame       DrawOp = "frame"
	DrawOpCell        DrawOp = "cell"
	DrawOpImage       DrawOp = "image"
	DrawOpGaugeH      DrawOp = "gauge_h"
	DrawOpGaugeV      DrawOp = "gauge_v"
	DrawOpBlockChrome DrawOp = "block_chrome"
	DrawOpHighlight   DrawOp = "highlight_rect"
	DrawOpRectColour  DrawOp = "rect_colour"
	DrawOpTileOverlay DrawOp = "tile_overlay"
	DrawOpTable       DrawOp = "table"
)

type DrawCommand struct {
	Op         DrawOp
	Flags      int32
	X          int32
	Y          int32
	Height     int32
	EndX       int32
	Char       string
	Fg         *types.Colour
	Bg         *types.Colour
	Width      int32
	Value      int32
	Max        int32
	Folded     bool
	Alpha      uint8
	ImageID    int64
	SrcWidth   int32
	SrcHeight  int32
	SrcScaleX  float64
	SrcScaleY  float64
	Boundaries []int32
	UlC        *types.Colour
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
		X: cellPos.X + tile.Left() + 1,
		Y: cellPos.Y + tile.Top(),
	}

	fg, bg := sgrOpts(cell.Sgr, false)
	width := int32(1)
	if cell.Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		width = 2
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:    DrawOpCell,
		Flags: int32(cell.Sgr.Bitwise),
		X:     pos.X,
		Y:     pos.Y,
		Char:  string(cell.Char),
		Fg:    fg,
		Bg:    bg,
		UlC:   cell.Sgr.UlC,
		Width: width,
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
	if length <= 0 {
		return
	}

	// Group adjacent cells with identical SGR style into a single draw
	// command whose Char field holds the complete run string.  When JS passes
	// a multi-character string to fillText() in a single call the underlying
	// OpenType shaper applies liga/calt substitutions automatically, enabling
	// ligatures without any bespoke pair table.

	runStart := int32(0)
	for runStart < length {
		// Skip nil cells at the start of a potential run.
		if cells[runStart] == nil || cells[runStart].Sgr == nil {
			runStart++
			continue
		}

		ref := cells[runStart]
		refFg, refBg := sgrOpts(ref.Sgr, false)
		refFlags := ref.Sgr.Bitwise

		// Accumulate printable characters that share the same style.

		runChars := []rune{ref.Char}
		runEnd := runStart + 1
		for runEnd < length {
			c := cells[runEnd]
			if c == nil || c.Sgr == nil {
				break
			}
			// Skip wide-char continuation cells (zero char) — they are already
			// accounted for by the preceding wide character's width field.
			if c.Char == 0 {
				runEnd++
				continue
			}
			cFg, cBg := sgrOpts(c.Sgr, false)
			if c.Sgr.Bitwise != refFlags || cFg != refFg || cBg != refBg || c.Sgr.UlC != ref.Sgr.UlC {
				break
			}
			if ref.Char > 0x7f || c.Char > 0x7f {
				// Only ASCII runs are grouped into one draw command so ligature shaping
				// does not affect Unicode cell alignment.
				break
			}
			runChars = append(runChars, c.Char)
			runEnd++
		}

		// Determine the pixel width of the run (in cells) accounting for
		// wide characters.
		width := int32(0)
		for i := runStart; i < runEnd; i++ {
			if cells[i] == nil {
				continue
			}
			if cells[i].Sgr != nil && cells[i].Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
				width += 2
			} else {
				width += 1
			}
		}
		if width == 0 {
			runStart = runEnd
			continue
		}

		pos := types.XY{
			X: cellPos.X + runStart + tile.Left() + 1,
			Y: cellPos.Y + tile.Top(),
		}

		wr.enqueueDrawCommand(DrawCommand{
			Op:    DrawOpCell,
			Flags: int32(refFlags),
			X:     pos.X,
			Y:     pos.Y,
			Char:  string(runChars),
			Fg:    refFg,
			Bg:    refBg,
			UlC:   ref.Sgr.UlC,
			Width: width,
		})

		runStart = runEnd
	}
}
