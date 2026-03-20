package rendererwebkit

import (
	"bytes"
	"fmt"

	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/clipboard"
)

type selectionPreviewMode uint8

const (
	selectionPreviewWrapped selectionPreviewMode = iota
	selectionPreviewSquare
	selectionPreviewLines
	selectionPreviewImage
)

const selectionHintText = "Drag to select, then release to choose a copy action"

type selectionState struct {
	tile        types.Tile
	start       *types.XY
	end         *types.XY
	moved       bool
	released    bool
	button      types.MouseButtonT
	previewMode selectionPreviewMode
	showFill    bool
	showBorder  bool
}

func cloneXY(pos *types.XY) *types.XY {
	if pos == nil {
		return nil
	}

	return &types.XY{X: pos.X, Y: pos.Y}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func normalizeSelection(start, end *types.XY) (left, top, right, bottom int32) {
	if start.X <= end.X {
		left, right = start.X, end.X
	} else {
		left, right = end.X, start.X
	}

	if start.Y <= end.Y {
		top, bottom = start.Y, end.Y
	} else {
		top, bottom = end.Y, start.Y
	}

	return
}

func isSelectionDragged(start, end *types.XY) bool {
	if start == nil || end == nil {
		return false
	}

	dx := abs32(end.X - start.X)
	dy := abs32(end.Y - start.Y)
	return dx > 1 || dy > 1
}

func (wr *webkitRender) beginSelection(tile types.Tile, pos *types.XY, button types.MouseButtonT) {
	if tile == nil || tile.GetTerm() == nil || pos == nil {
		return
	}

	wr.selection = &selectionState{
		tile:        tile,
		start:       cloneXY(pos),
		end:         cloneXY(pos),
		button:      button,
		previewMode: selectionPreviewWrapped,
		showFill:    true,
		showBorder:  false,
	}
	wr.StatusBarText(selectionHintText)
}

func (wr *webkitRender) updateSelection(tile types.Tile, pos *types.XY) {
	if wr.selection == nil || tile == nil || pos == nil {
		return
	}

	if wr.selection.tile == nil || wr.selection.tile.Id() != tile.Id() {
		wr.clearSelectionState()
		return
	}

	if wr.selection.start == nil {
		wr.selection.start = cloneXY(pos)
	}

	wr.selection.end = cloneXY(pos)
	if !wr.selection.moved {
		wr.selection.moved = isSelectionDragged(wr.selection.start, wr.selection.end)
	}
}

func (wr *webkitRender) endSelection(tile types.Tile, pos *types.XY) bool {
	if wr.selection == nil || tile == nil || pos == nil {
		return false
	}

	selection := wr.selection
	if selection.tile == nil || selection.tile.Id() != tile.Id() || selection.tile.GetTerm() == nil {
		wr.clearSelectionState()
		return false
	}

	selection.end = cloneXY(pos)
	selection.released = true
	if !selection.moved {
		selection.moved = isSelectionDragged(selection.start, selection.end)
	}
	if !selection.moved {
		wr.clearSelectionState()
		return false
	}

	webkitClipboardInit.Do(func() { _ = clipboard.Init() })
	wr.StatusBarText("Select copy action")
	wr.showSelectionContextMenu(selection)
	wr.TriggerRedraw()
	return true
}

func (wr *webkitRender) clearSelectionState() {
	wr.selection = nil
	wr.StatusBarText("")
	wr.TriggerRedraw()
}

func (wr *webkitRender) showSelectionContextMenu(selection *selectionState) {
	if selection == nil || selection.tile == nil || selection.tile.GetTerm() == nil {
		wr.clearSelectionState()
		return
	}

	menu := wr.NewContextMenu()
	menu.Append([]types.MenuItem{
		{
			Title: "Copy highlighted text to clipboard",
			Icon:  0xf0c5,
			Highlight: func() func() {
				return wr.setSelectionPreview(selectionPreviewWrapped, true, false)
			},
			Fn: func() {
				term := selection.tile.GetTerm()
				if term == nil {
					wr.clearSelectionState()
					return
				}
				copySelectionTextToClipboard(wr, term.CopyRange(cloneXY(selection.start), cloneXY(selection.end)))
				wr.clearSelectionState()
			},
		},
		{
			Title: "Copy text (rectangular region) to clipboard",
			Icon:  0xf850,
			Highlight: func() func() {
				return wr.setSelectionPreview(selectionPreviewSquare, true, false)
			},
			Fn: func() {
				term := selection.tile.GetTerm()
				if term == nil {
					wr.clearSelectionState()
					return
				}
				left, top, right, bottom := normalizeSelection(selection.start, selection.end)
				copySelectionTextToClipboard(wr, term.CopySquare(&types.XY{X: left, Y: top}, &types.XY{X: right, Y: bottom}))
				wr.clearSelectionState()
			},
		},
		{
			Title: "Copy text (selected lines) to clipboard",
			Icon:  0xf039,
			Highlight: func() func() {
				return wr.setSelectionPreview(selectionPreviewLines, true, false)
			},
			Fn: func() {
				term := selection.tile.GetTerm()
				if term == nil {
					wr.clearSelectionState()
					return
				}
				_, top, _, bottom := normalizeSelection(selection.start, selection.end)
				copySelectionTextToClipboard(wr, term.CopyLines(top, bottom))
				wr.clearSelectionState()
			},
		},
		{
			Title: "Copy image to clipboard",
			Icon:  0xf03e,
			Highlight: func() func() {
				return wr.setSelectionPreview(selectionPreviewImage, false, true)
			},
			Fn: func() {
				left, top, right, bottom := normalizeSelection(selection.start, selection.end)
				if wr.wapp != nil {
					runtime.EventsEmit(wr.wapp, "terminalCopyImageSelection", map[string]any{
						"x":      selection.tile.Left() + left + 1,
						"y":      selection.tile.Top() + top,
						"width":  right - left + 1,
						"height": bottom - top + 1,
					})
				}
				wr.clearSelectionState()
			},
		},
	}...)

	wr.openMenu(
		"Selection action",
		menu.Options(),
		menu.Icons(),
		func(i int) { menu.Highlight(i) },
		func(i int) { menu.Callback(i) },
		func(i int) {
			menu.Cancel(i)
			wr.clearSelectionState()
		},
	)
}

func (wr *webkitRender) setSelectionPreview(mode selectionPreviewMode, showFill, showBorder bool) func() {
	if wr.selection == nil {
		return func() {}
	}

	prevMode := wr.selection.previewMode
	prevFill := wr.selection.showFill
	prevBorder := wr.selection.showBorder

	wr.selection.previewMode = mode
	wr.selection.showFill = showFill
	wr.selection.showBorder = showBorder
	wr.TriggerRedraw()

	return func() {
		if wr.selection == nil {
			return
		}
		wr.selection.previewMode = prevMode
		wr.selection.showFill = prevFill
		wr.selection.showBorder = prevBorder
		wr.TriggerRedraw()
	}
}

func copySelectionTextToClipboard(wr *webkitRender, b []byte) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return
	}

	clipboard.Write(clipboard.FmtText, b)
	lines := bytes.Count(b, []byte{'\n'}) + 1
	wr.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Copied %d line(s) to clipboard", lines))
}

