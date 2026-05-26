package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseDefinitionResult_Location(t *testing.T) {
	raw := json.RawMessage(`{"uri":"file:///tmp/main.go","range":{"start":{"line":3,"character":5},"end":{"line":3,"character":11}}}`)
	items, err := parseDefinitionResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 location, got %d", len(items))
	}
	if items[0].Line != 3 || items[0].Character != 5 {
		t.Fatalf("unexpected location start: %+v", items[0])
	}
	if items[0].FilePath == "" {
		t.Fatalf("expected filepath to be derived from uri")
	}
}

func TestParseDefinitionResult_LocationLink(t *testing.T) {
	raw := json.RawMessage(`[{"targetUri":"file:///tmp/lib.go","targetRange":{"start":{"line":10,"character":1},"end":{"line":10,"character":8}},"targetSelectionRange":{"start":{"line":11,"character":2},"end":{"line":11,"character":9}}}]`)
	items, err := parseDefinitionResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 location, got %d", len(items))
	}
	if items[0].Line != 11 || items[0].Character != 2 {
		t.Fatalf("expected selection range start, got %+v", items[0])
	}
}
