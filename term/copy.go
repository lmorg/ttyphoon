package virtualterm

import (
	"strings"

	"github.com/lmorg/mxtty/types"
	"golang.design/x/clipboard"
)

func clone[T any](src []T) []T {
	s := make([]T, len(src))
	copy(s, src)
	return s
}

func (term *Term) CopyRange(topLeft, bottomRight *types.XY) []byte {
	if topLeft.X < 0 {
		topLeft.X = 0
	}
	if topLeft.Y < 0 {
		topLeft.Y = 0
	}
	if bottomRight.X > term.size.X {
		bottomRight.X = term.size.X
	}
	if bottomRight.X > term.size.X {
		bottomRight.Y = term.size.Y
	}

	// This is some ugly ass code. Sorry!
	// It is also called infrequently and not worth my time optimizing right now
	var (
		ix, iy int
		x, y   int32
		screen = term.visibleScreen()
		cell   *types.Cell
		b      []byte
		line   string
	)

	for iy = range screen {
		for ix, cell = range screen[iy].Cells {
			x, y = int32(ix), int32(iy)
			switch {
			case bottomRight.Y < topLeft.Y: // select up
				// start multiline
				if (x <= topLeft.X && y == topLeft.Y) ||
					// middle multiline
					(y < topLeft.Y && y > bottomRight.Y) ||
					// end multiline
					(x >= bottomRight.X && y == bottomRight.Y) {
					line += string(cell.Rune())
				}

			case topLeft.Y == bottomRight.Y: // midline
				if bottomRight.X < topLeft.X { //backwards
					if x <= topLeft.X && x >= bottomRight.X && y == topLeft.Y {
						line += string(cell.Rune())
					}
				} else { // forwards
					if x >= topLeft.X && x <= bottomRight.X && y == topLeft.Y {
						line += string(cell.Rune())
					}
				}

			default: // select down
				// start multiline
				if (x >= topLeft.X && y == topLeft.Y) ||
					// middle multiline
					(y > topLeft.Y && y < bottomRight.Y) ||
					// end multiline
					(x <= bottomRight.X && y == bottomRight.Y) {
					line += string(cell.Rune())
				}
			}
		}
		if len(line) > 0 {
			line = strings.TrimRight(line, " ")
			b = append(b, []byte(line+"\n")...)
			line = ""
		}
	}

	if len(b) > 0 {
		return b[:len(b)-1]
	}
	return b
}

func (term *Term) CopyLines(top, bottom int32) []byte {
	if top < 0 {
		top = 0
	}
	if bottom > term.size.Y {
		bottom = term.size.Y
	}

	screen := term.visibleScreen()
	var b []byte

	for y := top; y <= bottom; y++ {
		var line string
		for _, cell := range screen[y].Cells {
			line += string(cell.Rune())
		}
		line = strings.TrimRight(line, " ") + "\n"
		b = append(b, []byte(line)...)
	}

	if len(b) > 0 {
		return b[:len(b)-1]
	}
	return b
}

func (term *Term) CopySquare(begin *types.XY, end *types.XY) []byte {
	if begin.X < 0 {
		begin.X = 0
	}
	if begin.Y < 0 {
		begin.Y = 0
	}
	if end.X > term.size.X {
		end.X = term.size.X
	}
	if end.X > term.size.X {
		end.Y = term.size.Y
	}

	screen := term.visibleScreen()
	var b []byte

	for y := begin.Y; y <= end.Y; y++ {
		var line string
		for x := begin.X; x <= end.X; x++ {
			line += string(screen[y].Cells[x].Rune())
		}
		line = strings.TrimRight(line, " ") + "\n"
		b = append(b, []byte(line)...)
	}

	if len(b) > 0 {
		return b[:len(b)-1]
	}
	return b
}

func (term *Term) copyOutputBlock(blockPos [2]int32) []rune {
	var block []rune

	for i := int(blockPos[0]); i <= int(blockPos[1]); i++ {
		if i < len(term._scrollBuf) {
			block = append(block, *term._scrollBuf[i].Phrase...)
		} else {
			block = append(block, *term._normBuf[i-len(term._scrollBuf)].Phrase...)
		}
	}

	return block
}

func (term *Term) copyOutputBlockToClipboard(blockPos [2]int32) {
	clipboard.Write(clipboard.FmtText, []byte(string(term.copyOutputBlock(blockPos))))
}

func (term *Term) getCmdLine(pos int32) []rune {
	for i := int(pos); i > -1; i-- {
		if i < len(term._scrollBuf) {
			if term._scrollBuf[i].Meta.Is(types.ROW_OUTPUT_BLOCK_BEGIN) {
				return term._scrollBuf[i].Block.Query
			}

		} else {
			if term._normBuf[i-len(term._scrollBuf)].Meta.Is(types.ROW_OUTPUT_BLOCK_BEGIN) {
				return term._normBuf[i-len(term._scrollBuf)].Block.Query
			}

		}
	}

	return nil
}

func (term *Term) CmdLine(pos *types.XY) string {
	y := term.ConvertRelativeToAbsoluteY(pos)
	return string(term.getCmdLine(y))
}
