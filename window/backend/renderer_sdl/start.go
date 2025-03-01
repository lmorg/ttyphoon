package rendersdl

import (
	"log"

	"github.com/lmorg/mxtty/app"
	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/tmux"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/typeface"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/clipboard"
)

/*
	Reference documentation used:
	- https://github.com/veandco/go-sdl2-examples/tree/master/examples
*/

var (
	width  int32 = 1024
	height int32 = 768
	X      int32 = sdl.WINDOWPOS_UNDEFINED
	Y      int32 = sdl.WINDOWPOS_UNDEFINED
)

func init() {
	err := sdl.Init(sdl.INIT_VIDEO)
	if err != nil {
		panic(err.Error())
	}
}

func Initialise() (types.Renderer, *types.XY) {
	rect, err := sdl.GetDisplayUsableBounds(0)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	} else {
		X = rect.W / 2
		Y = 0
		width = rect.W / 2
		height = rect.H
	}

	sr := new(sdlRender)
	err = sr.createWindow(app.Title)
	if err != nil {
		panic(err.Error())
	}

	if config.Config.Window.StatusBar {
		sr.footer++
	}
	if config.Config.Tmux.Enabled {
		sr.footer++
	}

	sr._quit = make(chan bool)
	sr._redraw = make(chan bool)
	sr._resize = make(chan *types.XY)
	sr.keyIgnore = make(chan bool)

	sr.font = typeface.Deprecated_GetFont()
	sr.glyphSize = typeface.GetSize()
	_PANE_BLOCK_HIGHLIGHT = sr.glyphSize.X / 2
	_PANE_BLOCK_FOLDED = sr.glyphSize.X
	_PANE_LEFT_MARGIN = _PANE_LEFT_MARGIN_OUTER + _PANE_BLOCK_FOLDED

	sr.window.SetMinimumSize(
		(40*sr.glyphSize.X)+(_PANE_LEFT_MARGIN),
		(10*sr.glyphSize.Y)+(_PANE_TOP_MARGIN))

	err = clipboard.Init()
	if err != nil {
		panic(err)
	}

	sr.preloadNotificationGlyphs()
	sr.fontCache = NewFontCache(sr)

	return sr, sr.GetWindowSizeCells()
}

func (sr *sdlRender) createWindow(caption string) error {
	var err error

	// Create a window for us to draw the text on
	sr.window, err = sdl.CreateWindow(
		caption,             // window title
		X, Y, width, height, // window position & dimensions
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE|sdl.WINDOW_ALWAYS_ON_TOP|sdl.WINDOW_SHOWN, //|sdl.WINDOW_HIDDEN, // window properties
	)
	if err != nil {
		return err
	}

	sr.window.SetWindowOpacity(float32(config.Config.Window.Opacity) / 100)

	err = sr.setIcon()
	if err != nil {
		return err
	}

	sr.initBell()

	setLghtOrDarkMode()

	var renderFlags sdl.RendererFlags
	if config.Config.Window.UseGPU {
		renderFlags |= sdl.RENDERER_ACCELERATED
	} else {
		renderFlags |= sdl.RENDERER_SOFTWARE
	}

	sr.renderer, err = sdl.CreateRenderer(sr.window, -1, renderFlags)
	if err != nil {
		return err
	}

	err = sr.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	if err != nil {
		return err
	}

	//sr.ShowAndFocusWindow()
	return nil
}

func (sr *sdlRender) setIcon() error {
	rwops, err := sdl.RWFromMem(assets.Get(assets.ICON_APP))
	if err != nil {
		return err
	}

	icon, err := sdl.LoadBMPRW(rwops, true)
	if err != nil {
		return err
	}

	sr.window.SetIcon(icon)

	return nil
}

func setLghtOrDarkMode() {
	if config.Config.Terminal.LightMode {
		highlightBlendMode = sdl.BLENDMODE_ADD
		textShadow[_HLTEXTURE_NONE].Alpha = 32
		types.SGR_DEFAULT.Fg, types.SGR_DEFAULT.Bg = types.SGR_DEFAULT.Bg, types.SGR_DEFAULT.Fg
		types.SGR_COLOUR_WHITE, types.SGR_COLOUR_BLACK = types.SGR_COLOUR_BLACK, types.SGR_COLOUR_WHITE
		types.SGR_COLOUR_WHITE_BRIGHT, types.SGR_COLOUR_BLACK_BRIGHT = types.SGR_COLOUR_BLACK_BRIGHT, types.SGR_COLOUR_WHITE_BRIGHT

	} else {
		highlightBlendMode = sdl.BLENDMODE_ADD
		textShadow[_HLTEXTURE_NONE].Alpha = 255
	}
}

func (sr *sdlRender) Start(termWin *types.TermWindow, tmuxClient any) {
	switch tmuxClient.(type) {
	case *tmux.Tmux:
		sr.tmux = tmuxClient.(*tmux.Tmux)
	}

	sr.termWin = termWin

	sr.registerHotkey()
	go sr.refreshInterval()

	sr.eventLoop()
}
