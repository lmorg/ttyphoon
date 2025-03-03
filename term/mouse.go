package virtualterm

import (
	"strings"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
)

// MouseClick: pos X should be -1 when out of bounds
func (term *Term) MouseClick(pos *types.XY, button uint8, clicks uint8, pressed bool, callback types.EventIgnoredCallback) {
	term._mousePosRenderer.Set(nil)

	screen := term.visibleScreen()

	// this is used to determine whether to override ligatures with default font rendering
	term._mouseButtonDown = pressed

	if pos == nil {
		// this just exists to reset ligatures
		return
	}

	if pressed {
		callback()
		return
	}

	if pos.X < 0 {
		if button != 1 {
			callback()
			return
		}

		absPos := int32(len(term._scrollBuf)) - int32(term._scrollOffset) + pos.Y

		if len(screen[pos.Y].Hidden) > 0 {
			err := term.UnhideRows(absPos)
			if err != nil {
				term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
			}
			return
		}

		var block []int32
		for _, block = range term._cacheBlock {
			if block[0] <= pos.Y && block[0]+block[1] > pos.Y {
				goto isOutputBlock
			}
		}

		// not an output block
		return

	isOutputBlock:

		blockPos, _, err := term.outputBlocksFindStartEnd(absPos)
		debug.Log(blockPos)
		if err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
			return
		}

		if err = term.HideRows(blockPos[0], blockPos[1]+1); err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
		}
		return
	}

	if screen[pos.Y].Cells[pos.X].Element == nil {
		if button != 1 {
			callback()
			return
		}

		if h := term._mousePositionCodeFoldable(screen, pos); h != -1 {
			err := term.FoldAtIndent(pos)
			if err != nil {
				term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
			}
		}

		callback()
		return
	}

	screen[pos.Y].Cells[pos.X].Element.MouseClick(screen[pos.Y].Cells[pos.X].GetElementXY(), button, clicks, pressed, callback)
}

func (term *Term) MouseWheel(pos *types.XY, movement *types.XY) {
	term._mousePosRenderer.Set(nil)

	screen := term.visibleScreen()

	if screen[pos.Y].Cells[pos.X].Element == nil {
		term._mouseWheelCallback(movement)
		return
	}

	screen[pos.Y].Cells[pos.X].Element.MouseWheel(
		screen[pos.Y].Cells[pos.X].GetElementXY(),
		movement,
		func() { term._mouseWheelCallback(movement) },
	)
}

func (term *Term) _mouseWheelCallback(movement *types.XY) {
	if movement.Y == 0 {
		return
	}

	if term.IsAltBuf() {
		return
	}

	if len(term._scrollBuf) == 0 {
		return
	}

	term._scrollOffset += int(movement.Y * 2)
	term.updateScrollback()
}

func (term *Term) MouseMotion(pos *types.XY, movement *types.XY, callback types.EventIgnoredCallback) {
	term._mousePosRenderer.Set(nil)

	screen := term.visibleScreen()

	if pos.X < 0 {
		if term._mouseIn != nil {
			term._mouseIn.MouseOut()
		}

		if len(screen[pos.Y].Hidden) > 0 {
			cursor.Hand()
			return
		}

		var block []int32
		for _, block = range term._cacheBlock {
			if block[0] <= pos.Y && block[0]+block[1] > pos.Y {
				cursor.Hand()
				return
			}
		}

		cursor.Arrow()
		return
	}

	if height := term._mousePositionCodeFoldable(screen, pos); height >= 0 {
		cursor.Hand()
	} else {
		cursor.Arrow()
	}

	if screen[pos.Y].Cells[pos.X].Element == nil {
		if term._mouseIn != nil {
			term._mouseIn.MouseOut()
		}

		callback()
		return
	}

	if screen[pos.Y].Cells[pos.X].Element != term._mouseIn {
		if term._mouseIn != nil {
			term._mouseIn.MouseOut()
		}
		term._mouseIn = screen[pos.Y].Cells[pos.X].Element
	}

	screen[pos.Y].Cells[pos.X].Element.MouseMotion(screen[pos.Y].Cells[pos.X].GetElementXY(), movement, callback)
}

