package rendersdl

import (
	"github.com/lmorg/ttyphoon/config"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) cancelWInputBox() {
	if sr._cancelWInputBox != nil {
		sr._cancelWInputBox()
	}
}

func (sr *sdlRender) eventWindow(evt *sdl.WindowEvent) {
	sr.cacheBgTexture.Destroy(sr)

	switch evt.Event {
	case sdl.WINDOWEVENT_RESIZED:
		sr.cancelWInputBox()
		sr.windowResized()

	case sdl.WINDOWEVENT_FOCUS_GAINED:
		sr.cancelWInputBox()
		sr.termWin.Active.GetTerm().HasFocus(true)
		sr.window.SetWindowOpacity(float32(config.Config.Window.Opacity) / 100)
		sr.hkToggle = true
		if config.Config.Tmux.Enabled {
			sr.windowResized()
		}

	case sdl.WINDOWEVENT_FOCUS_LOST:
		sr.termWin.Active.GetTerm().HasFocus(false)
		sr.window.SetWindowOpacity(float32(config.Config.Window.InactiveOpacity) / 100)
		sr.hkToggle = false
	}
}
