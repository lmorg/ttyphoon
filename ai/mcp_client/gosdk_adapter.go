package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

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

// Call invokes a tool and returns a textual rendering of the result.
func (c *GoSDKClient) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	params := &mcp_sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	res, err := c.session.CallTool(ctx, params)
	if err != nil {
		return "", err
	}

	log.Printf("MCP tool result %q: content_items=%d structured_content=%t is_error=%t",
		name, len(res.Content), res.StructuredContent != nil, res.IsError)

	return printGoSDKToolResult(res), nil
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
