package sessiondb

import (
	"database/sql"
	"os"
	"testing"
)

// Sessions are ordered by last-updated so the most recently used prompt
// history surfaces first, without pinning the active session to the top
// regardless of when it was actually last touched.
func TestSetActiveSession_DoesNotChangeOrderingOrUpdatedField(t *testing.T) {
	workspace := "ordering-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	first, err := CreateSession(workspace, "first", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	second, err := CreateSession(workspace, "second", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Force distinct, known "updated" values so ordering is unambiguous:
	// second is older than first, even though second was created later.
	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`UPDATE sessions_meta SET updated = '2020-01-01T00:00:00Z' WHERE tableId = ?`, second.ActiveSessionID); err != nil {
		t.Fatalf("UPDATE second: %v", err)
	}
	if _, err := db.Exec(`UPDATE sessions_meta SET updated = '2030-01-01T00:00:00Z' WHERE tableId = ?`, first.ActiveSessionID); err != nil {
		t.Fatalf("UPDATE first: %v", err)
	}

	var wantUpdated string
	if err := db.QueryRow(`SELECT updated FROM sessions_meta WHERE tableId = ?`, second.ActiveSessionID).Scan(&wantUpdated); err != nil {
		t.Fatalf("SELECT updated: %v", err)
	}

	// Selecting the older session must not bump its "updated" value nor
	// reorder the list ahead of the newer one.
	state, err := SetActiveSession(workspace, second.ActiveSessionID, 24)
	if err != nil {
		t.Fatalf("SetActiveSession: %v", err)
	}

	if len(state.Sessions) != 2 {
		t.Fatalf("Sessions = %d, want 2", len(state.Sessions))
	}
	if state.Sessions[0].TableID != first.ActiveSessionID {
		t.Fatalf("Sessions[0].TableID = %d, want %d (most recently updated first)", state.Sessions[0].TableID, first.ActiveSessionID)
	}
	if state.Sessions[1].TableID != second.ActiveSessionID {
		t.Fatalf("Sessions[1].TableID = %d, want %d", state.Sessions[1].TableID, second.ActiveSessionID)
	}

	var gotUpdated string
	if err := db.QueryRow(`SELECT updated FROM sessions_meta WHERE tableId = ?`, second.ActiveSessionID).Scan(&gotUpdated); err != nil {
		t.Fatalf("SELECT updated after activation: %v", err)
	}
	if gotUpdated != wantUpdated {
		t.Fatalf("updated changed after SetActiveSession: got %q, want unchanged %q", gotUpdated, wantUpdated)
	}
}
