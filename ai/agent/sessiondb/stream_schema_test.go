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

	for _, table := range []string{"stream_prompts", "stream_blocks"} {
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
