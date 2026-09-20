package sessiondb

import (
	"os"
	"strings"
	"testing"
)

// A stray base64 image data URL previously sent through the generic "ask AI
// about this document" path (rather than the dedicated image attachment path)
// persisted as output_block and froze the AI Settings modal when rendered
// back out as one giant unbroken string. History fields must be capped
// regardless of how they got large, since a pre-existing row in someone's
// database cannot be edited after the fact.
func TestGetFrontendState_TruncatesOversizedHistoryFields(t *testing.T) {
	workspace := "history-truncation-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	giant := strings.Repeat("A", maxHistoryFieldChars*3)
	if _, _, err := AppendActiveSessionEntry(workspace, "prompt", "cmd", giant, "response"); err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}

	state, err := GetFrontendState(workspace, 24)
	if err != nil {
		t.Fatalf("GetFrontendState: %v", err)
	}

	if len(state.History) != 1 {
		t.Fatalf("History = %d items, want 1", len(state.History))
	}
	if len(state.History[0].OutputBlock) > maxHistoryFieldChars+len("\n\n[truncated for display]") {
		t.Fatalf("OutputBlock length = %d, want capped near %d", len(state.History[0].OutputBlock), maxHistoryFieldChars)
	}
	if !strings.Contains(state.History[0].OutputBlock, "[truncated for display]") {
		t.Fatalf("OutputBlock = %q, want a truncation marker", state.History[0].OutputBlock)
	}
}

func TestGetFrontendState_DoesNotTruncateNormalHistoryFields(t *testing.T) {
	workspace := "history-no-truncation-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, _, err := AppendActiveSessionEntry(workspace, "prompt", "cmd", "normal output", "response"); err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}

	state, err := GetFrontendState(workspace, 24)
	if err != nil {
		t.Fatalf("GetFrontendState: %v", err)
	}

	if state.History[0].OutputBlock != "normal output" {
		t.Fatalf("OutputBlock = %q, want unchanged", state.History[0].OutputBlock)
	}
}
