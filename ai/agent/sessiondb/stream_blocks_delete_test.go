package sessiondb

import (
	"os"
	"testing"
)

func TestDeleteActiveSessionEntry_RemovesStreamRows(t *testing.T) {
	workspace := "stream-block-delete-entry-test"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	sessionID, promptID, err := AppendActiveSessionEntry(workspace, "prompt", "", "", "response")
	if err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}
	if err := CreateStreamPrompt(workspace, sessionID, promptID, 7, "prompt", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	if err := CreateStreamBlock(workspace, AIStreamBlock{
		SessionID: sessionID, PromptID: promptID, RunID: 7, BlockID: "7-1",
		Kind: StreamBlockText, Ordinal: 0, Status: "closed", Content: "response",
	}, "finish"); err != nil {
		t.Fatalf("CreateStreamBlock: %v", err)
	}

	if _, err := DeleteActiveSessionEntry(workspace, promptID, 24); err != nil {
		t.Fatalf("DeleteActiveSessionEntry: %v", err)
	}
	if metas, err := ListStreamPromptMetas(workspace, sessionID); err != nil {
		t.Fatalf("ListStreamPromptMetas: %v", err)
	} else if len(metas) != 0 {
		t.Fatalf("stream prompts after deletion = %+v, want none", metas)
	}
	if _, err := GetStreamBlockContent(workspace, sessionID, promptID, "7-1"); err == nil {
		t.Fatal("GetStreamBlockContent succeeded after prompt deletion")
	}
}
