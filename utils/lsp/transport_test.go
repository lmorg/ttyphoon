package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Content-Length framing
// ---------------------------------------------------------------------------

func TestWriteAndReadMessage_roundtrip(t *testing.T) {
	payload := Message{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  json.RawMessage(`{"uri":"file:///main.go"}`),
	}

	var buf bytes.Buffer
	if err := WriteMessage(&buf, payload); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	got, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	if got.Method != payload.Method {
		t.Errorf("Method: got %q, want %q", got.Method, payload.Method)
	}

	if string(got.Params) != string(payload.Params) {
		t.Errorf("Params: got %s, want %s", got.Params, payload.Params)
	}
}

func TestReadMessage_missingContentLength(t *testing.T) {
	// Header with no Content-Length.
	raw := "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n{}"
	_, err := ReadMessage(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for missing Content-Length, got nil")
	}
}

func TestReadMessage_invalidContentLength(t *testing.T) {
	raw := "Content-Length: notanumber\r\n\r\n"
	_, err := ReadMessage(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid Content-Length, got nil")
	}
}

func TestWriteMessage_multipleMessages(t *testing.T) {
	var buf bytes.Buffer

	for i := range 5 {
		msg := Message{
			JSONRPC: "2.0",
			Method:  fmt.Sprintf("method/%d", i),
		}
		if err := WriteMessage(&buf, msg); err != nil {
			t.Fatalf("WriteMessage[%d]: %v", i, err)
		}
	}

	reader := bufio.NewReader(&buf)
	for i := range 5 {
		got, err := ReadMessage(reader)
		if err != nil {
			t.Fatalf("ReadMessage[%d]: %v", i, err)
		}
		want := fmt.Sprintf("method/%d", i)
		if got.Method != want {
			t.Errorf("[%d] Method: got %q, want %q", i, got.Method, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Transport request correlation
// ---------------------------------------------------------------------------

// pipeTransport wires two in-memory pipes to simulate a client+server.
func pipeTransport(t *testing.T) (client *Transport, serverReader *bufio.Reader, serverWriter *bytes.Buffer) {
	t.Helper()

	// client writes to clientToServer; server reads from it.
	var clientToServer bytes.Buffer
	// server writes to serverToClient; client reads from it.
	var serverToClient bytes.Buffer

	client = NewTransport(&clientToServer, &serverToClient)

	return client, bufio.NewReader(&clientToServer), &serverToClient
}

func TestTransport_callReceivesMatchingResponse(t *testing.T) {
	// Use io.Pipe for proper synchronisation so client writes reach the server
	// goroutine without data races on shared bytes.Buffer.
	clientR, clientW := io.Pipe() // server reads client requests from clientR
	serverR, serverW := io.Pipe() // client reads server responses from serverR

	client := NewTransport(clientW, serverR)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = client.ReadLoop(ctx)
	}()

	// Simulate server: read one request, write matching response.
	go func() {
		defer clientR.Close()
		defer serverW.Close()
		req, err := ReadMessage(bufio.NewReader(clientR))
		if err != nil {
			return
		}
		resp := Message{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"ok":true}`),
		}
		_ = WriteMessage(serverW, resp)
	}()

	resp, err := client.Call(ctx, "workspace/symbol", map[string]string{"query": "main"}, time.Second)
	cancel()
	wg.Wait()

	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}

	if string(resp.Result) != `{"ok":true}` {
		t.Errorf("Result: got %s, want %s", resp.Result, `{"ok":true}`)
	}
}

func TestTransport_callTimesOut(t *testing.T) {
	var clientToServer, serverToClient bytes.Buffer
	client := NewTransport(&clientToServer, &serverToClient)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = client.ReadLoop(ctx)
	}()

	// Server never responds — call should time out.
	_, err := client.Call(ctx, "textDocument/hover", nil, 50*time.Millisecond)
	cancel()
	wg.Wait()

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestTransport_notifyDoesNotBlock(t *testing.T) {
	var buf bytes.Buffer
	client := NewTransport(&buf, strings.NewReader(""))

	err := client.Notify("textDocument/didChange", map[string]string{"uri": "file:///a.go"})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage after Notify: %v", err)
	}

	if msg.Method != "textDocument/didChange" {
		t.Errorf("Method: got %q, want textDocument/didChange", msg.Method)
	}

	if msg.ID != nil {
		t.Errorf("Notification must have no ID, got %s", *msg.ID)
	}
}

// ---------------------------------------------------------------------------
// Message helpers
// ---------------------------------------------------------------------------

func TestMessage_IsNotification(t *testing.T) {
	notif := Message{JSONRPC: "2.0", Method: "$/progress"}
	if !notif.IsNotification() {
		t.Error("expected IsNotification() = true")
	}

	id := json.RawMessage(`1`)
	resp := Message{JSONRPC: "2.0", ID: (*json.RawMessage)(&id)}
	if resp.IsNotification() {
		t.Error("response with ID should not be a notification")
	}
}

func TestMessage_IsResponse(t *testing.T) {
	id := json.RawMessage(`42`)
	resp := Message{JSONRPC: "2.0", ID: (*json.RawMessage)(&id), Result: json.RawMessage(`null`)}
	if !resp.IsResponse() {
		t.Error("expected IsResponse() = true")
	}

	notif := Message{JSONRPC: "2.0", Method: "$/progress"}
	if notif.IsResponse() {
		t.Error("notification should not be a response")
	}
}
