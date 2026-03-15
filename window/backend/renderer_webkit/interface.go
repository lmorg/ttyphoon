package rendererwebkit

import (
	"context"
	"sync"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
)

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
	//fnScheduleM   sync.Mutex
}

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

func (wr *webkitRender) DrawHighlightRect(_ types.Tile, _ *types.XY, _ *types.XY) {}

func (wr *webkitRender) DrawRectWithColour(_ types.Tile, _ *types.XY, _ *types.XY, _ *types.Colour, _ bool) {
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

func (wr *webkitRender) RefreshWindowList() {}

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

func (wr *webkitRender) DisplayNotification(_ types.NotificationType, _ string) {}

func (wr *webkitRender) DisplaySticky(_ types.NotificationType, _ string, cancel func()) types.Notification {
	if cancel == nil {
		cancel = func() {}
	}
	return &notificationStub{cancel: cancel}
}

func (wr *webkitRender) DisplayInputBox(_ string, _ string, ok types.InputBoxCallbackT, _ types.InputBoxCallbackT) {
	if ok != nil {
		ok("")
	}
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

type notificationStub struct {
	message string
	cancel  func()
}

func (ns *notificationStub) SetMessage(message string) {
	ns.message = message
}

func (ns *notificationStub) UpdateCanceller(cancel func()) {
	if cancel == nil {
		ns.cancel = func() {}
		return
	}
	ns.cancel = cancel
}

func (ns *notificationStub) Close() {
	if ns.cancel != nil {
		ns.cancel()
	}
}
