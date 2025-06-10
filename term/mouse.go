package virtualterm

import (
	"fmt"
	"strings"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
)

// MouseClick: pos X should be -1 when out of bounds
func (term *Term) MouseClick(pos *types.XY, button types.MouseButtonT, clicks uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	term._mousePosRenderer.Set(nil)

	screen := term.visibleScreen()

	// this is used to determine whether to override ligatures with default font rendering
	term._mouseButtonDown = state == types.BUTTON_PRESSED

	if pos == nil {
		// this just exists to reset ligatures
		return
	}

	if state == types.BUTTON_PRESSED {
		callback()
		return
	}

	absPosY := len(term._scrollBuf) - term._scrollOffset + int(pos.Y)

	if button == types.MOUSE_BUTTON_RIGHT && !term.IsAltBuf() {
		if screen[pos.Y].Block.Meta != types.META_BLOCK_NONE {
			term._mouseClickContextMenuOutputBlock(absPosY)
		}
	}

	if pos.X < 0 {
		if button != types.MOUSE_BUTTON_LEFT {
			callback()
			return
		}

		if len(screen[pos.Y].Hidden) > 0 {
			err := term.UnhideRows(absPosY)
			if err != nil {
				term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
			}
			return
		}

		if screen[pos.Y].Block.Meta == types.META_BLOCK_NONE {
			return
		}
		absBlockPos := term.getBlockStartAndEndAbs(absPosY)
		if err := term.HideRows(absBlockPos[0], absBlockPos[1]+1); err != nil {
			term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
		}
		return
	}

	if screen[pos.Y].Cells[pos.X].Element == nil {
		if button != types.MOUSE_BUTTON_LEFT {
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

	screen[pos.Y].Cells[pos.X].Element.MouseClick(screen[pos.Y].Cells[pos.X].GetElementXY(), button, clicks, state, callback)
}

func (term *Term) _mouseClickContextMenuOutputBlock(absPosY int) {
	absBlockPos := term.getBlockStartAndEndAbs(absPosY)
	relBlockPos := term.getBlockStartAndEndRel(absBlockPos)
	meta := agent.Get(term.tile.Id())
	meta.Term = term
	meta.Renderer = term.renderer
	meta.CmdLine = string(term.getCmdLine(int(absBlockPos[0])))
	meta.Pwd = term.RowSrcFromScrollBack(absBlockPos[0]).Pwd
	meta.OutputBlock = string(term.copyOutputBlock(absBlockPos))
	meta.InsertAfterRowId = term.GetRowId(term.curPos().Y - 1)

	term.renderer.AddToContextMenu(
		[]types.MenuItem{
			{
				Title: "Copy output block to clipboard",
				Icon:  0xf0c5,
				Highlight: func() func() {
					return func() {
						term.renderer.DrawRectWithColour(term.tile, &types.XY{X: 0, Y: relBlockPos[0]}, &types.XY{X: term.size.X, Y: relBlockPos[1]}, types.COLOR_SELECTION, true)
					}
				},
				Fn: func() { term.copyOutputBlockToClipboard(absBlockPos) },
			},
			{
				Title: fmt.Sprintf("Explain output block (%s)", meta.ServiceName()),
				Icon:  0xf544,
				Highlight: func() func() {
					return func() {
						term.renderer.DrawRectWithColour(term.tile, &types.XY{X: 0, Y: relBlockPos[0]}, &types.XY{X: term.size.X, Y: relBlockPos[1]}, types.COLOR_SELECTION, true)
					}
				},
				Fn: func() { ai.Explain(meta, true /*false*/) },
			},
			/*{
				Title: fmt.Sprintf("Explain with custom prompt (%s)", meta.ServiceName()),
				Icon:  0xf6e8,
				Highlight: func() func() {
					return func() {
						term.renderer.DrawRectWithColour(term.tile, &types.XY{X: 0, Y: block[0]}, &types.XY{X: term.size.X, Y: block[1]}, types.COLOR_SELECTION, true)
					}
				},
				Fn: func() { ai.Explain(meta, true) },
			},*/
		}...)
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

		if screen[pos.Y].Block.Meta != types.META_BLOCK_NONE {
			cursor.Hand()
			return
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

		term.autoHotlink(screen[pos.Y])

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

func (term *Term) MouseHover(pos *types.XY) {
	if term._mousePosRenderer.Call() {
		return
	}

	defer term._mousePosRenderer.Call()

	screen := term.visibleScreen()

	if pos.X < 0 {
		if len(screen[pos.Y].Hidden) > 0 {
			colour := _outputBlockChromeColour(screen[pos.Y].Hidden[len(screen[pos.Y].Hidden)-1].Block.Meta)
			term._mousePosRenderer.Set(func() {
				term.renderer.DrawRectWithColour(term.tile,
					&types.XY{X: 0, Y: pos.Y},
					&types.XY{X: term.size.X, Y: 1},
					colour, true,
				)
			})
			return
		}

		if screen[pos.Y].Block.Meta == types.META_BLOCK_NONE {
			term._mousePosRenderer.Set(func() {})
			return
		}

		colour := _outputBlockChromeColour(screen[pos.Y].Block.Meta)
		relBlockPos := term.getBlockStartAndEndRel(term.getBlockStartAndEndAbs(int(term.convertRelPosToAbsPos(pos).Y)))
		term._mousePosRenderer.Set(func() {
			term.renderer.DrawRectWithColour(term.tile,
				&types.XY{X: 0, Y: relBlockPos[0]},
				&types.XY{X: term.size.X, Y: relBlockPos[1]},
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
				h := min(height-pos.Y, term.size.Y-pos.Y)
				term.renderer.DrawRectWithColour(term.tile,
					&types.XY{X: pos.X, Y: pos.Y},
					&types.XY{X: term.size.X - pos.X, Y: h},
					types.COLOR_FOLDED, false,
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
