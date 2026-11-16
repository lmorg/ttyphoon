package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMcpEnvironmentHelper(t *testing.T) {
	if os.Getenv("TTYPHOON_MCP_TEST_HELPER") != "1" {
		return
	}
	values := make(map[string]string)
	for _, name := range []string{
		"PATH", "TTYPHOON_TEST_HOST", "TTYPHOON_TEST_PRIORITY", "TTYPHOON_TEST_EXPLICIT",
		"TTYPHOON_TEST_EMPTY", "TTYPHOON_TEST_A_ONLY", "TTYPHOON_TEST_B_ONLY",
		"TTYPHOON_TEST_AI_ONLY", "TTYPHOON_TEST_LITERAL",
	} {
		if value, ok := os.LookupEnv(name); ok {
			values[name] = value
		}
	}
	data, err := json.Marshal(values)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Getenv("TTYPHOON_MCP_TEST_OUTPUT"), data, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	server := mcp_sdk.NewServer(&mcp_sdk.Implementation{Name: "env fixture", Version: "1"}, nil)
	if err := server.Run(context.Background(), &mcp_sdk.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestMcpCommandEnvironmentIsScopedAndInheritsHost(t *testing.T) {
	for key, value := range map[string]string{
		"TTYPHOON_TEST_HOST": "inherited", "TTYPHOON_TEST_PRIORITY": "host",
		"TTYPHOON_TEST_EXPLICIT": "host", "TTYPHOON_TEST_EMPTY": "host",
		"TTYPHOON_TEST_A_ONLY": "host-a", "TTYPHOON_TEST_B_ONLY": "host-b",
		"TTYPHOON_TEST_AI_ONLY": "host-ai",
	} {
		t.Setenv(key, value)
	}
	first, second := t.TempDir(), t.TempDir()
	previousDirs := xdg.ConfigDirs
	xdg.ConfigDirs = []string{first, second}
	t.Cleanup(func() { xdg.ConfigDirs = previousDirs })
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(first, app.DirName, "server.vars"), "TTYPHOON_TEST_PRIORITY=first\nTTYPHOON_TEST_A_ONLY=alpha-secret\nTTYPHOON_TEST_EXPLICIT=file\nTTYPHOON_TEST_EMPTY=file\nTTYPHOON_TEST_LITERAL=${input:token}\n")
	write(filepath.Join(second, app.DirName, "server.vars"), "TTYPHOON_TEST_PRIORITY=second\nTTYPHOON_TEST_EXPLICIT=later-file\n")
	otherFile := filepath.Join(t.TempDir(), "other.vars")
	write(otherFile, "TTYPHOON_TEST_B_ONLY=bravo-secret\n")
	aiFile := filepath.Join(t.TempDir(), "ai.vars")
	write(aiFile, "TTYPHOON_TEST_AI_ONLY=global-ai-secret\n")
	previousFiles := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{aiFile}
	services := config.Config.Ai.Services()
	previousService, hadService := services["test-mcp-scope"]
	services["test-mcp-scope"] = &config.AIServiceT{Label: "test-mcp-scope", Env: map[string]string{"TTYPHOON_TEST_AI_ONLY": "service-ai-secret"}}
	t.Cleanup(func() {
		config.Config.Ai.EnvFiles = previousFiles
		if hadService {
			services["test-mcp-scope"] = previousService
		} else {
			delete(services, "test-mcp-scope")
		}
	})
	renderer := &mcpLoadTestRenderer{started: make(chan func(), 4)}
	agt := &Agent{
		renderer: renderer, serviceName: "test-mcp-scope",
		_mcpServers: make(map[string]client), _mcpServerSources: make(map[string]string),
	}
	t.Cleanup(func() {
		for _, connection := range agt._mcpServers {
			_ = connection.Close()
		}
	})
	var logs bytes.Buffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previousLog)
	start := func(name string, files []string, extra mcp_config.EnvVarsT) map[string]string {
		t.Helper()
		output := filepath.Join(t.TempDir(), name+".json")
		extra["TTYPHOON_MCP_TEST_HELPER"] = "1"
		extra["TTYPHOON_MCP_TEST_OUTPUT"] = output
		server := mcp_config.ServerT{
			Type: "command", Command: os.Args[0], Args: []string{"-test.run=^TestMcpEnvironmentHelper$", "--", "${input:token}"},
			EnvFiles: files, Env: extra,
		}
		inputs := map[string]string{"token": "explicit-secret"}
		if err := agt.startServerFromConfigEntry("source", name, server, &inputs); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var values map[string]string
		if err := json.Unmarshal(data, &values); err != nil {
			t.Fatal(err)
		}
		return values
	}
	a := start("first", []string{"server.vars"}, mcp_config.EnvVarsT{
		"TTYPHOON_TEST_EXPLICIT": "${input:token}", "TTYPHOON_TEST_EMPTY": "",
	})
	b := start("second", []string{otherFile}, mcp_config.EnvVarsT{})
	for key, want := range map[string]string{
		"TTYPHOON_TEST_HOST": "inherited", "TTYPHOON_TEST_PRIORITY": "second",
		"TTYPHOON_TEST_EXPLICIT": "explicit-secret", "TTYPHOON_TEST_EMPTY": "",
		"TTYPHOON_TEST_A_ONLY": "alpha-secret", "TTYPHOON_TEST_B_ONLY": "host-b",
		"TTYPHOON_TEST_AI_ONLY": "host-ai", "TTYPHOON_TEST_LITERAL": "${input:token}",
	} {
		if got, ok := a[key]; !ok || got != want {
			t.Errorf("first server did not receive expected %s", key)
		}
	}
	for key, want := range map[string]string{
		"TTYPHOON_TEST_HOST": "inherited", "TTYPHOON_TEST_PRIORITY": "host",
		"TTYPHOON_TEST_A_ONLY": "host-a", "TTYPHOON_TEST_B_ONLY": "bravo-secret",
		"TTYPHOON_TEST_AI_ONLY": "host-ai",
	} {
		if b[key] != want {
			t.Errorf("second server did not receive expected %s", key)
		}
	}
	if a["PATH"] != os.Getenv("PATH") || b["PATH"] != os.Getenv("PATH") {
		t.Error("command servers did not inherit PATH")
	}
	if os.Getenv("TTYPHOON_TEST_PRIORITY") != "host" {
		t.Error("MCP environment resolution changed the host")
	}
	write(filepath.Join(second, app.DirName, "server.vars"), "TTYPHOON_TEST_PRIORITY=updated\n")
	if err := agt.SetMcpServerEnabled("source::first", false); err != nil {
		t.Fatal(err)
	}
	restarted := start("first", []string{"server.vars"}, mcp_config.EnvVarsT{})
	if restarted["TTYPHOON_TEST_PRIORITY"] != "updated" {
		t.Error("restarted MCP server did not reload its environment files")
	}
	log.SetOutput(previousLog)
	for _, secret := range []string{"alpha-secret", "bravo-secret", "explicit-secret", "global-ai-secret", "service-ai-secret"} {
		if strings.Contains(logs.String(), secret) {
			t.Error("MCP startup logged a scoped environment value")
		}
	}
}

