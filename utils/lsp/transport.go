package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ----------------------------------------------------------------------------
// JSON-RPC 2.0 wire types
// ----------------------------------------------------------------------------

// Message is the base JSON-RPC 2.0 envelope.
type Message struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("LSP RPC error %d: %s", e.Code, e.Message)
}

// IsNotification returns true when the message has no ID (server push).
func (m *Message) IsNotification() bool {
	return m.ID == nil && m.Method != ""
}

// IsResponse returns true when the message is a response to a prior request.
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == ""
}

// ----------------------------------------------------------------------------
// Content-Length framing
// ----------------------------------------------------------------------------

const (
	contentLengthHeader = "Content-Length"
	frameTerminator     = "\r\n\r\n"
)

// WriteMessage serialises msg as JSON and writes it with LSP Content-Length framing.
func WriteMessage(w io.Writer, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("lsp: marshal message: %w", err)
	}

	header := fmt.Sprintf("%s: %d%s", contentLengthHeader, len(body), frameTerminator)
	if _, err := io.WriteString(w, header); err != nil {
		return fmt.Errorf("lsp: write header: %w", err)
	}

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("lsp: write body: %w", err)
	}

	return nil
}

// ReadMessage reads one LSP-framed message from r.
func ReadMessage(r *bufio.Reader) (*Message, error) {
	contentLength := -1

	// Read headers terminated by blank line (\r\n\r\n).
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("lsp: read header line: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // blank line separates headers from body
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		if strings.TrimSpace(parts[0]) == contentLengthHeader {
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("lsp: invalid Content-Length %q: %w", parts[1], err)
			}

			contentLength = n
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("lsp: missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("lsp: read body: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("lsp: unmarshal message: %w", err)
	}

	return &msg, nil
}

// ----------------------------------------------------------------------------
// Pending request correlation
// ----------------------------------------------------------------------------

type pendingRequest struct {
	ch     chan *Message
	cancel context.CancelFunc
}

// Transport handles request correlation for a single server stdio connection.
type Transport struct {
	w        io.Writer
	r        *bufio.Reader
	mu       sync.Mutex
	pending  map[string]*pendingRequest
	nextID   atomic.Int64
	notifyCh chan *Message // incoming notifications / server pushes
}

// NewTransport creates a Transport connected to the given stdin/stdout streams.
func NewTransport(w io.Writer, r io.Reader) *Transport {
	return &Transport{
		w:        w,
		r:        bufio.NewReader(r),
		pending:  make(map[string]*pendingRequest),
		notifyCh: make(chan *Message, 64),
	}
}

// Notifications returns a channel that delivers all server-initiated messages
// (notifications and responses not matching a pending request).
func (t *Transport) Notifications() <-chan *Message {
	return t.notifyCh
}

// ReadLoop reads messages from the server and dispatches them.
// It blocks until ctx is cancelled or the reader returns an error.
func (t *Transport) ReadLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := ReadMessage(t.r)
		if err != nil {
			return err
		}

		if msg.IsResponse() {
			t.dispatch(msg)
			continue
		}

		// Notification or server request — non-blocking send, drop if full.
		select {
		case t.notifyCh <- msg:
		default:
		}
	}
}

// Call sends a request and waits for the matching response or context
// cancellation.
func (t *Transport) Call(ctx context.Context, method string, params any, timeout time.Duration) (*Message, error) {
	id := t.nextID.Add(1)
	idRaw, _ := json.Marshal(id)
	idKey := strconv.FormatInt(id, 10)

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("lsp: marshal params: %w", err)
	}

	msg := Message{
		JSONRPC: "2.0",
		ID:      (*json.RawMessage)(&idRaw),
		Method:  method,
		Params:  paramsRaw,
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	pr := &pendingRequest{ch: make(chan *Message, 1), cancel: cancel}

	t.mu.Lock()
	t.pending[idKey] = pr
	t.mu.Unlock()

	if err := t.writeMsg(msg); err != nil {
		t.mu.Lock()
		delete(t.pending, idKey)
		t.mu.Unlock()
		cancel()
		return nil, err
	}

	defer func() {
		t.mu.Lock()
		delete(t.pending, idKey)
		t.mu.Unlock()
		cancel()
	}()

	select {
	case resp := <-pr.ch:
		if resp.Error != nil {
			return resp, resp.Error
		}

		return resp, nil

	case <-callCtx.Done():
		// Send cancellation notification (best-effort).
		cancelMsg := Message{
			JSONRPC: "2.0",
			Method:  "$/cancelRequest",
			Params:  mustMarshal(map[string]any{"id": id}),
		}
		_ = t.writeMsg(cancelMsg)

		return nil, fmt.Errorf("lsp: call %q timed out or cancelled: %w", method, callCtx.Err())
	}
}

// Notify sends a JSON-RPC notification (no ID, no response expected).
func (t *Transport) Notify(method string, params any) error {
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("lsp: marshal params: %w", err)
	}

	return t.writeMsg(Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	})
}

func (t *Transport) dispatch(msg *Message) {
	if msg.ID == nil {
		return
	}

	var key string
	_ = json.Unmarshal(*msg.ID, &key)
	if key == "" {
		// ID may be encoded as a number.
		var n int64
		if err := json.Unmarshal(*msg.ID, &n); err == nil {
			key = strconv.FormatInt(n, 10)
		}
	}

	if key == "" {
		return
	}

	t.mu.Lock()
	pr, ok := t.pending[key]
	t.mu.Unlock()

	if ok {
		select {
		case pr.ch <- msg:
		default:
		}
	}
}

func (t *Transport) writeMsg(msg Message) error {
	return WriteMessage(t.w, msg)
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
