package mcp_client

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestOAuthCallbackCancellationReleasesListenerForRetry(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	redirect := "http://" + listener.Addr().String() + "/callback"
	listener.Close()
	for range 3 {
		server, err := StartOAuthCallbackServer(redirect)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, waitErr := server.WaitContext(ctx)
		server.Close()
		if !errors.Is(waitErr, context.Canceled) {
			t.Fatalf("callback cancellation returned %v", waitErr)
		}
	}
}
