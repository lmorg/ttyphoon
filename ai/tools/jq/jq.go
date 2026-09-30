package jq

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

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
func (t *Jq) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}

type jqInputT struct {
	Query string `json:"query"`
	JSON  any    `json:"json"`
}

func (t *Jq) InputType() reflect.Type { return reflect.TypeOf(jqInputT{}) }

func (t *Jq) Call(ctx context.Context, input string) (response string, err error) {
	var request jqInputT
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return fmt.Sprintf("Could not parse JSON input: %v", err), nil
	}
	return t.run(ctx, &request)
}

func (t *Jq) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*jqInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("jqScript received an invalid structured input %T", input)
	}
	return t.run(ctx, request)
}

func (t *Jq) run(ctx context.Context, input *jqInputT) (response string, err error) {
	serializedInput, _ := json.Marshal(input)
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), serializedInput)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}
	debug.Log(string(serializedInput))

	if strings.TrimSpace(input.Query) == "" {
		return "Could not parse JSON input: missing query", nil
	}
	jsonInput, err := json.Marshal(input.JSON)
	if err != nil {
		return "", err
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" is running `jq` query")

	response, err = execJq(string(jsonInput), input.Query)

	return response, err
}
