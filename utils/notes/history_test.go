package notes

import (
	"fmt"
	"testing"
)

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
	if h.History[0] != "a.md" {
		t.Fatalf("history[0] = %q, want %q", h.History[0], "a.md")
	}
	if h.Index != 0 {
		t.Fatalf("index = %d, want 0", h.Index)
	}
}

func TestHistoryListAdd_AppendsAndMovesIndexToEnd(t *testing.T) {
	projectRoot := t.TempDir()

	setHistoryList(projectRoot, &HistoryListT{
		History: []string{"a.md"},
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
	if h.History[0] != "a.md" || h.History[1] != "b.md" {
		t.Fatalf("history = %v, want [a.md b.md]", h.History)
	}
	if h.Index != 1 {
		t.Fatalf("index = %d, want 1", h.Index)
	}
}

func TestHistoryListAdd_TruncatesForwardHistoryFromCurrentIndex(t *testing.T) {
	projectRoot := t.TempDir()

	setHistoryList(projectRoot, &HistoryListT{
		History: []string{"a.md", "b.md", "c.md"},
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
	if h.History[0] != "a.md" || h.History[1] != "b.md" || h.History[2] != "x.md" {
		t.Fatalf("history = %v, want [a.md b.md x.md]", h.History)
	}
	if h.Index != 2 {
		t.Fatalf("index = %d, want 2", h.Index)
	}
}

func TestHistoryListAdd_CapsHistoryAtFiftyAndKeepsIndexAtLast(t *testing.T) {
	projectRoot := t.TempDir()

	history := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		history = append(history, fmt.Sprintf("file-%02d.md", i))
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
	if h.History[0] != "file-01.md" {
		t.Fatalf("history[0] = %q, want %q", h.History[0], "file-01.md")
	}
	if h.History[49] != "new.md" {
		t.Fatalf("history[49] = %q, want %q", h.History[49], "new.md")
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
		History: []string{"a.md", "b.md", "c.md"},
		Index:   1,
	})

	got := HistoryListCurrent(projectRoot)
	if got != "b.md" {
		t.Fatalf("HistoryListCurrent() = %q, want %q", got, "b.md")
	}
}
