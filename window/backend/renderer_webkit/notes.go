package rendererwebkit

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (wr *webkitRender) RefreshNotes() {
	if wr == nil || wr.wapp == nil || wr.termWin == nil || wr.termWin.Active == nil {
		// bit of a hack but this should only happen on application startup
		//time.Sleep(500 * time.Millisecond)
		return
	}
	runtime.EventsEmit(wr.wapp, "notesUpdate", wr.termWin.Active.GroupName())
}

func (wr *webkitRender) NotesEditFile(filename string) {
	/*for _, tab := range wr.TerminalPaneTabs() {
		if tab.ID == "notes" {
			wr.ActivateTerminalPaneTab("notes")
			break
		}
	}*/
	runtime.EventsEmit(wr.wapp, "viewFileInNotesEdit", filename)
}

func (wr *webkitRender) NotesCreateAndOpen(filename, content string) {
	runtime.EventsEmit(wr.wapp, "notesCreateAndOpen", map[string]string{
		"filename": filename,
		"contents": content,
	})
}
