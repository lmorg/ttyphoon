package mcp_client

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/mark3labs/mcp-go/client"
)

func ConnectCmdLine(overrides *mcp_config.OverrideT, envvars []string, command string, args ...string) (*Client, error) {
	c, err := client.NewStdioMCPClient(command, envvars, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return initClient(c, overrides)
}
