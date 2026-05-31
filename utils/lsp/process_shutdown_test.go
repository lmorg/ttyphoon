package lsp

import (
	"bufio"
	"context"
	"io"
	"sync"
	"testing"
)

func TestServerProcessStop_SendsShutdownAndExit(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	transport := NewTransport(clientToServerW, serverToClientR)
	sp := &ServerProcess{transport: transport, initialized: true, argv: []string{"fake-lsp"}}

	ctx, cancel := context.WithCancel(context.Background())
	sp.cancel = cancel

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = transport.ReadLoop(ctx)
	})

	serverDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(clientToServerR)

		shutdownReq, err := ReadMessage(reader)
		if err != nil {
			serverDone <- err
			return
		}
		if shutdownReq.Method != "shutdown" {
			serverDone <- methodError{got: shutdownReq.Method, want: "shutdown"}
			return
		}
		if shutdownReq.ID == nil {
			serverDone <- methodError{got: "<nil id>", want: "request id"}
			return
		}

		resp := Message{JSONRPC: "2.0", ID: shutdownReq.ID, Result: mustMarshal(map[string]any{})}
		if err := WriteMessage(serverToClientW, resp); err != nil {
			serverDone <- err
			return
		}

		exitMsg, err := ReadMessage(reader)
		if err != nil {
			serverDone <- err
			return
		}
		if exitMsg.Method != "exit" {
			serverDone <- methodError{got: exitMsg.Method, want: "exit"}
			return
		}
		if exitMsg.ID != nil {
			serverDone <- methodError{got: "request", want: "notification"}
			return
		}

		serverDone <- nil
	}()

	sp.Stop()

	if err := <-serverDone; err != nil {
		t.Fatalf("server flow failed: %v", err)
	}

	cancel()
	_ = clientToServerR.Close()
	_ = clientToServerW.Close()
	_ = serverToClientR.Close()
	_ = serverToClientW.Close()
	wg.Wait()
}
