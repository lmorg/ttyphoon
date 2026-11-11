package rendererwebkit

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

type terminalTab struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Index  int    `json:"index"`
	Active bool   `json:"active"`
}

type terminalTabsPayload struct {
	Tabs      []terminalTab `json:"tabs"`
	TileCount int           `json:"tileCount"`
}

type TerminalPane struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	X                 int32  `json:"x"`
	Y                 int32  `json:"y"`
	Width             int32  `json:"width"`
	Height            int32  `json:"height"`
	CanClose          bool   `json:"canClose"`
	Finished          bool   `json:"finished"`
	Zoomed            bool   `json:"zoomed"`
	AutoCloseAt       int64  `json:"autoCloseAt,omitempty"`
	AutoCloseDuration int64  `json:"autoCloseDuration,omitempty"`
}

type webkitRender struct {
	termWin         *types.AppWindowTerms
	paneLayoutMu    sync.Mutex
	paneLayout      []TerminalPane
	tmux            *tmux.Tmux
	app             interface{} // Reference to WApp for accessing methods like ListFiles()
	auxTabsMu       sync.RWMutex
	notesContextMu  sync.Mutex
	notesContext    notes.FileContext
	notesResolved   notes.FileContext
	notesPwd        string
	notesGroup      string
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
	renderPaused   atomic.Bool
	inFlightRedraw atomic.Uint64
	redrawAck      atomic.Uint64
	nextRedrawID   atomic.Uint64
	redrawDeferred atomic.Bool
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
	commandPalette *contextMenuStub
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

func (wr *webkitRender) EmitStyleUpdate() {
	if wr.wapp == nil {
		return
	}

	fontFamily := config.Config.TypeFace.FontName
	if fontFamily == "" {
		fontFamily = types.DefaultMono
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
			Accent:        *types.SGR_COLOR_ACCENT,
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
	Accent        types.Colour `json:"accent"`
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
		tabs := wr.tmuxTabs()
		tileCount := 0
		if wr.termWin != nil && wr.termWin.Tiles != nil {
			tileCount = len(wr.termWin.Tiles)
		}
		runtime.EventsEmit(wr.wapp, "terminalTabs", terminalTabsPayload{
			Tabs:      tabs,
			TileCount: tileCount,
		})
	}

	wr.TriggerRedraw()
	pane, err := wr.tmux.ActivePane()
	if err == nil {
		wr.setWindowTitleFromTile(pane)
	}
	wr.RefreshNotes()
}

// setWindowTitleFromTile falls back to the tmux window name for panes which
// have never reported a title.
func (wr *webkitRender) setWindowTitleFromTile(tile types.Tile) {
	if tile == nil {
		return
	}

	if title := tile.Name(); title != "" {
		wr.SetWindowTitle(title)
	} else {
		wr.SetWindowTitle(tile.GroupName())
	}
}

func (wr *webkitRender) GetWindowTabs() []terminalTab {
	if wr.tmux != nil {
		wr.termWin = wr.tmux.GetTermTiles()
	}

	return wr.tmuxTabs()
}

func (wr *webkitRender) GetPaneLayout() []TerminalPane {
	panes := make([]TerminalPane, 0)
	termWin := wr.termWin
	if termWin == nil {
		return panes
	}
	for _, tile := range termWin.Tiles {
		if tile == nil {
			continue
		}
		term := tile.GetTerm()
		if term == nil {
			continue
		}
		size := term.GetSize()
		if size == nil || size.X <= 0 || size.Y <= 0 {
			continue
		}
		pane := TerminalPane{ID: tile.Id(), Name: tile.Name(), X: tile.Left(), Y: tile.Top(), Width: size.X, Height: size.Y}
		if tmuxPane, ok := tile.(*tmux.PaneT); ok {
			pane.CanClose = true
			pane.Finished = tmuxPane.CommandFinished()
			pane.Zoomed = tmuxPane.IsZoomed()
			pane.AutoCloseAt = tmuxPane.AutoCloseAt()
			pane.AutoCloseDuration = config.PaneAutoCloseDelay().Milliseconds()
		}
		panes = append(panes, pane)
	}
	slices.SortFunc(panes, func(left, right TerminalPane) int { return strings.Compare(left.ID, right.ID) })
	return panes
}

func (wr *webkitRender) emitPaneLayout() {
	if wr.wapp == nil {
		return
	}
	panes := wr.GetPaneLayout()
	wr.paneLayoutMu.Lock()
	defer wr.paneLayoutMu.Unlock()
	if slices.Equal(wr.paneLayout, panes) {
		return
	}
	wr.paneLayout = panes
	runtime.EventsEmit(wr.wapp, "terminalPanes", panes)
}

func (wr *webkitRender) CancelPaneAutoClose(paneID string) error {
	if wr.tmux == nil {
		return fmt.Errorf("tmux not available")
	}
	return wr.tmux.CancelPaneAutoClose(paneID)
}

func (wr *webkitRender) CloseCompletedPane(paneID string) error {
	if wr.tmux == nil {
		return fmt.Errorf("tmux not available")
	}
	if err := wr.tmux.CloseCompletedPane(paneID); err != nil {
		return err
	}
	wr.RefreshWindowList()
	return nil
}

func (wr *webkitRender) ConfirmKillPane(paneID, paneName string) error {
	if wr.tmux == nil {
		return fmt.Errorf("tmux not available")
	}
	finished, err := wr.tmux.PaneCommandFinished(paneID)
	if err != nil {
		return err
	}
	if finished {
		return wr.CloseCompletedPane(paneID)
	}

	wr.DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           "Confirm pane close",
			Message:         fmt.Sprintf("Are you sure you wish to kill the active pane?\n\nID: %s\nName: %s", paneID, paneName),
			Confirmation:    true,
			ConfirmLabel:    "Kill",
			CancelLabel:     "Cancel",
			ConfirmIsDanger: true,
		},
		OkFunc: func(*types.InputBoxCallbackResultT) {
			if wr.tmux == nil {
				return
			}
			if err := wr.tmux.KillPane(paneID); err != nil {
				wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			}
		},
	})
	return nil
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

