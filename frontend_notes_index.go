package main

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/utils/notes"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

type NotesFilesSnapshot struct {
	Context   notes.FileContext         `json:"context"`
	Roots     []notes.FileIndexSnapshot `json:"roots"`
	Files     []string                  `json:"files"`
	State     string                    `json:"state"`
	Truncated bool                      `json:"truncated"`
	Error     string                    `json:"error"`
}

func (a *WApp) notesFileIndexLocked() *notes.FileIndex {
	if a.notesFileIndex == nil {
		a.notesFileIndex = notes.NewFileIndex(a.ctx, 16, 2, notes.ScanIndexRoot, func(event notes.FileIndexEvent) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "notesFileIndexChanged", event)
			}
			a.refreshGitStatusForFileIndex(event.Root, event)
		})
		a.notesIndexUnsubscribe = notes.SubscribeFileChanges(a.notesFileIndex.NotifyPaths)
		a.notesFileIndex.EnableWatching()
	}
	return a.notesFileIndex
}

func (a *WApp) GetNotesFilesSnapshot(roots notes.FileContext, refresh bool) (NotesFilesSnapshot, error) {
	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()
	if a.notesIndexClosed {
		return NotesFilesSnapshot{}, context.Canceled
	}
	if a.notesContext != roots {
		return NotesFilesSnapshot{}, context.Canceled
	}
	index := a.notesFileIndexLocked()
	paths := notes.FileContextRoots(roots)
	if refresh {
		index.Refresh(paths)
	}
	return composeNotesFilesSnapshot(roots, index.Snapshots(paths)), nil
}

func composeNotesFilesSnapshot(roots notes.FileContext, snapshots []notes.FileIndexSnapshot) NotesFilesSnapshot {
	result := NotesFilesSnapshot{Context: roots, Roots: snapshots, Files: []string{}, State: "ready"}
	byRoot := make(map[string]notes.FileIndexSnapshot)
	busy := false
	errors := []string{}
	for _, snapshot := range snapshots {
		byRoot[snapshot.Root] = snapshot
		busy = busy || snapshot.State == "scanning" || snapshot.State == "refreshing"
		result.Truncated = result.Truncated || snapshot.Truncated
		if snapshot.Error != "" {
			errors = append(errors, snapshot.Root+": "+snapshot.Error)
		}
	}
	for _, section := range []struct {
		path      string
		prefix    string
		recursive bool
	}{
		{roots.PathGlobalNotes, "$GLOBAL", false},
		{roots.PathUserNotes, "$NOTES", false},
		{roots.PathProjectRoot, "$PROJECT", true},
	} {
		if section.path == "" {
			continue
		}
		root, err := filepath.Abs(section.path)
		if err != nil {
			continue
		}
		for _, file := range byRoot[filepath.Clean(root)].Files {
			if !section.recursive && strings.ContainsAny(file, `/\`) {
				continue
			}
			result.Files = append(result.Files, section.prefix+string(filepath.Separator)+file)
		}
	}
	result.Error = strings.Join(errors, "\n")
	switch {
	case busy && len(result.Files) > 0:
		result.State = "refreshing"
	case busy:
		result.State = "scanning"
	case result.Error != "":
		result.State = "error"
	case result.Truncated:
		result.State = "truncated"
	}
	return result
}

func (a *WApp) prewarmNotesFiles(renderer interface{ NotesWorkspaceContexts() []notes.FileContext }) {
	if current, ok := renderwebkit.CurrentRenderer(); ok {
		current.NotesFileContext()
	}
	paths := []string{}
	for _, roots := range renderer.NotesWorkspaceContexts() {
		paths = append(paths, notes.FileContextRoots(roots)...)
	}
	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()
	if a.notesIndexClosed {
		return
	}
	a.notesFileIndexLocked().Prewarm(paths)
}
