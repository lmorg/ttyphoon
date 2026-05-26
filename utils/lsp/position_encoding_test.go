package lsp

import (
	"encoding/json"
	"testing"
)

func TestConvertCharacterAtLine_UTF16ToUTF8AndBack(t *testing.T) {
	content := "a😀z\n"

	serverChar := convertCharacterAtLine(content, 0, 3, PositionEncodingUTF16, PositionEncodingUTF8)
	if serverChar != 5 {
		t.Fatalf("utf16->utf8 character = %d, want 5", serverChar)
	}

	clientChar := convertCharacterAtLine(content, 0, serverChar, PositionEncodingUTF8, PositionEncodingUTF16)
	if clientChar != 3 {
		t.Fatalf("utf8->utf16 character = %d, want 3", clientChar)
	}
}

func TestConvertTextEdits_UTF8ToUTF16(t *testing.T) {
	content := "a😀z"
	edits := []TextEdit{{NewText: "X"}}
	edits[0].Range.Start.Line = 0
	edits[0].Range.Start.Character = 5 // utf-8 byte offset before trailing z
	edits[0].Range.End.Line = 0
	edits[0].Range.End.Character = 6

	converted := convertTextEdits(content, edits, PositionEncodingUTF8, PositionEncodingUTF16)
	if converted[0].Range.Start.Character != 3 {
		t.Fatalf("converted start = %d, want 3", converted[0].Range.Start.Character)
	}
	if converted[0].Range.End.Character != 4 {
		t.Fatalf("converted end = %d, want 4", converted[0].Range.End.Character)
	}

	got := ApplyTextEdits(content, converted)
	if got != "a😀X" {
		t.Fatalf("ApplyTextEdits result = %q, want %q", got, "a😀X")
	}
}

func TestParseInitializePositionEncoding_DefaultAndExplicit(t *testing.T) {
	if got := parseInitializePositionEncoding(json.RawMessage(`{"capabilities":{}}`)); got != PositionEncodingUTF16 {
		t.Fatalf("default parse = %q, want utf-16", got)
	}
	if got := parseInitializePositionEncoding(json.RawMessage(`{"capabilities":{"positionEncoding":"utf-8"}}`)); got != PositionEncodingUTF8 {
		t.Fatalf("explicit parse = %q, want utf-8", got)
	}
}
