package lsp

import (
	"encoding/json"
	"testing"
)

func TestWorkspaceEditEditsForURI_UsesChanges(t *testing.T) {
	var edit TextEdit
	edit.Range.Start.Line = 0
	edit.Range.Start.Character = 0
	edit.Range.End.Line = 0
	edit.Range.End.Character = 1
	edit.NewText = "X"

	w := workspaceEditWire{
		Changes: map[string][]TextEdit{
			"file:///tmp/main.go": {edit},
		},
	}

	got := w.editsForURI("file:///tmp/main.go")
	if len(got) != 1 {
		t.Fatalf("edits count = %d, want 1", len(got))
	}
	if got[0].NewText != "X" {
		t.Fatalf("unexpected edit newText: %q", got[0].NewText)
	}
}

func TestWorkspaceEditEditsForURI_UsesDocumentChanges(t *testing.T) {
	w := workspaceEditWire{}
	w.DocumentChanges = []json.RawMessage{
		json.RawMessage(`{"kind":"rename","oldUri":"file:///tmp/a.go","newUri":"file:///tmp/b.go"}`),
		json.RawMessage(`{"textDocument":{"uri":"file:///tmp/main.go","version":2},"edits":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"newText":"M"}]}`),
		json.RawMessage(`{"textDocument":{"uri":"file:///tmp/main.go","version":2},"edits":[{"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":1}},"newText":"N"}]}`),
		json.RawMessage(`{"textDocument":{"uri":"file:///tmp/other.go","version":1},"edits":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"newText":"Z"}]}`),
	}

	got := w.editsForURI("file:///tmp/main.go")
	if len(got) != 2 {
		t.Fatalf("edits count = %d, want 2", len(got))
	}
	if got[0].NewText != "M" || got[1].NewText != "N" {
		t.Fatalf("unexpected edit order/newText: %+v", got)
	}
}

func TestWorkspaceEditEditsForURI_DocumentChangesPreferred(t *testing.T) {
	w := workspaceEditWire{
		Changes: map[string][]TextEdit{
			"file:///tmp/main.go": {{NewText: "legacy"}},
		},
		DocumentChanges: []json.RawMessage{
			json.RawMessage(`{"textDocument":{"uri":"file:///tmp/main.go","version":2},"edits":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"newText":"preferred"}]}`),
		},
	}

	got := w.editsForURI("file:///tmp/main.go")
	if len(got) != 1 {
		t.Fatalf("edits count = %d, want 1", len(got))
	}
	if got[0].NewText != "preferred" {
		t.Fatalf("newText = %q, want preferred", got[0].NewText)
	}
}

func TestWorkspaceEditEditsForURI_AppliesToActiveContent(t *testing.T) {
	content := "foo\nbar\n"
	w := workspaceEditWire{}
	w.DocumentChanges = []json.RawMessage{
		json.RawMessage(`{"textDocument":{"uri":"file:///tmp/main.go","version":1},"edits":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}},"newText":"FOO"}]}`),
		json.RawMessage(`{"textDocument":{"uri":"file:///tmp/main.go","version":1},"edits":[{"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":3}},"newText":"BAR"}]}`),
	}

	next := ApplyTextEdits(content, w.editsForURI("file:///tmp/main.go"))
	if next != "FOO\nBAR\n" {
		t.Fatalf("updated content = %q, want %q", next, "FOO\\nBAR\\n")
	}
}
