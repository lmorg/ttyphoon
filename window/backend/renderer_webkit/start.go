package rendererwebkit

import (
	"context"
	"time"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var currentRenderer *webkitRender

func Initialise() (types.Renderer, *types.XY) {
	wr := &webkitRender{
		glyphSize:     nil,
		windowCells:   &types.XY{X: 120, Y: 40},
		windowTitle:   app.Name(),
		_redraw:       make(chan struct{}, 1),
		menuCallbacks: make(map[int]menuCallbacks),
	}

	currentRenderer = wr

	return wr, wr.windowCells
}

func (wr *webkitRender) Start(termWin *types.AppWindowTerms, tmuxClient any, wapp context.Context) {
	wr.termWin = termWin
	wr.wapp = wapp
	// app will be set separately by the caller with SetApp method

	if tc, ok := tmuxClient.(*tmux.Tmux); ok {
		wr.tmux = tc
	}

	cursor.Register(func(css string) {
		runtime.EventsEmit(wapp, "setCursor", css)
	})

	runtime.EventsEmit(wapp, "terminalStatusBarText", wr.statusBarText)

	wr.hotkeys()
	go wr.blinkSlowLoop()

	go func() {
		refreshInterval := time.Duration(config.Config.Window.RefreshInterval)
		for {
			select {
			case <-wr._redraw:
				if commands := wr.PopDrawCommands(); len(commands) > 0 {
					runtime.EventsEmit(wapp, "terminalRedraw", commands)
				}

			case <-time.After(refreshInterval * time.Millisecond):
				wr.TriggerRedraw()
			}
		}
	}()
}

func CurrentRenderer() (*webkitRender, bool) {
	if currentRenderer == nil {
		return nil, false
	}

	return currentRenderer, true
}

func (wr *webkitRender) SetApp(app interface{}) {
	wr.app = app
}
