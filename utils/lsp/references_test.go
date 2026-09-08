package lsp

import (
	"encoding/json"
	"testing"
)

func TestReferenceLocationWireShape(t *testing.T) {
	raw := json.RawMessage(`[{"uri":"file:///tmp/main.go","range":{"start":{"line":3,"character":5},"end":{"line":3,"character":11}}}]`)
	var items []referenceLocationWire
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 1 || items[0].Range.Start.Character != 5 || items[0].Range.End.Character != 11 {
		t.Fatalf("unexpected references: %+v", items)
	}
}
