package rendersdl

import (
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) eventMouseButton(evt *sdl.MouseButtonEvent) {
	switch {
	case sr.menu != nil:
		sr.menu.eventMouseButton(sr, evt)

	case sr.inputBox != nil:
		sr.inputBox.eventMouseButton(sr, evt)

	case sr.highlighter != nil:
		sr.highlighter.eventMouseButton(sr, evt)

	default:
		if !sr.notifications.eventMouseButton(sr, evt) {
			sr.termWidget.eventMouseButton(sr, evt)
		}
	}
}

func (sr *sdlRender) eventMouseWheel(evt *sdl.MouseWheelEvent) {
	switch {
	//case sr.menu != nil:
	//	sr.menu.eventMouseWheel(sr, evt)

	//case sr.inputBox != nil:
	//	sr.inputBox.eventMouseWheel(sr, evt)

	case sr.highlighter != nil:
		sr.highlighter.eventMouseWheel(sr, evt)

	default:
		sr.termWidget.eventMouseWheel(sr, evt)
	}
}

func (sr *sdlRender) eventMouseMotion(evt *sdl.MouseMotionEvent) {
	switch {
	case sr.menu != nil:
		sr.menu.eventMouseMotion(sr, evt)

	case sr.inputBox != nil:
		sr.inputBox.eventMouseMotion(sr, evt)

	case sr.highlighter != nil:
		sr.highlighter.eventMouseMotion(sr, evt)

	default:
		sr.termWidget.eventMouseMotion(sr, evt)
	}
}
