package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

// backoff constants for restart.
const (
	backoffMin        = 500 * time.Millisecond
	backoffMax        = 30 * time.Second
	backoffMultiplier = 2
)

// ServerProcess manages a single language-server OS process.
// It launches the server, wires stdio to a Transport, and restarts on crash.
type ServerProcess struct {
	argv        []string
	initOptions map[string]any
	transport   *Transport

	mu          sync.Mutex
	cmd         *exec.Cmd
	ctx         context.Context
	cancel      context.CancelFunc
	restarts    int
	stopped     bool
	initialized bool
	positionEnc PositionEncoding
	initMu      sync.Mutex

	notifyCh chan *Message // re-exported from transport
}

// NewServerProcess creates a ServerProcess but does not start it yet.
// initOptions is optional; when non-empty it is sent as initializationOptions
// during the LSP initialize handshake (env vars in string values are expanded).
func NewServerProcess(argv []string, initOptions map[string]any) (*ServerProcess, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("lsp: argv must not be empty")
	}

	return &ServerProcess{
		argv:        argv,
		initOptions: initOptions,
		notifyCh:    make(chan *Message, 64),
		positionEnc: PositionEncodingUTF16,
	}, nil
}

// expandEnvVarsInOptions recursively expands environment variables in string
// values within a map[string]any tree, leaving non-string values untouched.
func expandEnvVarsInOptions(v any) any {
	switch val := v.(type) {
	case string:
		return os.ExpandEnv(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			out[k] = expandEnvVarsInOptions(v2)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			out[i] = expandEnvVarsInOptions(v2)
		}
		return out
	default:
		return v
	}
}

// Start launches the language server and begins the read loop.
// Subsequent calls after a crash restart the process automatically (see Run).
func (sp *ServerProcess) Start(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.stopped {
		return fmt.Errorf("lsp: server process has been permanently stopped")
	}

	return sp.startLocked(ctx)
}

func (sp *ServerProcess) startLocked(ctx context.Context) error {
	log.Println("[info] starting lsp: ", sp.argv)
	procCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(procCtx, sp.argv[0], sp.argv[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("lsp: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("lsp: stdout pipe: %w", err)
	}

	// stderr is captured and forwarded to the log.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("lsp: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("lsp: start %q: %w", sp.argv[0], err)
	}

	transport := NewTransport(stdin, stdout)

	sp.cmd = cmd
	sp.ctx = procCtx
	sp.cancel = cancel
	sp.transport = transport
	sp.initialized = false
	sp.positionEnc = PositionEncodingUTF16

	// Forward server notifications to our own channel.
	go func() {
		for msg := range transport.Notifications() {
			select {
			case sp.notifyCh <- msg:
			default:
			}
		}
	}()

	// Drain stderr to log.
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[trace] lsp %s stderr: %s", sp.argv[0], scanner.Text())
		}
	}()

	// Read loop — restarts on exit unless permanently stopped.
	go func() {
		readErr := transport.ReadLoop(procCtx)
		cmd.Wait() //nolint:errcheck

		sp.mu.Lock()
		stopped := sp.stopped
		sp.mu.Unlock()

		if stopped {
			return
		}

		if readErr != nil {
			log.Printf("[warn] lsp[%s] read loop exited: %v — will restart", sp.argv[0], readErr)
		} else {
			log.Printf("[warn] lsp[%s] process exited — will restart", sp.argv[0])
		}

		sp.restartWithBackoff(ctx)
	}()

	return nil
}

func (sp *ServerProcess) restartWithBackoff(ctx context.Context) {
	sp.mu.Lock()
	if sp.stopped {
		sp.mu.Unlock()
		return
	}
	sp.restarts++
	attempt := sp.restarts
	sp.mu.Unlock()

	delay := backoffMin
	for i := 1; i < attempt; i++ {
		delay *= backoffMultiplier
		if delay > backoffMax {
			delay = backoffMax
			break
		}
	}

	log.Printf("[info] lsp[%s] restart #%d in %s", sp.argv[0], attempt, delay)

	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.stopped {
		return
	}

	if err := sp.startLocked(ctx); err != nil {
		log.Printf("[error] lsp[%s] restart failed: %v", sp.argv[0], err)
	}
}

