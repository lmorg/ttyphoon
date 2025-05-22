package mcp

import "github.com/lmorg/mxtty/ai/agent"

func StartServerCmdLine(server, command string, params ...string) error {
	client, err := connectCmdLine(command, params...)
	if err != nil {
		return err
	}

	c := &Client{client: client}

	err = c.listTools()
	if err != nil {
		return err
	}

	for name, desc := range c.tools {
		agent.ToolsAdd(&tool{
			client:      c,
			server:      server,
			name:        name,
			description: desc,
		})
	}

	return nil
}
