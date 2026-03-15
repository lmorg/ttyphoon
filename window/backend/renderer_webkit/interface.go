package rendererwebkit

import (
	"context"
	"sync"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type terminalTab struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Index  int    `json:"index"`
	Active bool   `json:"active"`
}

type webkitRender struct {
	termWin       *types.AppWindowTerms
	tmux          *tmux.Tmux
	glyphSize     *types.XY
	windowCells   *types.XY
	windowTitle   string
	blinkState    bool
	keyboardMode  types.KeyboardMode
	keyModifier   int
	statusBarText string
	//cmdMu         sync.Mutex
	drawCommands  []DrawCommand
	wapp          context.Context
	_redraw       chan struct{}
	fnSchedule    []func()
	contextMenu   types.ContextMenu
	menuMu        sync.Mutex
	menuNextID    int
	menuCallbacks map[int]menuCallbacks
	notifications notifyT
	inputBoxes    inputBoxesT
	//fnScheduleM   sync.Mutex
}

var (
	highlightBorderColour = &types.Colour{0x31, 0x6d, 0xb0, 0xff}
	highlightFillColour   = &types.Colour{0x1c, 0x3e, 0x64, 0xff}
)

func (wr *webkitRender) ShowAndFocusWindow() {}

func (wr *webkitRender) GetWindowSizeCells() *types.XY {
	return wr.windowCells
}

func (wr *webkitRender) GetGlyphSize() *types.XY {
	return wr.glyphSize
}

func (wr *webkitRender) GetBlinkState() bool {
	return wr.blinkState
}

func (wr *webkitRender) SetBlinkState(value bool) {
	wr.blinkState = value
}

func (wr *webkitRender) DrawGaugeH(tile types.Tile, topLeft *types.XY, width int32, value, max int, c *types.Colour) {
	if tile == nil || topLeft == nil || c == nil || width <= 0 || max <= 0 {
		return
	}

	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:    DrawOpGaugeH,
		X:     topLeft.X + tile.Left() + 1,
		Y:     topLeft.Y + tile.Top(),
		Width: width,
		Value: int32(value),
		Max:   int32(max),
		Fg:    c,
	})
}

func (wr *webkitRender) DrawGaugeV(tile types.Tile, topLeft *types.XY, height int32, value, max int, c *types.Colour) {
	if tile == nil || topLeft == nil || c == nil || height <= 0 || max <= 0 {
		return
	}

	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpGaugeV,
		X:      topLeft.X + tile.Left(),
		Y:      topLeft.Y + tile.Top(),
		Height: height,
		Value:  int32(value),
		Max:    int32(max),
		Fg:     c,
	})
}

func (wr *webkitRender) DrawTable(_ types.Tile, _ *types.XY, _ int32, _ []int32) {}

func (wr *webkitRender) DrawHighlightRect(tile types.Tile, _topLeftCell, bottomRightCell *types.XY) {
	if tile == nil || _topLeftCell == nil || bottomRightCell == nil {
		return
	}
	if bottomRightCell.X <= 0 || bottomRightCell.Y <= 0 {
		return
	}

	topLeftCell := &types.XY{
		X: _topLeftCell.X + tile.Left() + 1,
		Y: _topLeftCell.Y + tile.Top(),
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpHighlight,
		X:      topLeftCell.X,
		Y:      topLeftCell.Y,
		Width:  bottomRightCell.X,
		Height: bottomRightCell.Y,
		Fg:     highlightBorderColour,
		Bg:     highlightFillColour,
	})
}

func (wr *webkitRender) DrawRectWithColour(tile types.Tile, _topLeftCell, _bottomRightCell *types.XY, colour *types.Colour, incLeftMargin bool) {
	if tile == nil || _topLeftCell == nil || _bottomRightCell == nil || colour == nil {
		return
	}

	topLeftCell := &types.XY{
		X: _topLeftCell.X,
		Y: max(_topLeftCell.Y, 0),
	}

	bottomRightCell := &types.XY{
		X: _bottomRightCell.X,
		Y: _bottomRightCell.Y + min(_topLeftCell.Y, 0),
	}

	if tile.GetTerm() != nil && bottomRightCell.Y+topLeftCell.Y > tile.GetTerm().GetSize().Y {
		bottomRightCell.Y = tile.GetTerm().GetSize().Y - topLeftCell.Y
	}

	if bottomRightCell.X <= 0 || bottomRightCell.Y <= 0 {
		return
	}

	leftOffset := int32(1)
	if incLeftMargin {
		leftOffset = 0
	}

	wr.enqueueDrawCommand(DrawCommand{
		Op:     DrawOpRectColour,
		X:      topLeftCell.X + tile.Left() + leftOffset,
		Y:      topLeftCell.Y + tile.Top(),
		Width:  bottomRightCell.X,
		Height: bottomRightCell.Y,
		Fg:     colour,
		Bg:     colour,
	})
}

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

func (wr *webkitRender) GetWindowTitle() string {
	return wr.windowTitle
}

func (wr *webkitRender) SetWindowTitle(title string) {
	wr.windowTitle = title
}

func (wr *webkitRender) StatusBarText(text string) {
	wr.statusBarText = text
}

func (wr *webkitRender) tmuxTabs() []terminalTab {
	if wr.termWin == nil {
		return nil
	}

	tabs := make([]terminalTab, 0, len(wr.termWin.Tabs))
	for i := range wr.termWin.Tabs {
		tab := wr.termWin.Tabs[i]
		tabs = append(tabs, terminalTab{
			ID:     tab.Id(),
			Name:   tab.Name(),
			Index:  tab.Index(),
			Active: tab.Active(),
		})
	}

	return tabs
}

