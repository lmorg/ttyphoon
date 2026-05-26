package lsp

import "testing"

func TestApplyCodeActionResult_ApplyTextEditsChanged(t *testing.T) {
	content := "package main\nfunc main(){println(\"ok\")}\n"
	edits := []TextEdit{{NewText: "package main\n\nfunc main() { println(\"ok\") }\n"}}
	edits[0].Range.Start.Line = 0
	edits[0].Range.Start.Character = 0
	edits[0].Range.End.Line = 1
	edits[0].Range.End.Character = len("func main(){println(\"ok\")}")

	next := ApplyTextEdits(content, edits)
	if next == content {
		t.Fatalf("expected changed content")
	}
}