func (term *Term) MousePosition(pos *types.XY) {
	if term._mousePosRenderer.Call() {
		return
	}

	defer term._mousePosRenderer.Call()

	screen := term.visibleScreen()

	if pos.X < 0 {

		//absPos := int32(len(term._scrollBuf)) - int32(term._scrollOffset) + pos.Y

		if len(screen[pos.Y].Hidden) > 0 {
			term._mousePosRenderer.Set(func() {
				term.renderer.DrawRectWithColour(term.tileId,
					&types.XY{X: 0, Y: pos.Y},
					&types.XY{X: term.size.X, Y: 1},
					types.COLOUR_FOLDED, true,
				)
			})
			return
		}

		var block []int32
		for _, block = range term._cacheBlock {
			if block[0] <= pos.Y && block[0]+block[1] > pos.Y {
				goto isOutputBlock
			}
		}

		term._mousePosRenderer.Set(func() {})
		return

	isOutputBlock:
		var colour *types.Colour
		switch {
		case screen[block[0]+block[1]-1].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR):
			colour = types.COLOUR_ERROR
		case screen[block[0]+block[1]-1].Meta.Is(types.ROW_OUTPUT_BLOCK_END):
			colour = types.COLOUR_OK
		default:
			absScreen := append(term._scrollBuf, term._normBuf...)
			absPos := term.convertRelPosToAbsPos(pos)
			for y := absPos.Y + 1; int(y) < len(absScreen); y++ {
				switch {
				case absScreen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR):
					colour = types.COLOUR_ERROR
					goto drawRect
				case absScreen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_END):
					colour = types.COLOUR_OK
					goto drawRect
				default:
					continue
				}
			}
			term._mousePosRenderer.Set(func() {})
			return
		}

	drawRect:
		term._mousePosRenderer.Set(func() {
			term.renderer.DrawRectWithColour(term.tileId,
				&types.XY{X: 0, Y: block[0]},
				&types.XY{X: term.size.X, Y: block[1]},
				colour, true,
			)
		})
		return

	}

	if screen[pos.Y].Cells[pos.X].Element == nil {
		if height := term._mousePositionCodeFoldable(screen, pos); height >= 0 {
			cursor.Hand()
			term.renderer.StatusBarText("[Click] Fold branch")
			term._mousePosRenderer.Set(func() {
				term.renderer.DrawRectWithColour(term.tileId,
					&types.XY{X: pos.X, Y: pos.Y},
					&types.XY{X: term.size.X - pos.X, Y: height - pos.Y},
					types.COLOUR_FOLDED, false,
				)
			})
			return
		}
	}

	term._mousePosRenderer.Set(func() {})
}

func (term *Term) _mousePositionCodeFoldable(screen types.Screen, pos *types.XY) int32 {
	if pos.Y >= term.curPos().Y {
		return -1
	}

	if screen[pos.Y].Cells[pos.X].Char == ' ' {
		return -1
	}

	if pos.X > 0 && screen[pos.Y].Cells[pos.X-1].Char != ' ' {
		pos.X--
	}

	for x := pos.X - 1; x >= 0; x-- {
		if screen[pos.Y].Cells[x].Char != ' ' {
			return -1
		}
	}

	absScreen := append(term._scrollBuf, term._normBuf...)
	absPos := term.convertRelPosToAbsPos(pos)

	height, err := outputBlockFoldIndent(term, absScreen, absPos, false)
	if err != nil {
		return -1
	}

	height = height - absPos.Y + pos.Y

	if height-pos.Y == 2 && strings.TrimSpace(string(*screen[height-1].Phrase)) == "" {
		return -1
	}

	return height
}
