package jq

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/lmorg/murex/utils/which"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

//go:embed jq_description.md
var description string

type Jq struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	if which.Which("jq") != "" {
		agent.ToolsAdd(&Jq{})
	}
}

func (t Jq) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Jq{agent: agent, enabled: true}, nil
}

func (t *Jq) Enabled() bool { return t.enabled }
func (t *Jq) Toggle()       { t.enabled = !t.enabled }

func (t *Jq) Description() string {
	return description
}

func (t *Jq) Name() string { return "jqScript" }
func (t *Jq) Path() string { return "internal" }

func (t *Jq) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" is running `jq` query")

	v, err := parseJqXMLInput(input)
	if err != nil {
		return fmt.Sprintf("Could not parse XML input: %v", err), nil
	}

	response, err = execJq(v.JSON, v.Query)

	return response, err
}
