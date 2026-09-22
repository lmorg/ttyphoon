package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/app"
)

func TestAIStreamBlockWriter_ConcurrentParentedBlocks(t *testing.T) {
	workspace := "stream-writer-concurrency-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := sessiondb.CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 99, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}

	var eventsMu sync.Mutex
	var events []sessiondb.AIStreamBlock
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     99,
		nextID:    1,
		emit: func(block sessiondb.AIStreamBlock) {
			eventsMu.Lock()
			events = append(events, block)
			eventsMu.Unlock()
		},
	}

	parent := writer.Open(sessiondb.StreamBlockSubagent, "")
	if parent == nil {
		t.Fatal("Open parent returned nil")
	}
	child := writer.Open(sessiondb.StreamBlockText, parent.block.BlockID)
	if child == nil {
		t.Fatal("Open child returned nil")
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			parent.Append("parent ")
		}()
		go func() {
			defer wg.Done()
			child.Append("child ")
		}()
	}
	wg.Wait()
	parent.Close()
	child.Close()

	if err := sessiondb.FinalizeStreamPrompt(workspace, state.ActiveSessionID, 99, 42, "finish"); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}

	blocks, err := sessiondb.ListStreamBlockMeta(workspace, state.ActiveSessionID, 42)
	if err != nil {
		t.Fatalf("ListStreamBlockMeta: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(blocks))
	}
	if blocks[1].ParentID != blocks[0].BlockID {
		t.Fatalf("child parent = %q, want %q", blocks[1].ParentID, blocks[0].BlockID)
	}

	parentContent, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 42, blocks[0].BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent parent: %v", err)
	}
	childContent, err := sessiondb.GetStreamBlockContent(workspace, state.ActiveSessionID, 42, blocks[1].BlockID)
	if err != nil {
		t.Fatalf("GetStreamBlockContent child: %v", err)
	}
	if parentContent != strings.Repeat("parent ", 20) {
		t.Fatalf("parent content = %q, want 20 parent chunks", parentContent)
	}
	if childContent != strings.Repeat("child ", 20) {
		t.Fatalf("child content = %q, want 20 child chunks", childContent)
	}
}

func TestAIStreamBlockWriter_SplitsThinkingAroundToolBlock(t *testing.T) {
	workspace := "stream-writer-thinking-boundary-test"
	state, err := sessiondb.CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	path := filepath.Join(home, "Documents", app.DirName, "session."+workspace+".db")
	t.Cleanup(func() { _ = os.Remove(path) })

	var events []sessiondb.AIStreamBlock
	writer := &aiStreamBlockWriter{
		workspace: workspace,
		sessionID: state.ActiveSessionID,
		runID:     100,
		nextID:    1,
		emit: func(block sessiondb.AIStreamBlock) {
			events = append(events, block)
		},
	}

	writer.AppendThinking("before tool")
	firstThinkingID := writer.thinking.block.BlockID
	tool := writer.Open(sessiondb.StreamBlockToolCall, "")
	if tool == nil {
		t.Fatal("Open tool returned nil")
	}
	tool.Append("input")
	tool.Close()
	writer.AppendThinking("after tool")

	if writer.thinking == nil || writer.thinking.block.BlockID == tool.block.BlockID {
		t.Fatal("thinking block was not reopened after the tool block")
	}
	closedBeforeTool := false
	for _, event := range events {
		if event.BlockID == firstThinkingID && event.Status == "closed" {
			closedBeforeTool = true
			break
		}
	}
	if !closedBeforeTool {
		t.Fatalf("events = %+v, want thinking close before tool block", events)
	}
}
