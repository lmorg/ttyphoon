package rendersdl

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/layer"
	"github.com/lmorg/ttyphoon/window/backend/typeface"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/hotkey"
)

const ( // based at compile time
	_PANE_LEFT_MARGIN_OUTER = int32(1)
	_PANE_TOP_MARGIN        = int32(5)
	_WIDGET_INNER_MARGIN    = int32(5)
	_WIDGET_OUTER_MARGIN    = int32(10)
)

var ( // defined at runtime based on font size
	_PANE_BLOCK_HIGHLIGHT = int32(0)
	_PANE_BLOCK_FOLDED    = int32(0)
	_PANE_LEFT_MARGIN     = int32(0)
)

type sdlRender struct {
	window      *sdl.Window
	renderer    *sdl.Renderer
	fontCache   *fontCacheT
	glyphSize   *types.XY
	tmux        *tmux.Tmux
	limiter     sync.Mutex
	winCellSize *types.XY
	winTile     types.Tile

	// title
	title       string
	updateTitle int32

	// audio
	bell *mix.Music

	// events
	_quit         chan bool
	_redraw       chan bool
	_resize       chan *types.XY
	_redrawTimer  <-chan time.Time
	_deallocStack chan func()

	// notifications
	notifications  notifyT
	notifyIcon     map[int]types.Image
	notifyIconSize *types.XY

	// widgets
	termWin          *types.AppWindowTerms
	termWidget       *termWidgetT
	highlighter      *highlightWidgetT
	highlightAi      bool
	inputBox         *inputBoxWidgetT
	_cancelWInputBox func()
	menu             *menuWidgetT

	// render function stacks
	_elementStack []*layer.RenderStackT
	_overlayStack []*layer.RenderStackT
	contextMenu   types.ContextMenu

	// state
	keyboardMode    keyboardModeT
	keyModifier     sdl.Keymod
	keyIgnore       chan bool
	hidden          bool
	_redrawRequired atomic.Bool
	_blinkSlow      atomic.Bool

	// hotkey
	hk       *hotkey.Hotkey
	hkToggle bool

	// footer
	footer     int32
	footerText string
	windowTabs *tabListT // only a pointer so we can make it nil'able

	// caching
	cacheBgTexture bgT
}

type tabListT struct {
	tabs       *[]types.Tab
	boundaries []int32
	offset     *types.XY
	active     int
	mouseOver  int
	cells      []*types.Cell
	last       int
}

type keyboardModeT struct {
	keyboardMode int32
}

func (km *keyboardModeT) Set(mode types.KeyboardMode) {
	if config.Config.Tmux.Enabled {
		mode = types.KeysTmuxClient // override keyboard mode if in tmux control mode
	}
	atomic.StoreInt32(&km.keyboardMode, int32(mode))
}
func (km *keyboardModeT) Get() types.KeyboardMode {
	return types.KeyboardMode(atomic.LoadInt32(&km.keyboardMode))
}

func (sr *sdlRender) SetKeyboardFnMode(code types.KeyboardMode) {
	sr.keyboardMode.Set(code)
}

func (sr *sdlRender) TriggerQuit()  { go sr._triggerQuit() }
func (sr *sdlRender) _triggerQuit() { sr._quit <- true }

func (sr *sdlRender) TriggerDeallocation(fn func()) {
	go func() { sr._deallocStack <- fn }()
}

func (sr *sdlRender) TriggerRedraw() { go sr._triggerRedraw() }
func (sr *sdlRender) _triggerRedraw() {
	if sr.termWin == nil || !sr.limiter.TryLock() {
		//if sr.termWin == nil {
		return
	}

	/*if sr.renderLock.Swap(true) {
		return
	}*/

	sr._redraw <- true
}

func (sr *sdlRender) Close() {
	typeface.Close()
	sr.window.Destroy()

	if sr.bell != nil {
		sr.bell.Free()
		mix.CloseAudio()
		mix.Quit()
	}

	sdl.Quit()
}

func (sr *sdlRender) GetGlyphSize() *types.XY {
	return sr.glyphSize
}

type winTileT struct {
	sr *sdlRender
}

func (t *winTileT) Left() int32   { return 0 }
func (t *winTileT) Top() int32    { return 0 }
func (t *winTileT) Right() int32  { return t.sr.winCellSize.X }
func (t *winTileT) Bottom() int32 { return t.sr.winCellSize.Y }

func (t *winTileT) Name() string      { return app.Title }
func (t *winTileT) SetName(string)    {}
func (t *winTileT) GroupName() string { return app.Title }
func (t *winTileT) Id() string        { return "" }

func (t *winTileT) AtBottom() bool { return true }

func (t *winTileT) GetTerm() types.Term { return nil }
func (t *winTileT) SetTerm(types.Term)  { panic("writing to a readonly interface") }

func (t *winTileT) Pwd() string {
	pwd, _ := os.Getwd()
	return pwd
}

func (t *winTileT) Close() {}
