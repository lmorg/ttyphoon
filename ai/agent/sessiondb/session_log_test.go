package sessiondb

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/utils/file"
)

func TestPromptLogsUseOnlySQLiteAndLeaveRetiredFilesUntouched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspace := "sqlite-only-prompt-history-test"
	state, err := CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := state.ActiveSessionID
	dir := file.DocumentsDir(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, fmt.Sprintf("session-log.%s.%d.99.md", workspace, sessionID))
	const oldContent = "# Old transcript\n"
	if err := os.WriteFile(oldPath, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}
	var promptIDs []int64
	for i := 1; i <= 2; i++ {
		_, promptID, err := AppendActiveSessionEntry(workspace, fmt.Sprintf("prompt %d", i), "", "", "response")
		if err != nil {
			t.Fatal(err)
		}
		promptIDs = append(promptIDs, promptID)
		if err := CreateStreamPrompt(workspace, sessionID, promptID, uint64(i), "SQLite prompt", "start"); err != nil {
			t.Fatal(err)
		}
		if err := CreateStreamBlock(workspace, AIStreamBlock{
			SessionID: sessionID, PromptID: promptID, RunID: uint64(i), BlockID: fmt.Sprintf("%d-1", i),
			Kind: StreamBlockText, Status: "closed", Content: "SQLite content",
		}, "finish"); err != nil {
			t.Fatal(err)
		}
	}
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	metas := ListPromptLogs(workspace)
	if len(metas) != 2 || metas[0].PromptID != promptIDs[0] || metas[1].PromptID != promptIDs[1] {
		t.Fatalf("SQLite-only prompt metadata = %+v", metas)
	}
	if err := DeletePromptLog(workspace, sessionID, promptIDs[0]); err != nil {
		t.Fatal(err)
	}
	if metas := ListPromptLogs(workspace); len(metas) != 1 || metas[0].PromptID != promptIDs[1] {
		t.Fatalf("metadata after prompt deletion = %+v", metas)
	}
	blocks, err := ListStreamBlockMeta(workspace, sessionID, promptIDs[0])
	if err != nil || len(blocks) != 0 {
		t.Fatalf("deleted prompt blocks = %+v, error = %v", blocks, err)
	}
	if err := DeletePromptLog(workspace, sessionID, 999); err != nil {
		t.Fatal(err)
	}
	if err := ClearSessionLog(workspace, sessionID); err != nil {
		t.Fatal(err)
	}
	if metas := ListPromptLogs(workspace); len(metas) != 0 {
		t.Fatalf("metadata after session clearing = %+v", metas)
	}
	if blocks, err := ListStreamBlockMeta(workspace, sessionID, promptIDs[1]); err != nil || len(blocks) != 0 {
		t.Fatalf("cleared session blocks = %+v, error = %v", blocks, err)
	}
	if prompt, err := GetSessionPrompt(workspace, sessionID, promptIDs[0]); err != nil || prompt != "prompt 1" {
		t.Fatalf("session prompt after transcript cleanup = %q, error = %v", prompt, err)
	}
	history, err := GetFrontendState(workspace, 24)
	if err != nil || len(history.History) != 2 {
		t.Fatalf("session history after transcript cleanup = %+v, error = %v", history.History, err)
	}
	for _, entry := range history.History {
		if entry.Response != "response" {
			t.Fatalf("stored final response after transcript cleanup = %q", entry.Response)
		}
	}
	if content, err := os.ReadFile(oldPath); err != nil || string(content) != oldContent {
		t.Fatalf("retired transcript was changed: %q, error = %v", content, err)
	}
}
