package rendersdl

import (
	"sync"
	"sync/atomic"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/tmux"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/lmorg/mxtty/window/backend/typeface"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
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

	// deprecated
	font *ttf.Font

	// title
	title       string
	updateTitle int32

	// audio
	bell *mix.Music

	// events
	_quit   chan bool
	_redraw chan bool
	_resize chan *types.XY

	// notifications
	notifications  notifyT
	notifyIcon     map[int]types.Image
	notifyIconSize *types.XY

	// widgets
	termWin     *types.TermWindow
	termWidget  *termWidgetT
	highlighter *highlightWidgetT
	inputBox    *inputBoxWidgetT
	menu        *menuWidgetT

	// render function stacks
	_elementStack []*layer.RenderStackT
	_overlayStack []*layer.RenderStackT
	contextMenu   contextMenuT

	// state
	keyboardMode keyboardModeT
	keyModifier  sdl.Keymod
	keyIgnore    chan bool
	hidden       bool

	// hotkey
	hk       *hotkey.Hotkey
	hkToggle bool

	// footer
	footer     int32
	footerText string
	windowTabs *tabListT

	// caching
	cacheBgTexture *sdl.Texture
}

type tabListT struct {
	windows    []*tmux.WindowT
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

func (sr *sdlRender) TriggerRedraw() { go sr._triggerRedraw() }
func (sr *sdlRender) _triggerRedraw() {
	if sr.termWin != nil && sr.limiter.TryLock() {
		sr._redraw <- true
	}
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
