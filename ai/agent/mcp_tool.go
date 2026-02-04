package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

type mcpTool struct {
	client      *mcp_client.Client
	meta        *Meta
	server      string
	name        string
	path        string
	description string
	schema      []byte
	enabled     bool
}

func (t *mcpTool) New(meta *Meta) (Tool, error) {

	return &mcpTool{
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

func (t *mcpTool) Enabled() bool { return t.enabled }
func (t *mcpTool) Toggle()       { t.enabled = !t.enabled }
func (t *mcpTool) Close() error  { return t.client.Close() }

func (t *mcpTool) Name() string { return fmt.Sprintf("mcp.%s.%s", t.server, t.name) }
func (t *mcpTool) Path() string { return t.path }
func (t *mcpTool) Description() string {
	description := t.description //+ "\nInput MUST be a JSON object with the following schema:\n" + string(t.schema)

	if debug.Trace {
		log.Printf("MCP tool '%s' description:\n%s", t.Name(), description)
	}

	return description
}

func (t *mcpTool) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("MCP tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("MCP tool '%s' response:\n%s", t.Name(), response)
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

	response, err = t.client.Call(ctx, t.name, args)
	if err != nil {
		t.meta.Renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
	}
	return response, err
}
