package mcp

import (
	"fmt"
	"log"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/debug"
)

func StartServerCmdLine(cfgPath string, meta *agent.Meta, envvars []string, server, command string, args ...string) error {
	debug.Log(envvars)
	log.Printf("MCP server %s: %s %v", server, command, args)

	c, err := connectCmdLine(envvars, command, args...)
	if err != nil {
		return err
	}

	return startServer(cfgPath, meta, server, c)
}

func StartServerHttp(cfgPath string, meta *agent.Meta, server, url string) error {
	log.Printf("MCP server %s: %s", server, url)

	c, err := connectHttp(url)
	if err != nil {
		return err
	}

	return startServer(cfgPath, meta, server, c)
}

func startServer(cfgPath string, meta *agent.Meta, server string, c *Client) error {
	err := c.listTools()
	if err != nil {
		return err
	}

	meta.McpServerAdd(server, c)

	for i := range c.tools.Tools {
		jsonSchema, err := c.tools.Tools[i].MarshalJSON()
		if err != nil {
			return err
		}

		err = meta.ToolsAdd(&tool{
			client: c,
			server: server,
			path:   cfgPath,
			name:   c.tools.Tools[i].GetName(),
			schema: jsonSchema,
			description: fmt.Sprintf("%s\nInput schema: %s",
				c.tools.Tools[i].Description,
				string(jsonSchema),
			),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
