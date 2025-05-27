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
		err = meta.ToolsAdd(&tool{
			client: c,
			server: server,
			path:   cfgPath,
			name:   c.tools.Tools[i].GetName(),
			description: fmt.Sprintf("# Description:\n%s\n\n# Annotations:\n%s\n\n# Schema:\n%s",
				c.tools.Tools[i].Description,
				c.tools.Tools[i].Annotations.Title,
				string(c.tools.Tools[i].RawInputSchema),
			),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
