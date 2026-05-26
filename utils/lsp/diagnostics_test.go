package lsp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestDispatchDiagnostics_callsEmitter(t *testing.T) {
	params := publishDiagnosticsParams{
		URI: "file:///proj/main.go",
		Diagnostics: []Diagnostic{
			{
				Range:    Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 5}},
				Severity: SeverityError,
				Message:  "undefined: foo",
			},
		},
	}

	raw, _ := json.Marshal(params)
	msg := &Message{Method: "textDocument/publishDiagnostics", Params: raw}

	var got DiagnosticsPayload
	dispatchDiagnostics(msg, PositionEncodingUTF16, nil, func(p DiagnosticsPayload) { got = p })

	if got.URI != "file:///proj/main.go" {
		t.Errorf("URI: got %q, want file:///proj/main.go", got.URI)
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("Diagnostics count: got %d, want 1", len(got.Diagnostics))
	}
	if got.Diagnostics[0].Message != "undefined: foo" {
		t.Errorf("Message: got %q", got.Diagnostics[0].Message)
	}
	if got.Diagnostics[0].Severity != SeverityError {
		t.Errorf("Severity: got %d, want %d", got.Diagnostics[0].Severity, SeverityError)
	}
}

func TestDispatchDiagnostics_emptyList(t *testing.T) {
	params := publishDiagnosticsParams{URI: "file:///proj/clean.go", Diagnostics: nil}
	raw, _ := json.Marshal(params)
	msg := &Message{Method: "textDocument/publishDiagnostics", Params: raw}

	called := false
	dispatchDiagnostics(msg, PositionEncodingUTF16, nil, func(p DiagnosticsPayload) { called = true })
	if !called {
		t.Error("emitter should be called even for empty diagnostics list")
	}
}

func TestDispatchDiagnostics_invalidJSON(t *testing.T) {
	msg := &Message{Method: "textDocument/publishDiagnostics", Params: json.RawMessage(`not json`)}
	called := false
	dispatchDiagnostics(msg, PositionEncodingUTF16, nil, func(p DiagnosticsPayload) { called = true })
	if called {
		t.Error("emitter should not be called on invalid JSON")
	}
}

func TestDispatchDiagnostics_convertsServerUTF8ToClientUTF16(t *testing.T) {
	params := publishDiagnosticsParams{
		URI: "file:///proj/main.go",
		Diagnostics: []Diagnostic{
			{
				Range:    Range{Start: Position{Line: 0, Character: 5}, End: Position{Line: 0, Character: 6}},
				Severity: SeverityWarning,
				Message:  "test",
			},
		},
	}

	raw, _ := json.Marshal(params)
	msg := &Message{Method: "textDocument/publishDiagnostics", Params: raw}

	var got DiagnosticsPayload
	dispatchDiagnostics(msg, PositionEncodingUTF8, func(uri string) (string, bool) {
		if uri != "file:///proj/main.go" {
			return "", false
		}
		return "a😀z\n", true
	}, func(p DiagnosticsPayload) {
		got = p
	})

	if len(got.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(got.Diagnostics))
	}
	if got.Diagnostics[0].Range.Start.Character != 3 {
		t.Fatalf("start char = %d, want 3", got.Diagnostics[0].Range.Start.Character)
	}
	if got.Diagnostics[0].Range.End.Character != 4 {
		t.Fatalf("end char = %d, want 4", got.Diagnostics[0].Range.End.Character)
	}
}

func TestListenForDiagnostics_receivesViaChannel(t *testing.T) {
	sp := &ServerProcess{notifyCh: make(chan *Message, 4)}

	params := publishDiagnosticsParams{
		URI:         "file:///x/y.go",
		Diagnostics: []Diagnostic{{Message: "boom", Severity: SeverityWarning}},
	}
	raw, _ := json.Marshal(params)
	sp.notifyCh <- &Message{Method: "textDocument/publishDiagnostics", Params: raw}
	// Non-diagnostics notifications must be silently ignored.
	sp.notifyCh <- &Message{Method: "window/logMessage", Params: json.RawMessage(`{"message":"hi"}`)}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var mu sync.Mutex
	var received []DiagnosticsPayload

	go ListenForDiagnostics(ctx, sp, nil, func(p DiagnosticsPayload) {
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		cancel() // stop after first diagnostic
	})

	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 diagnostic payload, got %d", len(received))
	}
	if received[0].URI != "file:///x/y.go" {
		t.Errorf("URI: got %q", received[0].URI)
	}
}

func TestManager_has(t *testing.T) {
	m := NewManager()
	if m.Has("root", "go") {
		t.Error("Has should return false for unknown key")
	}
}
