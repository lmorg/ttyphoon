package rendersdl

import (
	"fmt"

	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/hotkey"
)

func (sr *sdlRender) _registerHotkey() {
	sr.hk = hotkey.New([]hotkey.Modifier{}, hotkey.KeyF12)
	err := sr.hk.Register()
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Unable to set hotkey: %s", err.Error()))
	}
}

func (sr *sdlRender) pollEventHotkey() <-chan hotkey.Event {
	if sr.hk != nil {
		return sr.hk.Keydown()
	}
	return nil
}

func (sr *sdlRender) eventHotkey() {
	if sr.hkToggle {
		sr.hideWindow()
	} else {
		sr.ShowAndFocusWindow()
	}
	sr.hkToggle = !sr.hkToggle
}
