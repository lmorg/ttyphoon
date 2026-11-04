package sessiondb

import (
	"database/sql"
	"testing"
)

func TestInitStreamTablesCreatesSchema(t *testing.T) {
	db, err := sql.Open(driverName, "file:stream-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("initDB: %v", err)
	}
	if err := initDB(db); err != nil {
		t.Fatalf("second initDB: %v", err)
	}

	for _, table := range []string{"stream_prompts", "stream_run_ids", "stream_blocks"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatalf("lookup table %q: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %q count = %d, want 1", table, count)
		}
	}

	var uniqueIndexCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'sqlite_autoindex_stream_blocks_1'`).Scan(&uniqueIndexCount); err != nil {
		t.Fatalf("lookup stream block unique index: %v", err)
	}
	if uniqueIndexCount != 1 {
		t.Fatalf("stream block unique index count = %d, want 1", uniqueIndexCount)
	}

	for _, index := range []string{"stream_blocks_prompt_idx", "stream_blocks_parent_idx"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&count); err != nil {
			t.Fatalf("lookup index %q: %v", index, err)
		}
		if count != 1 {
			t.Fatalf("index %q count = %d, want 1", index, count)
		}
	}
}

func TestInitDBMigratesPromptOutcomeStatuses(t *testing.T) {
	db, err := sql.Open(driverName, "file:stream-prompt-status-migration?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE stream_prompts (
			sessionId INTEGER NOT NULL,
			promptId INTEGER NOT NULL,
			runId INTEGER NOT NULL,
			heading TEXT NOT NULL,
			started TEXT NOT NULL,
			finished TEXT NOT NULL DEFAULT '',
			blockCount INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (sessionId, promptId)
		);
		INSERT INTO stream_prompts (sessionId, promptId, runId, heading, started, finished)
		VALUES (1, 1, 1, 'finished', 'start', 'finish'), (1, 2, 2, 'active', 'start', '');
	`); err != nil {
		t.Fatalf("create legacy stream prompt schema: %v", err)
	}
	if err := initDB(db); err != nil {
		t.Fatalf("initDB: %v", err)
	}

	var finishedStatus, activeStatus string
	if err := db.QueryRow(`SELECT status FROM stream_prompts WHERE promptId = 1`).Scan(&finishedStatus); err != nil {
		t.Fatalf("read migrated finished status: %v", err)
	}
	if err := db.QueryRow(`SELECT status FROM stream_prompts WHERE promptId = 2`).Scan(&activeStatus); err != nil {
		t.Fatalf("read migrated active status: %v", err)
	}
	if finishedStatus != PromptStatusSuccess || activeStatus != PromptStatusRunning {
		t.Fatalf("migrated statuses = %q, %q; want success, running", finishedStatus, activeStatus)
	}
}
