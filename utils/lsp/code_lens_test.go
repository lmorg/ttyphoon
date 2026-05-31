package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseCodeLensResult_ConvertsServerUTF8Character(t *testing.T) {
	rawItems := []json.RawMessage{
		json.RawMessage(`{
			"range":{"start":{"line":0,"character":5},"end":{"line":0,"character":5}},
			"command":{"title":"Run lens","command":"example.run"}
		}`),
	}

	items, err := parseCodeLensResult(rawItems, "a😀bc", PositionEncodingUTF8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Character != 3 {
		t.Fatalf("character = %d, want 3", items[0].Character)
	}
	if items[0].Title != "Run lens" {
		t.Fatalf("title = %q", items[0].Title)
	}
	if items[0].Index != 0 {
		t.Fatalf("index = %d", items[0].Index)
	}
}

func TestParseCodeLensResult_UsesFallbackTitle(t *testing.T) {
	rawItems := []json.RawMessage{
		json.RawMessage(`{
			"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":0}}
		}`),
		json.RawMessage(`{
			"range":{"start":{"line":2,"character":1},"end":{"line":2,"character":1}},
			"command":{"command":"example.command"}
		}`),
	}

	items, err := parseCodeLensResult(rawItems, "line1\nline2\nline3", PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Title != "Code lens" || items[0].Index != 0 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Title != "example.command" || items[1].Index != 1 {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}
