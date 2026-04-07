//go:build ignore
// +build ignore

package rendersdl

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
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
	//_elementStack []*layer.RenderStackT
	//_overlayStack []*layer.RenderStackT
	contextMenu types.ContextMenu

	// state
	keyboardMode    keyboardModeT
	keyModifier     sdl.Keymod
	keyIgnore       chan bool
	hidden          bool
	_redrawRequired atomic.Bool
	_blinkSlow      atomic.Bool

	// hotkeys
	hkEvent  chan *hotkeyFuncT
	hkToggle bool

	// footer
	footer     int32
	footerText string
	windowTabs *tabListT // only a pointer so we can make it nil'able

	// caching
	cacheBgTexture bgT
}
