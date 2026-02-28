package virtualterm

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/charset"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	sbh "github.com/lmorg/ttyphoon/utils/scrollback_history"
)

/*
	The virtual terminal emulator

	There is a distinct lack of unit tests in this package. That will change
	over time, however it is worth noting that the hardest part of the problem
	is understanding _what_ correct behaviour should look like, as opposed to
	any logic itself being complex. Therefore this package has has extensive
	manual testing against it via the following CLI applications:
	- vttest: https://invisible-island.net/vttest/vttest.html
	- vim
	- tmux
	- murex: https://murex.rocks
	- bash

	...as well as heavy reliance on documentation, as described in each source
	file.
*/

// Term is the display state of the virtual term
type Term struct {
	tile     types.Tile
	visible  bool
	size     *types.XY
	sgr      *types.Sgr
	renderer types.Renderer
	Pty      types.Pty
	_mutex   sync.Mutex

	screen        *types.Screen
	_normBuf      types.Screen
	_altBuf       types.Screen
	_scrollBuf    types.Screen
	_scrollOffset int
	historyDb     *sbh.ScrollbackHistory

	// smooth scroll
	_ssCounter   int32
	_ssFrequency int32
	_ssLargeBuf  atomic.Int32

	// tab stops
	_tabStops []int32
	_tabWidth int32

	// cursor and scrolling
	_curPos       types.XY
	_originMode   bool // Origin Mode (DECOM), VT100.
	_hideCursor   bool
	_savedCurPos  types.XY
	_scrollRegion *scrollRegionT

	// state
	_vtMode          _stateVtMode
	_insertOrReplace _stateIrmT
	_hasFocus        bool
	_activeElement   types.Element
	_mouseIn         types.Element
	_mouseButtonDown bool
	_rowSource       *types.RowSource
	_blockMeta       *types.BlockMeta
	_blockMetaId     atomic.Int64

	_apcStack uint

	// search
	_searchHighlight  bool
	_searchLastString string
	_searchHlHistory  []*types.Cell
	_searchResults    []searchResult

	// character sets
	_activeCharSet int
	_charSetG      [4]map[rune]rune

	// misc CSI configs
	_windowTitleStack []string
	_noAutoLineWrap   bool // No Auto-Wrap Mode (DECAWM), VT100.

	// cache
	_mousePosRenderer types.FuncMutex
}

type searchResult struct {
	rowId  uint64
	phrase string
}

type _stateVtMode int

const (
	_VT100   = 0
	_VT52    = 1
	_TEK4014 = 2
)

type _stateIrmT int

const (
	_STATE_IRM_REPLACE = 0
	_STATE_IRM_INSERT  = 1
)

func (term *Term) lfRedraw() {
	if term.renderer == nil {
		return
	}

	term._ssCounter++
	if term._ssCounter >= term._ssFrequency {
		term._ssCounter = 0
		term.renderer.TriggerRedraw()
	}
}

// NewTerminal creates a new virtual term
func NewTerminal(tile types.Tile, renderer types.Renderer, size *types.XY, visible bool) *Term {
	term := &Term{
		tile:     tile,
		renderer: renderer,
		size:     size,
		visible:  visible,
	}

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	pwd, err := os.Getwd()
	if err != nil {
		pwd = tile.Pwd()
	}

	term._rowSource = &types.RowSource{
		Host: host,
		Pwd:  pwd,
	}

	term.reset(size)
	tile.SetTerm(term)

	agent.New(renderer, tile)

	return term
}

func (term *Term) Start(pty types.Pty) {
	term.Pty = pty

	go term.Pty.ExecuteShell(term.renderer.TriggerQuit)
	go term.readLoop()
}

func NewRowBlockMeta(term *Term) *types.BlockMeta {
	return &types.BlockMeta{
		Id:        term._blockMetaId.Add(1),
		TimeStart: time.Now(),
	}
}

func (term *Term) reset(size *types.XY) {
	term.size = size
	term.resizePty()
	term._curPos = types.XY{}
	term._blockMeta = NewRowBlockMeta(term)

	term._normBuf = term.makeScreen()
	term._altBuf = term.makeScreen()
	term.eraseScrollBack()
	term.historyDb = sbh.New(term.tile.Id(), func(err error) {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	})

	term._tabWidth = 8
	term.csiResetTabStops()

	term.screen = &term._normBuf
	term.phraseSetToRowPos(_LINEFEED_CURSOR_MOVED)

	term.sgr = types.SGR_DEFAULT.Copy()

	term._charSetG[1] = charset.DecSpecialChar

	term.setJumpScroll()

	if config.Config.Tmux.Enabled {
		term.renderer.SetKeyboardFnMode(types.KeysTmuxClient)
	}
}

func (term *Term) makeScreen() types.Screen {
	screen := make(types.Screen, term.size.Y)
	for i := range screen {
		screen[i] = term.makeRow()
	}
	return screen
}

func (term *Term) makeRow() *types.Row {
	row := &types.Row{
		Id:    nextRowId(),
		Cells: term.makeCells(term.size.X),
		Block: term._blockMeta,
	}

	return row
}

func (term *Term) makeCells(length int32) []*types.Cell {
	cells := make([]*types.Cell, length)
	for i := range cells {
		cells[i] = new(types.Cell)
	}
	return cells
}

func (term *Term) GetSize() *types.XY {
	return term.size
}

