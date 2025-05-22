package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/types"
)

type tool struct {
	client      *Client
	meta        *agent.Meta
	server      string
	name        string
	description string
	schema      string
	enabled     bool
}

func (t *tool) New(meta *agent.Meta) (agent.Tool, error) {
	return &tool{
		client:      t.client,
		meta:        meta,
		server:      t.server,
		name:        t.name,
		description: t.description,
		enabled:     true,
	}, nil
}

func (t *tool) Enabled() bool { return t.enabled }
func (t *tool) Toggle()       { t.enabled = !t.enabled }

func (t *tool) Name() string        { return fmt.Sprintf("mcp.%s.%s", t.server, t.name) }
func (t *tool) Description() string { return t.description + "\nInput should be a JSON object." }

func (t *tool) Call(ctx context.Context, input string) (string, error) {
	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running an MCP tool: %s", t.meta.ServiceName(), t.Name()))

	// this code is a massive kludge!
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(input), &m)
	if err == nil {
		ret, err := t.client.call(ctx, t.name, m)
		if err != nil {
			return err.Error(), nil
		}
		return ret, nil
	}

	ret, err := t.client.call(ctx, t.name, map[string]interface{}{"input": input})
	if err != nil {
		return fmt.Sprintf("JSON input expected with!\n\n%v\n\nInput: %s", err, input), nil
	}
	return ret, nil
}
