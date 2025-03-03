package rendersdl

import (
	"log"
	"time"

	"github.com/lmorg/mxtty/config"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) refreshInterval() {
	if config.Config.Window.RefreshInterval == 0 {
		return
	}

	d := time.Duration(config.Config.Window.RefreshInterval) * time.Millisecond
	for {
		time.Sleep(d)
		sr.TriggerRedraw()
	}
}

func (sr *sdlRender) eventLoop() {
	for {

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch evt := event.(type) {

			case sdl.WindowEvent:
				sr.eventWindow(&evt)
				sr.TriggerRedraw()

			case sdl.TextInputEvent:
				sr.eventTextInput(&evt)
				sr.TriggerRedraw()

			case sdl.KeyboardEvent:
				sr.eventKeyPress(&evt)
				sr.TriggerRedraw()

			case sdl.MouseButtonEvent:
				sr.eventMouseButton(&evt)
				sr.TriggerRedraw()

			case sdl.MouseMotionEvent:
				sr.eventMouseMotion(&evt)
				// don't trigger redraw

			case sdl.MouseWheelEvent:
				sr.eventMouseWheel(&evt)
				sr.TriggerRedraw()

			case sdl.QuitEvent:
				sr.TriggerQuit()

			}
		}

		select {
		case size := <-sr._resize:
			sr._resizeWindow(size)

		case <-sr._redraw:
			err := render(sr)
			if err != nil {
				log.Printf("ERROR: %s", err.Error())
			}

		case <-sr.pollEventHotkey():
			sr.eventHotkey()

		case <-sr._quit:
			return

		case <-time.After(15 * time.Millisecond):
			continue
		}
	}
}
