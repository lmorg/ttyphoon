package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestApplyTextEdits_SingleLine(t *testing.T) {
	content := "func main(){return}"
	edits := []TextEdit{
		{NewText: "func main() { return }"},
	}
	edits[0].Range.Start.Line = 0
	edits[0].Range.Start.Character = 0
	edits[0].Range.End.Line = 0
	edits[0].Range.End.Character = len(content)

	got := ApplyTextEdits(content, edits)
	if got != "func main() { return }" {
		t.Fatalf("unexpected formatted content: %q", got)
	}
}

func TestApplyTextEdits_MultipleEdits(t *testing.T) {
	content := "a\nb\nc"

	edits := make([]TextEdit, 0, 2)

	var e1 TextEdit
	e1.Range.Start.Line = 1
	e1.Range.Start.Character = 0
	e1.Range.End.Line = 1
	e1.Range.End.Character = 1
	e1.NewText = "B"
	edits = append(edits, e1)

	var e2 TextEdit
	e2.Range.Start.Line = 2
	e2.Range.Start.Character = 0
	e2.Range.End.Line = 2
	e2.Range.End.Character = 1
	e2.NewText = "C"
	edits = append(edits, e2)

	got := ApplyTextEdits(content, edits)
	if got != "a\nB\nC" {
		t.Fatalf("unexpected formatted content: %q", got)
	}
}

func TestRequestRangeFormatting_SendsRangeFormattingAndAppliesEdits(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	transport := NewTransport(clientToServerW, serverToClientR)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = transport.ReadLoop(ctx)
	}()

	done := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(clientToServerR)
		msg, err := ReadMessage(reader)
		if err != nil {
			done <- err
			return
		}
		if msg.Method != "textDocument/rangeFormatting" {
			done <- methodError{got: msg.Method, want: "textDocument/rangeFormatting"}
			return
		}

		var params struct {
			Range struct {
				Start struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"start"`
				End struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"end"`
			} `json:"range"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			done <- err
			return
		}
		if params.Range.Start.Line != 1 || params.Range.Start.Character != 0 {
			done <- methodError{got: "start range", want: "line 1 char 0"}
			return
		}
		if params.Range.End.Line != 1 || params.Range.End.Character != 26 {
			done <- methodError{got: "end range", want: "line 1 char 26"}
			return
		}

		edits := []TextEdit{{NewText: "func main() { println(\"ok\") }"}}
		edits[0].Range.Start.Line = 1
		edits[0].Range.Start.Character = 0
		edits[0].Range.End.Line = 1
		edits[0].Range.End.Character = 26

		resp := Message{JSONRPC: "2.0", ID: msg.ID, Result: mustMarshal(edits)}
		if err := WriteMessage(serverToClientW, resp); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	content := "package main\nfunc main(){println(\"ok\")}"
	result, err := RequestRangeFormatting(context.Background(), transport, "file:///tmp/main.go", content, 1, 0, 1, 26, PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("RequestRangeFormatting failed: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected result changed")
	}
	if result.Content != "package main\nfunc main() { println(\"ok\") }" {
		t.Fatalf("unexpected content: %q", result.Content)
	}

	if err := <-done; err != nil {
		t.Fatalf("server flow failed: %v", err)
	}

	_ = clientToServerR.Close()
	_ = clientToServerW.Close()
	_ = serverToClientR.Close()
	_ = serverToClientW.Close()
}
