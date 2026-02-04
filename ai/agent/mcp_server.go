package agent

import (
	"fmt"
	"log"

	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/debug"
)

func startServerCmdLine(cfgPath string, meta *Meta, envvars []string, server, command string, args ...string) error {
	debug.Log(envvars)
	log.Printf("MCP server %s: %s %v", server, command, args)

	c, err := mcp_client.ConnectCmdLine(envvars, command, args...)
	if err != nil {
		return err
	}

	return startServer(cfgPath, meta, server, c)
}

func startServerHttp(cfgPath string, meta *Meta, server, url string) error {
	log.Printf("MCP server %s: %s", server, url)

	c, err := mcp_client.ConnectHttp(url)
	if err != nil {
		return err
	}

	return startServer(cfgPath, meta, server, c)
}

func startServer(cfgPath string, meta *Meta, server string, c *mcp_client.Client) error {
	err := c.ListTools()
	if err != nil {
		return err
	}

	meta.McpServerAdd(server, c)

	for i := range c.Tools.Tools {
		jsonSchema, err := c.Tools.Tools[i].MarshalJSON()
		if err != nil {
			return err
		}

		err = meta.ToolsAdd(&mcpTool{
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
