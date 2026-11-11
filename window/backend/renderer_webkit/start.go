package rendererwebkit

import (
	"context"
	"log"
	"time"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
)

var currentRenderer *webkitRender

// RedrawAckEvent is emitted by the frontend after applying a redraw batch.
const RedrawAckEvent = "ttyphoon:terminal-redraw-ack"

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
		emit := func(batchID uint64, commands []DrawOpTuple) {
			runtime.EventsEmit(wapp, "terminalRedraw", struct {
				ID  uint64        `json:"id"`
				Ops []DrawOpTuple `json:"ops"`
			}{ID: batchID, Ops: commands})
		}
		for {
			select {
			case <-wr._redraw:
				wr.redraw(emit)

			case <-time.After(refreshInterval * time.Millisecond):
				if !wr.renderPaused.Load() {
					wr.TriggerRedraw()
				}
			}
		}
	}()
}

// redraw renders every tile from current terminal state. While paused it
// renders nothing: terminal state keeps updating, so the first redraw after
// resuming is complete and nothing needs to be queued.
func (wr *webkitRender) redraw(emit func(uint64, []DrawOpTuple)) {
	if wr.renderPaused.Load() {
		return
	}
	if batchID := wr.inFlightRedraw.Load(); batchID != 0 {
		if wr.redrawAck.Load() != batchID {
			wr.redrawDeferred.Store(true)
			return
		}
		wr.inFlightRedraw.Store(0)
	}
	if commands := wr.PopDrawCommands(); len(commands) > 0 {
		batchID := wr.nextRedrawID.Add(1)
		wr.inFlightRedraw.Store(batchID)
		emit(batchID, encodeDrawCommands(commands))
	}
}

// AcknowledgeRedraw releases backpressure after the frontend has applied a
// batch to its offscreen canvas. It only requests a redraw if one was held
// back while waiting; an unconditional redraw would loop emit→ack→emit with
// no throttle, because every full render produces commands.
func (wr *webkitRender) AcknowledgeRedraw(batchID uint64) {
	if batchID == 0 || wr.inFlightRedraw.Load() != batchID {
		return
	}
	wr.redrawAck.Store(batchID)
	if wr.redrawDeferred.Swap(false) {
		wr.TriggerRedraw()
	}
}

// SetRenderingPaused stops rendering while the window cannot be seen and
// requests a full redraw when it can be seen again.
func (wr *webkitRender) SetRenderingPaused(paused bool) {
	if wr.renderPaused.Swap(paused) == paused {
		return
	}
	if paused {
		log.Printf("[trace] renderer: rendering paused (window not visible)")
		return
	}
	log.Printf("[trace] renderer: rendering resumed")
	wr.TriggerRedraw()
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
