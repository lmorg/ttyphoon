package mcp_client

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
)

func ConnectCmdLine(overrides *mcp_config.OverrideT, envvars []string, command string, args ...string) (*Client, error) {
	// Use the Go SDK adapter for stdio/command transport
	g, err := ConnectCmdLineGoSDK(overrides, envvars, command, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio MCP client: %w", err)
	}

	return initClientFromGoSDK(g, overrides)
}
