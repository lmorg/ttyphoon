package rendererwebkit

import (
	"context"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/typeface"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var currentRenderer *webkitRender

func Initialise() (types.Renderer, *types.XY) {
	glyphSize := calculateGlyphSize()

	wr := &webkitRender{
		glyphSize:    glyphSize,
		windowCells:  &types.XY{X: 120, Y: 40},
		windowTitle:  app.Name,
		keyboardMode: types.KeysNormal,
		_redraw:      make(chan struct{}, 1),
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

func GetConfiguredGlyphSize() *types.XY {
	return calculateGlyphSize()
}

func calculateGlyphSize() *types.XY {
	size, err := typeface.MeasureSize(config.Config.TypeFace.FontName, config.Config.TypeFace.FontSize)
	if err != nil {
		panic(err)
	}

	if size == nil || size.X <= 0 || size.Y <= 0 {
		panic("invalid glyph size from typography measurement")
	}

	size.X += int32(config.Config.TypeFace.AdjustCellWidth)
	size.Y += int32(config.Config.TypeFace.AdjustCellHeight)
	return size
}
