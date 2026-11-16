package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestLSPEnvironmentHelper(t *testing.T) {
	if os.Getenv("TTYPHOON_LSP_TEST_HELPER") != "1" {
		return
	}
	reader := bufio.NewReader(os.Stdin)
	var options any
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			os.Exit(1)
		}
		switch msg.Method {
		case "initialize":
			var params struct {
				Options any `json:"initializationOptions"`
			}
			if json.Unmarshal(msg.Params, &params) != nil {
				os.Exit(2)
			}
			options = params.Options
			fmt.Fprintln(os.Stderr, os.Getenv("TTYPHOON_LSP_SHARED"))
			_ = WriteMessage(os.Stdout, Message{JSONRPC: "2.0", Method: "window/logMessage", Params: mustMarshal(map[string]any{"type": 3, "message": os.Getenv("TTYPHOON_LSP_SHARED")})})
			_ = WriteMessage(os.Stdout, Message{JSONRPC: "2.0", ID: msg.ID, Result: mustMarshal(map[string]any{})})
		case "test/inspect":
			only, present := os.LookupEnv("TTYPHOON_LSP_ONLY")
			_ = WriteMessage(os.Stdout, Message{JSONRPC: "2.0", ID: msg.ID, Result: mustMarshal(map[string]any{
				"shared": os.Getenv("TTYPHOON_LSP_SHARED"), "empty": os.Getenv("TTYPHOON_LSP_EMPTY"),
				"host": os.Getenv("TTYPHOON_LSP_HOST"), "only": only, "onlyPresent": present, "options": options,
			})})
		case "test/error":
			_ = WriteMessage(os.Stdout, Message{JSONRPC: "2.0", ID: msg.ID, Error: &RPCError{Code: -32000, Message: os.Getenv("TTYPHOON_LSP_SHARED")}})
		case "shutdown":
			_ = WriteMessage(os.Stdout, Message{JSONRPC: "2.0", ID: msg.ID, Result: mustMarshal(nil)})
		case "exit":
			os.Exit(0)
		}
	}
}

func TestServerProcessScopedEnvironmentAndRestart(t *testing.T) {
	var logs lockedLogBuffer
	oldOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(oldOutput) })
	t.Setenv("TTYPHOON_LSP_TEST_HELPER", "1")
	t.Setenv("TTYPHOON_LSP_SHARED", "host-shared")
	t.Setenv("TTYPHOON_LSP_EMPTY", "host-nonempty")
	t.Setenv("TTYPHOON_LSP_HOST", "host-only")
	t.Setenv("TTYPHOON_LSP_ONLY", "")
	if err := os.Unsetenv("TTYPHOON_LSP_ONLY"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.vars"), filepath.Join(dir, "second.vars")
	write := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(first, "TTYPHOON_LSP_SHARED=first-value\nTTYPHOON_LSP_ONLY=scoped-only\nTTYPHOON_LSP_EMPTY=\n")
	write(second, "TTYPHOON_LSP_SHARED=second-value\n")
	options := map[string]any{"nested": []any{"$TTYPHOON_LSP_SHARED", "$TTYPHOON_LSP_EMPTY", "$TTYPHOON_LSP_HOST", "$TTYPHOON_LSP_ONLY"}}
	argv := []string{os.Args[0], "-test.run=^TestLSPEnvironmentHelper$"}
	manager := NewManager()
	t.Cleanup(manager.StopAll)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sp, err := manager.GetOrStart(ctx, dir, "one", argv, options, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	other, err := manager.GetOrStart(ctx, dir, "two", argv, options, nil)
	if err != nil {
		t.Fatal(err)
	}

	// A file change after process creation must not change its handshake.
	write(second, "TTYPHOON_LSP_SHARED=restart-value\n")
	check := func(server *ServerProcess, wantShared, wantEmpty, wantOnly string, onlyPresent bool) {
		t.Helper()
		if err := server.EnsureInitialized(ctx, dir); err != nil {
			t.Fatal(err)
		}
		response, err := server.Transport().Call(ctx, "test/inspect", nil, 2*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Shared, Empty, Host, Only string
			OnlyPresent               bool
			Options                   struct{ Nested []string }
		}
		if err := json.Unmarshal(response.Result, &got); err != nil {
			t.Fatal(err)
		}
		if got.Shared != wantShared || got.Empty != wantEmpty || got.Host != "host-only" || got.Only != wantOnly || got.OnlyPresent != onlyPresent {
			t.Fatal("server received the wrong environment")
		}
		want := []string{wantShared, wantEmpty, "host-only", wantOnly}
		if len(got.Options.Nested) != len(want) {
			t.Fatal("server received the wrong initialization options")
		}
		for i := range want {
			if got.Options.Nested[i] != want[i] {
				t.Fatal("initialization options did not match the process environment")
			}
		}
	}
	check(sp, "second-value", "", "scoped-only", true)
	check(other, "host-shared", "host-nonempty", "", false)
	if os.Getenv("TTYPHOON_LSP_SHARED") != "host-shared" {
		t.Fatal("scoped environment changed the host")
	}

	select {
	case notification := <-sp.Notifications():
		if strings.Contains(string(notification.Params), "second-value") {
			t.Fatal("scoped value leaked to a log notification")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("missing log notification")
	}
	_, err = sp.Transport().Call(ctx, "test/error", nil, 2*time.Second)
	if err == nil || strings.Contains(err.Error(), "second-value") {
		t.Fatal("scoped value leaked in an RPC error")
	}

	sp.mu.Lock()
	oldCmd := sp.cmd
	sp.mu.Unlock()
	if err := oldCmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		sp.mu.Lock()
		restarted := sp.cmd != oldCmd
		sp.mu.Unlock()
		if restarted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not restart")
		}
		time.Sleep(20 * time.Millisecond)
	}
	check(sp, "restart-value", "", "scoped-only", true)
	deadline = time.Now().Add(2 * time.Second)
	for !strings.Contains(logs.String(), "[REDACTED]") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	output := logs.String()
	if !strings.Contains(output, "[REDACTED]") || strings.Contains(output, "second-value") || strings.Contains(output, "restart-value") {
		t.Fatal("scoped values were not redacted from server stderr")
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	sp.mu.Lock()
	restartedCmd := sp.cmd
	sp.mu.Unlock()
	if err := restartedCmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(4 * time.Second)
	for !strings.Contains(logs.String(), "restart failed:") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(logs.String(), "restart failed:") {
		t.Fatal("environment file error during restart was not reported")
	}
}

func TestServerProcessRejectsInvalidEnvironmentFile(t *testing.T) {
	manager := NewManager()
	_, err := manager.GetOrStart(context.Background(), t.TempDir(), "test", []string{os.Args[0]}, nil, []string{filepath.Join(t.TempDir(), "missing.vars")})
	if err == nil || !strings.Contains(err.Error(), "environment file") {
		t.Fatal("missing environment file did not fail server startup")
	}
}
