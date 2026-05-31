package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestIntegration_DocumentLifecycleAndDiagnosticsRoundTrip(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	transport := NewTransport(clientToServerW, serverToClientR)
	sp := &ServerProcess{transport: transport, notifyCh: make(chan *Message, 16)}
	docs := NewDocumentStore()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer func() {
		_ = clientToServerR.Close()
		_ = clientToServerW.Close()
		_ = serverToClientR.Close()
		_ = serverToClientW.Close()
	}()

	go func() {
		_ = transport.ReadLoop(ctx)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-transport.Notifications():
				if msg == nil {
					continue
				}
				select {
				case sp.notifyCh <- msg:
				default:
				}
			}
		}
	}()

	fakeServerDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(clientToServerR)

		openMsg, err := ReadMessage(reader)
		if err != nil {
			fakeServerDone <- err
			return
		}
		if openMsg.Method != "textDocument/didOpen" {
			fakeServerDone <- errUnexpectedMethod(openMsg.Method, "textDocument/didOpen")
			return
		}

		changeMsg, err := ReadMessage(reader)
		if err != nil {
			fakeServerDone <- err
			return
		}
		if changeMsg.Method != "textDocument/didChange" {
			fakeServerDone <- errUnexpectedMethod(changeMsg.Method, "textDocument/didChange")
			return
		}

		closeMsg, err := ReadMessage(reader)
		if err != nil {
			fakeServerDone <- err
			return
		}
		if closeMsg.Method != "textDocument/didClose" {
			fakeServerDone <- errUnexpectedMethod(closeMsg.Method, "textDocument/didClose")
			return
		}

		params := publishDiagnosticsParams{
			URI: FilePathToURI("/proj/main.go"),
			Diagnostics: []Diagnostic{{
				Range: Range{
					Start: Position{Line: 1, Character: 0},
					End:   Position{Line: 1, Character: 7},
				},
				Severity: SeverityError,
				Message:  "undefined: foo",
			}},
		}
		raw, _ := json.Marshal(params)
		if err := WriteMessage(serverToClientW, Message{
			JSONRPC: "2.0",
			Method:  "textDocument/publishDiagnostics",
			Params:  raw,
		}); err != nil {
			fakeServerDone <- err
			return
		}

		fakeServerDone <- nil
	}()

	if err := docs.DidOpen(ctx, transport, "/proj/main.go", "go", "package main\nfunc main() {}\n"); err != nil {
		t.Fatalf("DidOpen: %v", err)
	}
	if err := docs.DidChange(ctx, transport, "/proj/main.go", "package main\nfunc main() { foo() }\n"); err != nil {
		t.Fatalf("DidChange: %v", err)
	}
	if err := docs.DidClose(ctx, transport, "/proj/main.go"); err != nil {
		t.Fatalf("DidClose: %v", err)
	}

	gotDiag := make(chan DiagnosticsPayload, 1)
	go ListenForDiagnostics(ctx, sp, nil, func(payload DiagnosticsPayload) {
		select {
		case gotDiag <- payload:
		default:
		}
	})

	select {
	case err := <-fakeServerDone:
		if err != nil {
			t.Fatalf("fake server failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for fake server flow")
	}

	select {
	case payload := <-gotDiag:
		if payload.URI != FilePathToURI("/proj/main.go") {
			t.Fatalf("diagnostic URI: got %q", payload.URI)
		}
		if len(payload.Diagnostics) != 1 {
			t.Fatalf("diagnostic count: got %d, want 1", len(payload.Diagnostics))
		}
		if payload.Diagnostics[0].Message != "undefined: foo" {
			t.Fatalf("diagnostic message: got %q", payload.Diagnostics[0].Message)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for diagnostics payload")
	}
}

type methodError struct {
	got  string
	want string
}

func (e methodError) Error() string {
	return "unexpected method: got " + e.got + ", want " + e.want
}

func errUnexpectedMethod(got, want string) error {
	return methodError{got: got, want: want}
}