func TestMcpCommandEnvironmentFailureDoesNotStartProcess(t *testing.T) {
	renderer := &mcpLoadTestRenderer{started: make(chan func(), 1)}
	agt := &Agent{renderer: renderer}
	output := filepath.Join(t.TempDir(), "output.json")
	server := mcp_config.ServerT{
		Command: os.Args[0], Args: []string{"-test.run=^TestMcpEnvironmentHelper$"},
		EnvFiles: []string{filepath.Join(t.TempDir(), "missing.vars")},
		Env:      mcp_config.EnvVarsT{"TTYPHOON_MCP_TEST_HELPER": "1", "TTYPHOON_MCP_TEST_OUTPUT": output},
	}
	err := agt.startServerFromConfigEntry("source", "missing", server, &map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "resolve MCP server environment") {
		t.Fatalf("missing environment file error = %v", err)
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("server started despite an environment error: %v", err)
	}
	if _, err := resolveMcpCommandEnvironment(nil, []string{"INVALID-KEY=secret-value"}); err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("invalid environment entry was not safely rejected: %v", err)
	}
}

func TestMcpHttpServerDoesNotLoadCommandEnvironmentFiles(t *testing.T) {
	server := mcpLifecycleFixture(t)
	renderer := &mcpLoadTestRenderer{started: make(chan func(), 1)}
	agt := &Agent{renderer: renderer}
	entry := mcp_config.ServerT{
		Type: "http", Url: server.URL, EnvFiles: []string{filepath.Join(t.TempDir(), "missing.vars")},
	}
	if err := agt.startServerFromConfigEntry("source", "http-server", entry, &map[string]string{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = agt._mcpServers["http-server"].Close() })
}

