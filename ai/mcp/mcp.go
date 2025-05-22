package mcp

import (
	"fmt"

	"github.com/lmorg/mxtty/ai/agent"
)

func StartServerCmdLine(server, command string, params ...string) error {
	c, err := connectCmdLine(command, params...)
	if err != nil {
		return err
	}

	err = c.listTools()
	if err != nil {
		return err
	}

	for i := range c.tools.Tools {
		agent.ToolsAdd(&tool{
			client: c,
			server: server,
			name:   c.tools.Tools[i].GetName(),
			description: fmt.Sprintf("# Description:\n%s\n\n# Annotations:\n%s\n\n# Schema:\n%s",
				c.tools.Tools[i].Description,
				c.tools.Tools[i].Annotations.Title,
				string(c.tools.Tools[i].RawInputSchema),
			),
		})
	}

	return nil
}
