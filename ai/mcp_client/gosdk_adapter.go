package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"

	authsdk "github.com/modelcontextprotocol/go-sdk/auth"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/utils/or"
)

// GoSDKClient is a thin adapter around github.com/modelcontextprotocol/go-sdk's ClientSession.
type GoSDKClient struct {
	client  *mcp_sdk.Client
	session *mcp_sdk.ClientSession
	Tools   *mcp_sdk.ListToolsResult
}

// ConnectCmdLineGoSDK connects to a local MCP server process using the Go SDK transports.
func ConnectCmdLineGoSDK(overrides *mcp_config.OverrideT, envvars []string, command string, args ...string) (*GoSDKClient, error) {
	impl := &mcp_sdk.Implementation{
		Name:    or.NotEmpty(overrides.AppName, app.Name()),
		Version: app.Version(),
	}

	client := mcp_sdk.NewClient(impl, nil)

	cmd := exec.Command(command, args...)
	// inherit envvars if provided
	if len(envvars) > 0 {
		cmd.Env = append(cmd.Env, envvars...)
	}

	transport := &mcp_sdk.CommandTransport{Command: cmd}

	sess, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("gosdk: connect command transport failed: %w", err)
	}

	return &GoSDKClient{client: client, session: sess}, nil
}

// ConnectHttpGoSDK connects to a streamable HTTP MCP endpoint. oauth may be nil.
func ConnectHttpGoSDK(overrides *mcp_config.OverrideT, endpoint string, oauth authsdk.OAuthHandler) (*GoSDKClient, error) {
	impl := &mcp_sdk.Implementation{
		Name:    or.NotEmpty(overrides.AppName, app.Name()),
		Version: app.Version(),
	}

	client := mcp_sdk.NewClient(impl, nil)

	transport := &mcp_sdk.StreamableClientTransport{
		Endpoint:     endpoint,
		HTTPClient:   http.DefaultClient,
		OAuthHandler: oauth,
	}

	sess, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("gosdk: connect http transport failed: %w", err)
	}

	return &GoSDKClient{client: client, session: sess}, nil
}

// ListTools calls ListTools on the underlying session and caches the result.
func (c *GoSDKClient) ListTools() error {
	res, err := c.session.ListTools(context.Background(), &mcp_sdk.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("gosdk: list tools failed: %w", err)
	}
	c.Tools = res
	return nil
}

// Call invokes a tool with retry logic for temporary connection failures.
// It uses exponential backoff with jitter to handle transient network issues.
func (c *GoSDKClient) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	params := &mcp_sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	// Retry configuration
	const (
		maxRetries  = 3
		initialWait = 200 * time.Millisecond
		maxWait     = 2 * time.Second
	)

	var lastErr error
	var backoff = initialWait

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Add jitter to backoff: backoff ± 25%
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			waitTime := backoff - backoff/4 + jitter
			if waitTime > maxWait {
				waitTime = maxWait
			}

			log.Printf("MCP tool %q attempt %d/%d: retrying after %v (error: %v)",
				name, attempt+1, maxRetries+1, waitTime, lastErr)

			select {
			case <-time.After(waitTime):
				// Continue to retry
			case <-ctx.Done():
				return "", ctx.Err()
			}

			// Double the backoff for next iteration (capped at maxWait)
			backoff = backoff * 2
			if backoff > maxWait {
				backoff = maxWait
			}
		}

		res, err := c.session.CallTool(ctx, params)
		if err == nil {
			if attempt > 0 {
				log.Printf("MCP tool %q: connection recovered after %d retries", name, attempt)
			}

			log.Printf("MCP tool result %q: content_items=%d structured_content=%t is_error=%t",
				name, len(res.Content), res.StructuredContent != nil, res.IsError)
			if res.IsError {
				msg := strings.TrimSpace(printGoSDKToolResult(res))
				if len(msg) > 2000 {
					msg = msg[:2000] + "..."
				}
				log.Printf("MCP tool error %q payload: %s", name, msg)
			}

			return printGoSDKToolResult(res), nil
		}

		lastErr = err

		// Check if the error is potentially transient (connection-related)
		// vs permanent (e.g. invalid tool name, JSON schema violation)
		if !isTransientError(err) {
			// Permanent error; don't retry
			return "", err
		}

		if attempt < maxRetries {
			continue
		}
	}

	// All retries exhausted
	return "", fmt.Errorf("MCP tool call failed after %d retries: %w", maxRetries+1, lastErr)
}

// isTransientError checks if an error is likely transient (network-related)
// and worth retrying. Connection drops, timeouts, and stream errors are transient;
// malformed input or invalid tool names are not.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Connection-related errors worth retrying
	transientPatterns := []string{
		"connection",
		"stream",
		"sockets",
		"connection reset",
		"connection refused",
		"connection closed",
		"broken pipe",
		"eof",
		"timeout",
		"temporary failure",
		"exceeded.*retries",
		"no progress",
		"i/o timeout",
		"read: connection",
		"write: connection",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func printGoSDKToolResult(result *mcp_sdk.CallToolResult) string {
	var out string
	for _, content := range result.Content {
		switch v := content.(type) {
		case *mcp_sdk.TextContent:
			out += v.Text + "\n"
		default:
			// best-effort fallback: try to format value
			out += fmt.Sprintf("%T: %+v\n", v, v)
		}
	}

	// Many MCP servers (e.g. Atlassian's Jira/Confluence tools) return their
	// payload in StructuredContent rather than as text Content. Without this,
	// a fully successful call renders as empty output.
	if result.StructuredContent != nil {
		if b, err := json.Marshal(result.StructuredContent); err == nil {
			out += string(b) + "\n"
		} else {
			out += fmt.Sprintf("%+v\n", result.StructuredContent)
		}
	}

	if result.IsError {
		out = "tool reported an error:\n" + out
	}

	return out
}

// Close closes the session.
func (c *GoSDKClient) Close() error {
	if c.session != nil {
		_ = c.session.Close()
	}
	return nil
}
