package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
)

func TestServerProcessEnsureInitialized(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	transport := NewTransport(clientToServerW, serverToClientR)
	sp := &ServerProcess{transport: transport}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = transport.ReadLoop(ctx)
	})

	wg.Go(func() {
		reader := bufio.NewReader(clientToServerR)

		initReq, err := ReadMessage(reader)
		if err != nil {
			t.Errorf("read initialize request: %v", err)
			return
		}
		if initReq.Method != "initialize" {
			t.Errorf("expected initialize method, got %q", initReq.Method)
			return
		}
		if initReq.ID == nil {
			t.Errorf("initialize request missing id")
			return
		}

		var initParams struct {
			Capabilities struct {
				TextDocument struct {
					SemanticTokens struct {
						TokenTypes     []string `json:"tokenTypes"`
						TokenModifiers []string `json:"tokenModifiers"`
					} `json:"semanticTokens"`
				} `json:"textDocument"`
			} `json:"capabilities"`
		}
		if err := json.Unmarshal(initReq.Params, &initParams); err != nil {
			t.Errorf("unmarshal initialize params: %v", err)
			return
		}
		if len(initParams.Capabilities.TextDocument.SemanticTokens.TokenTypes) == 0 {
			t.Errorf("initialize semanticTokens.tokenTypes should not be empty")
		}
		if len(initParams.Capabilities.TextDocument.SemanticTokens.TokenModifiers) == 0 {
			t.Errorf("initialize semanticTokens.tokenModifiers should not be empty")
		}

		resp := Message{JSONRPC: "2.0", ID: initReq.ID, Result: mustMarshal(map[string]any{})}
		if err := WriteMessage(serverToClientW, resp); err != nil {
			t.Errorf("write initialize response: %v", err)
			return
		}

		initializedMsg, err := ReadMessage(reader)
		if err != nil {
			t.Errorf("read initialized notify: %v", err)
			return
		}
		if initializedMsg.Method != "initialized" {
			t.Errorf("expected initialized method, got %q", initializedMsg.Method)
			return
		}
		if initializedMsg.ID != nil {
			t.Errorf("initialized should be notification")
		}
	})

	if err := sp.EnsureInitialized(context.Background(), "/tmp/workspace"); err != nil {
		t.Fatalf("EnsureInitialized failed: %v", err)
	}

	cancel()
	_ = clientToServerR.Close()
	_ = clientToServerW.Close()
	_ = serverToClientR.Close()
	_ = serverToClientW.Close()
	wg.Wait()
}

func TestServerProcessEnsureInitialized_NegotiatesPositionEncoding(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	transport := NewTransport(clientToServerW, serverToClientR)
	sp := &ServerProcess{transport: transport, positionEnc: PositionEncodingUTF16}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = transport.ReadLoop(ctx)
	})

	wg.Go(func() {
		reader := bufio.NewReader(clientToServerR)

		initReq, err := ReadMessage(reader)
		if err != nil {
			t.Errorf("read initialize request: %v", err)
			return
		}
		if initReq.Method != "initialize" || initReq.ID == nil {
			t.Errorf("unexpected initialize message: method=%q id=%v", initReq.Method, initReq.ID)
			return
		}

		result := map[string]any{
			"capabilities": map[string]any{
				"positionEncoding": string(PositionEncodingUTF8),
			},
		}
		raw, _ := json.Marshal(result)
		if err := WriteMessage(serverToClientW, Message{JSONRPC: "2.0", ID: initReq.ID, Result: raw}); err != nil {
			t.Errorf("write initialize response: %v", err)
			return
		}

		initializedMsg, err := ReadMessage(reader)
		if err != nil {
			t.Errorf("read initialized notify: %v", err)
			return
		}
		if initializedMsg.Method != "initialized" {
			t.Errorf("expected initialized method, got %q", initializedMsg.Method)
		}
	})

	if err := sp.EnsureInitialized(context.Background(), "/tmp/workspace"); err != nil {
		t.Fatalf("EnsureInitialized failed: %v", err)
	}

	if got := sp.PositionEncoding(); got != PositionEncodingUTF8 {
		t.Fatalf("PositionEncoding() = %q, want %q", got, PositionEncodingUTF8)
	}

	cancel()
	_ = clientToServerR.Close()
	_ = clientToServerW.Close()
	_ = serverToClientR.Close()
	_ = serverToClientW.Close()
	wg.Wait()
}
