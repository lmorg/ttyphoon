package sessiondb

import (
	"os"
	"testing"
)

// RenameSession is the only way the frontend can update a session's summary
// after creation, so it must persist and be reflected in the returned state.
func TestRenameSession_UpdatesSummaryAndReturnsState(t *testing.T) {
	workspace := "rename-session-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	created, err := CreateSession(workspace, "original name", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	renamed, err := RenameSession(workspace, created.ActiveSessionID, "  renamed session  ", 24)
	if err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	if renamed.ActiveSessionID != created.ActiveSessionID {
		t.Fatalf("ActiveSessionID = %d, want %d", renamed.ActiveSessionID, created.ActiveSessionID)
	}

	var got string
	for _, session := range renamed.Sessions {
		if session.TableID == created.ActiveSessionID {
			got = session.Summary
		}
	}
	if got != "renamed session" {
		t.Fatalf("Summary = %q, want %q", got, "renamed session")
	}
}

func TestRenameSession_EmptySummaryFallsBackToDefault(t *testing.T) {
	workspace := "rename-session-empty-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	created, err := CreateSession(workspace, "original name", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	renamed, err := RenameSession(workspace, created.ActiveSessionID, "   ", 24)
	if err != nil {
		t.Fatalf("RenameSession: %v", err)
	}

	var got string
	for _, session := range renamed.Sessions {
		if session.TableID == created.ActiveSessionID {
			got = session.Summary
		}
	}
	if got != defaultSessionTitle {
		t.Fatalf("Summary = %q, want default %q", got, defaultSessionTitle)
	}
}
