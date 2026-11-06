package mainwindow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/find"
	"github.com/lmorg/ttyphoon/utils/grep"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	"github.com/lmorg/ttyphoon/utils/notes"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) GetNotesMaxLogLines() int {
	return config.Config.Notes.MaxLogLines
}

// GetNotesStructViewMaxSizeKB returns the maximum file size (in kilobytes) for
// which the JSON/YAML structured "View" mode is rendered. Larger files display
// a "file too large" message instead.
func (a *WApp) GetNotesStructViewMaxSizeKB() int64 {
	return config.Config.Notes.StructViewMaxSize
}

// GetNotesLanguageTabIndent returns the number of spaces to use for tab indentation for a given language
func (a *WApp) GetNotesLanguageTabIndent(language string) int {
	return config.Config.Notes.Languages.GetTabIndent(language)
}

// GetNotesLanguageReservedWords returns the reserved words / built-in identifiers for a given language.
func (a *WApp) GetNotesLanguageReservedWords(language string) []string {
	if settings := config.Config.Notes.Languages[strings.ToLower(strings.TrimSpace(language))]; settings != nil && len(settings.ReservedWords) > 0 {
		return settings.ReservedWords
	}
	for _, alias := range jupyter.LanguageAliases(language) {
		if settings := config.Config.Notes.Languages[strings.ToLower(alias)]; settings != nil && len(settings.ReservedWords) > 0 {
			return settings.ReservedWords
		}
	}
	return nil
}

func (a *WApp) NotesKeyPress(key string, ctrl, alt, shift, meta bool) map[string]bool {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return map[string]bool{
			"consume":      false,
			"prefixActive": false,
		}
	}

	consume, prefixActive := renderer.HandleNotesKeyPress(key, ctrl, alt, shift, meta)
	return map[string]bool{
		"consume":      consume,
		"prefixActive": prefixActive,
	}
}

type FilterResultsT struct {
	List  []string
	Error error
}

// FilterStrings uses the find package to filter a list of strings by query.
// Supports: plain words (AND), "or word1 word2" (OR), "! word" (NOT), "rx regexp", "g glob".
// Returns the original list unchanged when query is empty or malformed.
func (a *WApp) FilterStrings(query string, items []string) *FilterResultsT {
	if strings.TrimSpace(query) == "" {
		return &FilterResultsT{List: items}
	}
	f, err := find.New(query)
	if err != nil {
		return &FilterResultsT{Error: err}
	}
	items = f.Filter(items)
	if len(items) == 0 {
		return &FilterResultsT{Error: fmt.Errorf("nothing matched the filter pattern: '%s'", query)}
	}
	return &FilterResultsT{List: items}
}

func (a *WApp) ListFiles() []string {
	files, _ := a.ListFilesForContext(a.GetNotesContext())
	return files
}

func (a *WApp) GetNotesContext() notes.FileContext {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return notes.FileContext{}
	}
	renderer.NotesFileContext()
	return a.currentNotesContext()
}

func (a *WApp) SyncNotesContext(roots notes.FileContext) {
	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()
	if a.notesIndexClosed {
		return
	}
	if a.notesContext == roots {
		return
	}
	if a.notesListCancel != nil {
		a.notesListCancel()
		a.notesListCancel = nil
	}
	a.notesListSeq++
	a.notesContext = roots
	a.globalNotes = roots.PathGlobalNotes
	a.usrNotesDir = roots.PathUserNotes
	a.projRoot = roots.PathProjectRoot
	a.groupName = roots.GroupName
	index := a.notesFileIndexLocked()
	paths := notes.FileContextRoots(roots)
	if !a.notesIndexActivated {
		index.Prewarm(paths)
		a.notesIndexActivated = true
	}
	index.Activate(paths)
}

func (a *WApp) currentNotesContext() notes.FileContext {
	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()
	return notes.FileContext{
		PathProjectRoot: a.projRoot,
		PathGlobalNotes: a.globalNotes,
		PathUserNotes:   a.usrNotesDir,
		GroupName:       a.groupName,
	}
}

func (a *WApp) ListFilesForContext(roots notes.FileContext) ([]string, error) {
	a.notesListMu.Lock()
	if a.notesContext != roots {
		a.notesListMu.Unlock()
		return nil, context.Canceled
	}
	if a.notesListCancel != nil {
		a.notesListCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.notesListCancel = cancel
	a.notesListSeq++
	scanID := a.notesListSeq
	a.notesListMu.Unlock()

	ulf := notes.ScanFiles(ctx, roots)

	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()
	if a.notesListSeq != scanID || a.notesContext != roots || ctx.Err() != nil {
		return nil, context.Canceled
	}
	a.notesListCancel = nil
	return ulf.Files, ulf.Err
}

// NotesGrepStream performs a streaming grep search, sending results via EventsEmit in batches.
// This is non-blocking and suitable for large result sets.
// The frontend should listen for "notesGrepBatch" events containing batched Results,
// "notesGrepError" for errors, and "notesGrepDone" when finished.
func (a *WApp) NotesGrepStream(query string, opts NotesGrepOptionsT) {
	go a.notesGrepStream(query, opts)
}

func (a *WApp) notesGrepStream(query string, opts NotesGrepOptionsT) {
	searchRoot := strings.TrimSpace(a.GetCurrentProject())
	if searchRoot == "" {
		runtime.EventsEmit(a.ctx, "notesGrepError", "search root is empty")
		return
	}

	pathMapper := func(absPath string) string {
		mapped := a.notesPathForHistory(absPath)
		if mapped != "" {
			return mapped
		}
		return absPath
	}

	resultsChan := make(chan []*grep.Result, 1)

	errChan := make(chan error, 1)
	go func() {
		timeout, _ := context.WithTimeout(context.Background(), 5*time.Second)
		errChan <- grep.BatchedStreamResults(timeout, searchRoot, query, opts, pathMapper, resultsChan)
	}()

	// Emit batches as they arrive
	for batch := range resultsChan {
		runtime.EventsEmit(a.ctx, "notesGrepBatch", batch)
	}

	// Check for error
	if err := <-errChan; err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[debug] NotesGrepStream canceled: %v", err)
		} else {
			runtime.EventsEmit(a.ctx, "notesGrepError", err.Error())
		}
	}

	runtime.EventsEmit(a.ctx, "notesGrepDone", nil)
}

