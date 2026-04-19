package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/utils/or"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Client struct {
	client     *client.Client
	initResult mcp.InitializeResult
	Tools      *mcp.ListToolsResult
}

func initClient(c *client.Client, overrides *mcp_config.OverrideT) (*Client, error) {
	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    or.NotEmpty(overrides.AppName, app.Name()),
		Version: app.Version(),
	}

	initResult, err := c.Initialize(context.Background(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	client := &Client{
		client:     c,
		initResult: *initResult,
	}

	return client, nil
}

func (c *Client) ListTools() error {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.client.ListTools(context.Background(), toolsRequest)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	c.Tools = tools

	return nil
}

func (c *Client) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	req := mcp.CallToolRequest{
		Request: mcp.Request{Method: "tools/call"},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		return "", err
	}

	return printToolResult(result), nil
}

// Helper function to print tool results
func printToolResult(result *mcp.CallToolResult) string {
	var results string

	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			results += textContent.Text + "\n"
		} else {
			jsonBytes, _ := json.MarshalIndent(content, "", "  ")
			results += string(jsonBytes) + "\n"
		}
	}

	debug.Log(results)
	return results
}

func (c *Client) Close() error {
	return c.client.Close()
}
