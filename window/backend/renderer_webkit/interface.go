package rendererwebkit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

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
	termWin         *types.AppWindowTerms
	tmux            *tmux.Tmux
	app             interface{} // Reference to WApp for accessing methods like ListFiles()
	auxTabsMu       sync.RWMutex
	auxTerminalTabs []types.TerminalPaneTab
	glyphSize       *types.XY
	windowCells     *types.XY
	windowTitle     string
	blinkState      atomic.Bool
	keyboardMode    keyboardModeT
	keyModifier     int
	statusBarText   string
	selection       *selectionState
	//cmdMu         sync.Mutex
	drawCommands   []DrawCommand
	wapp           context.Context
	_redraw        chan struct{}
	fnSchedule     []func()
	contextMenu    types.ContextMenu
	menuMu         sync.Mutex
	menuNextID     int
	menuCallbacks  map[int]menuCallbacks
	menuHoverFn    func()
	menuHoverClear func()
	menuHoverDrawn bool
	notifications  notifyT
	inputBoxes     inputBoxesT
	nextImageID    atomic.Int64
	lastMouseCellX atomic.Int32
	lastMouseCellY atomic.Int32
	lastMouseValid atomic.Bool
	//fnScheduleM   sync.Mutex
}

var (
	highlightBorderColour = &types.Colour{0x31, 0x6d, 0xb0, 0xff}
	highlightFillColour   = &types.Colour{0x1c, 0x3e, 0x64, 0xff}
)

func (wr *webkitRender) GetWindowSizeCells() *types.XY {
	return wr.windowCells
}

func (wr *webkitRender) GetGlyphSize() *types.XY {
	return wr.glyphSize
}

func (wr *webkitRender) SetGlyphSize(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}

	wr.glyphSize = &types.XY{X: width, Y: height}
}

func (wr *webkitRender) GetBlinkState() bool {
	return wr.blinkState.Load()
}

func (wr *webkitRender) SetBlinkState(value bool) {
	wr.blinkState.Store(value)
}

func (wr *webkitRender) DrawTable(_ types.Tile, _ *types.XY, _ int32, _ []int32) {}

func (wr *webkitRender) EmitStyleUpdate() {
	if wr.wapp == nil {
		return
	}

	fontFamily := config.Config.TypeFace.FontName
	if fontFamily == "" {
		fontFamily = "Fira Code"
	}

	style := struct {
		Colours          *coloursPayload `json:"colors"`
		StatusBar        bool            `json:"statusBar"`
		FontFamily       string          `json:"fontFamily"`
		FontSize         int             `json:"fontSize"`
		AdjustCellWidth  int             `json:"adjustCellWidth"`
		AdjustCellHeight int             `json:"adjustCellHeight"`
	}{
		Colours: &coloursPayload{
			Fg:            *types.SGR_DEFAULT.Fg,
			Bg:            *types.SGR_DEFAULT.Bg,
			Black:         *types.SGR_COLOR_BLACK,
			Red:           *types.SGR_COLOR_RED,
			Green:         *types.SGR_COLOR_GREEN,
			Yellow:        *types.SGR_COLOR_YELLOW,
			Blue:          *types.SGR_COLOR_BLUE,
			Magenta:       *types.SGR_COLOR_MAGENTA,
			Cyan:          *types.SGR_COLOR_CYAN,
			White:         *types.SGR_COLOR_WHITE,
			BlackBright:   *types.SGR_COLOR_BLACK_BRIGHT,
			RedBright:     *types.SGR_COLOR_RED_BRIGHT,
			GreenBright:   *types.SGR_COLOR_GREEN_BRIGHT,
			YellowBright:  *types.SGR_COLOR_YELLOW_BRIGHT,
			BlueBright:    *types.SGR_COLOR_BLUE_BRIGHT,
			MagentaBright: *types.SGR_COLOR_MAGENTA_BRIGHT,
			CyanBright:    *types.SGR_COLOR_CYAN_BRIGHT,
			WhiteBright:   *types.SGR_COLOR_WHITE_BRIGHT,
			Selection:     *types.COLOR_SELECTION,
			Link:          *types.SGR_COLOR_BLUE,
			Error:         *types.COLOR_ERROR,
			SearchResult:  *types.COLOR_SEARCH_RESULT,
		},
		StatusBar:        config.Config.Window.StatusBar,
		FontFamily:       fontFamily,
		FontSize:         config.Config.TypeFace.FontSize,
		AdjustCellWidth:  config.Config.TypeFace.AdjustCellWidth,
		AdjustCellHeight: config.Config.TypeFace.AdjustCellHeight,
	}

	runtime.EventsEmit(wr.wapp, "terminalStyleUpdate", style)
}

