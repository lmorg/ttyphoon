package tools

import (
	"context"
	"log"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type DateTime struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&DateTime{})
}

func (t DateTime) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &DateTime{agent: agent, enabled: true}, nil
}

func (t *DateTime) Enabled() bool { return t.enabled }
func (t *DateTime) Toggle()       { t.enabled = !t.enabled }

func (t *DateTime) Name() string { return "dateTime" }
func (t *DateTime) Path() string { return "internal" }
func (t *DateTime) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}
func (t *DateTime) Description() string {
	return `Returns current date, time and timezone`
}

func (t *DateTime) Call(ctx context.Context, input string) (response string, err error) {
	log.Println("[debug] ai tool: dateTime")

	return time.Now().String(), nil
}
