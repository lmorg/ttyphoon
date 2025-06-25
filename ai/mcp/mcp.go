package mcp

import (
	"fmt"
	"log"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
)

func StartServerCmdLine(cfgPath string, meta *agent.Meta, envvars []string, server, command string, args ...string) error {
	debug.Log(envvars)
	log.Printf("MCP server %s: %s %v", server, command, args)

	c, err := connectCmdLine(envvars, command, args...)
	if err != nil {
		return err
	}

	err = c.listTools()
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
