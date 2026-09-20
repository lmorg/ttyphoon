package sessiondb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// EntryCount is read directly from sessions_meta rather than recomputed with a
// per-session COUNT(1) scan, so it must stay in sync with actual inserts.
// A stale/incorrect count would silently reintroduce the full-table-scan
// regression this maintains against (see migrateEntryCountColumn).
func TestAppendActiveSessionEntry_IncrementsEntryCount(t *testing.T) {
	workspace := "entrycount-append-test-workspace"
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

	if len(state.Sessions) != 1 {
		t.Fatalf("Sessions = %d, want 1", len(state.Sessions))
	}
	if state.Sessions[0].EntryCount != 3 {
		t.Fatalf("EntryCount = %d, want 3", state.Sessions[0].EntryCount)
	}
}

func TestClearActiveSession_ResetsEntryCount(t *testing.T) {
	workspace := "entrycount-clear-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, _, err := AppendActiveSessionEntry(workspace, "prompt", "", "", "response"); err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}

	state, err := ClearActiveSession(workspace, 24)
	if err != nil {
		t.Fatalf("ClearActiveSession: %v", err)
	}

	if len(state.Sessions) != 1 || state.Sessions[0].EntryCount != 0 {
		t.Fatalf("Sessions = %+v, want a single session with EntryCount 0", state.Sessions)
	}
}

// A database created before entryCount existed must be migrated once, not left
// to fall back to the slow per-session scan on every subsequent read.
func TestMigrateEntryCountColumn_BackfillsFromExistingHistory(t *testing.T) {
	workspace := "entrycount-migration-test-workspace"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	// Recreate the pre-migration schema (no entryCount column) and manually
	// populate a session with history rows, bypassing the current code path.
	if _, err := db.Exec(`
		CREATE TABLE sessions_meta (
			tableId INTEGER PRIMARY KEY AUTOINCREMENT,
			summary TEXT NOT NULL,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
		);
		CREATE UNIQUE INDEX sessions_meta_active_idx ON sessions_meta(active) WHERE active = 1;
		INSERT INTO sessions_meta (summary, created, updated, active) VALUES ('legacy', 'now', 'now', 1);
		CREATE TABLE "session_1" (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt TEXT NOT NULL,
			command_line TEXT NOT NULL,
			output_block TEXT NOT NULL,
			llm_response TEXT NOT NULL
		);
		INSERT INTO "session_1" (prompt, command_line, output_block, llm_response) VALUES ('a', '', '', ''), ('b', '', '', '');
	`); err != nil {
		t.Fatalf("seed legacy schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}

	state, err := GetFrontendState(workspace, 24)
	if err != nil {
		t.Fatalf("GetFrontendState: %v", err)
	}

	if len(state.Sessions) != 1 || state.Sessions[0].EntryCount != 2 {
		t.Fatalf("Sessions = %+v, want a single session backfilled with EntryCount 2", state.Sessions)
	}
}
