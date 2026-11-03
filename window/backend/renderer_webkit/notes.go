package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

func (wr *webkitRender) RefreshNotes() {
	if wr == nil || wr.wapp == nil {
		return
	}
	wr.NotesFileContext()
}

func (wr *webkitRender) NotesFileContext() notes.FileContext {
	if wr == nil {
		return notes.FileContext{}
	}
	wr.notesContextMu.Lock()
	defer wr.notesContextMu.Unlock()
	var tile types.Tile = wr.ActiveTile()
	if wr.tmux != nil {
		if pane, err := wr.tmux.ActivePane(); err == nil && pane != nil {
			tile = pane
		}
	}
	if tile == nil {
		return notes.FileContext{}
	}
	pwd, group := tile.Pwd(), tile.GroupName()
	if wr.notesResolved.PathGlobalNotes == "" || pwd != wr.notesPwd || group != wr.notesGroup {
		wr.notesPwd, wr.notesGroup = pwd, group
		wr.notesResolved = notes.ResolveFileContext(tile)
	}
	roots := wr.notesResolved
	if roots != wr.notesContext {
		wr.notesContext = roots
		if app, ok := wr.app.(interface{ SyncNotesContext(notes.FileContext) }); ok {
			app.SyncNotesContext(roots)
		}
		if wr.wapp != nil {
			runtime.EventsEmit(wr.wapp, "notesContextChanged", roots)
		}
	}
	return roots
}

func (wr *webkitRender) NotesWorkspaceContexts() []notes.FileContext {
	if wr == nil || wr.tmux == nil {
		return nil
	}
	contexts := []notes.FileContext{}
	for _, directory := range wr.tmux.WorkspaceDirectories() {
		contexts = append(contexts, notes.ResolveDirectoryContext(directory.Pwd, directory.GroupName))
	}
	return contexts
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
	context := wr.NotesFileContext()
	notes.MigrateWorkspaceHistory(context.GroupName, context.PathProjectRoot)
	recent := notes.HistoryRecentFiles(context.GroupName)

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

func (wr *webkitRender) NotesLspGoToWorkspaceSymbol() {
	if wr == nil || wr.wapp == nil {
		return
	}

	runtime.EventsEmit(wr.wapp, "notesRunLspGoToWorkspaceSymbol")
}
