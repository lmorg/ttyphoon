package mainwindow

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) NotesRecentFiles() []string {
	files := notes.HistoryRecentFiles(a.notesHistoryWorkspace())
	for index, filename := range files {
		files[index] = a.notesPathForHistory(filename)
	}
	return files
}

func (a *WApp) notesHistoryWorkspace() string {
	context := a.currentNotesContext()
	notes.MigrateWorkspaceHistory(context.GroupName, context.PathProjectRoot)
	return context.GroupName
}

func (a *WApp) GetNotesFindFieldValues(fieldName string) []string {
	return notes.GetFindFieldValues(fieldName)
}

func (a *WApp) AddNotesFindFieldValue(fieldName, value string) []string {
	return notes.AddFindFieldValue(fieldName, value)
}

func (a *WApp) NotesHistoryAdd(filename string) {
	a.NotesHistoryAddPosition(filename, 1)
}

func (a *WApp) NotesHistoryAddPosition(filename string, line int) {
	expanded := a.filePath(filename)
	if err := notes.HistoryListAddAt(a.notesHistoryWorkspace(), expanded, line); err != nil {
		if renderer, ok := renderwebkit.CurrentRenderer(); ok {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		} else {
			panic(err)
		}
	}
}

func (a *WApp) NotesHistoryCurrent() string {
	return a.NotesHistoryCurrentPosition().File
}

func (a *WApp) NotesHistoryPrevious() (string, error) {
	entry, err := a.NotesHistoryPreviousPosition()
	if err != nil {
		return "", err
	}

	return entry.File, nil
}

func (a *WApp) NotesHistoryNext() (string, error) {
	entry, err := a.NotesHistoryNextPosition()
	if err != nil {
		return "", err
	}

	return entry.File, nil
}

type NotesHistoryPosition struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func (a *WApp) NotesHistoryCurrentPosition() NotesHistoryPosition {
	return a.notesHistoryPosition(notes.HistoryListCurrentEntry(a.notesHistoryWorkspace()))
}

func (a *WApp) NotesHistoryPreviousPosition() (NotesHistoryPosition, error) {
	entry, err := notes.HistoryListPreviousEntry(a.notesHistoryWorkspace())
	if err != nil {
		return NotesHistoryPosition{}, err
	}
	return a.notesHistoryPosition(entry), nil
}

func (a *WApp) NotesHistoryNextPosition() (NotesHistoryPosition, error) {
	entry, err := notes.HistoryListNextEntry(a.notesHistoryWorkspace())
	if err != nil {
		return NotesHistoryPosition{}, err
	}
	return a.notesHistoryPosition(entry), nil
}

func (a *WApp) notesHistoryPosition(entry notes.HistoryEntryT) NotesHistoryPosition {
	if entry.Line < 1 {
		entry.Line = 1
	}
	return NotesHistoryPosition{
		File: a.notesPathForHistory(entry.Filename),
		Line: entry.Line,
	}
}
