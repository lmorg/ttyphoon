package sessiondb

import (
	"database/sql"
	"os"
	"testing"
)

func TestStreamBlockLifecycle(t *testing.T) {
	workspace := "stream-block-lifecycle-test"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if err := initDB(db); err != nil {
		t.Fatalf("initDB: %v", err)
	}

	if err := CreateStreamPrompt(workspace, 1, 0, 7, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	if err := CreateStreamBlock(workspace, AIStreamBlock{
		SessionID: 1,
		RunID:     7,
		BlockID:   "7-1",
		Kind:      StreamBlockText,
		Ordinal:   1,
		Content:   "hello",
	}, "start"); err != nil {
		t.Fatalf("CreateStreamBlock: %v", err)
	}
	if err := AppendStreamBlock(workspace, 1, 7, "7-1", " world", "closed", "finish"); err != nil {
		t.Fatalf("AppendStreamBlock: %v", err)
	}
	if err := FinalizeStreamPrompt(workspace, 1, 7, 42, "finish"); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}

	var promptID, blockPromptID int64
	var content, status string
	if err := db.QueryRow(`SELECT promptId FROM stream_prompts WHERE sessionId = 1 AND promptId = 42`).Scan(&promptID); err != nil {
		t.Fatalf("read stream prompt: %v", err)
	}
	if err := db.QueryRow(`SELECT promptId, content, status FROM stream_blocks WHERE blockId = '7-1'`).Scan(&blockPromptID, &content, &status); err != nil {
		t.Fatalf("read stream block: %v", err)
	}
	if promptID != 42 || blockPromptID != 42 || content != "hello world" || status != "closed" {
		t.Fatalf("prompt=%d blockPrompt=%d content=%q status=%q", promptID, blockPromptID, content, status)
	}
}

func TestDatabaseAllocatesPersistentRunAndBlockIDs(t *testing.T) {
	workspace := "stream-database-owned-id-test"
	state, err := CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 7, "existing", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	firstRunID, err := AllocateStreamRunID(workspace, "next")
	if err != nil {
		t.Fatalf("AllocateStreamRunID first: %v", err)
	}
	secondRunID, err := AllocateStreamRunID(workspace, "later")
	if err != nil {
		t.Fatalf("AllocateStreamRunID second: %v", err)
	}
	if firstRunID <= 7 || secondRunID <= firstRunID {
		t.Fatalf("allocated run ids = %d, %d; want increasing IDs above existing run 7", firstRunID, secondRunID)
	}

	block := AIStreamBlock{
		SessionID: state.ActiveSessionID,
		RunID:     firstRunID,
		Kind:      StreamBlockToolCall,
		Status:    "open",
	}
	firstBlockID, err := CreateStreamBlockWithDatabaseID(workspace, block, "start")
	if err != nil {
		t.Fatalf("CreateStreamBlockWithDatabaseID first: %v", err)
	}
	secondBlockID, err := CreateStreamBlockWithDatabaseID(workspace, block, "start")
	if err != nil {
		t.Fatalf("CreateStreamBlockWithDatabaseID second: %v", err)
	}
	if firstBlockID == "" || secondBlockID == "" || firstBlockID == secondBlockID {
		t.Fatalf("allocated block ids = %q, %q; want distinct non-empty IDs", firstBlockID, secondBlockID)
	}
}