func (wr *webkitRender) RefreshWindowList() {
	if wr.tmux != nil {
		wr.termWin = wr.tmux.GetTermTiles()
	}

	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalTabs", wr.tmuxTabs())
	}

	wr.TriggerRedraw()
}

func (wr *webkitRender) GetWindowTabs() []terminalTab {
	if wr.tmux != nil {
		wr.termWin = wr.tmux.GetTermTiles()
	}

	return wr.tmuxTabs()
}

func (wr *webkitRender) SelectWindow(windowID string) {
	if windowID == "" || wr.tmux == nil {
		return
	}

	if wr.windowCells == nil {
		wr.windowCells = &types.XY{X: 120, Y: 40}
	}

	if err := wr.tmux.SelectAndResizeWindow(windowID, wr.windowCells); err != nil {
		return
	}

	wr.RefreshWindowList()
}

func (wr *webkitRender) Bell() {}

func (wr *webkitRender) TriggerRedraw() {
	select {
	case wr._redraw <- struct{}{}:
	default:
	}
}

func (wr *webkitRender) TriggerLazyRedraw() {
	wr.TriggerRedraw()
}

func (wr *webkitRender) TriggerDeallocation(fn func()) {
	//wr.fnScheduleM.Lock()
	wr.fnSchedule = append(wr.fnSchedule, fn)
	//wr.fnScheduleM.Unlock()
}

func (wr *webkitRender) TriggerQuit() {}

func (wr *webkitRender) NewElement(_ types.Tile, _ types.ElementID) types.Element {
	return &elementStub{}
}



func (wr *webkitRender) GetWindowMeta() any {
	return nil
}

func (wr *webkitRender) ResizeWindow(size *types.XY) {
	if size == nil {
		return
	}
	wr.windowCells = size
}

func (wr *webkitRender) WindowResized(cols, rows int32) {
	size := &types.XY{X: cols, Y: rows}
	wr.windowCells = size

	if wr.tmux != nil {
		_ = wr.tmux.RefreshClient(size)
		_ = wr.tmux.SelectAndResizeWindow(wr.tmux.ActiveWindow().Id(), size)
		go wr.RefreshWindowList()
		return
	}

	if !config.Config.Tmux.Enabled && wr.termWin != nil && wr.termWin.Active != nil {
		term := wr.termWin.Active.GetTerm()
		if term != nil {
			term.Resize(size)
		}
	}
}

func (wr *webkitRender) SetKeyboardFnMode(mode types.KeyboardMode) {
	wr.keyboardMode = mode
}

func (wr *webkitRender) GetKeyboardModifier() int {
	return wr.keyModifier
}

func (wr *webkitRender) NotesCreateAndOpen(_ string, _ string) {}

func (wr *webkitRender) Close() {}

func (wr *webkitRender) enqueueDrawCommand(cmd DrawCommand) {
	//wr.cmdMu.Lock()
	wr.drawCommands = append(wr.drawCommands, cmd)
	//wr.cmdMu.Unlock()
}

func (wr *webkitRender) PopDrawCommands() []DrawCommand {
	//wr.enqueueDrawCommand(DrawCommand{Op: DrawOpFrame})
	//wr.drawCommands = []DrawCommand{{Op: DrawOpFrame}}
	if wr.termWin != nil {
		for i := range wr.termWin.Tiles {
			term := wr.termWin.Tiles[i].GetTerm()
			if term == nil {
				continue
			}
			_ = term.Render()
		}

		wr.enqueueInactiveTileOverlays()
	}

	if len(wr.drawCommands) == 0 {
		return nil
	}

	for _, fn := range wr.fnSchedule {
		fn()
	}
	wr.fnSchedule = []func(){}

	//wr.cmdMu.Lock()
	commands := append([]DrawCommand{{Op: DrawOpFrame}}, wr.drawCommands...)
	wr.drawCommands = nil
	//wr.cmdMu.Unlock()

	//if len(commands) == 0 {
	//	return []DrawCommand{}
	//}

	return commands
}

func (wr *webkitRender) enqueueInactiveTileOverlays() {
	if wr.termWin == nil || len(wr.termWin.Tiles) <= 1 || wr.termWin.Active == nil {
		return
	}

	for i := range wr.termWin.Tiles {
		tile := wr.termWin.Tiles[i]
		if tile == nil || tile.GetTerm() == nil {
			continue
		}

		if tile.Id() == wr.termWin.Active.Id() {
			continue
		}

		termSize := tile.GetTerm().GetSize()
		if termSize == nil || termSize.X <= 0 || termSize.Y <= 0 {
			continue
		}

		wr.enqueueDrawCommand(DrawCommand{
			Op:     DrawOpTileOverlay,
			X:      tile.Left(),
			Y:      tile.Top(),
			Width:  termSize.X + 1,
			Height: termSize.Y,
			Alpha:  51,
		})
	}
}

type elementStub struct{}

func (es *elementStub) Generate(_ *types.ApcSlice) error { return nil }
func (es *elementStub) Write(_ rune) error               { return nil }
func (es *elementStub) Rune(_ *types.XY) rune            { return 0 }
func (es *elementStub) Size() *types.XY                  { return &types.XY{} }
func (es *elementStub) Draw(_ *types.XY)                 {}
func (es *elementStub) MouseClick(_ *types.XY, _ types.MouseButtonT, _ uint8, _ types.ButtonStateT, _ types.EventIgnoredCallback) {
}
func (es *elementStub) MouseWheel(_ *types.XY, _ *types.XY, _ types.EventIgnoredCallback) {}
func (es *elementStub) MouseMotion(_ *types.XY, _ *types.XY, _ types.EventIgnoredCallback) {
}
func (es *elementStub) MouseHover(_ *types.XY, _ *types.XY) func() { return func() {} }
func (es *elementStub) MouseOut()                                  {}
