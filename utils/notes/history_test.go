package notes

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/utils/cache"
)

func TestWorkspaceHistoryMigrationPreservesPositionsAndDoesNotReplaceHistory(t *testing.T) {
	root := t.TempDir()
	workspace := t.Name() + root
	legacy := HistoryListT{History: []HistoryEntryT{
		{Filename: filepath.Join(root, "a.md"), Line: 12},
		{Filename: filepath.Join(root, "b.md"), Line: 30},
	}, Index: 0}
	var existing HistoryListT
	cache.Read(cache.NS_NOTESW_HISTORY, root, &existing)
	cache.Write(cache.NS_NOTESW_HISTORY, root, &legacy, cache.Days(30))
	MigrateWorkspaceHistory(workspace, root)
	if got := getHistoryList(workspace); !reflect.DeepEqual(*got, legacy) {
		t.Fatalf("migrated history = %+v, want %+v", got, legacy)
	}
	otherRoot := t.TempDir()
	outsideFile := filepath.Join(otherRoot, "outside.md")
	if err := HistoryListAddAt(workspace, outsideFile, 9); err != nil {
		t.Fatal(err)
	}
	MigrateWorkspaceHistory(workspace, otherRoot)
	if got := HistoryListCurrentEntry(workspace); got != (HistoryEntryT{Filename: outsideFile, Line: 9}) {
		t.Fatalf("root switch replaced workspace history: %+v", got)
	}
	MigrateWorkspaceHistory(workspace+"-other", root)
	if got := HistoryListCurrentEntry(workspace + "-other"); got != legacy.History[0] {
		t.Fatalf("workspace histories are not independent: %+v", got)
	}
}

