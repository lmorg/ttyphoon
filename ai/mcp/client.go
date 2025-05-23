package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lmorg/mxtty/app"
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

func (c *Client) call(ctx context.Context, name string, input any) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = input
	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		return "", err
	}

	return printToolResult(result), nil
}

/*
	// List allowed directories
	fmt.Println("Listing allowed directories...")
	listDirRequest := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	listDirRequest.Params.Name = "list_allowed_directories"

	result, err := c.CallTool(ctx, listDirRequest)
	if err != nil {
		log.Fatalf("Failed to list allowed directories: %v", err)
	}
	printToolResult(result)
	fmt.Println()

	// List /tmp
	fmt.Println("Listing /tmp directory...")
	listTmpRequest := mcp.CallToolRequest{}
	listTmpRequest.Params.Name = "list_directory"
	listTmpRequest.Params.Arguments = map[string]any{
		"path": "/tmp",
	}

	result, err = c.CallTool(ctx, listTmpRequest)
	if err != nil {
		log.Fatalf("Failed to list directory: %v", err)
	}
	printToolResult(result)
	fmt.Println()

	// Create mcp directory
	fmt.Println("Creating /tmp/mcp directory...")
	createDirRequest := mcp.CallToolRequest{}
	createDirRequest.Params.Name = "create_directory"
	createDirRequest.Params.Arguments = map[string]any{
		"path": "/tmp/mcp",
	}

	result, err = c.CallTool(ctx, createDirRequest)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	printToolResult(result)
	fmt.Println()

	// Create hello.txt
	fmt.Println("Creating /tmp/mcp/hello.txt...")
	writeFileRequest := mcp.CallToolRequest{}
	writeFileRequest.Params.Name = "write_file"
	writeFileRequest.Params.Arguments = map[string]any{
		"path":    "/tmp/mcp/hello.txt",
		"content": "Hello World",
	}

	result, err = c.CallTool(ctx, writeFileRequest)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	printToolResult(result)
	fmt.Println()

	// Verify file contents
	fmt.Println("Reading /tmp/mcp/hello.txt...")
	readFileRequest := mcp.CallToolRequest{}
	readFileRequest.Params.Name = "read_file"
	readFileRequest.Params.Arguments = map[string]any{
		"path": "/tmp/mcp/hello.txt",
	}

	result, err = c.CallTool(ctx, readFileRequest)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	printToolResult(result)

	// Get file info
	fmt.Println("Getting info for /tmp/mcp/hello.txt...")
	fileInfoRequest := mcp.CallToolRequest{}
	fileInfoRequest.Params.Name = "get_file_info"
	fileInfoRequest.Params.Arguments = map[string]any{
		"path": "/tmp/mcp/hello.txt",
	}

	result, err = c.CallTool(ctx, fileInfoRequest)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	printToolResult(result)
}*/

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

	return results
}