func (wr *webkitRender) drawSelectionPreview() {
	if wr.selection == nil || wr.selection.tile == nil || wr.selection.tile.GetTerm() == nil || wr.selection.start == nil || wr.selection.end == nil {
		return
	}

	termSize := wr.selection.tile.GetTerm().GetSize()
	start := cloneXY(wr.selection.start)
	end := cloneXY(wr.selection.end)

	if start.X < 0 {
		start.X = 0
	}
	if end.X < 0 {
		end.X = 0
	}

	if start.X >= termSize.X {
		start.X = termSize.X - 1
	}
	if end.X >= termSize.X {
		end.X = termSize.X - 1
	}

	if start.Y < 0 {
		start.Y = 0
	}
	if end.Y < 0 {
		end.Y = 0
	}

	if start.Y >= termSize.Y {
		start.Y = termSize.Y - 1
	}
	if end.Y >= termSize.Y {
		end.Y = termSize.Y - 1
	}

	switch wr.selection.previewMode {
	case selectionPreviewSquare:
		drawSquareSelectionPreview(wr, wr.selection.tile, start, end, wr.selection.showFill, wr.selection.showBorder)
	case selectionPreviewLines:
		drawLineSelectionPreview(wr, wr.selection.tile, start, end, termSize.X, wr.selection.showFill, wr.selection.showBorder)
	case selectionPreviewImage:
		drawSquareSelectionPreview(wr, wr.selection.tile, start, end, false, wr.selection.showBorder)
	case selectionPreviewWrapped:
		fallthrough
	default:
		drawWrappedSelectionPreview(wr, wr.selection.tile, start, end, termSize.X, wr.selection.showFill, wr.selection.showBorder)
	}

	if !wr.selection.released {
		// While dragging, always show the image-copy rectangular bounds as an extra border overlay.
		drawSquareSelectionPreview(wr, wr.selection.tile, start, end, false, true)
	}
}