func TestHistoryRecentFilesReturnsNewestTenDistinctFilesAcrossRoots(t *testing.T) {
	root := t.TempDir()
	workspace := t.Name() + root
	for index := 0; index < 12; index++ {
		if err := HistoryListAddAt(workspace, filepath.Join(root, fmt.Sprintf("file-%02d.md", index)), 1); err != nil {
			t.Fatal(err)
		}
	}
	outsideFile := filepath.Join(t.TempDir(), "outside.md")
	for _, line := range []int{1, 30, 60} {
		if err := HistoryListAddAt(workspace, outsideFile, line); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{outsideFile}
	for index := 11; index >= 3; index-- {
		want = append(want, filepath.Join(root, fmt.Sprintf("file-%02d.md", index)))
	}
	if got := HistoryRecentFiles(workspace); !reflect.DeepEqual(got, want) {
		t.Fatalf("recent = %v, want %v", got, want)
	}
	if got := HistoryRecentFiles(workspace + "-other"); len(got) != 0 {
		t.Fatalf("another workspace leaked recent files: %v", got)
	}
}

func TestHistoryListRenameUpdatesFolderDescendants(t *testing.T) {
	root := t.TempDir()
	workspace := t.Name() + root
	oldPath := filepath.Join(root, "docs")
	newPath := filepath.Join(root, "renamed")
	history := &HistoryListT{History: []HistoryEntryT{
		{Filename: oldPath, Line: 1},
		{Filename: filepath.Join(oldPath, "guide.md"), Line: 12},
		{Filename: filepath.Join(oldPath, "nested", "api.md"), Line: 30},
		{Filename: filepath.Join(root, "docs-old", "other.md"), Line: 7},
	}, Index: 2}
	setHistoryList(workspace, history)
	if err := HistoryListRename(workspace, oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	want := HistoryListT{History: []HistoryEntryT{
		{Filename: newPath, Line: 1},
		{Filename: filepath.Join(newPath, "guide.md"), Line: 12},
		{Filename: filepath.Join(newPath, "nested", "api.md"), Line: 30},
		{Filename: filepath.Join(root, "docs-old", "other.md"), Line: 7},
	}, Index: 2}
	if got := getHistoryList(workspace); !reflect.DeepEqual(*got, want) {
		t.Fatalf("renamed history = %+v, want %+v", *got, want)
	}
}

func TestHistoryListDelete_SelectsSurvivingDocument(t *testing.T) {
	for _, test := range []struct {
		name    string
		history []HistoryEntryT
		index   int
		deleted string
		want    HistoryEntryT
	}{
		{"current middle", []HistoryEntryT{{Filename: "a.md", Line: 7}, {Filename: "b.md", Line: 3}, {Filename: "c.md", Line: 1}}, 1, "b.md", HistoryEntryT{Filename: "a.md", Line: 7}},
		{"current first", []HistoryEntryT{{Filename: "a.md", Line: 7}, {Filename: "b.md", Line: 3}}, 0, "a.md", HistoryEntryT{Filename: "b.md", Line: 3}},
		{"other document", []HistoryEntryT{{Filename: "a.md", Line: 7}, {Filename: "b.md", Line: 3}}, 1, "a.md", HistoryEntryT{Filename: "b.md", Line: 3}},
		{"repeated current", []HistoryEntryT{{Filename: "a.md", Line: 7}, {Filename: "b.md", Line: 3}, {Filename: "b.md", Line: 25}, {Filename: "c.md", Line: 1}}, 2, "b.md", HistoryEntryT{Filename: "a.md", Line: 7}},
		{"last document", []HistoryEntryT{{Filename: "a.md", Line: 7}}, 0, "a.md", HistoryEntryT{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			getHistoryList(projectRoot)
			setHistoryList(projectRoot, &HistoryListT{History: test.history, Index: test.index})
			if err := HistoryListDelete(projectRoot, test.deleted); err != nil {
				t.Fatal(err)
			}
			if got := HistoryListCurrentEntry(projectRoot); got != test.want {
				t.Fatalf("current = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestHistoryListAdd_FirstItemSetsIndex(t *testing.T) {
	projectRoot := t.TempDir()

	err := HistoryListAdd(projectRoot, "a.md")
	if err != nil {
		t.Fatalf("HistoryListAdd returned error: %v", err)
	}

	h := getHistoryList(projectRoot)
	if len(h.History) != 1 {
		t.Fatalf("history length = %d, want 1", len(h.History))
	}
	if h.History[0].Filename != "a.md" || h.History[0].Line != 1 {
		t.Fatalf("history[0] = %+v, want a.md at line 1", h.History[0])
	}
	if h.Index != 0 {
		t.Fatalf("index = %d, want 0", h.Index)
	}
}

func TestHistoryListAdd_AppendsAndMovesIndexToEnd(t *testing.T) {
	projectRoot := t.TempDir()

	setHistoryList(projectRoot, &HistoryListT{
		History: []HistoryEntryT{{Filename: "a.md", Line: 1}},
		Index:   0,
	})

	err := HistoryListAdd(projectRoot, "b.md")
	if err != nil {
		t.Fatalf("HistoryListAdd returned error: %v", err)
	}

	h := getHistoryList(projectRoot)
	if len(h.History) != 2 {
		t.Fatalf("history length = %d, want 2", len(h.History))
	}
	if h.History[0].Filename != "a.md" || h.History[1].Filename != "b.md" {
		t.Fatalf("history = %v, want [a.md b.md]", h.History)
	}
	if h.Index != 1 {
		t.Fatalf("index = %d, want 1", h.Index)
	}
}

func TestHistoryListAdd_TruncatesForwardHistoryFromCurrentIndex(t *testing.T) {
	projectRoot := t.TempDir()

	setHistoryList(projectRoot, &HistoryListT{
		History: []HistoryEntryT{{Filename: "a.md", Line: 1}, {Filename: "b.md", Line: 1}, {Filename: "c.md", Line: 1}},
		Index:   1,
	})

	err := HistoryListAdd(projectRoot, "x.md")
	if err != nil {
		t.Fatalf("HistoryListAdd returned error: %v", err)
	}

	h := getHistoryList(projectRoot)
	if len(h.History) != 3 {
		t.Fatalf("history length = %d, want 3", len(h.History))
	}
	if h.History[0].Filename != "a.md" || h.History[1].Filename != "b.md" || h.History[2].Filename != "x.md" {
		t.Fatalf("history = %v, want [a.md b.md x.md]", h.History)
	}
	if h.Index != 2 {
		t.Fatalf("index = %d, want 2", h.Index)
	}
}

func TestHistoryListAdd_CapsHistoryAtFiftyAndKeepsIndexAtLast(t *testing.T) {
	projectRoot := t.TempDir()

	history := make([]HistoryEntryT, 0, 50)
	for i := 0; i < 50; i++ {
		history = append(history, HistoryEntryT{Filename: fmt.Sprintf("file-%02d.md", i), Line: 1})
	}
	setHistoryList(projectRoot, &HistoryListT{
		History: history,
		Index:   49,
	})

	err := HistoryListAdd(projectRoot, "new.md")
	if err != nil {
		t.Fatalf("HistoryListAdd returned error: %v", err)
	}

	h := getHistoryList(projectRoot)
	if len(h.History) != 50 {
		t.Fatalf("history length = %d, want 50", len(h.History))
	}
	if h.History[0].Filename != "file-01.md" {
		t.Fatalf("history[0] = %+v, want file-01.md", h.History[0])
	}
	if h.History[49].Filename != "new.md" {
		t.Fatalf("history[49] = %+v, want new.md", h.History[49])
	}
	if h.Index != 49 {
		t.Fatalf("index = %d, want 49", h.Index)
	}
}

func TestHistoryListAdd_RejectsEmptyFilename(t *testing.T) {
	projectRoot := t.TempDir()

	err := HistoryListAdd(projectRoot, "")
	if err == nil {
		t.Fatalf("expected error for empty filename")
	}
}

func TestHistoryListCurrent_ReturnsCurrentItem(t *testing.T) {
	projectRoot := t.TempDir()

	setHistoryList(projectRoot, &HistoryListT{
		History: []HistoryEntryT{{Filename: "a.md", Line: 1}, {Filename: "b.md", Line: 1}, {Filename: "c.md", Line: 1}},
		Index:   1,
	})

	got := HistoryListCurrent(projectRoot)
	if got != "b.md" {
		t.Fatalf("HistoryListCurrent() = %q, want %q", got, "b.md")
	}
}

func TestHistoryListAddAt_StoresSameFilePositionsTwentyLinesApart(t *testing.T) {
	projectRoot := t.TempDir()
	if err := HistoryListAddAt(projectRoot, "a.md", 10); err != nil {
		t.Fatal(err)
	}
	if err := HistoryListAddAt(projectRoot, "a.md", 29); err != nil {
		t.Fatal(err)
	}
	if err := HistoryListAddAt(projectRoot, "a.md", 30); err != nil {
		t.Fatal(err)
	}

	h := getHistoryList(projectRoot)
	if len(h.History) != 2 {
		t.Fatalf("history length = %d, want 2", len(h.History))
	}
	if h.History[0].Line != 10 || h.History[1].Line != 30 {
		t.Fatalf("history lines = [%d %d], want [10 30]", h.History[0].Line, h.History[1].Line)
	}
}

func TestHistoryListLegacyCacheMigratesToLineOne(t *testing.T) {
	var history HistoryListT
	if err := json.Unmarshal([]byte(`{"History":["a.md"],"Index":0}`), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.History) != 1 || history.History[0] != (HistoryEntryT{Filename: "a.md", Line: 1}) {
		t.Fatalf("legacy history = %+v, want a.md at line 1", history.History)
	}
}

func TestHistoryListPreviousNextEntriesRetainLineNumbers(t *testing.T) {
	projectRoot := t.TempDir()
	if err := HistoryListAddAt(projectRoot, "a.md", 5); err != nil {
		t.Fatal(err)
	}
	if err := HistoryListAddAt(projectRoot, "a.md", 30); err != nil {
		t.Fatal(err)
	}

	previous, err := HistoryListPreviousEntry(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if previous.Filename != "a.md" || previous.Line != 5 {
		t.Fatalf("previous entry = %+v, want a.md at line 5", previous)
	}

	next, err := HistoryListNextEntry(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if next.Filename != "a.md" || next.Line != 30 {
		t.Fatalf("next entry = %+v, want a.md at line 30", next)
	}
}
