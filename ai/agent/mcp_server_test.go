package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/types"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

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
