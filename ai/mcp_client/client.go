package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/debug"
)

type Client struct {
	gosdk *GoSDKClient
	Tools *mcp_sdk.ListToolsResult
}

// initClient is deprecated and kept only for backward compatibility reference.
// Use initClientFromGoSDK instead.
func initClient(overrides *mcp_config.OverrideT) (*Client, error) {
	return nil, fmt.Errorf("initClient is no longer supported; use go-sdk adapter")
}

// initClientFromGoSDK wraps an existing GoSDKClient into the same exported Client type
func initClientFromGoSDK(g *GoSDKClient, overrides *mcp_config.OverrideT) (*Client, error) {
	// We don't need to call Initialize here because the Go SDK handles session setup
	return &Client{gosdk: g}, nil
}

func (c *Client) ListTools() error {
	if c.gosdk != nil {
		err := c.gosdk.ListTools()
		if err == nil {
			// Propagate tools from gosdk to Client struct for backward compatibility
			c.Tools = c.gosdk.Tools
		}
		return err
	}

	return fmt.Errorf("no underlying client available")
}

func (c *Client) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	if c.gosdk != nil {
		return c.gosdk.Call(ctx, name, args)
	}

	return "", fmt.Errorf("no underlying client available")
}

// Close closes the underlying client connection
func (c *Client) Close() error {
	if c.gosdk != nil {
		return c.gosdk.Close()
	}
	return fmt.Errorf("no underlying client available")
}

// Helper function to print tool results
func printToolResult(result interface{}) string {
	var results string

	// Marshal result to JSON for display
	if data, err := json.MarshalIndent(result, "", "  "); err == nil {
		results = string(data) + "\n"
	}

	debug.Log(results)
	return results
}
