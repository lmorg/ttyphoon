package main

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/notes"
	"golang.design/x/clipboard"
)

func TestNotesHistoryUsesWindowNameAcrossProjectRoots(t *testing.T) {
	firstRoot, secondRoot := t.TempDir(), t.TempDir()
	workspace := t.Name() + firstRoot
	app := &WApp{projRoot: firstRoot, groupName: workspace, homeDir: t.TempDir()}
	app.NotesHistoryAddPosition("$PROJECT/README.md", 4)
	app.projRoot = secondRoot
	app.NotesHistoryAddPosition("$PROJECT/README.md", 30)
	firstFile := filepath.Join(firstRoot, "README.md")
	if got, err := app.NotesHistoryPreviousPosition(); err != nil || got != (NotesHistoryPosition{File: firstFile, Line: 4}) {
		t.Fatalf("cross-root previous = %+v, %v", got, err)
	}
	if got := app.NotesRecentFiles(); !reflect.DeepEqual(got, []string{"$PROJECT/README.md", firstFile}) {
		t.Fatalf("cross-root recent files = %v", got)
	}
	app.groupName = workspace + "-other"
	if got := app.NotesHistoryCurrentPosition(); got.File != "" {
		t.Fatalf("another window inherited history: %+v", got)
	}
	if _, err := app.NotesHistoryNextPosition(); err == nil {
		t.Fatal("empty workspace accepted Next navigation")
	}
	app.NotesHistoryAddPosition("$PROJECT/other.md", 8)
	app.groupName = workspace
	if got := app.NotesHistoryCurrentPosition(); got != (NotesHistoryPosition{File: firstFile, Line: 4}) {
		t.Fatalf("switching windows changed the original history: %+v", got)
	}
}

func TestNotesReservedWordsResolveFenceAliasesAndRuntimeDescriptions(t *testing.T) {
	app := &WApp{}
	for _, test := range []struct{ language, word string }{
		{"docker", "FROM"}, {"dockerfile", "FROM"}, {" DOCKERFILE ", "FROM"},
		{"Dockerfile (build and run)", "FROM"}, {"node", "function"},
		{"Javascript (node.js)", "function"}, {"py", "def"},
	} {
		words := app.GetNotesLanguageReservedWords(test.language)
		found := false
		for _, word := range words {
			found = found || strings.EqualFold(word, test.word)
		}
		if !found {
			t.Errorf("reserved words for %q omit %q", test.language, test.word)
		}
	}
	if words := app.GetNotesLanguageReservedWords("unknown-synthetic-language"); len(words) != 0 {
		t.Fatalf("unknown language returned words: %v", words)
	}
}

func TestNotesReservedWordsPreferExactConfiguredLanguage(t *testing.T) {
	previous, existed := config.Config.Notes.Languages["dockerfile"]
	t.Cleanup(func() {
		if existed {
			config.Config.Notes.Languages["dockerfile"] = previous
		} else {
			delete(config.Config.Notes.Languages, "dockerfile")
		}
	})
	config.Config.Notes.Languages["dockerfile"] = &config.LanguageOptionsT{ReservedWords: []string{"custom-direct-word"}}
	if got := (&WApp{}).GetNotesLanguageReservedWords("dockerfile"); !reflect.DeepEqual(got, []string{"custom-direct-word"}) {
		t.Fatalf("exact configured words were overridden: %v", got)
	}
}