func (a *WApp) CancelNotesListFiles() {
	a.notesListMu.Lock()
	defer a.notesListMu.Unlock()

	if a.notesListCancel != nil {
		a.notesListCancel()
		a.notesListCancel = nil
	}
}

func (a *WApp) GetCurrentGroupName() string {
	return a.currentNotesContext().GroupName
}

func (a *WApp) expandMappingFuncWithProject(s, projectPath string) string {
	switch s {
	case "PROJECT":
		if projectPath != "" {
			return projectPath
		}
		return a.GetCurrentProject()
	case "NOTES":
		return a.currentNotesContext().PathUserNotes
	case "HOME":
		return a.homeDir
	case "GLOBAL":
		return a.currentNotesContext().PathGlobalNotes
	//case "HISTORY":
	//	return a.historyDir
	default:
		return "error"
	}
}

func (a *WApp) notesPathForHistory(filename string) string {
	cleanFile := filepath.Clean(strings.TrimSpace(filename))
	if cleanFile == "." || cleanFile == "" {
		return ""
	}

	replacements := []struct {
		prefix   string
		variable string
	}{
		{prefix: filepath.Clean(a.GetCurrentProject()), variable: "$PROJECT"},
		{prefix: filepath.Clean(a.currentNotesContext().PathUserNotes), variable: "$NOTES"},
		{prefix: filepath.Clean(a.currentNotesContext().PathGlobalNotes), variable: "$GLOBAL"},
		{prefix: filepath.Clean(a.homeDir), variable: "$HOME"},
	}

	for _, replacement := range replacements {
		if replacement.prefix == "." || replacement.prefix == "" {
			continue
		}
		if cleanFile == replacement.prefix {
			return replacement.variable
		}

		prefix := replacement.prefix + string(filepath.Separator)
		if strings.HasPrefix(cleanFile, prefix) {
			return replacement.variable + string(filepath.Separator) + strings.TrimPrefix(cleanFile, prefix)
		}
	}

	return cleanFile
}

func (a *WApp) filePath(filename string) string {
	filename = os.Expand(filename, func(s string) string {
		return a.expandMappingFuncWithProject(s, "")
	})
	if filepath.IsLocal(filename) {
		filename = a.currentNotesContext().PathUserNotes + string(filepath.Separator) + filename
	}
	return filename
}

// filePathWithProject returns the expanded file path using a specific project root.
// This uses the same expansion logic as filePath but allows overriding $PROJECT.
func (a *WApp) filePathWithProject(filename string, projectPath string) string {
	filename = os.Expand(filename, func(s string) string {
		return a.expandMappingFuncWithProject(s, projectPath)
	})
	if filepath.IsLocal(filename) {
		filename = a.currentNotesContext().PathUserNotes + string(filepath.Separator) + filename
	}
	return filename
}

func (a *WApp) ResolveFilePath(filename string) string {
	return a.filePath(filename)
}

// GetCurrentProject returns the absolute path of the current project root.
// This is used by the frontend to track which project a file is associated with,
// preventing issues where a file opened in one project could be overwritten
// if the user switches projects before autosave completes.
func (a *WApp) GetCurrentProject() string {
	return a.currentNotesContext().PathProjectRoot
}

// ViewFileInNotes displays a popup menu (in Go) to select a file to view in the Notes pane.
// On selection it emits:
//  1. "viewFileInNotesOpen" — tells the frontend to load the chosen file.
//  2. "terminalActivateAuxTab" with id "notes" — switches to the Notes tab if it is
//     currently registered as an auxiliary terminal pane tab.
func (a *WApp) ViewFileInNotes() {
	files := a.ListFiles()
	if len(files) == 0 || a.ctx == nil {
		return
	}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	onSelect := func(i int) {
		if i < 0 || i >= len(files) {
			return
		}
		filename := files[i]

		// If Notes is registered as an auxiliary tab, activate it.
		/*for _, tab := range renderer.TerminalPaneTabs() {
			if tab.ID == "notes" {
				renderer.ActivateTerminalPaneTab("notes")
				break
			}
		}*/

		// Tell the frontend to open the file in the Notes pane.
		runtime.EventsEmit(a.ctx, "viewFileInNotesOpen", filename)
	}

	renderer.DisplayMenu("Select file to view in Notes", files, nil, onSelect, nil)
}
