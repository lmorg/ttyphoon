package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func startServerCmdLine(cfgPath string, agent *Agent, envvars []string, server string, svr mcp_config.ServerT) error {
	debug.Log(envvars)
	log.Printf("MCP server %s: %s %v", server, svr.Command, svr.Args)

	c, err := mcp_client.ConnectCmdLine(&svr.Override, envvars, svr.Command, svr.Args...)
	if err != nil {
		return err
	}

	return startServer(cfgPath, agent, server, c)
}

func startServerHttp(cfgPath string, agent *Agent, server string, svr mcp_config.ServerT) error {
	serverURL := svr.Url
	log.Printf("MCP server %s: %s", server, serverURL)
	hooks := mcp_client.OAuthUIHooks{
		OpenBrowser: func(authURL string) {
			rctx := agent.Renderer().GetContext()
			if rctx != nil {
				runtime.BrowserOpenURL(rctx, authURL)
			}
		},
		PromptCallbackURL: func() (string, error) {
			return promptString(agent, "Paste OAuth callback URL")
		},
		OnAutoCallbackUnavailable: func(callbackErr error) {
			agent.Renderer().DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Automatic OAuth callback unavailable (%v). Falling back to pasted callback URL.", callbackErr))
		},
	}

	return mcp_client.ConnectAndUseHttp(
		&svr.Override,
		server,
		serverURL,
		svr.OAuth,
		hooks,
		func() {
			agent.Renderer().DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("MCP server %s requires OAuth. Starting browser authentication...", server))
		},
		func(c *mcp_client.Client) error {
			return startServer(cfgPath, agent, server, c)
		},
	)
}

func promptString(agent *Agent, prompt string) (string, error) {
	ch := make(chan string)
	errCh := make(chan error)

	agent.Renderer().DisplayInputBox(prompt, "",
		func(s string) { ch <- s },
		func(_ string) { errCh <- fmt.Errorf("OAuth authorization canceled") },
	)

	select {
	case v := <-ch:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("OAuth authorization canceled")
		}
		return v, nil
	case err := <-errCh:
		return "", err
	}
}

func startServer(cfgPath string, agent *Agent, server string, c *mcp_client.Client) error {
	err := c.ListTools()
	if err != nil {
		return err
	}

	agent.McpServerAdd(server, c)

	for i := range c.Tools.Tools {
		jsonSchema, err := c.Tools.Tools[i].MarshalJSON()
		if err != nil {
			return err
		}

		err = agent.ToolsAdd(&mcpTool{
			client: c,
			server: server,
			path:   cfgPath,
			name:   c.Tools.Tools[i].GetName(),
			schema: jsonSchema,
			description: fmt.Sprintf("%s\nInput schema: %s",
				c.Tools.Tools[i].Description,
				string(jsonSchema),
			),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
