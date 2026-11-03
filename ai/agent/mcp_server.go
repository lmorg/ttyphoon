package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func startServerCmdLine(ctx context.Context, cfgPath string, agent *Agent, envvars []string, server string, svr mcp_config.ServerT) error {
	debug.Log(envvars)
	log.Printf("[info] MCP server %s: %s %v", server, svr.Command, svr.Args)

	c, err := mcp_client.ConnectCmdLineContext(ctx, &svr.Override, envvars, svr.Command, svr.Args...)
	if err != nil {
		return err
	}

	if err := startServer(ctx, cfgPath, agent, server, svr, c); err != nil {
		_ = c.Close()
		return err
	}
	return nil
}

func startServerHttp(ctx context.Context, cfgPath string, agent *Agent, server string, svr mcp_config.ServerT) error {
	serverURL := svr.Url
	log.Printf("[info] MCP server %s: %s", server, serverURL)
	hooks := mcp_client.OAuthUIHooks{
		OpenBrowser: func(authURL string) {
			rctx := agent.Renderer().GetWindowContext()
			if rctx != nil {
				runtime.BrowserOpenURL(rctx, authURL)
			}
		},
	}

	return mcp_client.ConnectAndUseHttpContext(
		ctx,
		&svr.Override,
		agent.Workspace(),
		server,
		serverURL,
		svr.OAuth,
		hooks,
		func() {
			agent.Renderer().DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("MCP server %s requires OAuth. Starting browser authentication...", server))
		},
		func(c *mcp_client.Client) error {
			return startServer(ctx, cfgPath, agent, server, svr, c)
		},
	)
}

func startServer(ctx context.Context, cfgPath string, agent *Agent, server string, svr mcp_config.ServerT, c *mcp_client.Client) error {
	err := c.ListToolsContext(ctx)
	if err != nil {
		return err
	}

	return agent.registerMcpServer(ctx, cfgPath, server, svr, c, c.Tools.Tools)
}

func (agent *Agent) registerMcpServer(ctx context.Context, cfgPath, server string, svr mcp_config.ServerT, c *mcp_client.Client, advertised []*mcp_sdk.Tool) error {
	tools := make([]aitypes.Tool, 0, len(advertised))
	seen := make(map[string]bool)

	toolNames := make([]string, len(advertised))
	for i := range advertised {
		toolNames[i] = advertised[i].Name
	}
	log.Printf("[debug] MCP server %s: %d tools advertised: %s", server, len(toolNames), strings.Join(toolNames, ", "))

	for i := range advertised {
		tool := advertised[i]
		if seen[tool.Name] {
			return fmt.Errorf("MCP server %q advertised duplicate tool %q", server, tool.Name)
		}
		seen[tool.Name] = true

		// Marshal the InputSchema to JSON
		jsonSchema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return err
		}

		prepared, err := (&mcpTool{
			client: c,
			server: server,
			path:   cfgPath,
			name:   tool.Name,
			schema: jsonSchema,
			description: fmt.Sprintf("%s\nInput schema: %s",
				tool.Description,
				string(jsonSchema),
			),
			permissions: aitypes.DefaultPermissions{
				Invocation: svr.DefaultPermissions.Invocation,
				Subagents:  svr.DefaultPermissions.Subagents,
			},
		}).New(agent)
		if err != nil {
			return err
		}
		tools = append(tools, prepared)
	}
	agent.mcpMu.Lock()
	if agent.mcpClosed {
		agent.mcpMu.Unlock()
		return fmt.Errorf("MCP agent has been closed")
	}
	if err := ctx.Err(); err != nil {
		agent.mcpMu.Unlock()
		return err
	}
	if _, exists := agent._mcpServers[server]; exists {
		agent.mcpMu.Unlock()
		return fmt.Errorf("MCP server %q is already loaded", server)
	}
	if agent._mcpServers == nil {
		agent._mcpServers = make(map[string]client)
		agent._mcpServerSources = make(map[string]string)
	}
	agent.toolMu.Lock()
	agent._tools = append(agent._tools, tools...)
	agent._mcpServers[server] = c
	agent._mcpServerSources[server] = cfgPath
	agent.toolMu.Unlock()
	agent.mcpMu.Unlock()
	agent.Reload()
	return nil
}