func drawSquareSelectionPreview(wr *webkitRender, tile types.Tile, start, end *types.XY, showFill, showBorder bool) {
	left, top, right, bottom := normalizeSelection(start, end)
	width := right - left + 1
	height := bottom - top + 1

	if showFill {
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: left, Y: top},
			&types.XY{X: width, Y: height},
			types.COLOR_SELECTION,
			false,
		)
	}

	if showBorder {
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: left, Y: top},
			&types.XY{X: width, Y: height},
		)
	}
}

func drawLineSelectionPreview(wr *webkitRender, tile types.Tile, start, end *types.XY, width int32, showFill, showBorder bool) {
	_, top, _, bottom := normalizeSelection(start, end)
	height := bottom - top + 1

	if showFill {
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: 0, Y: top},
			&types.XY{X: width, Y: height},
			types.COLOR_SELECTION,
			false,
		)
	}

	if showBorder {
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: 0, Y: top},
			&types.XY{X: width, Y: height},
		)
	}
}

func drawWrappedSelectionPreview(wr *webkitRender, tile types.Tile, start, end *types.XY, width int32, showFill, showBorder bool) {
	if showFill {
		drawWrappedRangeFill(wr, tile, start, end, width)
	}

	if !showBorder {
		return
	}

	if start.Y == end.Y {
		left, _, right, _ := normalizeSelection(start, end)
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: left, Y: start.Y},
			&types.XY{X: right - left + 1, Y: 1},
		)
		return
	}

	if end.Y > start.Y {
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: start.X, Y: start.Y},
			&types.XY{X: width - start.X, Y: 1},
		)
		if end.Y-start.Y > 1 {
			wr.DrawHighlightRect(
				tile,
				&types.XY{X: 0, Y: start.Y + 1},
				&types.XY{X: width, Y: end.Y - start.Y - 1},
			)
		}
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: 0, Y: end.Y},
			&types.XY{X: end.X + 1, Y: 1},
		)
		return
	}

	wr.DrawHighlightRect(
		tile,
		&types.XY{X: end.X, Y: end.Y},
		&types.XY{X: width - end.X, Y: 1},
	)
	if start.Y-end.Y > 1 {
		wr.DrawHighlightRect(
			tile,
			&types.XY{X: 0, Y: end.Y + 1},
			&types.XY{X: width, Y: start.Y - end.Y - 1},
		)
	}
	wr.DrawHighlightRect(
		tile,
		&types.XY{X: 0, Y: start.Y},
		&types.XY{X: start.X + 1, Y: 1},
	)
}

func drawWrappedRangeFill(wr *webkitRender, tile types.Tile, start, end *types.XY, width int32) {
	if start.Y == end.Y {
		left, _, right, _ := normalizeSelection(start, end)
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: left, Y: start.Y},
			&types.XY{X: right - left + 1, Y: 1},
			types.COLOR_SELECTION,
			false,
		)
		return
	}

	if end.Y > start.Y {
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: start.X, Y: start.Y},
			&types.XY{X: width - start.X, Y: 1},
			types.COLOR_SELECTION,
			false,
		)
		if end.Y-start.Y > 1 {
			wr.DrawRectWithColour(
				tile,
				&types.XY{X: 0, Y: start.Y + 1},
				&types.XY{X: width, Y: end.Y - start.Y - 1},
				types.COLOR_SELECTION,
				false,
			)
		}
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: 0, Y: end.Y},
			&types.XY{X: end.X + 1, Y: 1},
			types.COLOR_SELECTION,
			false,
		)
		return
	}

	wr.DrawRectWithColour(
		tile,
		&types.XY{X: end.X, Y: end.Y},
		&types.XY{X: width - end.X, Y: 1},
		types.COLOR_SELECTION,
		false,
	)
	if start.Y-end.Y > 1 {
		wr.DrawRectWithColour(
			tile,
			&types.XY{X: 0, Y: end.Y + 1},
			&types.XY{X: width, Y: start.Y - end.Y - 1},
			types.COLOR_SELECTION,
			false,
		)
	}
	wr.DrawRectWithColour(
		tile,
		&types.XY{X: 0, Y: start.Y},
		&types.XY{X: start.X + 1, Y: 1},
		types.COLOR_SELECTION,
		false,
	)
}
