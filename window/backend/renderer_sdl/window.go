package rendersdl

import (
	"fmt"
	"sync/atomic"

	"github.com/lmorg/mxtty/app"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) SetWindowTitle(title string) {
	sr.title = fmt.Sprintf("%s: %s", app.Name, title)
	atomic.CompareAndSwapInt32(&sr.updateTitle, 0, 1)
}

func (sr *sdlRender) GetWindowTitle() string {
	return sr.window.GetTitle()
}

func (sr *sdlRender) GetWindowMeta() any {
	return sr.window
}

func (sr *sdlRender) ShowAndFocusWindow() {
	defer sr.window.Raise()
	defer sr.window.Show()

	sr.hidden = false

	displayNum := screenUnderCursor()
	if displayNum == -1 {
		return
	}
	displayBounds, err := sdl.GetDisplayUsableBounds(displayNum)
	if err != nil {
		return
	}

	winW, winH := sr.window.GetSize()

	posX := displayBounds.W - winW
	if width < 0 {
		winW, posX = displayBounds.W, 0
	}

	if winH > displayBounds.H {
		winH = displayBounds.H
	}

	sr.window.SetPosition(posX, displayBounds.Y)
	sr.window.SetSize(winW, winH)
	go sr.RefreshWindowList()
}

func (sr *sdlRender) hideWindow() {
	sr.window.Hide()
	sr.hidden = true
}

func screenUnderCursor() int {
	displayCount, err := sdl.GetNumVideoDisplays()
	if err != nil {
		return -1
	}

	x, y, _ := sdl.GetGlobalMouseState()
	for i := 0; i < displayCount; i++ {
		displayBounds, err := sdl.GetDisplayBounds(i)
		if err != nil {
			return -1
		}

		if x >= displayBounds.X && x <= displayBounds.W &&
			y >= displayBounds.Y && y <= displayBounds.H {
			return i
		}
	}

	return -1
}
