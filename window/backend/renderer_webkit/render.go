package rendererwebkit

func (wr *webkitRender) enqueueDrawCommand(cmd DrawCommand) {
	//wr.cmdMu.Lock()
	wr.drawCommands = append(wr.drawCommands, cmd)
	//wr.cmdMu.Unlock()
}

func (wr *webkitRender) PopDrawCommands() []DrawCommand {
	for _, tile := range wr.termWin.Tiles {
		if !tile.GetTerm().Render() || tile.GetTerm().IsFocused() {
			continue
		}

		termSize := tile.GetTerm().GetSize()

		wr.enqueueDrawCommand(DrawCommand{
			Op:     DrawOpTileOverlay,
			X:      tile.Left(),
			Y:      tile.Top(),
			Width:  termSize.X + 1,
			Height: termSize.Y,
			Alpha:  51,
		})
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
