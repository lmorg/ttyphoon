package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/utils/notes"
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

func (wr *webkitRender) NotesRecentFiles() {
	tile := wr.ActiveTile()
	if tile == nil {
		return
	}

	project := notes.DirProjectRoot(tile.Pwd())
	recent := notes.GetRecentList(project)

	okFn := func(i int) {
		runtime.EventsEmit(wr.wapp, "viewFileInNotesOpen", recent[i])
	}

	wr.openMenu("Recent files", recent, nil, nil, okFn, nil, false)
}

func (wr *webkitRender) NotesLspOptions() {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "notesShowLspOptions")
}

func (wr *webkitRender) NotesLspFormatDocument() {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "notesRunLspFormatDocument")
}

func (wr *webkitRender) NotesLspGoToSymbol() {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "notesRunLspGoToSymbol")
}