// Stop permanently shuts down the process. Safe to call multiple times.
func (sp *ServerProcess) Stop() {
	sp.mu.Lock()
	if sp.stopped {
		sp.mu.Unlock()
		return
	}

	sp.stopped = true
	t := sp.transport
	initialized := sp.initialized
	cancel := sp.cancel
	sp.cancel = nil
	sp.mu.Unlock()

	if initialized && t != nil {
		ctx, cancelShutdown := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := t.Call(ctx, "shutdown", map[string]any{}, 400*time.Millisecond)
		if err != nil {
			log.Printf("[error] lsp[%s] shutdown request failed: %v", sp.argv[0], err)
		}
		if err := t.Notify("exit", map[string]any{}); err != nil {
			log.Printf("[error] lsp[%s] exit notify failed: %v", sp.argv[0], err)
		}
		cancelShutdown()
	}

	if cancel != nil {
		cancel()
	}
}

// Transport returns the active Transport, or nil if not yet started.
func (sp *ServerProcess) Transport() *Transport {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.transport
}

// Notifications returns the channel of server-initiated messages.
func (sp *ServerProcess) Notifications() <-chan *Message {
	return sp.notifyCh
}

// EnsureInitialized sends initialize/initialized once per active process.
// Safe to call multiple times.
func (sp *ServerProcess) EnsureInitialized(ctx context.Context, workspaceRoot string) error {
	sp.initMu.Lock()
	defer sp.initMu.Unlock()

	sp.mu.Lock()
	if sp.stopped {
		sp.mu.Unlock()
		return fmt.Errorf("lsp: server process has been permanently stopped")
	}

	if sp.initialized {
		sp.mu.Unlock()
		return nil
	}

	t := sp.transport
	sp.mu.Unlock()

	if t == nil {
		return fmt.Errorf("lsp: transport is not available")
	}

	params := map[string]any{
		"processId": os.Getpid(),
		"rootUri":   FilePathToURI(workspaceRoot),
		"capabilities": map[string]any{
			"general": map[string]any{
				"positionEncodings": []string{string(PositionEncodingUTF16), string(PositionEncodingUTF8)},
			},
			"textDocument": map[string]any{
				"inlayHint": map[string]any{
					"dynamicRegistration": false,
				},
				"semanticTokens": map[string]any{
					"dynamicRegistration": false,
					"tokenTypes": []string{
						"namespace", "type", "class", "enum", "interface", "struct", "typeParameter",
						"parameter", "variable", "property", "enumMember", "event", "function", "method",
						"macro", "keyword", "modifier", "comment", "string", "number", "regexp", "operator",
					},
					"tokenModifiers": []string{
						"declaration", "definition", "readonly", "static", "deprecated",
						"abstract", "async", "modification", "documentation", "defaultLibrary",
					},
					"requests": map[string]any{
						"full": true,
					},
					"formats":                 []string{"relative"},
					"multilineTokenSupport":   false,
					"overlappingTokenSupport": true,
				},
			},
			"workspace": map[string]any{},
		},
		"offsetEncoding": []string{string(PositionEncodingUTF16), string(PositionEncodingUTF8)},
		"clientInfo": map[string]any{
			"name": app.Name(),
		},
	}

	if len(sp.initOptions) > 0 {
		params["initializationOptions"] = expandEnvVarsInOptions(sp.initOptions)
	}

	resp, err := t.Call(ctx, "initialize", params, 8*time.Second)
	if err != nil {
		return fmt.Errorf("lsp: initialize failed: %w", err)
	}

	positionEnc := PositionEncodingUTF16
	if resp != nil && len(resp.Result) > 0 && string(resp.Result) != "null" {
		positionEnc = parseInitializePositionEncoding(resp.Result)
	}

	if err := t.Notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("lsp: initialized notify failed: %w", err)
	}

	sp.mu.Lock()
	if sp.transport == t {
		sp.initialized = true
		sp.positionEnc = positionEnc
	}
	sp.mu.Unlock()

	return nil
}

// PositionEncoding returns the currently negotiated server position encoding.
func (sp *ServerProcess) PositionEncoding() PositionEncoding {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.positionEnc == "" {
		return PositionEncodingUTF16
	}
	return sp.positionEnc
}

func parseInitializePositionEncoding(raw json.RawMessage) PositionEncoding {
	var result struct {
		Capabilities struct {
			PositionEncoding string `json:"positionEncoding"`
		} `json:"capabilities"`
		OffsetEncoding string `json:"offsetEncoding"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return PositionEncodingUTF16
	}

	if enc := normalizePositionEncoding(result.Capabilities.PositionEncoding); enc != PositionEncodingUTF16 || result.Capabilities.PositionEncoding == string(PositionEncodingUTF16) {
		return enc
	}

	if enc := normalizePositionEncoding(result.OffsetEncoding); enc != PositionEncodingUTF16 || result.OffsetEncoding == string(PositionEncodingUTF16) {
		return enc
	}

	return PositionEncodingUTF16
}
