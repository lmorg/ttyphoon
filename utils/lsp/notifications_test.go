package lsp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestParseLogMessage(t *testing.T) {
	payload, ok := parseLogMessage(json.RawMessage(`{"type":3,"message":"hello"}`))
	if !ok {
		t.Fatal("expected parse success")
	}
	if payload.Type != 3 {
		t.Fatalf("Type = %d, want 3", payload.Type)
	}
	if payload.Message != "hello" {
		t.Fatalf("Message = %q, want hello", payload.Message)
	}
}

func TestParseProgressMessage_StringToken(t *testing.T) {
	payload, ok := parseProgressMessage(json.RawMessage(`{"token":"abc","value":{"kind":"report","message":"working","percentage":50}}`))
	if !ok {
		t.Fatal("expected parse success")
	}
	if payload.Token != "abc" {
		t.Fatalf("Token = %q, want abc", payload.Token)
	}
	if payload.Kind != "report" {
		t.Fatalf("Kind = %q, want report", payload.Kind)
	}
	if payload.Percentage != 50 {
		t.Fatalf("Percentage = %d, want 50", payload.Percentage)
	}
}

func TestListenForNotifications_dispatchesKnownMethods(t *testing.T) {
	sp := &ServerProcess{notifyCh: make(chan *Message, 8)}

	diagRaw, _ := json.Marshal(publishDiagnosticsParams{
		URI:         "file:///x/y.go",
		Diagnostics: []Diagnostic{{Message: "boom", Severity: SeverityWarning}},
	})
	sp.notifyCh <- &Message{Method: "textDocument/publishDiagnostics", Params: diagRaw}
	sp.notifyCh <- &Message{Method: "window/logMessage", Params: json.RawMessage(`{"type":2,"message":"log"}`)}
	sp.notifyCh <- &Message{Method: "$/progress", Params: json.RawMessage(`{"token":"t1","value":{"kind":"begin","title":"Indexing"}}`)}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var mu sync.Mutex
	diagCount := 0
	logCount := 0
	progressCount := 0

	go ListenForNotifications(ctx, sp, nil,
		func(payload DiagnosticsPayload) {
			mu.Lock()
			diagCount++
			mu.Unlock()
		},
		func(payload LspProgressPayload) {
			mu.Lock()
			progressCount++
			mu.Unlock()
		},
		func(payload LspLogPayload) {
			mu.Lock()
			logCount++
			if diagCount > 0 && progressCount > 0 {
				cancel()
			}
			mu.Unlock()
		},
	)

	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()
	if diagCount != 1 {
		t.Fatalf("diagCount = %d, want 1", diagCount)
	}
	if logCount != 1 {
		t.Fatalf("logCount = %d, want 1", logCount)
	}
	if progressCount != 1 {
		t.Fatalf("progressCount = %d, want 1", progressCount)
	}
}
