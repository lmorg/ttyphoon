package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseInlayHintsResult_LabelParts(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"position":{"line":0,"character":5},
			"label":[{"value":": "},{"value":"string"}],
			"tooltip":{"value":"inferred type"},
			"kind":1,
			"paddingLeft":true
		}
	]`)

	items, err := parseInlayHintsResult(raw, "a😀bc", PositionEncodingUTF8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(items))
	}
	if items[0].Label != ": string" {
		t.Fatalf("label = %q", items[0].Label)
	}
	if items[0].Tooltip != "inferred type" {
		t.Fatalf("tooltip = %q", items[0].Tooltip)
	}
	if items[0].Character != 3 {
		t.Fatalf("character = %d, want 3", items[0].Character)
	}
	if !items[0].PaddingLeft || items[0].PaddingRight {
		t.Fatalf("unexpected padding flags: %+v", items[0])
	}
}

func TestParseInlayHintsResult_StringLabel(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"position":{"line":1,"character":4},
			"label":"name:",
			"kind":2,
			"paddingRight":true
		}
	]`)

	items, err := parseInlayHintsResult(raw, "func main() {\nname\n}\n", PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(items))
	}
	if items[0].Label != "name:" || items[0].Line != 1 || items[0].Character != 4 {
		t.Fatalf("unexpected parsed hint: %+v", items[0])
	}
	if items[0].Kind != 2 || !items[0].PaddingRight {
		t.Fatalf("unexpected hint metadata: %+v", items[0])
	}
}
