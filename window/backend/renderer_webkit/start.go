package rendererwebkit

import (
	"context"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var currentRenderer *webkitRender

func Initialise() (types.Renderer, *types.XY) {
	wr := &webkitRender{
		glyphSize:     nil,
		windowCells:   &types.XY{X: 120, Y: 40},
		windowTitle:   app.Name,
		keyboardMode:  types.KeysNormal,
		_redraw:       make(chan struct{}, 1),
		menuCallbacks: make(map[int]menuCallbacks),
	}

	currentRenderer = wr

	return wr, wr.windowCells
}

func (wr *webkitRender) Start(termWin *types.AppWindowTerms, _ any, wapp context.Context) {
	wr.termWin = termWin
	wr.wapp = wapp

	go func() {
		for {
			select {
			case <-wr._redraw:
				commands := wr.PopDrawCommands()
				if len(commands) == 0 {
					continue
				}
				runtime.EventsEmit(wapp, "terminalRedraw", commands)
				//case <-time.After(15 * time.Millisecond):
				//	runtime.EventsEmit(wapp, "terminalRedraw", wr.PopDrawCommands())
				//wr.TriggerRedraw()
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
