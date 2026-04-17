package rendererwebkit

import (
	"fmt"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (wr *webkitRender) GetWindowTitle() string {
	return wr.windowTitle
}

func (wr *webkitRender) SetWindowTitle(title string) {
	wr.windowTitle = title
	runtime.WindowSetTitle(wr.wapp, fmt.Sprintf("%s: %s", app.Name(), title))
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

func (wr *webkitRender) toggleNotesPane() {
	if wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "toggleNotesPane")
}

func (wr *webkitRender) ResizeWindow(size *types.XY) {
	if size == nil {
		return
	}
	wr.windowCells = size

	// Physically resize the OS window so the column/row constraint imposed by
	// DECCOLM (resize80/resize132) is actually enforced. Without this the
	// buffer is reset to the new width but the window pixel size is unchanged,
	// so the next WindowResized event overwrites the constraint.
	/*if wr.wapp != nil && wr.glyphSize != nil && wr.glyphSize.X > 0 && wr.glyphSize.Y > 0 {
		px := int(size.X * wr.glyphSize.X)
		py := int(size.Y * wr.glyphSize.Y)
		runtime.WindowSetSize(wr.wapp, px, py)
	}*/

	if wr.tmux != nil {
		_ = wr.tmux.RefreshClient(size)
		_ = wr.tmux.SelectAndResizeWindow(wr.tmux.ActiveWindow().Id(), size)
	}
}

func (wr *webkitRender) WindowResized(cols, rows int32) {
	size := &types.XY{X: cols, Y: rows}

	wr.ResizeWindow(size)

	if !config.Config.Tmux.Enabled && wr.termWin != nil && wr.termWin.Active != nil {
		term := wr.termWin.Active.GetTerm()
		if term != nil {
			term.Resize(size)
		}
	}

	go wr.RefreshWindowList()
}