func (wr *webkitRender) ZoomPane(paneID string) error {
	if wr.tmux == nil {
		return fmt.Errorf("tmux not available")
	}
	if paneID == "" {
		return fmt.Errorf("pane ID is required")
	}
	if err := wr.tmux.ZoomPane(paneID); err != nil {
		return err
	}

	// ZoomPane blocks until tmux has applied the (un)zoom, so re-read the tile
	// layout now to pick up the new pane geometry.
	wr.RefreshWindowList()
	return nil
}

func (wr *webkitRender) ZoomPaneAndMaximize(paneID string) error {
	if err := wr.ZoomPane(paneID); err != nil {
		return err
	}
	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalFullsizeToggle")
	}
	return nil
}

func (wr *webkitRender) GetActivePaneId() string {
	if wr.tmux == nil {
		return ""
	}

	pane, err := wr.tmux.ActivePane()
	if err != nil || pane == nil {
		return ""
	}

	return pane.Id()
}

func (wr *webkitRender) Bell() {
	if wr == nil {
		return
	}

	if config.Config.Window.BellVisualNotification {
		wr.DisplayNotification(types.NOTIFY_INFO, "DING! System bell received")
	}

	if !config.Config.Window.BellPlayAudio || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "terminalBell", map[string]any{
		"sound": "ding",
	})
}

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

func (wr *webkitRender) ClonePane(command string, callback func(output []byte, exitCode int, err error)) error {
	if wr == nil || wr.tmux == nil {
		return fmt.Errorf("terminal is unavailable")
	}
	return wr.tmux.ClonePane(command, callback)
}

func (wr *webkitRender) ClonePaneAtCurrentDirectory() error {
	if wr == nil || wr.tmux == nil {
		return fmt.Errorf("terminal is unavailable")
	}
	return wr.tmux.ClonePaneAtCurrentDirectory()
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

func (wr *webkitRender) GetWindowContext() context.Context {
	return wr.wapp
}

func (wr *webkitRender) DisplayMarkdownModel(id, markdownContent string) {
	runtime.EventsEmit(wr.wapp, "showMarkdownModal", map[string]string{
		"id":       id,
		"markdown": markdownContent,
	})
}

func (wr *webkitRender) ShowTooltip(text string) {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "tooltipShow", map[string]any{
		"text": text,
	})
}

func (wr *webkitRender) CloseTooltip() {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "tooltipClose")
}
