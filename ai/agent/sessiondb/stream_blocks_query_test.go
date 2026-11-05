package sessiondb

import (
	"os"
	"testing"
)

func TestStreamBlockQueriesReturnMetadataWithoutContent(t *testing.T) {
	workspace := "stream-block-query-test"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := CreateStreamPrompt(workspace, 1, 42, 7, "request", "02-11-26 13:15"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	if err := CreateStreamBlock(workspace, AIStreamBlock{
		SessionID: 1, PromptID: 42, RunID: 7, BlockID: "7-1", ParentID: "",
		Kind: StreamBlockToolOutput, Ordinal: 0, Status: "closed", Content: "private output",
	}, "start"); err != nil {
		t.Fatalf("CreateStreamBlock: %v", err)
	}

	prompts, err := ListStreamPromptMetas(workspace, 1)
	if err != nil {
		t.Fatalf("ListStreamPromptMetas: %v", err)
	}
	if len(prompts) != 1 || prompts[0].PromptID != 42 || prompts[0].Heading != "request" || prompts[0].Started != "02-11-26 13:15" {
		t.Fatalf("prompts = %+v, want one SQLite prompt metadata row", prompts)
	}

	meta, err := ListStreamBlockMeta(workspace, 1, 42)
	if err != nil {
		t.Fatalf("ListStreamBlockMeta: %v", err)
	}
	if len(meta) != 1 || meta[0].BlockID != "7-1" || meta[0].Size != int64(len("private output")) {
		t.Fatalf("metadata = %+v, want one block with its size", meta)
	}

	content, err := GetStreamBlockContent(workspace, 1, 42, "7-1")
	if err != nil {
		t.Fatalf("GetStreamBlockContent: %v", err)
	}
	if content != "private output" {
		t.Fatalf("content = %q, want %q", content, "private output")
	}
}

func TestLiveStreamBlockMetadataMarksOnlyInflightRunsActive(t *testing.T) {
	workspace := "stream-block-active-metadata-test"
	state, err := CreateSession(workspace, "", 24)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	t.Cleanup(func() {
		aiSessionLogStore.Lock()
		delete(aiSessionLogStore.byWorkspace, normalizeWorkspaceName(workspace))
		aiSessionLogStore.Unlock()
		_ = os.Remove(path)
	})

	if err := CreateStreamPrompt(workspace, state.ActiveSessionID, 0, 7, "request", "start"); err != nil {
		t.Fatalf("CreateStreamPrompt: %v", err)
	}
	if err := CreateStreamBlock(workspace, AIStreamBlock{
		SessionID: state.ActiveSessionID,
		RunID:     7,
		BlockID:   "7-1",
		Kind:      StreamBlockText,
		Status:    "open",
		Content:   "response",
	}, "start"); err != nil {
		t.Fatalf("CreateStreamBlock: %v", err)
	}
	aiSessionLogStore.Lock()
	aiSessionLogStore.byWorkspace[normalizeWorkspaceName(workspace)] = &sessionLogState{
		workspace: workspace, sessionID: state.ActiveSessionID, runID: 7, active: true,
	}
	aiSessionLogStore.Unlock()

	blocks, err := ListLiveStreamBlockMeta(workspace)
	if err != nil {
		t.Fatalf("ListLiveStreamBlockMeta active: %v", err)
	}
	if len(blocks) != 1 || !blocks[0].Active {
		t.Fatalf("active blocks = %+v, want one active block", blocks)
	}

	if err := FinalizeStreamPrompt(workspace, state.ActiveSessionID, 7, 42, "finish", PromptStatusSuccess); err != nil {
		t.Fatalf("FinalizeStreamPrompt: %v", err)
	}
	blocks, err = ListLiveStreamBlockMeta(workspace)
	if err != nil {
		t.Fatalf("ListLiveStreamBlockMeta before finish event: %v", err)
	}
	if len(blocks) != 1 || !blocks[0].Active {
		t.Fatalf("finalizing blocks = %+v, want active until finish event", blocks)
	}
	aiSessionLogStore.Lock()
	aiSessionLogStore.byWorkspace[normalizeWorkspaceName(workspace)].active = false
	aiSessionLogStore.Unlock()

	blocks, err = ListLiveStreamBlockMeta(workspace)
	if err != nil {
		t.Fatalf("ListLiveStreamBlockMeta completed: %v", err)
	}
	if len(blocks) != 1 || blocks[0].Active {
		t.Fatalf("completed blocks = %+v, want one inactive block", blocks)
	}
}
