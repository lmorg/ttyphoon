package agent

import (
	"context"
	"fmt"
	"sort"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/utils/file"
)

func (agent *Agent) ConfiguredMcpServers() ([]aitypes.McpServer, error) {
	files := file.GetConfigFiles("mcp", ".json")
	sort.Strings(files)
	servers := make([]aitypes.McpServer, 0)
	for _, path := range files {
		config, err := mcp_config.ReadJson(path)
		if err != nil {
			return nil, fmt.Errorf("read MCP config %s: %w", path, err)
		}
		for name, server := range config.Mcp.Servers {
			servers = append(servers, aitypes.McpServer{
				Key: makeMcpServerKey(path, name), Name: name, Description: server.Description,
			})
		}
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Key < servers[j].Key })
	return servers, nil
}

func (agent *Agent) StartConfiguredMcpServer(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	servers, err := agent.ConfiguredMcpServers()
	if err != nil {
		return "", err
	}
	for _, server := range servers {
		if server.Key != key {
			continue
		}
		source, name, err := parseMcpServerKey(server.Key)
		if err != nil {
			return "", err
		}
		config, err := mcp_config.ReadJson(source)
		if err != nil {
			return "", err
		}
		config.Source = source
		if err := agent.startServerFromConfigServer(ctx, config, name); err != nil {
			return "", err
		}
		return name, nil
	}
	return "", fmt.Errorf("MCP server key %q is not in the configured server catalog", key)
}
