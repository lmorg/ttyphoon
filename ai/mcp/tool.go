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
	path        string
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
		path:        t.path,
		description: t.description,
		enabled:     true,
	}, nil
}

func (t *tool) Enabled() bool { return t.enabled }
func (t *tool) Toggle()       { t.enabled = !t.enabled }
func (t *tool) Close() error  { return t.client.client.Close() }

func (t *tool) Name() string { return fmt.Sprintf("mcp.%s.%s", t.server, t.name) }
func (t *tool) Path() string { return t.path }
func (t *tool) Description() string {
	return t.description + "\nInput MUST be a JSON object with the following schema:\n" + t.schema
}

func (t *tool) Call(ctx context.Context, input string) (string, error) {
	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running an MCP tool: %s", t.meta.ServiceName(), t.Name()))

	// this code is a massive kludge!
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(input), &m)
	if err == nil {
		ret, err := t.client.call(ctx, t.name, m)
		if err != nil {
			t.meta.Renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
			return err.Error(), nil
		}
		return ret, nil
	}

	ret, err := t.client.call(ctx, t.name, map[string]interface{}{"input": input})
	if err != nil {
		t.meta.Renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
		return fmt.Sprintf("JSON input expected with!\n\n%v\n\nInput: %s", err, input), nil
	}
	return ret, nil
}
