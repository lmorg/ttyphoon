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

	if err := CreateStreamPrompt(workspace, 1, 42, 7, "request", "start"); err != nil {
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
	if len(prompts) != 1 || prompts[0].PromptID != 42 || prompts[0].Heading != "request" {
		t.Fatalf("prompts = %+v, want one SQLite prompt metadata row", prompts)
	}

	promptContent, err := GetStreamPromptContent(workspace, 1, 42)
	if err != nil {
		t.Fatalf("GetStreamPromptContent: %v", err)
	}
	if promptContent != "private output" {
		t.Fatalf("prompt content = %q, want %q", promptContent, "private output")
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
