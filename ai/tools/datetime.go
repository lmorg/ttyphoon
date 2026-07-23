package tools

import (
	"context"
	"log"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
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
func (t *DateTime) Description() string {
	return `Returns current date, time and timezone`
}

func (t *DateTime) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	log.Println("[debug] ai tool: dateTime")

	return time.Now().String(), nil
}
