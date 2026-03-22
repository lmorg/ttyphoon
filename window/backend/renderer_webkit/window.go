package rendererwebkit

import (
	"fmt"

	"github.com/lmorg/ttyphoon/app"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (wr *webkitRender) GetWindowTitle() string {
	return wr.windowTitle
}

func (wr *webkitRender) SetWindowTitle(title string) {
	wr.windowTitle = title
	runtime.WindowSetTitle(wr.wapp, fmt.Sprintf("%s: %s", app.Name, title))
}

func (wr *webkitRender) ShowAndFocusWindow() {
	runtime.WindowShow(wr.wapp)
}

func (wr *webkitRender) StatusBarText(text string) {
	wr.statusBarText = text
	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalStatusBarText", text)
	}
}