func TestNotesContextChangesBeforeScanAndCancelsOldScan(t *testing.T) {
	app := &WApp{}
	oldContext := notes.FileContext{PathProjectRoot: t.TempDir(), GroupName: "Old"}
	newContext := notes.FileContext{PathProjectRoot: t.TempDir(), GroupName: "New"}
	app.SyncNotesContext(oldContext)
	t.Cleanup(func() { app.notesIndexUnsubscribe(); app.notesFileIndex.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.notesListCancel = cancel
	sequence := app.notesListSeq

	app.SyncNotesContext(newContext)
	if ctx.Err() != context.Canceled {
		t.Fatal("workspace change did not cancel the previous scan")
	}
	if app.GetCurrentProject() != newContext.PathProjectRoot || app.GetCurrentGroupName() != "New" {
		t.Fatal("workspace context was not updated before scanning")
	}
	if app.notesListSeq <= sequence {
		t.Fatal("workspace change did not invalidate the old scan generation")
	}
	if files, err := app.ListFilesForContext(oldContext); !errors.Is(err, context.Canceled) || files != nil {
		t.Fatalf("stale context listing = %v, %v; want cancellation", files, err)
	}
	sequence = app.notesListSeq
	app.SyncNotesContext(newContext)
	if app.notesListSeq != sequence {
		t.Fatal("unchanged context invalidated the scan generation")
	}
}

func TestNotesFilesSnapshotIsNonblockingAndDoesNotRescan(t *testing.T) {
	started := make(chan struct{}, 1)
	finish := make(chan struct{})
	events := make(chan notes.FileIndexEvent, 4)
	index := notes.NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, root string) notes.RootScanResult {
		started <- struct{}{}
		select {
		case <-finish:
			return notes.RootScanResult{Files: []string{"ready.md"}}
		case <-ctx.Done():
			return notes.RootScanResult{Err: ctx.Err()}
		}
	}, func(event notes.FileIndexEvent) { events <- event })
	t.Cleanup(index.Close)
	app := &WApp{notesFileIndex: index}
	roots := notes.FileContext{PathProjectRoot: t.TempDir(), GroupName: "Workspace"}
	app.SyncNotesContext(roots)
	<-started
	snapshot, err := app.GetNotesFilesSnapshot(roots, false)
	if err != nil || snapshot.State != "scanning" {
		t.Fatalf("initial snapshot = %+v, %v", snapshot, err)
	}
	close(finish)
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("snapshot was not published")
	}
	snapshot, err = app.GetNotesFilesSnapshot(roots, false)
	if err != nil || !reflect.DeepEqual(snapshot.Files, []string{"$PROJECT/ready.md"}) || snapshot.State != "ready" {
		t.Fatalf("published snapshot = %+v, %v", snapshot, err)
	}
	select {
	case <-started:
		t.Fatal("reading a cached snapshot started another scan")
	default:
	}
}

func TestComposeNotesFilesSnapshotMapsSectionsAndReportsTruncation(t *testing.T) {
	roots := notes.FileContext{PathProjectRoot: t.TempDir(), PathGlobalNotes: t.TempDir(), PathUserNotes: t.TempDir()}
	result := composeNotesFilesSnapshot(roots, []notes.FileIndexSnapshot{
		{Root: roots.PathProjectRoot, Files: []string{"src/main.go"}, State: "truncated", Truncated: true},
		{Root: roots.PathGlobalNotes, Files: []string{"global.md", "group/note.md"}, State: "ready"},
		{Root: roots.PathUserNotes, Files: []string{"todo.md", "nested/note.md"}, State: "ready"},
	})
	want := []string{"$GLOBAL/global.md", "$NOTES/todo.md", "$PROJECT/src/main.go"}
	if !reflect.DeepEqual(result.Files, want) || result.State != "truncated" || !result.Truncated {
		t.Fatalf("snapshot = %+v, want sections %v with truncation", result, want)
	}
}

func TestSaveBinaryFilePublishesWriteThroughHint(t *testing.T) {
	app := &WApp{}
	path := filepath.Join(t.TempDir(), "clipboard.png")
	var changed []string
	unsubscribe := notes.SubscribeFileChanges(func(paths ...string) { changed = append(changed, paths...) })
	defer unsubscribe()
	if err := app.SaveBinaryFile(path, "aW1hZ2U="); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(changed, []string{path}) {
		t.Fatalf("file change hints = %v, want %s", changed, path)
	}
}

func TestFormattedClipboardItemsIncludeRTFHTMLAndText(t *testing.T) {
	items := formattedClipboardItems("<table></table>", `{\rtf1 table}`, "table")
	if len(items) != 3 {
		t.Fatalf("clipboard representations = %d, want 3", len(items))
	}
	wantFormats := []clipboard.Format{clipboard.Register("text/rtf"), clipboard.Register("text/html"), clipboard.FmtText}
	wantBytes := []string{`{\rtf1 table}`, "<table></table>", "table"}
	for position, item := range items {
		if item.Format != wantFormats[position] || string(item.Bytes) != wantBytes[position] {
			t.Fatalf("clipboard item %d = %+v, want format %v and %q", position, item, wantFormats[position], wantBytes[position])
		}
	}
	if legacy := formattedClipboardItems("html", "", "text"); len(legacy) != 2 {
		t.Fatalf("HTML-only compatibility representations = %d, want 2", len(legacy))
	}
}