func (term *Term) currentCell() *types.Cell {
	pos := term.curPos()

	return (*term.screen)[pos.Y].Cells[pos.X]
}

func (term *Term) previousCell() (*types.Cell, *types.XY) {
	pos := term.curPos()
	pos.X--

	if pos.X < 0 {
		pos.X = term.size.X - 1
		pos.Y--
	} else if pos.X >= term.size.X {
		pos.X = term.size.X - 1
	}

	if pos.Y < 0 {
		pos.Y = 0
	}

	return (*term.screen)[pos.Y].Cells[pos.X], pos
}

func (term *Term) curPos() *types.XY {
	var y int32
	switch {
	case term._curPos.Y < 0:
		y = 0
	case term._curPos.Y >= term.size.Y: // should this be >= or > ??
		y = term.size.Y - 1
	default:
		y = term._curPos.Y
	}

	var x int32
	switch {
	case term._curPos.X < 0:
		x = 0
	case term._curPos.X >= term.size.X:
		x = term.size.X - 1
	default:
		x = term._curPos.X
	}

	return &types.XY{X: x, Y: y}
}

func (term *Term) scrollToRowId(id uint64, offset int) {
	if term.IsAltBuf() {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Cannot jump rows from within the alt buffer")
		return
	}

	term._mutex.Lock()
	defer term._mutex.Unlock()

	for i := range term._normBuf {
		if id == term._normBuf[i].Id {
			term._scrollOffset = 0
			term.updateScrollback()
			return
		}
	}

	for i := range term._scrollBuf {
		if id == term._scrollBuf[i].Id {
			term._scrollOffset = len(term._scrollBuf) - i + offset
			term.updateScrollback()
			return
		}
	}

	term.renderer.DisplayNotification(types.NOTIFY_WARN, "Row not found")
}

type scrollRegionT struct {
	Top    int32
	Bottom int32
}

func (term *Term) Close() {
	term.Pty.Close()
	term.tile.Close()
}

func (term *Term) Reply(b []byte) {
	if term._scrollOffset != 0 && config.Config.Terminal.ScrollbackCloseKeyPress {
		term._scrollOffset = 0
		term.updateScrollback()
	}

	err := term.Pty.Write(b)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Cannot write to PTY: %s", err.Error()))
	}
}

func (term *Term) visibleScreen() types.Screen {
	if term._scrollOffset == 0 {
		return *term.screen
	}

	// render scrollback buffer
	start := len(term._scrollBuf) - term._scrollOffset
	screen := term._scrollBuf[start:]
	if len(screen) < int(term.size.Y) {
		screen = append(screen, term._normBuf[:int(term.size.Y)-term._scrollOffset]...)
	}

	return screen
}

func (term *Term) updateScrollback() {
	if term._scrollOffset > len(term._scrollBuf) {
		term._scrollOffset = len(term._scrollBuf)
	}

	if term._scrollOffset < 0 {
		term._scrollOffset = 0
	}
}

func (term *Term) HasFocus(state bool) {
	term._hasFocus = state
	//term._slowBlinkState = true
	term.renderer.SetBlinkState(true)
}

func (term *Term) MakeVisible(visible bool) {
	term.visible = visible
}

func (term *Term) IsAltBuf() bool {
	return unsafe.Pointer(term.screen) != unsafe.Pointer(&term._normBuf)
}

func (term *Term) GetTermContents() []byte {
	var b []byte

	if term.IsAltBuf() {

		term._mutex.Lock()
		b = []byte(term._altBuf.String())
		term._mutex.Unlock()

	} else {

		term._mutex.Lock()
		b = append([]byte(term._scrollBuf.String()), []byte(term._normBuf.String())...)
		term._mutex.Unlock()

	}

	return b
}

func (term *Term) Host(pos *types.XY) string {
	src := term.visibleScreen()[pos.Y].Source
	if src != nil {
		return src.Host
	}

	return "localhost"
}

func (term *Term) Pwd(pos *types.XY) string {
	src := term.visibleScreen()[pos.Y].Source
	if src != nil {
		return src.Pwd
	}

	return term.tile.Pwd()
}

func (term *Term) RowSrcFromScrollBack(absY int) (src *types.RowSource) {
	if absY < len(term._scrollBuf) {
		src = term._scrollBuf[absY].Source
	} else {
		src = term._normBuf[absY-len(term._scrollBuf)].Source
	}

	if src == nil {
		return &types.RowSource{}
	}

	return src
}

func (term *Term) ConvertRelativeToAbsoluteY(pos *types.XY) int32 {
	return int32(len(term._scrollBuf) + int(pos.Y) - term._scrollOffset)
}

func (term *Term) GetCursorPosition() *types.XY {
	return term.curPos()
}

func (term *Term) GetRowId(y int32) uint64 {
	if y < 0 {
		y = 0
	}
	return (*term.screen)[y].Id
}

func (term *Term) GetSgr() *types.Sgr {
	return term.sgr
}

func (term *Term) GetCellSgr(cell *types.XY) *types.Sgr {
	screen := term.visibleScreen()
	if cell.Y < 0 || int(cell.Y) >= len(screen) {
		return term.sgr
	}
	if cell.X < 0 || int(cell.X) >= len(screen[cell.Y].Cells) {
		return term.sgr
	}

	return screen[cell.Y].Cells[cell.X].Sgr
}

func (term *Term) Tile() types.Tile {
	return term.tile
}
