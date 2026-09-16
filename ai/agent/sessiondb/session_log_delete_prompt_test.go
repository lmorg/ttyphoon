package sessiondb

import (
	"os"
	"testing"
	"time"
)

func TestDeletePromptLog_RemovesOnlyThatPromptFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	now := time.Now()
	if err := writePromptLogHeader(mustPromptLogPath(t, "ws", 1, 5), "ws", 1, now); err != nil {
		t.Fatalf("writePromptLogHeader: %v", err)
	}
	if err := writePromptLogHeader(mustPromptLogPath(t, "ws", 1, 6), "ws", 1, now); err != nil {
		t.Fatalf("writePromptLogHeader: %v", err)
	}

	if err := DeletePromptLog("ws", 1, 5); err != nil {
		t.Fatalf("DeletePromptLog: %v", err)
	}

	if _, err := os.Stat(mustPromptLogPath(t, "ws", 1, 5)); !os.IsNotExist(err) {
		t.Fatalf("prompt 5 log still exists: err=%v", err)
	}
	if _, err := os.Stat(mustPromptLogPath(t, "ws", 1, 6)); err != nil {
		t.Fatalf("prompt 6 log was removed: %v", err)
	}
}

func TestDeletePromptLog_MissingFileIsANoop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := DeletePromptLog("ws", 1, 999); err != nil {
		t.Fatalf("DeletePromptLog: %v", err)
	}
}

func mustPromptLogPath(t *testing.T, workspace string, sessionID, promptID int64) string {
	t.Helper()
	path, err := sessionLogPromptPath(workspace, sessionID, promptID)
	if err != nil {
		t.Fatalf("sessionLogPromptPath: %v", err)
	}
	return path
}
