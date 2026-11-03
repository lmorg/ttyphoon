package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) enqueueDrawCommand(cmd DrawCommand) {
	//wr.cmdMu.Lock()
	wr.drawCommands = append(wr.drawCommands, cmd)
	//wr.cmdMu.Unlock()
}

func (wr *webkitRender) PopDrawCommands() []DrawCommand {
	wr.emitPaneLayout()
	if wr.termWin != nil {
		for _, tile := range wr.termWin.Tiles {
			if tile == nil {
				continue
			}

			term := tile.GetTerm()
			if term == nil || !term.Render() || term.IsFocused() {
				continue
			}

			termSize := term.GetSize()

			wr.enqueueDrawCommand(DrawCommand{
				Op:     DrawOpTileOverlay,
				X:      tile.Left(),
				Y:      tile.Top(),
				Width:  termSize.X + 1,
				Height: termSize.Y,
				Alpha:  51,
			})
			var dividerEdges int32
			for _, other := range wr.termWin.Tiles {
				if other == nil || other == tile || tile.Right() <= other.Left() || other.Right() <= tile.Left() {
					continue
				}
				switch {
				case other.Top() == tile.Bottom()+2:
					dividerEdges |= tileOverlayTopHalf
				case tile.Top() == other.Bottom()+2:
					dividerEdges |= tileOverlayBottomHalf
				}
			}
			if dividerEdges&tileOverlayTopHalf != 0 {
				wr.enqueueDividerTileOverlay(tile, tile.Bottom()+1, tileOverlayTopHalf)
			}
			if dividerEdges&tileOverlayBottomHalf != 0 {
				wr.enqueueDividerTileOverlay(tile, tile.Top()-1, tileOverlayBottomHalf)
			}
		}

	}

	wr.drawSelectionPreview()
	wr.applyMenuHover()

	wr.applyMouseHoverFromLastPosition()

	if len(wr.drawCommands) == 0 {
		return nil
	}

	for _, fn := range wr.fnSchedule {
		fn()
	}
	wr.fnSchedule = []func(){}

	//wr.cmdMu.Lock()
	commands := wr.drawCommands

	wr.drawCommands = nil
	//wr.cmdMu.Unlock()

	return commands
}

func (wr *webkitRender) enqueueDividerTileOverlay(tile types.Tile, row, flags int32) {
	term := tile.GetTerm()
	if term == nil {
		return
	}
	size := term.GetSize()
	if size == nil {
		return
	}
	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpTileOverlay,
		Flags:  flags,
		X:      tile.Left(),
		Y:      row,
		Width:  size.X + 1,
		Height: 1,
		Alpha:  51,
	})
}
