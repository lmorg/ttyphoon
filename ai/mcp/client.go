package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Client struct {
	client     *client.Client
	initResult mcp.InitializeResult
	tools      *mcp.ListToolsResult
}

func connectCmdLine(envvars []string, command string, args ...string) (*Client, error) {
	c, err := client.NewStdioMCPClient(command, envvars, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return initClient(c)
}

func connectHttp(url string) (*Client, error) {
	c, err := client.NewStreamableHttpClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}
	return initClient(c)
}

func initClient(c *client.Client) (*Client, error) {
	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    app.Name,
		Version: app.Version(),
	}

	initResult, err := c.Initialize(context.Background(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %v", err)
	}

	client := &Client{
		client:     c,
		initResult: *initResult,
	}

	return client, nil
}

func (c *Client) listTools() error {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.client.ListTools(context.Background(), toolsRequest)
	if err != nil {
		return fmt.Errorf("failed to list tools: %v", err)
	}

	c.tools = tools

	return nil
}

func (c *Client) call(ctx context.Context, name string, args map[string]any) (string, error) {
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
