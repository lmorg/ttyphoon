package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

type tool struct {
	client      *Client
	meta        *agent.Meta
	server      string
	name        string
	path        string
	description string
	schema      []byte
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
		schema:      t.schema,
		enabled:     true,
	}, nil
}

func (t *tool) Enabled() bool { return t.enabled }
func (t *tool) Toggle()       { t.enabled = !t.enabled }
func (t *tool) Close() error  { return t.client.client.Close() }

func (t *tool) Name() string { return fmt.Sprintf("mcp.%s.%s", t.server, t.name) }
func (t *tool) Path() string { return t.path }
func (t *tool) Description() string {
	return t.description + "\nInput MUST be a JSON object with the following schema:\n" + string(t.schema)
}

func (t *tool) Call(ctx context.Context, input string) (ret string, err error) {
	if debug.Trace {
		log.Printf("MCP tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("MCP tool '%s' response:\n%s", t.Name(), ret)
			log.Printf("MCP tool '%s' error: %v", t.Name(), err)
		}()
	}

	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running an MCP tool: %s", t.meta.ServiceName(), t.Name()))

	var args map[string]any
	err = json.Unmarshal([]byte(input), &args)
	if err != nil {
		err = nil
		return "call the tool error: input must be valid json, retry tool calling with correct json", nil
	}

	ret, err = t.client.call(ctx, t.name, args)
	if err != nil {
		t.meta.Renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
	}
	return ret, err
}
