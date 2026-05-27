package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseSemanticTokensResult_ConvertsServerUTF8CharacterAndLength(t *testing.T) {
	raw := json.RawMessage(`{
		"data": [
			0, 5, 4, 2, 0
		]
	}`)

	items, err := parseSemanticTokensResult(raw, "a😀beta", PositionEncodingUTF8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 token, got %d", len(items))
	}
	if items[0].Character != 3 {
		t.Fatalf("character = %d, want 3", items[0].Character)
	}
	if items[0].Length != 4 {
		t.Fatalf("length = %d, want 4", items[0].Length)
	}
	if items[0].TokenType != 2 {
		t.Fatalf("token type = %d", items[0].TokenType)
	}
}

func TestParseSemanticTokensResult_DeltaLineAndDeltaStart(t *testing.T) {
	raw := json.RawMessage(`{
		"data": [
			0, 0, 4, 1, 0,
			1, 2, 3, 3, 5
		]
	}`)

	items, err := parseSemanticTokensResult(raw, "name\n  val\n", PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(items))
	}

	if items[0].Line != 0 || items[0].Character != 0 || items[0].Length != 4 {
		t.Fatalf("unexpected token[0]: %+v", items[0])
	}
	if items[1].Line != 1 || items[1].Character != 2 || items[1].Length != 3 {
		t.Fatalf("unexpected token[1]: %+v", items[1])
	}
	if items[1].TokenType != 3 || items[1].TokenMods != 5 {
		t.Fatalf("unexpected token[1] metadata: %+v", items[1])
	}
}
