package sessiondb

import (
	"os"
	"testing"
	"time"
)

func TestListPromptLogsMergesLegacyAndSQLitePrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspace := "mixed-prompt-history-test"
	state, err := CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	sessionID := state.ActiveSessionID
	legacyPath, err := sessionLogPromptPath(workspace, sessionID, 99)
	if err != nil {
		t.Fatalf("sessionLogPromptPath: %v", err)
	}
	if err := writePromptLogHeader(legacyPath, workspace, sessionID, time.Now()); err != nil {
		t.Fatalf("writePromptLogHeader: %v", err)
	}
	if err := appendSessionLog(legacyPath, "\n<!-- request heading: legacy prompt -->\n"); err != nil {
		t.Fatalf("appendSessionLog: %v", err)
	}
	_, promptID, err := AppendActiveSessionEntry(workspace, "new prompt", "", "", "response")
	if err != nil {
		t.Fatalf("AppendActiveSessionEntry: %v", err)
	}
	if err := CreateStreamPrompt(workspace, sessionID, promptID, 7, "SQLite prompt", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	if err := CreateStreamBlock(workspace, AIStreamBlock{
		SessionID: sessionID, PromptID: promptID, RunID: 7, BlockID: "7-1",
		Kind: StreamBlockText, Ordinal: 0, Status: "closed", Content: "SQLite content",
	}, "finish"); err != nil {
		t.Fatalf("CreateStreamBlock: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(legacyPath)
		if path, pathErr := dbPath(workspace); pathErr == nil {
			_ = os.Remove(path)
		}
	})

	metas := ListPromptLogs(workspace)
	if len(metas) != 2 || metas[0].Heading != "SQLite prompt" || metas[1].Heading != "legacy prompt" {
		t.Fatalf("merged prompt metadata = %+v", metas)
	}
}
