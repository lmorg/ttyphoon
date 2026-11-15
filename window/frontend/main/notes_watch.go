package mainwindow

import (
	"github.com/lmorg/ttyphoon/utils/file/watcher"
)

// NotesWatchedFileT is an open Notes document whose file is watched for
// changes made outside Notes. Key is the document's frontend key.
type NotesWatchedFileT struct {
	Key     string `json:"key"`
	Path    string `json:"path"`
	Project string `json:"project"`
}

// NotesFileChangedT is the notesFileChanged event: a watched file changed on
// disk, and Keys are the open documents of that file.
type NotesFileChangedT struct {
	Path string   `json:"path"`
	Keys []string `json:"keys"`
}

// WatchNotesFiles replaces the watched files with those of the open documents
// (ADR 0100). Calls may be handled out of order, so a call with an older
// generation than the last one applied is ignored.
func (a *WApp) WatchNotesFiles(generation int, files []NotesWatchedFileT) {
	keys := make(map[string][]string, len(files))
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.Path == "" {
			continue
		}
		path := watcher.CleanPath(a.filePathWithProject(file.Path, file.Project))
		if _, ok := keys[path]; !ok {
			paths = append(paths, path)
		}
		keys[path] = append(keys[path], file.Key)
	}

	a.notesWatchMu.Lock()
	defer a.notesWatchMu.Unlock()
	if generation < a.notesWatchGeneration {
		return
	}
	a.notesWatchGeneration = generation
	a.notesWatchKeys = keys
	watcher.SetFiles(paths)
}

func (a *WApp) notesFileChangedEvent(path string) NotesFileChangedT {
	a.notesWatchMu.Lock()
	defer a.notesWatchMu.Unlock()
	return NotesFileChangedT{
		Path: path,
		Keys: append([]string{}, a.notesWatchKeys[path]...),
	}
}
