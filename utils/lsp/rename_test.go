package lsp

import (
	"encoding/json"
	"testing"
)

func TestRenameResult_ApplyTextEditsChanged(t *testing.T) {
	content := "let value = 1\nlet output = value\n"

	var edit TextEdit
	edit.Range.Start.Line = 1
	edit.Range.Start.Character = 13
	edit.Range.End.Line = 1
	edit.Range.End.Character = 18
	edit.NewText = "renamed"

	next := ApplyTextEdits(content, []TextEdit{edit})
	if next == content {
		t.Fatalf("expected changed content")
	}
	if next != "let value = 1\nlet output = renamed\n" {
		t.Fatalf("unexpected rename content: %q", next)
	}
}

func TestParsePrepareRenameResult_RangeWithPlaceholder(t *testing.T) {
	raw := json.RawMessage(`{"range":{"start":{"line":1,"character":2},"end":{"line":1,"character":9}},"placeholder":"myVar"}`)
	got, err := parsePrepareRenameResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.CanRename {
		t.Fatalf("expected CanRename=true")
	}
	if got.Placeholder != "myVar" {
		t.Fatalf("unexpected placeholder: %q", got.Placeholder)
	}
}

func TestParsePrepareRenameResult_DefaultBehavior(t *testing.T) {
	raw := json.RawMessage(`{"defaultBehavior":true}`)
	got, err := parsePrepareRenameResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.CanRename {
		t.Fatalf("expected CanRename=true")
	}
}

func TestParsePrepareRenameResult_DirectRange(t *testing.T) {
	raw := json.RawMessage(`{"start":{"line":0,"character":0},"end":{"line":0,"character":7}}`)
	got, err := parsePrepareRenameResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.CanRename {
		t.Fatalf("expected CanRename=true")
	}
}