func mcpLifecycleFixture(t *testing.T) *httptest.Server {
	t.Helper()
	server := mcp_sdk.NewServer(&mcp_sdk.Implementation{Name: "lifecycle", Version: "1"}, nil)
	server.AddTool(&mcp_sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}},
		func(context.Context, *mcp_sdk.CallToolRequest) (*mcp_sdk.CallToolResult, error) {
			return &mcp_sdk.CallToolResult{}, nil
		})
	httpServer := httptest.NewServer(mcp_sdk.NewStreamableHTTPHandler(func(*http.Request) *mcp_sdk.Server { return server }, nil))
	t.Cleanup(httpServer.Close)
	return httpServer
}

func TestMcpRegistrationIsAtomicAndConcurrentRegistrationCannotDuplicateTools(t *testing.T) {
	server := mcpLifecycleFixture(t)
	connection, err := mcp_client.ConnectHttp(&mcp_config.OverrideT{}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	agt := &Agent{_mcpServers: make(map[string]client), _mcpServerSources: make(map[string]string)}
	valid := &mcp_sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}
	invalid := &mcp_sdk.Tool{Name: "invalid", InputSchema: make(chan int)}
	if err := agt.registerMcpServer(context.Background(), "source", "server", mcp_config.ServerT{}, connection, []*mcp_sdk.Tool{valid, invalid}); err == nil {
		t.Fatal("invalid tool schema was accepted")
	}
	if agt.McpServerExists("server") || len(agt._tools) != 0 {
		t.Fatal("failed setup left partial MCP state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := agt.registerMcpServer(ctx, "source", "server", mcp_config.ServerT{}, connection, []*mcp_sdk.Tool{valid}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled registration returned %v", err)
	}
	var group sync.WaitGroup
	var successes atomic.Int32
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if agt.registerMcpServer(context.Background(), "source", "server", mcp_config.ServerT{}, connection, []*mcp_sdk.Tool{valid}) == nil {
				successes.Add(1)
			}
		}()
	}
	group.Wait()
	if successes.Load() != 1 || len(agt._tools) != 1 || len(agt._mcpServers) != 1 {
		t.Fatalf("duplicate registration: successful=%d tools=%d servers=%d", successes.Load(), len(agt._tools), len(agt._mcpServers))
	}
}

type mcpLoadTestNotification struct{ types.Notification }

func (*mcpLoadTestNotification) Close() {}

type mcpLoadTestRenderer struct {
	types.Renderer
	started chan func()
}

func (renderer *mcpLoadTestRenderer) DisplaySticky(_ types.NotificationType, _ string, cancel func()) types.Notification {
	renderer.started <- cancel
	return &mcpLoadTestNotification{}
}

func (renderer *mcpLoadTestRenderer) DisplayNotification(types.NotificationType, string) {}
func (renderer *mcpLoadTestRenderer) GetWindowContext() context.Context                  { return context.Background() }

func TestMcpLoadReservationClearsAfterCancellationAndCanRetry(t *testing.T) {
	release := make(chan struct{})
	blocking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = io.Copy(io.Discard, request.Body)
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer func() { close(release); blocking.Close() }()
	healthy := mcpLifecycleFixture(t)
	renderer := &mcpLoadTestRenderer{started: make(chan func(), 2)}
	agt := &Agent{renderer: renderer, _mcpServers: make(map[string]client), _mcpServerSources: make(map[string]string)}
	result := make(chan error, 1)
	go func() {
		result <- agt.startServerFromConfigEntry("source", "server", mcp_config.ServerT{Type: "http", Url: blocking.URL}, &map[string]string{})
	}()
	var cancel func()
	select {
	case cancel = <-renderer.started:
	case <-time.After(time.Second):
		t.Fatal("load did not start")
	}
	if err := agt.startServerFromConfigEntry("source", "server", mcp_config.ServerT{Type: "http", Url: blocking.URL}, &map[string]string{}); err == nil || !strings.Contains(err.Error(), "already loading") {
		t.Fatalf("concurrent load was not rejected: %v", err)
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled load returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelled load did not stop")
	}
	if len(agt.mcpLoads) != 0 || agt.McpServerExists("server") {
		t.Fatal("cancelled load left registered or pending state")
	}
	if err := agt.startServerFromConfigEntry("source", "server", mcp_config.ServerT{Type: "http", Url: healthy.URL}, &map[string]string{}); err != nil {
		t.Fatal(err)
	}
	defer agt._mcpServers["server"].Close()
	if len(agt._tools) != 1 || !agt.McpServerExists("server") {
		t.Fatal("retry did not register a healthy MCP server")
	}
}
