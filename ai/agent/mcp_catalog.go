package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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
				Key:         makeMcpServerKey(path, name),
				Name:        strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + "::" + name,
				Description: server.Description,
			})
		}
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Key < servers[j].Key })
	return servers, nil
}

func (agent *Agent) StartConfiguredMcpServer(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	servers, err := agent.ConfiguredMcpServers()
	if err != nil {
		return "", err
	}
	var match *aitypes.McpServer
	for i := range servers {
		if servers[i].Name != name {
			continue
		}
		if match != nil {
			return "", fmt.Errorf("MCP server name %q is ambiguous in the configured server catalog", name)
		}
		match = &servers[i]
	}
	if match == nil {
		return "", fmt.Errorf("MCP server name %q is not in the configured server catalog", name)
	}

	source, name, err := parseMcpServerKey(match.Key)
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
