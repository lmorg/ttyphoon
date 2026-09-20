package sessiondb

import (
	"fmt"
	"os"
	"testing"
)

// The frontend transcript shows the newest prompt first, while
// ActiveSessionEntries (context reconstruction for the LLM) needs
// chronological order - the two must not regress into the same order.
func TestGetFrontendState_HistoryIsNewestFirst(t *testing.T) {
	workspace := "history-order-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	for i := 0; i < 3; i++ {
		if _, _, err := AppendActiveSessionEntry(workspace, fmt.Sprintf("prompt %d", i), "", "", "response"); err != nil {
			t.Fatalf("AppendActiveSessionEntry: %v", err)
		}
	}

	state, err := GetFrontendState(workspace, 24)
	if err != nil {
		t.Fatalf("GetFrontendState: %v", err)
	}

	if len(state.History) != 3 {
		t.Fatalf("History = %d items, want 3", len(state.History))
	}
	want := []string{"prompt 2", "prompt 1", "prompt 0"}
	for i, w := range want {
		if state.History[i].Prompt != w {
			t.Fatalf("History[%d].Prompt = %q, want %q (newest first)", i, state.History[i].Prompt, w)
		}
	}

	entries, err := ActiveSessionEntries(workspace, 24)
	if err != nil {
		t.Fatalf("ActiveSessionEntries: %v", err)
	}
	wantChronological := []string{"prompt 0", "prompt 1", "prompt 2"}
	for i, w := range wantChronological {
		if entries[i].Prompt != w {
			t.Fatalf("entries[%d].Prompt = %q, want %q (chronological, unaffected)", i, entries[i].Prompt, w)
		}
	}
}
