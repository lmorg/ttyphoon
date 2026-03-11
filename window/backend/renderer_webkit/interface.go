package rendererwebkit

import (
	"sync"

	"github.com/lmorg/ttyphoon/types"
)

type webkitRender struct {
	termWin       *types.AppWindowTerms
	glyphSize     *types.XY
	windowCells   *types.XY
	windowTitle   string
	blinkState    bool
	keyboardMode  types.KeyboardMode
	keyModifier   int
	statusBarText string
	cmdMu         sync.Mutex
	drawCommands  []DrawCommand
}

func (wr *webkitRender) Start(termWin *types.AppWindowTerms, _ any) {
	wr.termWin = termWin
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

func (wr *webkitRender) DrawGaugeH(_ types.Tile, _ *types.XY, _ int32, _ int, _ int, _ *types.Colour) {
}

func (wr *webkitRender) DrawGaugeV(_ types.Tile, _ *types.XY, _ int32, _ int, _ int, _ *types.Colour) {
}

func (wr *webkitRender) DrawTable(_ types.Tile, _ *types.XY, _ int32, _ []int32) {}

func (wr *webkitRender) DrawHighlightRect(_ types.Tile, _ *types.XY, _ *types.XY) {}

func (wr *webkitRender) DrawRectWithColour(_ types.Tile, _ *types.XY, _ *types.XY, _ *types.Colour, _ bool) {
}

func (wr *webkitRender) DrawOutputBlockChrome(_ types.Tile, _ int32, _ int32, _ *types.Colour, _ bool) {
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
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpFrame})
}

func (wr *webkitRender) TriggerLazyRedraw() {}

func (wr *webkitRender) TriggerDeallocation(fn func()) {
	if fn != nil {
		fn()
	}
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

func (wr *webkitRender) DisplayMenu(_ string, items []string, _ types.MenuCallbackT, ok types.MenuCallbackT, _ types.MenuCallbackT) {
	if len(items) > 0 && ok != nil {
		ok(0)
	}
}

func (wr *webkitRender) NewContextMenu() types.ContextMenu {
	return &contextMenuStub{}
}

func (wr *webkitRender) AddToContextMenu(_ ...types.MenuItem) {}

func (wr *webkitRender) GetWindowMeta() any {
	return nil
}

func (wr *webkitRender) ResizeWindow(size *types.XY) {
	if size == nil {
		return
	}
	wr.windowCells = size
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
	wr.cmdMu.Lock()
	wr.drawCommands = append(wr.drawCommands, cmd)
	wr.cmdMu.Unlock()
}

func (wr *webkitRender) PopDrawCommands() []DrawCommand {
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpFrame})

	if wr.termWin != nil {
		for i := range wr.termWin.Tiles {
			term := wr.termWin.Tiles[i].GetTerm()
			if term == nil {
				continue
			}
			_ = term.Render()
		}
	}

	wr.cmdMu.Lock()
	commands := wr.drawCommands
	wr.drawCommands = nil
	wr.cmdMu.Unlock()

	if len(commands) == 0 {
		return []DrawCommand{}
	}

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

type contextMenuStub struct {
	items []types.MenuItem
}

func (cms *contextMenuStub) Append(items ...types.MenuItem) {
	cms.items = append(cms.items, items...)
}

func (cms *contextMenuStub) DisplayMenu(_ string) {}

func (cms *contextMenuStub) Options() []string {
	options := make([]string, len(cms.items))
	for i := range cms.items {
		options[i] = cms.items[i].Title
	}
	return options
}

func (cms *contextMenuStub) Icons() []rune {
	icons := make([]rune, len(cms.items))
	for i := range cms.items {
		icons[i] = cms.items[i].Icon
	}
	return icons
}

func (cms *contextMenuStub) Highlight(i int) {
	if i < 0 || i >= len(cms.items) {
		return
	}
	if cms.items[i].Highlight != nil {
		cancel := cms.items[i].Highlight()
		if cancel != nil {
			cancel()
		}
	}
}

func (cms *contextMenuStub) Callback(i int) {
	if i < 0 || i >= len(cms.items) {
		return
	}
	if cms.items[i].Fn != nil {
		cms.items[i].Fn()
	}
}

func (cms *contextMenuStub) Cancel(_ int) {}

func (cms *contextMenuStub) MenuItems() []types.MenuItem {
	return append([]types.MenuItem(nil), cms.items...)
}