type coloursPayload struct {
	Fg            types.Colour `json:"fg"`
	Bg            types.Colour `json:"bg"`
	Black         types.Colour `json:"black"`
	Red           types.Colour `json:"red"`
	Green         types.Colour `json:"green"`
	Yellow        types.Colour `json:"yellow"`
	Blue          types.Colour `json:"blue"`
	Magenta       types.Colour `json:"magenta"`
	Cyan          types.Colour `json:"cyan"`
	White         types.Colour `json:"white"`
	BlackBright   types.Colour `json:"blackBright"`
	RedBright     types.Colour `json:"redBright"`
	GreenBright   types.Colour `json:"greenBright"`
	YellowBright  types.Colour `json:"yellowBright"`
	BlueBright    types.Colour `json:"blueBright"`
	MagentaBright types.Colour `json:"magentaBright"`
	CyanBright    types.Colour `json:"cyanBright"`
	WhiteBright   types.Colour `json:"whiteBright"`
	Selection     types.Colour `json:"selection"`
	Link          types.Colour `json:"link"`
	Error         types.Colour `json:"error"`
	SearchResult  types.Colour `json:"searchResult"`
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
	wr.updateNotes()
}

func (wr *webkitRender) updateNotes() {
	if wr.termWin == nil {
		// bit of a hack but this should only happen on application startup
		time.Sleep(500 * time.Millisecond)
	}
	runtime.EventsEmit(wr.wapp, "notesUpdate", wr.termWin.Active.GroupName())
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

func (wr *webkitRender) GetWindowMeta() any {
	return wr.TerminalPaneTabs()
}

func (wr *webkitRender) SetTerminalPaneTabs(tabs []types.TerminalPaneTab) {
	wr.auxTabsMu.Lock()
	wr.auxTerminalTabs = append([]types.TerminalPaneTab(nil), tabs...)
	wr.auxTabsMu.Unlock()
}

func (wr *webkitRender) TerminalPaneTabs() []types.TerminalPaneTab {
	wr.auxTabsMu.RLock()
	defer wr.auxTabsMu.RUnlock()

	if len(wr.auxTerminalTabs) == 0 {
		return nil
	}

	return append([]types.TerminalPaneTab(nil), wr.auxTerminalTabs...)
}

func (wr *webkitRender) ActivateTerminalPaneTab(tabID string) {
	if wr.wapp == nil || tabID == "" {
		return
	}

	runtime.EventsEmit(wr.wapp, "terminalActivateAuxTab", map[string]string{"id": tabID})
}

func (wr *webkitRender) NotesCreateAndOpen(filename, content string) {
	runtime.EventsEmit(wr.wapp, "notesCreateAndOpen", map[string]string{
		"filename": filename,
		"contents": content,
	})
}

func (wr *webkitRender) EmitAIResponseChunk(chunk string) {
	if wr.wapp == nil || chunk == "" {
		return
	}
	runtime.EventsEmit(wr.wapp, "aiResponseStream", chunk)
}

func (wr *webkitRender) DisplayImageFullscreen(dataURL string, sourceWidth, sourceHeight int32) {
	if wr.wapp == nil || dataURL == "" {
		return
	}
	runtime.EventsEmit(wr.wapp, "imageDisplayFullscreen", map[string]any{
		"dataURL":      dataURL,
		"sourceWidth":  sourceWidth,
		"sourceHeight": sourceHeight,
	})
}
func (wr *webkitRender) Close() {}

func (wr *webkitRender) ActiveTile() types.Tile {
	if wr == nil || wr.termWin == nil {
		return nil
	}
	return wr.termWin.Active
}

func (wr *webkitRender) activeTerm() types.Term {
	if wr.termWin == nil {
		return nil
	}

	if wr.termWin.Active != nil && wr.termWin.Active.GetTerm() != nil {
		return wr.termWin.Active.GetTerm()
	}

	for _, tile := range wr.termWin.Tiles {
		if tile != nil && tile.GetTerm() != nil {
			return tile.GetTerm()
		}
	}

	return nil
}
