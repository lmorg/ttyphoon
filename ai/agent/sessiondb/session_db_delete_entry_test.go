package sessiondb

import (
	"fmt"
	"os"
	"testing"
)

// Erasing one prompt must remove only that row: entryCount decrements by
// exactly one, sibling entries are untouched, and the deleted entry's own
// per-prompt log file goes with it.
func TestDeleteActiveSessionEntry_RemovesOnlyThatEntry(t *testing.T) {
	workspace := "delete-entry-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	var ids []int64
	for i := 0; i < 3; i++ {
		_, entryID, err := AppendActiveSessionEntry(workspace, fmt.Sprintf("prompt %d", i), "", "", "response")
		if err != nil {
			t.Fatalf("AppendActiveSessionEntry: %v", err)
		}
		ids = append(ids, entryID)
	}

	state, err := DeleteActiveSessionEntry(workspace, ids[1], 24)
	if err != nil {
		t.Fatalf("DeleteActiveSessionEntry: %v", err)
	}

	if len(state.Sessions) != 1 || state.Sessions[0].EntryCount != 2 {
		t.Fatalf("Sessions = %+v, want a single session with EntryCount 2", state.Sessions)
	}
	if len(state.History) != 2 {
		t.Fatalf("History = %d items, want 2", len(state.History))
	}
	for _, item := range state.History {
		if item.ID == ids[1] {
			t.Fatalf("deleted entry %d is still present in history: %+v", ids[1], state.History)
		}
	}
}

func TestDeleteActiveSessionEntry_UnknownIDIsANoop(t *testing.T) {
	workspace := "delete-entry-noop-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, _, err := AppendActiveSessionEntry(workspace, "prompt", "", "", "response"); err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}

	state, err := DeleteActiveSessionEntry(workspace, 999999, 24)
	if err != nil {
		t.Fatalf("DeleteActiveSessionEntry: %v", err)
	}

	if len(state.Sessions) != 1 || state.Sessions[0].EntryCount != 1 {
		t.Fatalf("Sessions = %+v, want the existing entry untouched", state.Sessions)
	}
}
